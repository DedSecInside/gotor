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

// create a simple tor client, this can be modified to allow the user to
// set their address and port at some point
func createTorClient() (*http.Client, error) {
	proxyStr := "socks5://127.0.0.1:9050"
	proxyUrl, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyUrl),
	}
	return &http.Client{
		Transport: transport,
	}, nil
}

// parses the links from a reader
func parseLinks(r io.Reader) ([]string, error) {
	links := make([]string, 0)
	tokenizer := html.NewTokenizer(r)

	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			return links, tokenizer.Err()
		case html.StartTagToken:
			token := tokenizer.Token()
			for _, attr := range token.Attr {
				if attr.Key == "href" {
					if u, err := url.ParseRequestURI(attr.Val); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
						links = append(links, attr.Val)
					}
				}
			}
		}
	}
}

// parses the links from a reader
func streamLinks(r io.Reader) chan string {
	linkChan := make(chan string, 10)
	tokenizer := html.NewTokenizer(r)
	go func() {
		for {
			tokenType := tokenizer.Next()
			switch tokenType {
			case html.ErrorToken:
				err := tokenizer.Err()
				if err != io.EOF {
					log.Fatal(err)
				}
				close(linkChan)
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
		status = fmt.Sprintf("Link: %4s Status: %4s", link, markError((status)))
	} else {
		status = fmt.Sprintf("Link: %4s Status: %4s", link, markSuccess((status)))
	}
	return status, nil
}

func main() {
	var link string
	var depthInput string
	flag.StringVar(&link, "l", "", "Root used for searching. Requred.")
	flag.StringVar(&depthInput, "d", "1", "Depth of search. Defaults to 1.")
	flag.Parse()
	if link == "" {
		log.Fatal("-l (link) argument is required.")
		return
	}

	depth, err := strconv.Atoi(depthInput)
	if err != nil {
		log.Fatal("Invalid depth found. Depth: ", depth)
		return
	}

	client, err := createTorClient()
	if err != nil {
		log.Fatal(err)
		return
	}
	resp, err := client.Get(link)
	if err != nil {
		log.Fatal(err)
		return
	}

	linkChan := streamLinks(resp.Body)
	wg := new(sync.WaitGroup)
	for link := range linkChan {
		go func(l string) {
			defer wg.Done()
			status, err := getStatus(client, l)
			if err != nil {
				log.Fatal(err)
				return
			}
			fmt.Println(status)
		}(link)
		wg.Add(1)
	}
	wg.Wait()
}
