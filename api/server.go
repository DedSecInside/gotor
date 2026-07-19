package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/DedSecInside/gotor/internal/crawl"
)

const APIBasePath = "/api/v1"

// Server exposes crawl job control over a local HTTP API.
type Server struct {
	host        string
	port        int
	manager     *JobManager
	readyWriter io.Writer
}

func NewServer(host string, port int) *Server {
	return &Server{
		host:        host,
		port:        port,
		manager:     NewJobManager(),
		readyWriter: os.Stdout,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /_healthz", s.handleHealth)
	mux.HandleFunc("GET /_readyz", s.handleReady)
	mux.HandleFunc("GET "+APIBasePath+"/tor/status", s.handleTorStatus)
	mux.HandleFunc("POST "+APIBasePath+"/jobs", s.handleStartJob)
	mux.HandleFunc("GET "+APIBasePath+"/jobs/{id}", s.handleGetJob)
	mux.HandleFunc("POST "+APIBasePath+"/jobs/{id}/cancel", s.handleCancelJob)
	return loggingMiddleware(mux)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		next.ServeHTTP(writer, request)
		slog.Info("api request",
			"method", request.Method,
			"path", request.URL.Path,
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
	})
}

func (s *Server) Run(ctx context.Context) error {
	address := net.JoinHostPort(s.host, strconv.Itoa(s.port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverURL := "http://" + listener.Addr().String()
	if s.readyWriter != nil {
		_ = json.NewEncoder(s.readyWriter).Encode(map[string]string{
			"event": "ready",
			"url":   serverURL,
		})
	}
	slog.Info("gotor API ready", "url", serverURL)

	stopped := make(chan error, 1)
	go func() {
		stopped <- httpServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		s.manager.CancelAll()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err = <-stopped
	case err = <-stopped:
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) handleHealth(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{
		"status": "ok",
		"engine": "gotor",
	})
}

func (s *Server) handleReady(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleStartJob(writer http.ResponseWriter, request *http.Request) {
	body := http.MaxBytesReader(writer, request.Body, 1<<20)
	defer body.Close()

	var crawlRequest crawl.Request
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&crawlRequest); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	job, err := s.manager.Start(crawlRequest)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(writer, http.StatusAccepted, job)
}

func (s *Server) handleGetJob(writer http.ResponseWriter, request *http.Request) {
	job, ok := s.manager.Get(request.PathValue("id"))
	if !ok {
		writeAPIError(writer, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(writer, http.StatusOK, job)
}

func (s *Server) handleCancelJob(writer http.ResponseWriter, request *http.Request) {
	job, ok := s.manager.Cancel(request.PathValue("id"))
	if !ok {
		writeAPIError(writer, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(writer, http.StatusOK, job)
}

func writeAPIError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		slog.Error("encode API response", "error", err)
	}
}

func (s *Server) String() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}
