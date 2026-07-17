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
	effectiveBase := documentBaseURL(doc, base)
	collectDocumentLinks(doc, effectiveBase, seen, &links)

	sort.Slice(links, func(i, j int) bool {
		return links[i].String() < links[j].String()
	})
	return links, nil
}

func nodeAttribute(node *html.Node, name string) string {
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, name) {
			return attribute.Val
		}
	}
	return ""
}

func documentBaseURL(node *html.Node, fallback *url.URL) *url.URL {
	if node.Type == html.ElementNode && node.Data == "base" {
		if parsed, err := fallback.Parse(strings.TrimSpace(nodeAttribute(node, "href"))); err == nil {
			return parsed
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if result := documentBaseURL(child, fallback); result != fallback {
			return result
		}
	}
	return fallback
}

func appendDocumentLink(
	rawURL string,
	base *url.URL,
	seen map[string]struct{},
	links *[]*url.URL,
) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || strings.HasPrefix(rawURL, "#") {
		return
	}
	parsedURL, err := base.Parse(rawURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return
	}
	parsedURL.Fragment = ""
	value := parsedURL.String()
	if _, ok := seen[value]; ok {
		return
	}
	seen[value] = struct{}{}
	*links = append(*links, parsedURL)
}

func metaRefreshURL(node *html.Node) string {
	if !strings.EqualFold(strings.TrimSpace(nodeAttribute(node, "http-equiv")), "refresh") {
		return ""
	}
	content := nodeAttribute(node, "content")
	separator := strings.Index(content, ";")
	if separator < 0 {
		return ""
	}
	target := strings.TrimSpace(content[separator+1:])
	if !strings.HasPrefix(strings.ToLower(target), "url=") {
		return ""
	}
	return strings.Trim(strings.TrimSpace(target[4:]), `"'`)
}

func collectDocumentLinks(
	node *html.Node,
	base *url.URL,
	seen map[string]struct{},
	links *[]*url.URL,
) {
	if node.Type == html.ElementNode {
		switch node.Data {
		case "a":
			appendDocumentLink(nodeAttribute(node, "href"), base, seen, links)
		case "meta":
			appendDocumentLink(metaRefreshURL(node), base, seen, links)
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		collectDocumentLinks(child, base, seen, links)
	}
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

	pageMeta, body, skippedReason, err := c.readResponse(resp, t.URL)
	if err != nil {
		return err
	}
	if skippedReason != "" {
		c.storeSkippedPage(t, pageMeta, skippedReason)
		c.logFetch(t, resp, startedAt, skippedReason)
		return nil
	}

	pageResult, links, err := analyzeHTMLPage(t, resp, body)
	if err != nil {
		return err
	}
	c.store.addResult(pageResult)
	c.logFetch(t, resp, startedAt, "")
	c.enqueueChildren(t, links)
	return nil
}

func (c *Crawler) readResponse(
	response *http.Response,
	pageURL *url.URL,
) (parse.PageMeta, []byte, string, error) {
	contentType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	pageMeta := parse.PageMeta{
		URL:           pageURL.String(),
		Status:        response.StatusCode,
		ContentType:   contentType,
		ContentLength: response.ContentLength,
	}

	if response.ContentLength > c.opts.MaxResponseBytes {
		reason := fmt.Sprintf(
			"response is larger than %d bytes",
			c.opts.MaxResponseBytes,
		)
		return pageMeta, nil, reason, nil
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, c.opts.MaxResponseBytes+1))
	if err != nil {
		return pageMeta, nil, "", err
	}
	pageMeta.ContentLength = int64(len(body))
	if int64(len(body)) > c.opts.MaxResponseBytes {
		reason := fmt.Sprintf(
			"response exceeded %d bytes while downloading",
			c.opts.MaxResponseBytes,
		)
		return pageMeta, nil, reason, nil
	}

	if contentType == "" {
		contentType, _, _ = mime.ParseMediaType(http.DetectContentType(body))
		pageMeta.ContentType = contentType
	}
	isHTML := contentType == "text/html" || contentType == "application/xhtml+xml"
	if !c.opts.AllowNonHTML && !isHTML {
		reason := "non-HTML content type: " + contentType
		return pageMeta, nil, reason, nil
	}
	return pageMeta, body, "", nil
}

func (c *Crawler) storeSkippedPage(task Task, metadata parse.PageMeta, reason string) {
	c.store.addResult(PageResult{
		ParentURL:     task.ParentURL,
		Depth:         task.Depth,
		SkippedReason: reason,
		Metadata:      metadata,
	})
}

func analyzeHTMLPage(
	task Task,
	response *http.Response,
	body []byte,
) (PageResult, []*url.URL, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return PageResult{}, nil, fmt.Errorf("parse HTML: %w", err)
	}

	pageMeta := parse.ExtractPageMeta(response, task.URL, doc)
	pageMeta.ContentLength = int64(len(body))
	links, err := extractLinks(doc, task.URL)
	if err != nil {
		return PageResult{}, nil, fmt.Errorf("extract links: %w", err)
	}

	linkStrings := make([]string, 0, len(links))
	for _, link := range links {
		linkStrings = append(linkStrings, link.String())
	}
	return PageResult{
		ParentURL: task.ParentURL,
		Depth:     task.Depth,
		Links:     linkStrings,
		Metadata:  pageMeta,
	}, links, nil
}

func (c *Crawler) enqueueChildren(task Task, links []*url.URL) {
	nextDepth := task.Depth + 1
	if nextDepth <= c.opts.MaxDepth {
		for _, child := range links {
			_ = c.frontier.EnqueueIfNew(Task{
				URL:       child,
				ParentURL: task.URL.String(),
				Depth:     nextDepth,
			})
		}
	}
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
