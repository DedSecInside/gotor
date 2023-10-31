// This file contains HTTP REST API handlers
package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Server struct {
	host   string
	port   int
	client *http.Client
}

func New(client *http.Client, host string, port int) *Server {
	return &Server{
		client: client,
		host:   host,
		port:   port,
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Println(r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

// RunServer starts the server for the API, using the given client and port. Pass `nil` to use the default client, port must be specified
func (s Server) Run() {
	router := mux.NewRouter()

	router.HandleFunc("/ip", s.handleGetIP).Methods(http.MethodGet)
	router.HandleFunc("/tree", s.handleGetTreeNode).Methods(http.MethodGet)
	router.HandleFunc("/emails", s.handleGetEmails).Methods(http.MethodGet)
	router.HandleFunc("/phone", s.handleGetPhoneNumbers).Methods(http.MethodGet)
	router.HandleFunc("/content", s.handleGetWebsiteContent).Methods(http.MethodGet)

	log.Printf("Attempting to start local gotor server. Host: %s - Port: %d\n", s.host, s.port)

	router.Use(loggingMiddleware)

	address := fmt.Sprintf("%s:%d", s.host, s.port)
	err := http.ListenAndServe(address, router)
	if err != nil {
		log.Fatalf("Unable to start server. Error: %+v.\n", err)
		return
	}
}
