package crawl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func directRequest(target string, depth int) Request {
	return Request{
		URL:               target,
		Depth:             depth,
		UseTor:            false,
		SOCKS5Host:        "127.0.0.1",
		SOCKS5Port:        9050,
		Workers:           8,
		QueueSize:         64,
		RequestsPerSecond: 1000,
		Burst:             1000,
		PerHostParallel:   2,
		MaxResponseBytes:  1024 * 1024,
	}
}

func TestExecuteReturnsRelationshipsContactsAndSkippedContent(t *testing.T) {
	t.Parallel()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/":
			writer.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(writer, `
				<html lang="en">
					<head>
						<title>Root</title>
						<base href="%s/nested/">
						<meta http-equiv="refresh" content="30; url=/refresh">
					</head>
					<body>
						<a href="child">Child</a>
						<a href="/binary">Binary</a>
						<a href="mailto:ops@example.com">Email</a>
						<a href="tel:+12025550100">Phone</a>
					</body>
				</html>`, server.URL)
		case "/nested/child":
			writer.Header().Set("Content-Type", "text/html")
			fmt.Fprint(writer, "<html><title>Child</title></html>")
		case "/refresh":
			writer.Header().Set("Content-Type", "text/html")
			fmt.Fprint(writer, "<html><title>Refresh</title></html>")
		case "/binary":
			writer.Header().Set("Content-Type", "application/octet-stream")
			fmt.Fprint(writer, "not html")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	report, err := Execute(context.Background(), directRequest(server.URL, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("unexpected crawl errors: %#v", report.Errors)
	}
	if len(report.Pages) != 4 {
		t.Fatalf("expected four pages, got %d: %#v", len(report.Pages), report.Pages)
	}

	root := report.Pages[0]
	if root.Metadata.Title != "Root" {
		t.Fatalf("unexpected root metadata: %#v", root.Metadata)
	}
	if len(root.Metadata.Emails) != 1 || root.Metadata.Emails[0] != "ops@example.com" {
		t.Fatalf("unexpected emails: %#v", root.Metadata.Emails)
	}
	if len(root.Metadata.PhoneNumbers) != 1 || root.Metadata.PhoneNumbers[0] != "+12025550100" {
		t.Fatalf("unexpected phone numbers: %#v", root.Metadata.PhoneNumbers)
	}

	foundChild := false
	foundRefresh := false
	foundSkippedBinary := false
	for _, page := range report.Pages {
		if page.ParentURL != "" && page.ParentURL != server.URL {
			t.Fatalf("unexpected parent URL %q", page.ParentURL)
		}
		switch page.Metadata.URL {
		case server.URL + "/nested/child":
			foundChild = true
		case server.URL + "/refresh":
			foundRefresh = true
		case server.URL + "/binary":
			foundSkippedBinary = strings.Contains(page.SkippedReason, "non-HTML")
		}
	}
	if !foundChild || !foundRefresh || !foundSkippedBinary {
		t.Fatalf(
			"missing expected results: child=%v refresh=%v skippedBinary=%v",
			foundChild,
			foundRefresh,
			foundSkippedBinary,
		)
	}
}

func TestPerHostParallelLimit(t *testing.T) {
	t.Parallel()
	var active int32
	var maximum int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/" {
			writer.Header().Set("Content-Type", "text/html")
			fmt.Fprint(writer, `<a href="/a">A</a><a href="/b">B</a><a href="/c">C</a>`)
			return
		}

		current := atomic.AddInt32(&active, 1)
		for {
			observed := atomic.LoadInt32(&maximum)
			if current <= observed || atomic.CompareAndSwapInt32(&maximum, observed, current) {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
		atomic.AddInt32(&active, -1)
		writer.Header().Set("Content-Type", "text/html")
		fmt.Fprint(writer, "<html></html>")
	}))
	defer server.Close()

	request := directRequest(server.URL, 1)
	request.PerHostParallel = 1
	if _, err := Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&maximum); got > 1 {
		t.Fatalf("expected at most one request per host, observed %d", got)
	}
}
