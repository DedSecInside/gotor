package crawl

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/DedSecInside/gotor/internal/parse"
)

type Task struct {
	URL       *url.URL
	ParentURL string
	Depth     int
}

type Options struct {
	Workers            int           // N workers
	QueueSize          int           // bounded frontier size
	Rate               float64       // global RPS
	Burst              int           // global limiter burst
	MaxDepth           int           // inclusive max depth
	ReqTimeout         time.Duration // per-request timeout
	DialTimeout        time.Duration // dial timeout (plumbed to netx)
	PerHostParallel    int           // maximum concurrent requests per host
	PerHostDelay       time.Duration // minimum delay between requests to a host
	MaxResponseBytes   int64         // maximum response body read into memory
	AllowNonHTML       bool          // parse content without an HTML content type
	RandomizeUserAgent bool          // rotate a small set of browser user agents
}

// PageResult is the stable, machine-readable representation of one crawled page.
type PageResult struct {
	ParentURL     string         `json:"parentUrl,omitempty"`
	Depth         int            `json:"depth"`
	Links         []string       `json:"links,omitempty"`
	SkippedReason string         `json:"skippedReason,omitempty"`
	Metadata      parse.PageMeta `json:"metadata"`
}

// CrawlError records a page that could not be fetched or parsed without
// aborting the rest of the crawl.
type CrawlError struct {
	URL       string `json:"url"`
	ParentURL string `json:"parentUrl,omitempty"`
	Depth     int    `json:"depth"`
	Message   string `json:"message"`
}

// Report is the versioned response shared by the CLI and HTTP API.
type Report struct {
	SchemaVersion int          `json:"schemaVersion"`
	Engine        string       `json:"engine"`
	Target        string       `json:"target"`
	MaxDepth      int          `json:"maxDepth"`
	UsesTor       bool         `json:"usesTor"`
	StartedAt     time.Time    `json:"startedAt"`
	DurationMS    int64        `json:"durationMs"`
	Pages         []PageResult `json:"pages"`
	Errors        []CrawlError `json:"errors"`
}

// Request configures a complete crawl.
type Request struct {
	URL               string  `json:"url"`
	Depth             int     `json:"depth"`
	UseTor            bool    `json:"useTor"`
	SOCKS5Host        string  `json:"socks5Host"`
	SOCKS5Port        int     `json:"socks5Port"`
	Workers           int     `json:"workers,omitempty"`
	QueueSize         int     `json:"queueSize,omitempty"`
	RequestsPerSecond float64 `json:"requestsPerSecond,omitempty"`
	Burst             int     `json:"burst,omitempty"`
	PerHostParallel   int     `json:"perHostParallel,omitempty"`
	PerHostDelayMS    int     `json:"perHostDelayMs,omitempty"`
	MaxResponseBytes  int64   `json:"maxResponseBytes,omitempty"`
	AllowNonHTML      bool    `json:"allowNonHtml,omitempty"`
	RandomizeHeaders  bool    `json:"randomizeHeaders,omitempty"`
}

func (r Request) options() Options {
	workers := r.Workers
	if workers <= 0 {
		workers = 16
	}
	queueSize := r.QueueSize
	if queueSize <= 0 {
		queueSize = 2048
	}
	requestsPerSecond := r.RequestsPerSecond
	if requestsPerSecond <= 0 {
		requestsPerSecond = 5
	}
	burst := r.Burst
	if burst <= 0 {
		burst = 5
	}
	perHostParallel := r.PerHostParallel
	if perHostParallel <= 0 {
		perHostParallel = 2
	}
	maxResponseBytes := r.MaxResponseBytes
	if maxResponseBytes <= 0 {
		maxResponseBytes = 5 * 1024 * 1024
	}

	return Options{
		Workers:            workers,
		QueueSize:          queueSize,
		Rate:               requestsPerSecond,
		Burst:              burst,
		MaxDepth:           r.Depth,
		ReqTimeout:         30 * time.Second,
		DialTimeout:        10 * time.Second,
		PerHostParallel:    perHostParallel,
		PerHostDelay:       time.Duration(r.PerHostDelayMS) * time.Millisecond,
		MaxResponseBytes:   maxResponseBytes,
		AllowNonHTML:       r.AllowNonHTML,
		RandomizeUserAgent: r.RandomizeHeaders,
	}
}

func (r Request) withDefaults() Request {
	if r.SOCKS5Host == "" {
		r.SOCKS5Host = "127.0.0.1"
	}
	if r.SOCKS5Port == 0 {
		r.SOCKS5Port = 9050
	}
	return r
}

// Validate checks API and CLI inputs before work is scheduled.
func (r Request) Validate() error {
	parsedURL, err := url.Parse(r.URL)
	if err != nil || parsedURL.Host == "" ||
		(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return &InvalidSeedError{URL: r.URL}
	}
	if r.Depth < 0 || r.Depth > 10 {
		return fmt.Errorf("depth must be between 0 and 10")
	}
	if r.SOCKS5Host == "" {
		return fmt.Errorf("SOCKS5 host cannot be blank")
	}
	if r.SOCKS5Port < 1 || r.SOCKS5Port > 65535 {
		return fmt.Errorf("SOCKS5 port must be between 1 and 65535")
	}
	if r.PerHostParallel < 0 || r.PerHostDelayMS < 0 {
		return fmt.Errorf("per-host limits cannot be negative")
	}
	return nil
}

// Execute validates and runs a crawl using the shared crawler implementation.
func Execute(ctx context.Context, request Request) (Report, error) {
	request = request.withDefaults()
	startedAt := time.Now().UTC()
	report := Report{
		SchemaVersion: 1,
		Engine:        "gotor",
		Target:        request.URL,
		MaxDepth:      request.Depth,
		UsesTor:       request.UseTor,
		StartedAt:     startedAt,
		Pages:         []PageResult{},
		Errors:        []CrawlError{},
	}

	if err := request.Validate(); err != nil {
		return report, err
	}

	crawler, err := NewCrawler(
		ctx,
		request.options(),
		request.UseTor,
		request.SOCKS5Host,
		request.SOCKS5Port,
	)
	if err != nil {
		return report, err
	}

	if crawler.Seed(request.URL) == 0 {
		return report, &InvalidSeedError{URL: request.URL}
	}

	err = crawler.Run(ctx)
	report.Pages = crawler.Results()
	report.Errors = crawler.Errors()
	report.DurationMS = time.Since(startedAt).Milliseconds()
	return report, err
}

// InvalidSeedError indicates that the requested root URL is not absolute HTTP(S).
type InvalidSeedError struct {
	URL string
}

func (e *InvalidSeedError) Error() string {
	return "invalid seed URL: " + e.URL
}

type resultStore struct {
	mu      sync.Mutex
	results []PageResult
	errors  []CrawlError
}

func (s *resultStore) addResult(result PageResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results = append(s.results, result)
}

func (s *resultStore) addError(task Task, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors = append(s.errors, CrawlError{
		URL:       task.URL.String(),
		ParentURL: task.ParentURL,
		Depth:     task.Depth,
		Message:   err.Error(),
	})
}

func (s *resultStore) snapshot() ([]PageResult, []CrawlError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	results := make([]PageResult, len(s.results))
	copy(results, s.results)
	errors := make([]CrawlError, len(s.errors))
	copy(errors, s.errors)
	sort.Slice(results, func(i, j int) bool {
		if results[i].Depth == results[j].Depth {
			return results[i].Metadata.URL < results[j].Metadata.URL
		}
		return results[i].Depth < results[j].Depth
	})
	sort.Slice(errors, func(i, j int) bool {
		if errors[i].Depth == errors[j].Depth {
			return errors[i].URL < errors[j].URL
		}
		return errors[i].Depth < errors[j].Depth
	})
	return results, errors
}
