package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/DedSecInside/gotor/internal/netx"
)

// returns the body of a website as a string
func collectWebsiteContent(client *http.Client, link string) (string, error) {
	req, err := netx.NewRequest(context.Background(), "GET", link, "GoTor/1.0")
	if err != nil {
		log.Printf("Unable to GET URL. URL %s. Error: %+v\n", link, err)
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to GET URL. URL %s. Error: %+v\n", link, err)
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
