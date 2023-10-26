package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"

	"github.com/DedSecInside/gotor/api"
	"github.com/DedSecInside/gotor/internal/logger"
	"github.com/DedSecInside/gotor/linktree"
	"github.com/mgutz/ansi"
	"github.com/xuri/excelize/v2"
)

// creates a http client using socks5 proxy
func newTorClient(host string, port int) (*http.Client, error) {
	proxyStr := fmt.Sprintf("socks5://%s:%d", host, port)
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

func writeList(client *http.Client, node *linktree.Node, depth int) {
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

func downloadExcel(client *http.Client, node *linktree.Node, depth int) {
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
	url := flag.String("url", "", "URL used to initiate search. Root of the LinkTree. Required ")
	depth := flag.Int("depth", 1, "Depth of search. Defaults to 1.")

	// socks6 configuration
	socks5Host := flag.String("socks5-host", "127.0.0.1", "Host used for the SOCKS5 proxy. Defaults to localhost (127.0.0.1.)")
	socks5Port := flag.Int("socks5-port", 9050, "Port used for the SOCKS5 proxy. Defaults to 9050.")

	// server configuraiton
	serverHost := flag.String("server-host", "127.0.0.1", "Host used for the GoTor server. Defaults to localhost (127.0.0.1.)")
	serverPort := flag.Int("server-port", 8081, "Port used for the GoTor server. Defaults to 8081")

	outputFmt := flag.String("f", "list", "Determines how results will be printed. Options are list or tree")
	download := flag.Bool("d", false, "Downloads results as Excel spreadsheet. (.xlsx)")
	serve := flag.Bool("s", false, "Determines if the program will behave as an HTTP server.")
	disableTor := flag.Bool("disable-socks5", false, "Disable the use of SOCKS5 proxy.")
	flag.Parse()

	// If not serving and no URL is passed, there's nothing to do
	if !*serve && *url == "" {
		flag.CommandLine.Usage()
		return
	}

	var client *http.Client = http.DefaultClient
	var err error

	// overwrite client with tor client
	if !*disableTor {
		logger.Info("connecting to tor",
			"host", socks5Host,
			"port", socks5Port,
		)
		client, err = newTorClient(*socks5Host, *socks5Port)
		if err != nil {
			logger.Error("unable to connect to tor",
				"error", err.Error(),
			)
			return
		}
	}

	// If the server flag is passed then all other flags are ignored.
	if *serve {
		api.RunServer(client, *serverHost, *serverPort)
		return
	}

	logger.Info("starting tree with root",
		"root", *url,
		"depth", *depth,
		"output", *outputFmt,
	)

	node := linktree.NewNode(client, *url)
	switch *outputFmt {
	case "list":
		writeList(client, node, *depth)
	case "tree":
		writeTree(node, *depth)
	}

	if *download {
		downloadExcel(client, node, *depth)
	}
}
