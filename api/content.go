package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/DedSecInside/gotor/internal/logger"
)

// returns the body of a website as a string
func collectWebsiteContent(client *http.Client, link string) (string, error) {
	resp, err := client.Get(link)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (s Server) handleGetWebsiteContent(w http.ResponseWriter, r *http.Request) {
	link := r.URL.Query().Get("link")
	logger.Info("attempting to collect website content",
		"link", link,
	)
	content, err := collectWebsiteContent(s.client, link)
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
