package netx

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

var browserUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/127.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36",
}

// ClientOpts configures the shared HTTP client.
type ClientOpts struct {
	// Transport / dialing
	UseTor             bool   // route via SOCKS5 (Tor)
	SocksHost          string // e.g. "127.0.0.1"
	SocksPort          int    // e.g. 9050
	DisableHTTP2       bool   // often desirable over SOCKS5
	DisableCompression bool
	InsecureTLS        bool // only for testing!

	// Pooling / timeouts
	DialTimeout  time.Duration // TCP connect timeout
	ReqTimeout   time.Duration // whole request timeout
	MaxIdleConns int           // shared connection pool size

	// Headers
	UserAgent string
}

// NewHTTPClient creates a single reusable *http.Client with a shared Transport.
// UseTor switches the Dialer to SOCKS5; otherwise a direct Dialer is used.
func NewHTTPClient(o ClientOpts) (*http.Client, error) {
	baseDialer := &net.Dialer{Timeout: o.DialTimeout}

	var d proxy.Dialer = baseDialer
	if o.UseTor {
		addr := fmt.Sprintf("%s:%d", o.SocksHost, o.SocksPort)
		socks, err := proxy.SOCKS5("tcp", addr, nil, baseDialer)
		if err != nil {
			return nil, fmt.Errorf("socks5 dialer: %w", err)
		}
		d = socks
	}

	tr := &http.Transport{
		Proxy: nil, // IMPORTANT: ignore env proxies when you say “-disable-socks5”
		// Use our dialer (direct or SOCKS5)
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return d.Dial(network, address)
		},
		MaxIdleConns:        o.MaxIdleConns,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 30 * time.Second,
		DisableCompression:  o.DisableCompression,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: o.InsecureTLS}, //nolint:gosec // allow opt-in for testing
		ForceAttemptHTTP2:   !o.DisableHTTP2,
	}

	return &http.Client{Transport: tr, Timeout: o.ReqTimeout}, nil
}

// NewRequest builds a request with UA applied. Use this everywhere for consistency.
func NewRequest(ctx context.Context, method, url, ua string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")

	return req, nil
}

// UserAgent returns a stable crawler identifier or a randomized browser user
// agent when header rotation is explicitly enabled.
func UserAgent(randomize bool) string {
	if !randomize {
		return "GoTor/3.0"
	}

	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return browserUserAgents[0]
	}
	index := binary.LittleEndian.Uint64(value[:]) % uint64(len(browserUserAgents))
	return browserUserAgents[index]
}
