package api

import (
	"io"
	"net/http"

	"github.com/DedSecInside/gotor/internal/logger"
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
	logger.Info("retrieving local tor IP")
	ip, err := getTorIP(s.client)
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
