package linktree

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/KingAkeem/gotor/internal/logger"
	"golang.org/x/net/html"
)

// Node represents a single URL
type Node struct {
	URL        string       `json:"url"`
	StatusCode int          `json:"status_code"`
	Status     string       `json:"status"`
	Children   []*Node      `json:"children"`
	client     *http.Client `json:"-"`
	loaded     bool         `json:"-"`
	lastLoaded time.Time    `json:"-"`
}

// PrintTree ...
func (n *Node) PrintTree() {
	fmt.Printf("%s has %d children.\n", n.URL, len(n.Children))
	for _, child := range n.Children {
		fmt.Printf("- %s\n", child.URL)
	}
	for _, child := range n.Children {
		child.PrintTree()
	}
}

// UpdateStatus gets the current status of the node's URL
func (n *Node) updateStatus() {
	logger.Debug("updating status",
		"url", n.URL,
		"current status", n.Status,
		"current status code", n.StatusCode,
	)
	if resp, err := n.client.Get(n.URL); err != nil {
		n.Status = http.StatusText(http.StatusInternalServerError)
		n.StatusCode = http.StatusInternalServerError
	} else {
		n.Status = http.StatusText(resp.StatusCode)
		n.StatusCode = resp.StatusCode
	}
	logger.Debug("status updated",
		"url", n.URL,
		"new status", n.Status,
		"new status code", n.StatusCode,
	)
}

func isValidURL(URL string) bool {
	if u, err := url.ParseRequestURI(URL); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return true
	}
	return false
}

// NewNode returns a new Link object
func NewNode(client *http.Client, URL string) *Node {
	n := &Node{
		URL:    URL,
		client: client,
	}
	n.updateStatus()
	return n
}

// streams start tag tokens found within HTML content at the given link
func streamTokens(client *http.Client, link string) chan html.Token {
	TOKEN_CHAN_SIZE := 100
	logger.Debug("streaming tokens",
		"link", link,
		"channel size", TOKEN_CHAN_SIZE,
	)
	tokenStream := make(chan html.Token, TOKEN_CHAN_SIZE)
	go func() {
		defer close(tokenStream)
		resp, err := client.Get(link)
		if err != nil {
			log.Println(err)
			return
		}
		tokenizer := html.NewTokenizer(resp.Body)
		for {
			tokenType := tokenizer.Next()
			switch tokenType {
			case html.ErrorToken:
				err := tokenizer.Err()
				if err != io.EOF {
					log.Println(err)
				}
				return
			case html.StartTagToken:
				token := tokenizer.Token()
				logger.Debug("queue token stream",
					"token", token,
				)
				tokenStream <- token
			}
		}
	}()
	return tokenStream
}

// filters tokens from the stream that do not pass the given tokenfilter
func filterTokens(tokenStream chan html.Token, filter *TokenFilter) chan string {
	FILTER_CHAN_SIZE := 10
	logger.Debug("Filtering tokens",
		"filter", filter,
		"channel size", FILTER_CHAN_SIZE,
	)
	filterStream := make(chan string, FILTER_CHAN_SIZE)

	filterAttributes := func(token html.Token) {
		// check if token passes filter
		for _, attr := range token.Attr {
			if _, foundAttribute := filter.attributes[attr.Key]; foundAttribute {
				logger.Debug("queue filter stream",
					"data", attr.Val,
				)
				filterStream <- attr.Val
			}
		}
	}

	go func() {
		defer close(filterStream)
		for token := range tokenStream {
			logger.Debug("dequeue token stream",
				"token", token,
			)
			if len(filter.tags) == 0 {
				logger.Debug("queue filter stream",
					"data", token.Data,
				)
				filterStream <- token.Data
			}

			// check if token passes tag filter or tag filter is empty
			if _, foundTag := filter.tags[token.Data]; foundTag {
				// emit attributes if there is a filter, otherwise emit token
				if len(filter.attributes) > 0 {
					filterAttributes(token)
				} else {
					logger.Debug("queue filter stream",
						"data", token.Data,
					)
					filterStream <- token.Data
				}
			}
		}
	}()

	return filterStream
}

// TokenFilter determines which tokens will be filtered from a stream,
// 1. There are zero to many attributes per tag.
// if the tag is included then those tags will be used (e.g. all anchor tags)
// if the attribute is included then those attributes will be used (e.g. all href attributes)
// if both are specified then the combination will be used (e.g. all href attributes within anchor tags only)
// if neither is specified then all tokens will be used (e.g. all tags found)
type TokenFilter struct {
	tags       map[string]bool
	attributes map[string]bool
}

// builds a tree for the parent node using the incoming links as children (repeated until depth has been exhausted)
func buildTree(parent *Node, depth int, childLinks chan string, wg *sync.WaitGroup, filter *TokenFilter) {
	logger.Debug("building tree",
		"parent", parent,
		"children", childLinks,
		"filter", filter,
	)
	for link := range childLinks {
		if isValidURL(link) {
			wg.Add(1)
			go func(parent *Node, link string, depth int) {
				defer wg.Done()
				// Do not add the link as it's own child
				if parent.URL != link {
					n := NewNode(parent.client, link)
					parent.Children = append(parent.Children, n)
					if depth > 1 {
						depth--
						tokenStream := streamTokens(n.client, n.URL)
						filteredStream := filterTokens(tokenStream, filter)
						buildTree(n, depth, filteredStream, wg, filter)
					}
				}
			}(parent, link, depth)
		}
	}
}

// Load places the tree within memory.
func (n *Node) Load(depth int) {
	logger.Debug("attempting to load node",
		"node", n,
	)
	tokenStream := streamTokens(n.client, n.URL)
	filter := &TokenFilter{
		tags:       map[string]bool{"a": true},
		attributes: map[string]bool{"href": true},
	}
	filteredStream := filterTokens(tokenStream, filter)
	wg := new(sync.WaitGroup)
	buildTree(n, depth, filteredStream, wg, filter)
	wg.Wait()
	n.loaded = true
	n.lastLoaded = time.Now().UTC()
	logger.Debug("loaded node",
		"node", n,
	)
}

// perform work on each token stream until the deapth has been reached
func crawl(client *http.Client, wg *sync.WaitGroup, linkChan <-chan string, depth int, filter *TokenFilter, doWork func(link string)) {
	for link := range linkChan {
		go func(currentLink string, currentDepth int) {
			defer wg.Done()
			doWork(currentLink)
			if currentDepth > 1 {
				currentDepth--
				tokenStream := streamTokens(client, currentLink)
				filteredStream := filterTokens(tokenStream, filter)
				crawl(client, wg, filteredStream, currentDepth, filter, doWork)
			}
		}(link, depth)
		wg.Add(1)
	}
}

// Crawl traverses the children of a node without storing it in memory
func (n *Node) Crawl(depth int, work func(link string)) {
	tokenStream := streamTokens(n.client, n.URL)
	filter := &TokenFilter{
		tags:       map[string]bool{"a": true},
		attributes: map[string]bool{"href": true},
	}
	filteredStream := filterTokens(tokenStream, filter)
	wg := new(sync.WaitGroup)
	crawl(n.client, wg, filteredStream, depth, filter, work)
	wg.Wait()
}
