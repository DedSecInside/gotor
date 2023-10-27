package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/DedSecInside/gotor/internal/logger"
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
	link := r.URL.Query().Get("link")
	logger.Info("attempting to collect phone numbers",
		"link", link,
	)
	phone := collectPhoneNumbers(s.client, link)
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
