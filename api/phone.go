package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/DedSecInside/gotor/pkg/linktree"
)

// returns a slice of phone numbers found at a link
func collectPhoneNumbers(client *http.Client, link string) []string {
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
func (s Server) handleGetPhoneNumbers(w http.ResponseWriter, r *http.Request) {
	link := strings.TrimSpace(r.URL.Query().Get("link"))
	if link == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Link must be specified."))
		return
	}
	phone := collectPhoneNumbers(s.client, link)
	err := json.NewEncoder(w).Encode(phone)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Unable to marshal. Error: %+v\n", err)
		return
	}
}
