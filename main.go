package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mgutz/ansi"
	"github.com/xuri/excelize/v2"
	"golang.org/x/net/html"
)

// LinkNode ...
type LinkNode struct {
	client     *http.Client
	URL        string      `json:"url"`
	StatusCode int         `json:"status_code"`
	Status     string      `json:"status"`
	Children   []*LinkNode `json:"children"`
}

func newNode(client *http.Client, link string) *LinkNode {
	l := &LinkNode{
		URL:    link,
		client: client,
	}
	l.UpdateStatus()
	return l
}

// UpdateStatus ...
func (l *LinkNode) UpdateStatus() {
	fmt.Printf("Checking %s\n ", ansi.Color(l.URL, "blue"))
	resp, err := l.client.Get(l.URL)
	if err != nil {
		log.Fatal(err)
		return
	}
	l.Status = http.StatusText(resp.StatusCode)
	l.StatusCode = resp.StatusCode
}

// creates a http client using socks5 proxy
func newTorClient(host, port string) (*http.Client, error) {
	proxyStr := fmt.Sprintf("socks5://%s:%s", host, port)
	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	return &http.Client{
		Transport: transport,
	}, nil
}

// streams the child nodes of a link
func streamLinks(client *http.Client, link string) chan string {
	linkChan := make(chan string, 10)
	go func() {
		resp, err := client.Get(link)
		if err != nil {
			log.Fatal(err)
			return
		}
		tokenizer := html.NewTokenizer(resp.Body)
		defer close(linkChan)
		for {
			tokenType := tokenizer.Next()
			switch tokenType {
			case html.ErrorToken:
				err := tokenizer.Err()
				if err != io.EOF {
					log.Fatal(err)
				}
				return
			case html.StartTagToken:
				token := tokenizer.Token()
				if token.Data == "a" {
					for _, attr := range token.Attr {
						if attr.Key == "href" {
							if u, err := url.ParseRequestURI(attr.Val); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
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

// streams the status of the links from the channel until the depth has reached 0
func crawl(client *http.Client, linkChan <-chan string, depth int, wg *sync.WaitGroup, doWork func(link string)) {
	for link := range linkChan {
		go func(l string) {
			defer wg.Done()
			doWork(l)
			if depth > 1 {
				depth--
				subLinkChan := streamLinks(client, l)
				crawl(client, subLinkChan, depth, wg, doWork)
			}
		}(link)
		wg.Add(1)
	}
}

// streams the status of the links from the channel until the depth has reached 0
func buildTree(linkChan <-chan string, depth int, wg *sync.WaitGroup, node *LinkNode) {
	for link := range linkChan {
		go func(l string, node *LinkNode) {
			defer wg.Done()
			// Do not add the link as it's own child
			if node.URL != l {
				n := newNode(node.client, l)
				node.Children = append(node.Children, n)
				if depth > 1 {
					depth--
					subLinkChan := streamLinks(node.client, l)
					buildTree(subLinkChan, depth, wg, n)
				}
			}
		}(link, node)
		wg.Add(1)
	}
}

func printTree(n *LinkNode) {
	fmt.Printf("%s has %d children.\n", n.URL, len(n.Children))
	for _, child := range n.Children {
		fmt.Printf("- %s\n", child.URL)
	}
	for _, child := range n.Children {
		printTree(child)
	}
}

// Crawler ...
type Crawler struct {
	client   *http.Client
	linkChan chan string
	wg       *sync.WaitGroup
}

func newCrawler(client *http.Client, linkChan chan string) *Crawler {
	return &Crawler{
		client:   client,
		linkChan: linkChan,
		wg:       new(sync.WaitGroup),
	}
}

// Crawl ...
func (c *Crawler) Crawl(work func(link string), depth int) {
	crawl(c.client, c.linkChan, depth, c.wg, work)
	c.wg.Wait()
}

func writeTerminal(crawler *Crawler, depth int) {
	printStatus := func(link string) {
		l := newNode(crawler.client, link)
		markError := ansi.ColorFunc("red")
		markSuccess := ansi.ColorFunc("green")
		if l.StatusCode != 200 {
			fmt.Printf("Link: %20s Status: %d %s\n", l.URL, l.StatusCode, markError(l.Status))
		} else {
			fmt.Printf("Link: %20s Status: %d %s\n", l.URL, l.StatusCode, markSuccess(l.Status))
		}
	}
	crawler.Crawl(printStatus, depth)
}

func writeExcel(crawler *Crawler, depth int, filename string) {
	f := excelize.NewFile()
	err := f.SetCellStr(f.GetSheetName(0), "A1", "Link")
	if err != nil {
		log.Fatal(err)
		return
	}
	err = f.SetCellStr(f.GetSheetName(0), "B1", "Status")
	if err != nil {
		log.Fatal(err)
		return
	}
	row := 2
	addRow := func(link string) {
		node := newNode(crawler.client, link)
		linkCell := fmt.Sprintf("A%d", row)
		statusCell := fmt.Sprintf("B%d", row)
		err = f.SetCellStr(f.GetSheetName(0), linkCell, node.URL)
		if err != nil {
			log.Fatal(err)
			return
		}
		err = f.SetCellStr(f.GetSheetName(0), statusCell, fmt.Sprintf("%d %s", node.StatusCode, node.Status))
		if err != nil {
			log.Fatal(err)
			return
		}
		row++
	}
	crawler.Crawl(addRow, depth)
	err = f.SaveAs(filename)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func runServer(host, port string) {
	router := mux.NewRouter()

	client, err := newTorClient(host, port)
	if err != nil {
		log.Fatal(err)
		return
	}

	router.HandleFunc("/children", func(w http.ResponseWriter, r *http.Request) {
		queryMap := r.URL.Query()
		// decode depth
		depthInput := queryMap.Get("depth")
		depth, err := strconv.Atoi(depthInput)
		if err != nil {
			_, err := w.Write([]byte("Invalid depth. Must be an integer."))
			if err != nil {
				log.Println(err.Error())
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		link := queryMap.Get("link")
		log.Printf("processing link %s at a depth of %d\n", link, depth)
		linkChan := streamLinks(client, link)
		crawler := newCrawler(client, linkChan)
		node := newNode(client, link)
		buildTree(linkChan, depth, crawler.wg, node)
		crawler.wg.Wait()

		log.Printf("tree built: %+v\n", node)
		err = json.NewEncoder(w).Encode(node)
		if err != nil {
			log.Printf("Unable to encode the link node. Error: %+v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}).Methods(http.MethodGet)

	log.Println("Listening on port 8081")
	err = http.ListenAndServe(":8081", router)
	if err != nil {
		log.Println(err)
		return
	}
}

func main() {
	var root string
	var host string
	var port string
	var depthInput string
	var output string
	var serve bool
	flag.StringVar(&root, "l", "", "Root used for searching. Required. (Must be a valid URL)")
	flag.StringVar(&depthInput, "d", "1", "Depth of search. Defaults to 1. (Must be an integer)")
	flag.StringVar(&host, "h", "127.0.0.1", "The host used for the SOCKS5 proxy. Defaults to localhost (127.0.0.1.)")
	flag.StringVar(&port, "p", "9050", "The port used for the SOCKS5 proxy. Defaults to 9050.")
	flag.StringVar(&output, "o", "terminal", "The method of output being used. Defaults to terminal.")
	flag.BoolVar(&serve, "server", false, "Determines if the program will behave as an HTTP server.")
	flag.Parse()

	// If the server flag is passed then all other flags are ignored.
	if serve {
		runServer(host, port)
		return
	}

	if root == "" {
		flag.CommandLine.Usage()
		return
	}

	depth, err := strconv.Atoi(depthInput)
	if err != nil {
		flag.CommandLine.Usage()
		return
	}

	client, err := newTorClient(host, port)
	if err != nil {
		log.Fatal(err)
		return
	}

	linkChan := streamLinks(client, root)
	crawler := newCrawler(client, linkChan)
	switch output {
	case "terminal":
		writeTerminal(crawler, depth)
	case "excel":
		u, err := url.Parse(root)
		if err != nil {
			log.Fatal(err)
			return
		}
		filename := fmt.Sprintf("%s_depth_%d.xlsx", u.Hostname(), depth)
		writeExcel(crawler, depth, filename)
	case "tree":
		r := newNode(client, root)
		buildTree(linkChan, depth, crawler.wg, r)
		crawler.wg.Wait()
		printTree(r)
	}
}
