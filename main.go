package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/mgutz/ansi"
	"golang.org/x/net/html"
)

// creates a http client using socks5 proxy
func createTorClient(host, port string) (*http.Client, error) {
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
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						if u, err := url.ParseRequestURI(attr.Val); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
							linkChan <- attr.Val
						}
					}
				}
			}
		}
	}()
	return linkChan
}

// returns a formatted string of the HTTP status for the link
func getStatus(client *http.Client, link string) (string, error) {
	markError := ansi.ColorFunc("red")
	markSuccess := ansi.ColorFunc("green")
	fmt.Printf("Checking %s\n ", ansi.Color(link, "blue"))
	resp, err := client.Get(link)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	status := fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	if resp.StatusCode != 200 {
		status = fmt.Sprintf("Link: %20s Status: %15s", link, markError((status)))
	} else {
		status = fmt.Sprintf("Link: %20s Status: %15s", link, markSuccess((status)))
	}
	return status, nil
}

// streams the status of the links from the channel until the depth has reached 0
func streamStatus(client *http.Client, linkChan <-chan string, depth int, wg *sync.WaitGroup) {
	for link := range linkChan {
		go func(l string) {
			defer wg.Done()
			status, err := getStatus(client, l)
			if err != nil {
				log.Fatal(err)
				return
			}
			fmt.Println(status)
			if depth > 0 {
				depth--
				subLinkChan := streamLinks(client, l)
				streamStatus(client, subLinkChan, depth, wg)
			}
		}(link)
		wg.Add(1)
	}
}

func main() {
	var link string
	var host string
	var port string
	var depthInput string
	flag.StringVar(&link, "l", "", "Root used for searching. Required. (Must be a valid URL)")
	flag.StringVar(&depthInput, "d", "1", "Depth of search. Defaults to 1. (Must be an integer)")
	flag.StringVar(&host, "h", "127.0.0.1", "The address used for the SOCKS5 proxy. Defaults to localhost (127.0.0.1.)")
	flag.StringVar(&port, "p", "9050", "The port used for the SOCKS5 proxy. Defaults to 9050.")
	flag.Parse()
	if link == "" {
		flag.CommandLine.Usage()
		return
	}

	depth, err := strconv.Atoi(depthInput)
	if err != nil {
		flag.CommandLine.Usage()
		return
	}

	client, err := createTorClient(host, port)
	linkChan := streamLinks(client, link)
	wg := new(sync.WaitGroup)
	streamStatus(client, linkChan, depth, wg)
	wg.Wait()
}
