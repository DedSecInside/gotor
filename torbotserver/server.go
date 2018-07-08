package main

import (
	"./goBot"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Input struct {
	Website string `json:"website"`
	Option  string `json:"option"`
}

type Output struct {
	Websites []string `json:"websites"`
}

func linksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var links []string
		resp := Input{}
		err := json.NewDecoder(r.Body).Decode(&resp)
		if err != nil {
			log.Printf("Error: %v", err)
		}
		if !strings.Contains(resp.Website, ".onion") {
			links, err = geturls.GetLinks(resp.Website, "", "", 15)
		} else {
			links, err = geturls.GetLinks(resp.Website, "127.0.0.1", "9050", 15)
		}
		if err != nil {
			log.Printf("Error: %v", err)
		}
		outResp := Output{Websites: links}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		err = json.NewEncoder(w).Encode(&outResp)
	}
}

func main() {
	http.HandleFunc("/LIVE", linksHandler)
	fmt.Println("Serving on localhost:8008/LIVE")
	log.Fatal(http.ListenAndServe(":8008", nil))
}
