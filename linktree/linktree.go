package linktree

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"

	"golang.org/x/net/html"
)

// Node represents a single URL
type Node struct {
	manager    *NodeManager
	URL        string  `json:"url"`
	StatusCode int     `json:"status_code"`
	Status     string  `json:"status"`
	Children   []*Node `json:"children"`
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
	resp, err := n.manager.client.Get(n.URL)
	if err != nil {
		n.Status = "UNKNOWN"
		n.StatusCode = http.StatusInternalServerError
		return
	}
	n.Status = http.StatusText(resp.StatusCode)
	n.StatusCode = resp.StatusCode
}

// NewNode returns a new Link object
func NewNode(manager *NodeManager, URL string) *Node {
	n := &Node{
		URL:     URL,
		manager: manager,
	}
	n.updateStatus()
	return n
}

// NodeManager ...
type NodeManager struct {
	client *http.Client
	wg     *sync.WaitGroup
}

func isValidURL(URL string) bool {
	if u, err := url.ParseRequestURI(URL); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return true
	}
	return false
}

// streams the child nodes of a link
func (m *NodeManager) streamUrls(link string) chan string {
	linkChan := make(chan string, 100)
	go func() {
		defer close(linkChan)
		resp, err := m.client.Get(link)
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
				if token.Data == "a" {
					for _, attr := range token.Attr {
						if attr.Key == "href" {
							if isValidURL(attr.Val) {
								linkChan <- attr.Val
							}
						}
					}
				}
			}
		}
	}()
	return linkChan
}

// LoadNode ...
func (m *NodeManager) LoadNode(root string, depth int) *Node {
	node := NewNode(m, root)
	rootChan := m.streamUrls(root)
	m.buildTree(rootChan, depth, node)
	m.wg.Wait()
	return node
}

// streams the status of the links from the channel until the depth has reached 0
func (m *NodeManager) crawl(linkChan <-chan string, depth int, doWork func(link string)) {
	for link := range linkChan {
		go func(l string) {
			defer m.wg.Done()
			doWork(l)
			if depth > 1 {
				depth--
				subLinkChan := m.streamUrls(l)
				m.crawl(subLinkChan, depth, doWork)
			}
		}(link)
		m.wg.Add(1)
	}
}

// builds a tree from the given link channel
func (m *NodeManager) buildTree(linkChan <-chan string, depth int, node *Node) {
	for link := range linkChan {
		go func(l string, node *Node) {
			defer m.wg.Done()
			// Do not add the link as it's own child
			if node.URL != l {
				n := NewNode(m, l)
				node.Children = append(node.Children, n)
				if depth > 1 {
					depth--
					subLinkChan := m.streamUrls(l)
					m.buildTree(subLinkChan, depth, n)
				}
			}
		}(link, node)
		m.wg.Add(1)
	}
}

// NewNodeManager ...
func NewNodeManager(client *http.Client) *NodeManager {
	return &NodeManager{
		client: client,
		wg:     new(sync.WaitGroup),
	}
}

// Crawl ...
func (m *NodeManager) Crawl(root string, depth int, work func(link string)) {
	rootChan := m.streamUrls(root)
	m.crawl(rootChan, depth, work)
	m.wg.Wait()
}
