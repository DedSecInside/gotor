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

	"github.com/KingAkeem/gotor/linktree"
	"github.com/gorilla/mux"
	"github.com/mgutz/ansi"
	"github.com/xuri/excelize/v2"
	"golang.org/x/net/html"
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

// streams the child nodes of a link
func getTorIP(client *http.Client) (string, error) {
	resp, err := client.Get("https://check.torproject.org/")
	if err != nil {
		return "", err
	}
	tokenizer := html.NewTokenizer(resp.Body)
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			err := tokenizer.Err()
			if err != io.EOF {
				return "", err
			}
			return "", nil
		case html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == "strong" {
				tokenizer.Next()
				ipToken := tokenizer.Token()
				return ipToken.Data, nil
			}
		}
	}
}

func writeTerminal(manager *linktree.NodeManager, root string, depth int) {
	printStatus := func(link string) {
		n := linktree.NewNode(manager, link)
		markError := ansi.ColorFunc("red")
		markSuccess := ansi.ColorFunc("green")
		if n.StatusCode != 200 {
			fmt.Printf("Link: %20s Status: %d %s\n", n.URL, n.StatusCode, markError(n.Status))
		} else {
			fmt.Printf("Link: %20s Status: %d %s\n", n.URL, n.StatusCode, markSuccess(n.Status))
		}
	}
	manager.Crawl(root, depth, printStatus)
}

func writeExcel(manager *linktree.NodeManager, root string, depth int) {
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
		node := linktree.NewNode(manager, link)
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
	manager.Crawl(root, depth, addRow)
	u, err := url.Parse(root)
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

	router.HandleFunc("/ip", func(w http.ResponseWriter, r *http.Request) {
		ip, err := getTorIP(client)
		if err != nil {
			_, err = w.Write([]byte(err.Error()))
			if err != nil {
				logErr(err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write([]byte(ip))
		if err != nil {
			logErr(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}).Methods(http.MethodGet)
	router.HandleFunc("/children", func(w http.ResponseWriter, r *http.Request) {
		queryMap := r.URL.Query()
		// decode depth
		depthInput := queryMap.Get("depth")
		depth, err := strconv.Atoi(depthInput)
		if err != nil {
			_, err := w.Write([]byte("Invalid depth. Must be an integer."))
			if err != nil {
				logErr(err)
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		link := queryMap.Get("link")
		logInfo(fmt.Sprintf("processing link %s at a depth of %d", link, depth))
		manager := linktree.NewNodeManager(client)
		node := manager.LoadNode(link, depth)
		logInfo(fmt.Sprintf("Tree built for %s at depth %d", node.URL, depth))
		err = json.NewEncoder(w).Encode(node)
		if err != nil {
			logErr(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}).Methods(http.MethodGet)

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

	manager := linktree.NewNodeManager(client)
	switch output {
	case "terminal":
		writeTerminal(manager, root, depth)
	case "excel":
		writeExcel(manager, root, depth)
	case "tree":
		node := manager.LoadNode(root, depth)
		node.PrintTree()
	}
}
