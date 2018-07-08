package main

import (
	"./goBot"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Input respresents the input received from client
type Input struct {
	Website string `json:"website"`
	Option  string `json:"option"`
}

// Output represents the JSON that will be served as response
type Output struct {
	Websites map[string]bool `json:"websites"`
}

func linksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Receive user input
		resp := Input{}
		err := json.NewDecoder(r.Body).Decode(&resp)
		if err != nil {
			log.Printf("Error: %v", err)
		}

		// If onion website then use Tor
		var links map[string]bool
		if !strings.Contains(resp.Website, ".onion") {
			links, err = gobot.GetLinks(resp.Website, "", "", 15)
		} else {
			links, err = gobot.GetLinks(resp.Website, "127.0.0.1", "9050", 15)
		}
		if err != nil {
			log.Printf("Error: %v", err)
		}

		// Serve Response
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
