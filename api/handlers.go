package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/KingAkeem/gotor/linktree"
	"github.com/patrickmn/go-cache"
	"golang.org/x/net/html"
)

// GetTreeNode writes a tree using the root and depth given, has a cache to avoid excessive lookups because building a tree is an expensive operation
func GetTreeNode(client *http.Client) func(w http.ResponseWriter, r *http.Request) {
	nodeCache := cache.New(5*time.Minute, 10*time.Minute)

	getNode := func(link string, depth int) (*linktree.Node, error) {
		parsedLink, err := url.Parse(link)
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%s-%d", parsedLink.Host, depth)
		if recentNode, found := nodeCache.Get(key); found {
			return recentNode.(*linktree.Node), nil
		}

		log.Printf("processing link %s at a depth of %d\n", link, depth)
		node := linktree.NewNode(client, link)
		node.Load(depth)
		log.Printf("Tree built for %s at depth %d\n", node.URL, depth)
		nodeCache.Set(key, node, cache.DefaultExpiration)
		return node, nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		writeNode := func(node *linktree.Node) {
			err := json.NewEncoder(w).Encode(node)
			if err != nil {
				log.Printf("Error: %+v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}

		queryMap := r.URL.Query()
		link := queryMap.Get("link")

		depthInput := queryMap.Get("depth")
		depth, err := strconv.Atoi(depthInput)
		if err != nil {
			_, err := w.Write([]byte("Invalid depth. Must be an integer."))
			if err != nil {
				log.Printf("Error: %+v\n", err)
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// attempt to pull node from cache
		node, err := getNode(link, depth)
		if err != nil {
			log.Printf("Error: %+v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		writeNode(node)
	}
}

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// isEmailValid checks if the email provided passes the required structure
// and length test. It also checks the domain has a valid MX record.
func isEmailValid(e string) bool {
	if len(e) < 3 && len(e) > 254 {
		return false
	}
	if !emailRegex.MatchString(e) {
		return false
	}
	parts := strings.Split(e, "@")
	mx, err := net.LookupMX(parts[1])
	if err != nil || len(mx) == 0 {
		return false
	}
	return true
}

// gets any email addresses on the url passed
func getEmails(client *http.Client, link string) []string {
	links := []string{}
	node := linktree.NewNode(client, link)
	depth := 1
	collectLinks := func(childLink string) {
		linkPieces := strings.Split(childLink, "mailto:")
		if len(linkPieces) > 1 && isEmailValid(linkPieces[1]) {
			links = append(links, linkPieces[1])
		}
	}
	node.Crawl(depth, collectLinks)
	return links
}

// GetEmails ...
func GetEmails(c *http.Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		queryMap := r.URL.Query()
		link := queryMap.Get("link")
		emails := getEmails(c, link)
		err := json.NewEncoder(w).Encode(emails)
		if err != nil {
			log.Println("Error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// gets any phone number addresses on the url passed
func getPhoneNumbers(client *http.Client, link string) []string {
	phone := []string{}
	node := linktree.NewNode(client, link)
	depth := 1
	collectLinks := func(childLink string) {
		linkPieces := strings.Split(childLink, "tel:")
		if len(linkPieces) > 1 {
			if len(linkPieces[1]) > 0 {
				phone = append(phone, linkPieces[1])
			}
		}
	}
	node.Crawl(depth, collectLinks)
	return phone
}

// GetPhone number ...
func GetPhoneNumbers(c *http.Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		queryMap := r.URL.Query()
		link := queryMap.Get("link")
		phone := getPhoneNumbers(c, link)
		err := json.NewEncoder(w).Encode(phone)
		if err != nil {
			log.Println("Error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// gets the current IP adress of the Tor client
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

// GetIP ...
func GetIP(c *http.Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, err := getTorIP(c)
		if err != nil {
			_, err = w.Write([]byte(err.Error()))
			if err != nil {
				log.Println("Error:", err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = w.Write([]byte(ip))
		if err != nil {
			log.Println("Error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
