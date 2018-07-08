package gobot

import (
	"log"
	"net/http"
	urllib "net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type Client interface {
	Get(string) (*http.Response, error)
}

type dualClient struct {
	regClient *http.Client
	torClient *http.Client
}

// Establishes tor connection for tcp
func newTorClient(addr string, port string, timeout int) *dualClient {
	var torProxy = "socks5://" + addr + ":" + port
	torProxyURL, err := urllib.Parse(torProxy)
	if err != nil {
		log.Fatal("Error parsing Tor Proxy:", err)
	}
	torTransport := &http.Transport{Proxy: http.ProxyURL(torProxyURL)}

	// Creating both clients for regular browsing and Tor
	torc := http.Client{Transport: torTransport, Timeout: time.Second * time.Duration(timeout)}
	regc := http.Client{Timeout: time.Second * time.Duration(timeout)}
	return &dualClient{regClient: &regc, torClient: &torc}

}

func (d *dualClient) Get(url string) (*http.Response, error) {
	if strings.Contains(url, ".onion") {
		return d.torClient.Get(url)
	} else {
		return d.regClient.Get(url)
	}
}

// Sends string to channel that contains a message that explains the
// status of the url passed
func checkURL(client Client, url string) (bool, error) {
	var err error
	var isURLAlive bool

	resp, err := client.Get(url)
	if err == nil && resp.StatusCode < 400 {
		isURLAlive = true
	} else {
		isURLAlive = false
	}
	return isURLAlive, err
}

// Parses html attributes to find urls
func parseAttrs(attributes []html.Attribute) []string {
	var foundUrls = make([]string, 0)
	for i := 0; i < len(attributes); i++ {
		if attributes[i].Key == "href" {
			foundUrls = append(foundUrls, attributes[i].Val)
		}
	}

	return foundUrls
}

// GetLinks returns a map that contains the links as keys and their statuses as values
func GetLinks(searchURL string, addr string, port string, timeout int) (map[string]bool, error) {
	// Creating new Tor connection
	client := newTorClient(addr, port, timeout)
	resp, err := client.Get(searchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Begin parsing HTML
	tokenizer := html.NewTokenizer(resp.Body)
	var urls []string
	for notEnd := true; notEnd; {
		currentTokenType := tokenizer.Next()
		switch {
		case currentTokenType == html.ErrorToken:
			notEnd = false
		case currentTokenType == html.StartTagToken:
			token := tokenizer.Token()
			// If link tag is found, append it to slice
			if token.Data == "a" {
				found := parseAttrs(token.Attr)
				urls = append(urls, found...)
			}
		}
	}

	if len(urls) == 0 {
		return nil, nil
	}

	// Check all links and assign their status
	linksWithStatus := make(map[string]bool, len(urls))
	for _, url := range urls {
		linksWithStatus[url], err = checkURL(client, url)
		if err != nil {
			panic(err)
		}
	}

	return linksWithStatus, nil
}
