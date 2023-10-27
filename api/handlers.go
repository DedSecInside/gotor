// This file contains HTTP REST API handlers
package api

import (
	"fmt"
	"net/http"

	"github.com/DedSecInside/gotor/internal/logger"
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

// RunServer starts the server for the API, using the given client and port. Pass `nil` to use the default client, port must be specified
func (s Server) Run() {
	router := mux.NewRouter()

	router.HandleFunc("/ip", s.handleGetIP).Methods(http.MethodGet)
	router.HandleFunc("/tree", s.handleGetTreeNode).Methods(http.MethodGet)
	router.HandleFunc("/emails", s.handleGetEmails).Methods(http.MethodGet)
	router.HandleFunc("/phone", s.handleGetPhoneNumbers).Methods(http.MethodGet)
	router.HandleFunc("/content", s.handleGetWebsiteContent).Methods(http.MethodGet)

	logger.Info("attempting to start local gotor server",
		"host", s.host,
		"port", s.port,
	)

	address := fmt.Sprintf("%s:%d", s.host, s.port)
	err := http.ListenAndServe(address, router)
	if err != nil {
		logger.Fatal("unable to start server",
			"error", err.Error(),
		)
	}
}
