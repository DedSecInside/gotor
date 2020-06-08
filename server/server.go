package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	gobot "./goBot"
)

func linksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	if r.Method == "GET" {
		website := r.URL.Query().Get("url")
		links, err := gobot.GetLinks(website, "127.0.0.1", "9050", 60)
		if err != nil {
			log.Printf("Unable to retrieve links for %s. Error: %v", website, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(&links)
		if err != nil {
			log.Printf("Error: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func main() {
	http.HandleFunc("/", linksHandler)
	fmt.Println("Serving on localhost:3050")
	log.Fatal(http.ListenAndServe(":3050", nil))
}
