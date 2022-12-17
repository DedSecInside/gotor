// This file contains HTTP REST handlers for interacting with links
package api

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/KingAkeem/gotor/internal/logger"
	"github.com/KingAkeem/gotor/pkg/linktree"
	"golang.org/x/net/html"
)

// GetTreeNode returns a LinkTree with the specified depth passed to the query parameter.
func GetTreeNode(client *http.Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		depthInput := r.URL.Query().Get("depth")
		depth, err := strconv.Atoi(depthInput)
		if err != nil {
			msg := "invalid depth, must be an integer"
			logger.Error(msg, "error", err.Error())
			w.Write([]byte(msg))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		link := r.URL.Query().Get("link")
		if link == "" {
			logger.Error("found blank link")
			w.Write([]byte("Link cannot be blank."))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		logger.Info("attempting to build new tree from request",
			"root", link,
			"depth", depth,
		)
		node := linktree.NewNode(client, link)
		node.Load(depth)
		logger.Info("build successful",
			"node", node,
		)

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(node)
		if err != nil {
			logger.Error("unable to marshal node",
				"error", err,
			)
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
		link := r.URL.Query().Get("link")
		logger.Info("attempting to collect emails",
			"link", link,
		)
		emails := getEmails(c, link)
		logger.Info("emails collected",
			"link", link,
			"emails", emails,
		)
		err := json.NewEncoder(w).Encode(emails)
		if err != nil {
			logger.Error("unable to marshal",
				"error", err,
			)
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
		link := r.URL.Query().Get("link")
		logger.Info("attempting to collect phone numbers",
			"link", link,
		)
		phone := getPhoneNumbers(c, link)
		logger.Info("numbers collected",
			"link", link,
			"numbers", phone,
		)
		err := json.NewEncoder(w).Encode(phone)
		if err != nil {
			logger.Error("unable to marshal",
				"error", err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// returns the body of a website as a string
func getWebsiteContent(client *http.Client, link string) (string, error) {
	resp, err := client.Get(link)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func GetWebsiteContent(client *http.Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		link := r.URL.Query().Get("link")
		logger.Info("attempting to collect website content",
			"link", link,
		)
		content, err := getWebsiteContent(client, link)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.Info("content collected",
			"link", link,
			"content", content,
		)
		err = json.NewEncoder(w).Encode(content)
		if err != nil {
			logger.Error("unable to marshal",
				"error", err,
			)
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
		logger.Info("retrieving local tor IP")
		ip, err := getTorIP(c)
		if err != nil {
			logger.Error("unable to retrieve IP",
				"error", err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte(ip))
		if err != nil {
			logger.Error("unable to write IP",
				"error", err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
