package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Result struct {
	Message string `json:"msg"`
	ID      int    `json:"id"`
}

func linksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		res := Result{"Hello World", 1}
		jData, err := json.Marshal(res)
		if err != nil {
			panic(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(jData)
	}
}

func main() {
	http.HandleFunc("/LIVE", linksHandler)
	fmt.Println("Serving on localhost:8080/LIVE")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
