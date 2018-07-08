package geturls

import (
	"fmt"
	"log"
	"net/http"
	urllib "net/url"
	"regexp"
	"time"

	"github.com/mgutz/ansi"
	"golang.org/x/net/html"
)

// Checks for valid .onion domain names
func validOnionURL(url string) bool {
	pattern := `^https?:\/\/(www\.)?([a-z,A-Z,0-9]*)\.onion/?(.*)`
	re := regexp.MustCompile(pattern)
	return re.Match([]byte(url))
}

type netClient interface {
	Get(string) (*http.Response, error)
	Head(string) (*http.Response, error)
}

// Sends string to channel that contains a message that explains the
// status of the url passed
func checkURL(client netClient, url string, ch chan<- string) {
	red := ansi.ColorFunc("red")
	resp, err := client.Get(url)
	if err == nil && resp.StatusCode < 400 {
		ch <- fmt.Sprintf("%v is reachable.\n", url)
	} else {
		ch <- red(fmt.Sprintf("%v is not reachable.\n", url))
	}
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

// Establishes tor connection for tcp
func newTorConn(addr string, port string, timeout int) *http.Client {
	var torProxy = "socks5://" + addr + ":" + port
	torProxyURL, err := urllib.Parse(torProxy)
	if err != nil {
		log.Fatal("Error parsing URL: ", err)
	}
	torTransport := &http.Transport{Proxy: http.ProxyURL(torProxyURL)}
	return &http.Client{Transport: torTransport, Timeout: time.Second * time.Duration(timeout)}
}

func GetLinks(searchURL string, addr string, port string, timeout int) ([]string, error) {
	var client *http.Client
	if port == "" && addr == "" {
		client = &http.Client{Timeout: time.Second * time.Duration(timeout)}
	} else {
		client = newTorConn(addr, port, timeout)
	}
	resp, err := client.Get(searchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	tokenizer := html.NewTokenizer(resp.Body)
	var urls []string
	var found []string
	for notEnd := true; notEnd; {
		currentTokenType := tokenizer.Next()
		switch {
		case currentTokenType == html.ErrorToken:
			notEnd = false
		case currentTokenType == html.StartTagToken:
			token := tokenizer.Token()
			if token.Data == "a" {
				attributes := token.Attr
				found = parseAttrs(attributes)
				urls = append(urls, found...)
			}
		}
	}
	if len(urls) == 0 {
		return nil, nil
	}
	return urls, nil
}
