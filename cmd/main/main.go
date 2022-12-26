package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/KingAkeem/gotor/api"
	"github.com/KingAkeem/gotor/internal/config"
	"github.com/KingAkeem/gotor/internal/logger"
	"github.com/KingAkeem/gotor/linktree"
	"github.com/mgutz/ansi"
	"github.com/xuri/excelize/v2"
)

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

func writeTerminal(client *http.Client, node *linktree.Node, depth int) {
	printStatus := func(link string) {
		n := linktree.NewNode(client, link)
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

func writeExcel(client *http.Client, node *linktree.Node, depth int) {
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
		node := linktree.NewNode(client, link)
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
	node.Crawl(depth, addRow)
	u, err := url.Parse(node.URL)
	if err != nil {
		logger.Fatal("unable to parse node URL",
			"url", node.URL,
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

func main() {
	cfg := config.GetConfig()
	var root string
	flag.StringVar(&root, "l", "", "Root used for searching. Required. (Must be a valid URL)")

	var depthInput string
	flag.StringVar(&depthInput, "d", "1", "Depth of search. Defaults to 1. (Must be an integer)")

	var host string
	flag.StringVar(&host, "h", cfg.Proxy.Host, "The host used for the SOCKS5 proxy. Defaults to localhost (127.0.0.1.)")

	var port string
	flag.StringVar(&port, "p", cfg.Proxy.Port, "The port used for the SOCKS5 proxy. Defaults to 9050.")

	var output string
	flag.StringVar(&output, "o", "terminal", "The method of output being used. Defaults to terminal. Options are terminal, excel sheet (using xlsx) or tree (a tree representation will be visually printed in text)")

	var serve bool
	flag.BoolVar(&serve, "server", false, "Determines if the program will behave as an HTTP server.")

	flag.Parse()

	// If not serving and not root is passed, there's nothing to do
	if !serve && root == "" {
		flag.CommandLine.Usage()
		return
	}

	var client *http.Client = http.DefaultClient
	var err error

	// overwrite client with tor client
	if cfg.UseTor {
		logger.Info("connecting to tor",
			"host", host,
			"port", port,
		)
		client, err = newTorClient(host, port)
		if err != nil {
			logger.Error("unable to connect to tor",
				"error", err.Error(),
			)
			return
		}
	}

	// If the server flag is passed then all other flags are ignored.
	if serve {
		serverPort := 8081
		api.RunServer(client, serverPort)
		return
	}

	depth, err := strconv.Atoi(depthInput)
	if err != nil {
		flag.CommandLine.Usage()
		return
	}

	logger.Info("starting tree with root",
		"root", root,
		"depth", depth,
		"output", output,
	)
	node := linktree.NewNode(client, root)
	switch output {
	case "terminal":
		writeTerminal(client, node, depth)
	case "excel":
		writeExcel(client, node, depth)
	case "tree":
		writeTree(node, depth)
	}
}
