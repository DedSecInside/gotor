package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/KingAkeem/gotor/api"
	"github.com/KingAkeem/gotor/linktree"
	"github.com/gorilla/mux"
	"github.com/mgutz/ansi"
	"github.com/xuri/excelize/v2"
)

func logInfo(msg string) {
	log.Println(ansi.Color(msg, "blue"))
}

func logErr(err error) {
	log.Println(ansi.Color(err.Error(), "red"))
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

func writeTree(node *linktree.Node, depth int) {
	node.Load(depth)
	node.PrintTree()
}

func writeTerminal(node *linktree.Node, depth int) {
	printStatus := func(link string) {
		n := linktree.NewNode(node.Client, link)
		markError := ansi.ColorFunc("red")
		markSuccess := ansi.ColorFunc("green")
		if n.StatusCode != 200 {
			fmt.Printf("Link: %20s Status: %d %s\n", n.URL, n.StatusCode, markError(n.Status))
		} else {
			fmt.Printf("Link: %20s Status: %d %s\n", n.URL, n.StatusCode, markSuccess(n.Status))
		}
	}
	node.Crawl(depth, printStatus)
}

func writeExcel(node *linktree.Node, depth int) {
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
		node := linktree.NewNode(node.Client, link)
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
	node.Crawl(depth, addRow)
	u, err := url.Parse(node.URL)
	if err != nil {
		log.Fatal(err)
		return
	}
	filename := fmt.Sprintf("%s_depth_%d.xlsx", u.Hostname(), depth)
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

	router.HandleFunc("/ip", api.GetIP(client)).Methods(http.MethodGet)
	router.HandleFunc("/emails", api.GetEmails(client)).Methods(http.MethodGet)
	router.HandleFunc("/phone", api.GetPhoneNumbers(client)).Methods(http.MethodGet)
	router.HandleFunc("/tree", api.GetTreeNode(client)).Methods(http.MethodGet)
	router.HandleFunc("/content", api.GetWebsiteContent(client)).Methods(http.MethodGet)

	logInfo("Listening on port 8081")
	err = http.ListenAndServe(":8081", router)
	if err != nil {
		logErr(err)
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

	node := linktree.NewNode(client, root)
	switch output {
	case "terminal":
		writeTerminal(node, depth)
	case "excel":
		writeExcel(node, depth)
	case "tree":
		writeTree(node, depth)
	}
}
