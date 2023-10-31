package api

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/DedSecInside/gotor/pkg/linktree"
)

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
func collectEmails(client *http.Client, link string) []string {
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
func (s Server) handleGetEmails(w http.ResponseWriter, r *http.Request) {
	link := strings.TrimSpace(r.URL.Query().Get("link"))
	if link == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Link must be specified."))
		return
	}
	emails := collectEmails(s.client, link)
	err := json.NewEncoder(w).Encode(emails)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to serve email response."))
		log.Printf("Unable to marshal. Error: %+v\n", err)
		return
	}
}
