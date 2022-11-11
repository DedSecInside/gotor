// This file contains HTTP REST handlers for interacting with links
package api

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/KingAkeem/gotor/pkg/linktree"
	"golang.org/x/net/html"
)

// GetTreeNode returns a LinkTree with the specified depth passed to the query parameter.
func GetTreeNode(client *http.Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		queryMap := r.URL.Query()
		depthInput := queryMap.Get("depth")
		depth, err := strconv.Atoi(depthInput)
		if err != nil {
			_, err := w.Write([]byte("Invalid depth. Must be an integer."))
			if err != nil {
				log.Printf("Unable to write error message. Error: %+v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		link := queryMap.Get("link")
		log.Printf("processing link %s at a depth of %d\n", link, depth)
		node := linktree.NewNode(client, link)
		node.Load(depth)
		log.Printf("tree built for %s at depth %d\n", node.URL, depth)

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(node)
		if err != nil {
			log.Printf("Unable to marshal link node. Error: %+v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// isEmailValid checks if the email provided passes the required structure and length test.
// It also checks the domain has a valid DNS MX record.
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
	emails := []string{}
	node := linktree.NewNode(client, link)
	depth := 1
	collectEmails := func(childLink string) {
		linkPieces := strings.Split(childLink, "mailto:")
		if len(linkPieces) > 1 && isEmailValid(linkPieces[1]) {
			emails = append(emails, linkPieces[1])
		}
	}
	node.Crawl(depth, collectEmails)
	return emails
}

// GetEmails writes an array of emails found on the given "link" passed in the query parameters by the client
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

// returns a slice of phone numbers found at a link
func getPhoneNumbers(client *http.Client, link string) []string {
	phone := []string{}
	node := linktree.NewNode(client, link)
	depth := 1
	collectNumbers := func(childLink string) {
		linkPieces := strings.Split(childLink, "tel:")
		if len(linkPieces) > 1 {
			if len(linkPieces[1]) > 0 {
				phone = append(phone, linkPieces[1])
			}
		}
	}
	node.Crawl(depth, collectNumbers)
	return phone
}

// GetPhoneNumbers writes a list of phone numbers using the `tel:` tag
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

// returns the body of a website as a string
func getWebsiteContent(client *http.Client, link string) (string, error) {
	resp, err := client.Get(link)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return string(body), nil
}

func GetWebsiteContent(client *http.Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		queryMap := r.URL.Query()
		link := queryMap.Get("link")
		content, err := getWebsiteContent(client, link)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = json.NewEncoder(w).Encode(content)
		if err != nil {
			log.Println("Error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// gets the current IP address of the Tor client
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

// GetIP writes the IP address of the current TOR connection being used
func GetIP(c *http.Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, err := getTorIP(c)
		if err != nil {
			log.Println("Error:", err)
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
