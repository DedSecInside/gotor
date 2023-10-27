// This package contains functionality to interact with linktrees
package linktree

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/DedSecInside/gotor/internal/logger"
	"github.com/mgutz/ansi"
	"github.com/xuri/excelize/v2"
)

// Node represents a single node of a LinkTree
type Node struct {
	URL        string       `json:"url"`
	StatusCode int          `json:"status_code"`
	Status     string       `json:"status"`
	Children   []*Node      `json:"children"`
	client     *http.Client `json:"-"`
	loaded     bool         `json:"-"`
	lastLoaded time.Time    `json:"-"`
}

func (n *Node) DownloadExcel(depth int) {
	f := excelize.NewFile()
	err := f.SetCellStr(f.GetSheetName(0), "A1", "Link")
	if err != nil {
		logger.Fatal("unable to set sheet name", "error", err)
	}

	err = f.SetCellStr(f.GetSheetName(0), "B1", "Status")
	if err != nil {
		logger.Fatal(err.Error())
	}

	row := 2
	addRow := func(link string) {
		node := NewNode(n.client, link)
		linkCell := fmt.Sprintf("A%d", row)
		statusCell := fmt.Sprintf("B%d", row)
		err = f.SetCellStr(f.GetSheetName(0), linkCell, node.URL)
		if err != nil {
			logger.Fatal(err.Error())
		}

		err = f.SetCellStr(f.GetSheetName(0), statusCell, fmt.Sprintf("%d %s", node.StatusCode, node.Status))
		if err != nil {
			logger.Fatal(err.Error())
		}
		row++
	}
	n.Crawl(depth, addRow)
	u, err := url.Parse(n.URL)
	if err != nil {
		logger.Fatal("unable to parse node URL",
			"url", n.URL,
			"error", err.Error(),
		)
	}

	filename := fmt.Sprintf("%s_depth_%d.xlsx", u.Hostname(), depth)
	err = f.SaveAs(filename)
	if err != nil {
		logger.Fatal("unable to save excel file",
			"filename", filename,
			"error", err.Error(),
		)
	}
}

// PrintTree prints a visual representation of a tree using the std terminal
func (n *Node) PrintTree() {
	fmt.Printf("%s has %d children.\n", n.URL, len(n.Children))
	for _, child := range n.Children {
		fmt.Printf("- %s\n", child.URL)
	}
	for _, child := range n.Children {
		child.PrintTree()
	}
}

func (n *Node) PrintList(depth int) {
	printStatus := func(link string) {
		n := NewNode(n.client, link)
		markError := ansi.ColorFunc("red")
		markSuccess := ansi.ColorFunc("green")
		if n.StatusCode != 200 {
			fmt.Printf("Link: %20s Status: %d %s\n", n.URL, n.StatusCode, markError(n.Status))
		} else {
			fmt.Printf("Link: %20s Status: %d %s\n", n.URL, n.StatusCode, markSuccess(n.Status))
		}
	}
	n.Crawl(depth, printStatus)
}

// UpdateStatus gets the current status of the node's URL
func (n *Node) updateStatus() error {
	logger.Debug("updating status",
		"url", n.URL,
		"current status", n.Status,
		"current status code", n.StatusCode,
	)
	resp, err := n.client.Get(n.URL)
	if err != nil {
		logger.Warn("unable to get url", "url", n, "error", err)
		return err
	}
	defer resp.Body.Close()

	n.Status = http.StatusText(resp.StatusCode)
	n.StatusCode = resp.StatusCode
	logger.Debug("status updated",
		"url", n.URL,
		"new status", n.Status,
		"new status code", n.StatusCode,
	)

	return nil
}

func isValidURL(URL string) bool {
	if u, err := url.ParseRequestURI(URL); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return true
	}
	return false
}

// NewNode returns a new node object after setting it's status, this is the primary mode of creating new nodes
func NewNode(client *http.Client, URL string) *Node {
	n := &Node{
		URL:    URL,
		client: client,
	}

	err := n.updateStatus()
	if err != nil {
		n.Status = http.StatusText(http.StatusInternalServerError)
		n.StatusCode = http.StatusInternalServerError
	}

	return n
}

// builds a tree for the parent node using the incoming links as children (repeated until depth has been exhausted)
func buildTree(parent *Node, depth int, childLinks chan string, wg *sync.WaitGroup, filter *tokenFilter) {
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

// Load constructs a LinkTree using the given depth specified
func (n *Node) Load(depth int) {
	logger.Debug("attempting to load node",
		"node", n,
	)
	tokenStream := streamTokens(n.client, n.URL)
	filter := &tokenFilter{
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

// perform work on each token stream until the specified depth has been reached
func crawl(client *http.Client, wg *sync.WaitGroup, linkChan <-chan string, depth int, filter *tokenFilter, doWork func(link string)) {
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
	fmt.Println(n.client, n.URL)
	tokenStream := streamTokens(n.client, n.URL)
	filter := &tokenFilter{
		tags:       map[string]bool{"a": true},
		attributes: map[string]bool{"href": true},
	}
	filteredStream := filterTokens(tokenStream, filter)
	wg := new(sync.WaitGroup)
	crawl(n.client, wg, filteredStream, depth, filter, work)
	wg.Wait()
}
