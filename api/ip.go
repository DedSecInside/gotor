package api

import (
	"io"
	"log"
	"net/http"

	"golang.org/x/net/html"
)

// gets the current IP address of the Tor client
func getTorIP(client *http.Client) (string, error) {
	resp, err := client.Get("https://check.torproject.org/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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
func (s Server) handleGetIP(w http.ResponseWriter, r *http.Request) {
	ip, err := getTorIP(s.client)
	if err != nil {
		log.Printf("Unable to retrieve IP. Error: %+v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte(ip))
	if err != nil {
		log.Printf("Unable to write IP. Error: %+v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
