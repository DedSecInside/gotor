package linktree

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// Node represents a single URL
type Node struct {
	URL        string  `json:"url"`
	StatusCode int     `json:"status_code"`
	Status     string  `json:"status"`
	Children   []*Node `json:"children"`
	Client     *http.Client
	Loaded     bool
	LastLoaded time.Time
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

// UpdateStatus updates the status of the URL
func (n *Node) updateStatus() {
	resp, err := n.Client.Get(n.URL)
	if err != nil {
		n.Status = "UNKNOWN"
		n.StatusCode = http.StatusInternalServerError
		return
	}
	n.Status = http.StatusText(resp.StatusCode)
	n.StatusCode = resp.StatusCode
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
		Client: client,
	}
	n.updateStatus()
	return n
}

// streams start tag tokens found within HTML content at the given link
func streamTokens(client *http.Client, link string) chan html.Token {
	tokenStream := make(chan html.Token, 100)
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
				tokenStream <- token
			}
		}
	}()
	return tokenStream
}

// filters tokens from the stream that do not pass the given tokenfilter
func filterTokens(tokenStream chan html.Token, filter *TokenFilter) chan string {
	filterStream := make(chan string)

	filterAttributes := func(token html.Token) {
		// check if token passes filter
		for _, attr := range token.Attr {
			if _, foundAttribute := filter.attributes[attr.Key]; foundAttribute {
				filterStream <- attr.Val
			}
		}
	}

	go func() {
		defer close(filterStream)
		for token := range tokenStream {
			if len(filter.tags) == 0 {
				filterStream <- token.Data
			}

			// check if token passes tag filter or tag filter is empty
			if _, foundTag := filter.tags[token.Data]; foundTag {
				// emit attributes if there is a filter, otherwise emit token
				if len(filter.attributes) > 0 {
					filterAttributes(token)
				} else {
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
func buildTree(parent *Node, depth int, childLinks chan string, wg *sync.WaitGroup) {
	for link := range childLinks {
		if isValidURL(link) {
			wg.Add(1)
			go func(parent *Node, link string, depth int) {
				defer wg.Done()
				// Do not add the link as it's own child
				if parent.URL != link {
					n := NewNode(parent.Client, link)
					parent.Children = append(parent.Children, n)
					if depth > 1 {
						depth--
						tokenStream := streamTokens(n.Client, n.URL)
						filteredStream := filterTokens(tokenStream, &TokenFilter{
							tags:       map[string]bool{"a": true},
							attributes: map[string]bool{"href": true},
						})
						buildTree(n, depth, filteredStream, wg)
					}
				}
			}(parent, link, depth)
		}
	}
}

// Load places the tree within memory.
func (n *Node) Load(depth int) {
	tokenStream := streamTokens(n.Client, n.URL)
	filteredStream := filterTokens(tokenStream, &TokenFilter{
		tags:       map[string]bool{"a": true},
		attributes: map[string]bool{"href": true},
	})

	wg := new(sync.WaitGroup)
	buildTree(n, depth, filteredStream, wg)
	wg.Wait()
	n.Loaded = true
	n.LastLoaded = time.Now().UTC()
}

// perform work on each token stream until the deapth has been reached
func crawl(client *http.Client, wg *sync.WaitGroup, linkChan <-chan string, depth int, doWork func(link string)) {
	for link := range linkChan {
		go func(currentLink string, currentDepth int) {
			defer wg.Done()
			doWork(currentLink)
			if currentDepth > 1 {
				currentDepth--
				tokenStream := streamTokens(client, currentLink)
				filteredStream := filterTokens(tokenStream, &TokenFilter{
					tags:       map[string]bool{"a": true},
					attributes: map[string]bool{"href": true},
				})
				crawl(client, wg, filteredStream, currentDepth, doWork)
			}
		}(link, depth)
		wg.Add(1)
	}
}

// Crawl traverses the children of a node without storing it in memory
func (n *Node) Crawl(depth int, work func(link string)) {
	tokenStream := streamTokens(n.Client, n.URL)
	filteredStream := filterTokens(tokenStream, &TokenFilter{
		tags:       map[string]bool{"a": true},
		attributes: map[string]bool{"href": true},
	})
	wg := new(sync.WaitGroup)
	crawl(n.Client, wg, filteredStream, depth, work)
	wg.Wait()
}
