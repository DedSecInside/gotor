package main

import (
	"fmt"
	"net/http"
)

func linksHandler(r *http.Request, w http.ResponseWriter) {
	if r.Method == "GET" {
		fmt.Println("Being Reached")
	}
}

func main() {
	http.Handle("/links", linksHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
