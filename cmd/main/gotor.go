package main

import (
	"flag"
	"log"
	"time"

	"github.com/DedSecInside/gotor/api"
	"github.com/DedSecInside/gotor/internal/netx"
	"github.com/DedSecInside/gotor/pkg/linktree"
)

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
	client, err := netx.NewHTTPClient(netx.ClientOpts{
		UseTor:       !*disableTor,
		SocksHost:    *socks5Host,
		SocksPort:    *socks5Port,
		ReqTimeout:   30 * time.Second,
		DialTimeout:  10 * time.Second,
		MaxIdleConns: 10,
	})

	if err != nil {
		log.Fatalf("Unable to connect Tor. Error: %+v\n", err)
		return
	}

	// If the server flag is passed then all other flags are ignored.
	if *serve {
		server := api.New(client, *serverHost, *serverPort)
		server.Run()
		return
	}

	log.Printf("Building tree - Root: %s Depth %d Format %s\n", *url, *depth, *outputFmt)
	node := linktree.NewNode(client, *url)
	switch *outputFmt {
	case "list":
		node.PrintList(*depth)
	case "tree":
		node.Load(*depth)
		node.PrintTree()
	}

	if *download {
		node.DownloadExcel(*depth)
	}
}
