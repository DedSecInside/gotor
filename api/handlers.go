// This file contains HTTP REST API handlers
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/KingAkeem/gotor/internal/logger"
	"github.com/KingAkeem/gotor/linktree"
	"github.com/gorilla/mux"
	"golang.org/x/net/html"
)

// global HTTP client used for all requests, can be overwritten when starting server
var client = http.DefaultClient

// RunServer starts the server for the API, using the given client and port. Pass `nil` to use the default client, port must be specified
func RunServer(overrideClient *http.Client, port int) {
	router := mux.NewRouter()

	if overrideClient != nil {
		client = overrideClient
	}

	router.HandleFunc("/ip", GetIP).Methods(http.MethodGet)
	router.HandleFunc("/emails", GetEmails).Methods(http.MethodGet)
	router.HandleFunc("/phone", GetPhoneNumbers).Methods(http.MethodGet)
	router.HandleFunc("/tree", GetTreeNode).Methods(http.MethodGet)
	router.HandleFunc("/content", GetWebsiteContent).Methods(http.MethodGet)

	logger.Info("attempting to start local gotor server",
		"port", port,
	)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), router)
	if err != nil {
		logger.Fatal("unable to start server",
			"error", err.Error(),
		)
	}
}

// GetTreeNode returns a LinkTree with the specified depth passed to the query parameter.
func GetTreeNode(w http.ResponseWriter, r *http.Request) {
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
func getEmails(link string) []string {
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
func GetEmails(w http.ResponseWriter, r *http.Request) {
	link := r.URL.Query().Get("link")
	logger.Info("attempting to collect emails",
		"link", link,
	)
	emails := getEmails(link)
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

// returns a slice of phone numbers found at a link
func getPhoneNumbers(link string) []string {
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
func GetPhoneNumbers(w http.ResponseWriter, r *http.Request) {
	link := r.URL.Query().Get("link")
	logger.Info("attempting to collect phone numbers",
		"link", link,
	)
	phone := getPhoneNumbers(link)
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

// returns the body of a website as a string
func getWebsiteContent(link string) (string, error) {
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

func GetWebsiteContent(w http.ResponseWriter, r *http.Request) {
	link := r.URL.Query().Get("link")
	logger.Info("attempting to collect website content",
		"link", link,
	)
	content, err := getWebsiteContent(link)
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

// gets the current IP address of the Tor client
func getTorIP() (string, error) {
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
func GetIP(w http.ResponseWriter, r *http.Request) {
	logger.Info("retrieving local tor IP")
	ip, err := getTorIP()
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
	logger.Info("Successfully sent IP", "ip", ip)
}
