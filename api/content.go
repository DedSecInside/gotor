package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
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
	content, err := collectWebsiteContent(s.client, link)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(content)
	if err != nil {
		log.Printf("Unable to marshal. Error: %+v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
