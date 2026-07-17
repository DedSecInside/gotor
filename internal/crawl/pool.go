package crawl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/DedSecInside/gotor/internal/netx"
	"github.com/DedSecInside/gotor/internal/parse"
)

type Crawler struct {
	opts      Options
	frontier  *Frontier
	client    *http.Client
	limiter   *rate.Limiter
	store     resultStore
	usesTor   bool
	hostMu    sync.Mutex
	hostGates map[string]*hostGate
}

type hostGate struct {
	tokens chan struct{}
	mu     sync.Mutex
	next   time.Time
}

func (g *hostGate) acquire(ctx context.Context, delay time.Duration) (func(), error) {
	select {
	case g.tokens <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	g.mu.Lock()
	now := time.Now()
	readyAt := now
	if g.next.After(now) {
		readyAt = g.next
	}
	g.next = readyAt.Add(delay)
	g.mu.Unlock()

	timer := time.NewTimer(time.Until(readyAt))
	defer timer.Stop()
	select {
	case <-timer.C:
		return func() { <-g.tokens }, nil
	case <-ctx.Done():
		<-g.tokens
		return nil, ctx.Err()
	}
}

func NewCrawler(ctx context.Context, opts Options, useTor bool, socksHost string, socksPort int) (*Crawler, error) {
	if opts.Workers <= 0 {
		opts.Workers = 16
	}
	if opts.QueueSize <= 0 {
		opts.QueueSize = 2048
	}
	if opts.Rate <= 0 {
		opts.Rate = 5
	}
	if opts.Burst <= 0 {
		opts.Burst = 5
	}
	if opts.PerHostParallel <= 0 {
		opts.PerHostParallel = 2
	}
	if opts.MaxResponseBytes <= 0 {
		opts.MaxResponseBytes = 5 * 1024 * 1024
	}

	httpClient, err := netx.NewHTTPClient(netx.ClientOpts{
		UseTor:       useTor,
		SocksHost:    socksHost,
		SocksPort:    socksPort,
		ReqTimeout:   opts.ReqTimeout,
		DialTimeout:  opts.DialTimeout,
		MaxIdleConns: 32,
	})
	if err != nil {
		return nil, err
	}
	return &Crawler{
		opts:      opts,
		frontier:  NewFrontier(opts.QueueSize),
		client:    httpClient,
		limiter:   rate.NewLimiter(rate.Limit(opts.Rate), opts.Burst),
		usesTor:   useTor,
		hostGates: make(map[string]*hostGate),
	}, nil
}

func (c *Crawler) gateFor(host string) *hostGate {
	c.hostMu.Lock()
	defer c.hostMu.Unlock()
	gate, ok := c.hostGates[host]
	if !ok {
		gate = &hostGate{tokens: make(chan struct{}, c.opts.PerHostParallel)}
		c.hostGates[host] = gate
	}
	return gate
}

// Seed one or more starting URLs at depth 0.
func (c *Crawler) Seed(rawURLs ...string) int {
	added := 0
	for _, ru := range rawURLs {
		u, err := url.Parse(ru)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			continue
		}
		if c.frontier.EnqueueIfNew(Task{URL: u, Depth: 0}) {
			added++
		}
	}
	return added
}

// Run crawls until all queued pages have been processed or the context is cancelled.
func (c *Crawler) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for i := 0; i < c.opts.Workers; i++ {
		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case t, ok := <-c.frontier.Next():
					if !ok {
						return nil // frontier closed -> worker exits
					}

					// global rate-limit
					if err := c.limiter.Wait(ctx); err != nil {
						return err
					}

					// track in-flight
					c.frontier.markStart()
					if err := c.process(ctx, t); err != nil {
						slog.Warn("page crawl failed",
							"url", t.URL.String(),
							"depth", t.Depth,
							"error", err,
						)
						c.store.addError(t, err)
					}
					c.frontier.markDone()
				}
			}
		})
	}

	done := make(chan struct{})

	go func() {
		t := time.NewTicker(200 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_, _, pending, _ := c.frontier.Stats()
				if pending == 0 {
					c.frontier.CloseOnce()
					close(done)
					return
				}
			}
		}
	}()

	select {
	case <-done:
		// graceful completion (max depth reached & drained)
		return g.Wait()
	case <-ctx.Done():
		// external cancel (e.g., SIGINT)
		c.frontier.CloseOnce()
		return g.Wait()
	}

}

// Results returns a deterministic snapshot of successfully crawled pages.
func (c *Crawler) Results() []PageResult {
	results, _ := c.store.snapshot()
	return results
}

// Errors returns a deterministic snapshot of page-level crawl failures.
func (c *Crawler) Errors() []CrawlError {
	_, errors := c.store.snapshot()
	return errors
}

// extractLinks parses <a href> tags from the body, resolves them against base,
// and returns absolute URLs suitable for enqueueing.
func extractLinks(doc *html.Node, base *url.URL) ([]*url.URL, error) {
	seen := make(map[string]struct{})
	var links []*url.URL
	effectiveBase := base

	var findBase func(*html.Node)
	findBase = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "base" {
			for _, attribute := range n.Attr {
				if attribute.Key != "href" {
					continue
				}
				if parsed, err := base.Parse(strings.TrimSpace(attribute.Val)); err == nil {
					effectiveBase = parsed
				}
				return
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			findBase(child)
		}
	}
	findBase(doc)

	appendLink := func(rawURL string) {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" || strings.HasPrefix(rawURL, "#") {
			return
		}
		parsedURL, err := effectiveBase.Parse(rawURL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return
		}
		parsedURL.Fragment = ""
		value := parsedURL.String()
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		links = append(links, parsedURL)
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					appendLink(a.Val)
				}
			}
		}
		if n.Type == html.ElementNode && n.Data == "meta" {
			var httpEquiv, content string
			for _, attribute := range n.Attr {
				switch strings.ToLower(attribute.Key) {
				case "http-equiv":
					httpEquiv = strings.ToLower(strings.TrimSpace(attribute.Val))
				case "content":
					content = attribute.Val
				}
			}
			if httpEquiv == "refresh" {
				if separator := strings.Index(content, ";"); separator >= 0 {
					target := strings.TrimSpace(content[separator+1:])
					if strings.HasPrefix(strings.ToLower(target), "url=") {
						appendLink(strings.Trim(strings.TrimSpace(target[4:]), `"'`))
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	sort.Slice(links, func(i, j int) bool {
		return links[i].String() < links[j].String()
	})

	return links, nil
}

func (c *Crawler) process(ctx context.Context, t Task) error {
	startedAt := time.Now()
	gate := c.gateFor(t.URL.Hostname())
	release, err := gate.acquire(ctx, c.opts.PerHostDelay)
	if err != nil {
		return err
	}
	defer release()

	req, err := netx.NewRequest(
		ctx,
		http.MethodGet,
		t.URL.String(),
		netx.UserAgent(c.opts.RandomizeUserAgent),
	)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	contentType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	pageMeta := parse.PageMeta{
		URL:           t.URL.String(),
		Status:        resp.StatusCode,
		ContentType:   contentType,
		ContentLength: resp.ContentLength,
	}

	if resp.ContentLength > c.opts.MaxResponseBytes {
		reason := fmt.Sprintf(
			"response is larger than %d bytes",
			c.opts.MaxResponseBytes,
		)
		c.store.addResult(PageResult{
			ParentURL:     t.ParentURL,
			Depth:         t.Depth,
			SkippedReason: reason,
			Metadata:      pageMeta,
		})
		c.logFetch(t, resp, startedAt, reason)
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.opts.MaxResponseBytes+1))
	if err != nil {
		return err
	}
	pageMeta.ContentLength = int64(len(body))
	if int64(len(body)) > c.opts.MaxResponseBytes {
		reason := fmt.Sprintf(
			"response exceeded %d bytes while downloading",
			c.opts.MaxResponseBytes,
		)
		c.store.addResult(PageResult{
			ParentURL:     t.ParentURL,
			Depth:         t.Depth,
			SkippedReason: reason,
			Metadata:      pageMeta,
		})
		c.logFetch(t, resp, startedAt, reason)
		return nil
	}

	if contentType == "" {
		contentType, _, _ = mime.ParseMediaType(http.DetectContentType(body))
		pageMeta.ContentType = contentType
	}
	isHTML := contentType == "text/html" || contentType == "application/xhtml+xml"
	if !c.opts.AllowNonHTML && !isHTML {
		reason := "non-HTML content type: " + contentType
		c.store.addResult(PageResult{
			ParentURL:     t.ParentURL,
			Depth:         t.Depth,
			SkippedReason: reason,
			Metadata:      pageMeta,
		})
		c.logFetch(t, resp, startedAt, reason)
		return nil
	}

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("parse HTML: %w", err)
	}

	pageMeta = parse.ExtractPageMeta(resp, t.URL, doc)
	pageMeta.ContentLength = int64(len(body))
	links, err := extractLinks(doc, t.URL)
	if err != nil {
		return fmt.Errorf("extract links: %w", err)
	}

	linkStrings := make([]string, 0, len(links))
	for _, link := range links {
		if link.Scheme == "http" || link.Scheme == "https" {
			linkStrings = append(linkStrings, link.String())
		}
	}
	c.store.addResult(PageResult{
		ParentURL: t.ParentURL,
		Depth:     t.Depth,
		Links:     linkStrings,
		Metadata:  pageMeta,
	})
	c.logFetch(t, resp, startedAt, "")

	// Enqueue children if we’re not at max depth yet.
	nextDepth := t.Depth + 1
	if nextDepth <= c.opts.MaxDepth {
		for _, child := range links {
			if child.Scheme != "http" && child.Scheme != "https" {
				continue
			}
			_ = c.frontier.EnqueueIfNew(Task{
				URL:       child,
				ParentURL: t.URL.String(),
				Depth:     nextDepth,
			})
		}
	}
	return nil
}

func (c *Crawler) logFetch(
	task Task,
	response *http.Response,
	startedAt time.Time,
	skippedReason string,
) {
	via := "direct"
	if c.usesTor {
		via = "tor"
	}
	slog.Info("fetch",
		"host", task.URL.Hostname(),
		"url", task.URL.String(),
		"code", response.StatusCode,
		"bytes", response.ContentLength,
		"duration_ms", time.Since(startedAt).Milliseconds(),
		"via", via,
		"retry", 0,
		"skipped_reason", skippedReason,
	)
}
