package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
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
	link := strings.TrimSpace(r.URL.Query().Get("link"))
	if link == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Link must be specified."))
		return
	}

	content, err := collectWebsiteContent(s.client, link)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to collect website content."))
		log.Printf("Error: %+v\n", err)
		return
	}
	err = json.NewEncoder(w).Encode(content)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error encountered attempting to serve response."))
		log.Printf("Unable to marshal content. Error: %+v\n", err)
		return
	}
}
