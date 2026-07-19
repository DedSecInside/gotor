package parse

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestExtractPageMetaIncludesContactsAndLanguage(t *testing.T) {
	t.Parallel()
	document, err := html.Parse(strings.NewReader(`
		<html lang="EN-US">
			<head>
				<title>Example</title>
				<meta name="description" content="A test page">
			</head>
			<body>
				<a href="mailto:hello@example.com?subject=Hi">Email</a>
				<a href="mailto:hello@example.com">Duplicate email</a>
				<a href="tel:+1-202-555-0100">Call</a>
			</body>
		</html>
	`))
	if err != nil {
		t.Fatal(err)
	}
	base, _ := url.Parse("https://example.com")
	response := &http.Response{
		StatusCode:    http.StatusOK,
		ContentLength: 42,
		Header:        http.Header{"Content-Type": []string{"text/html"}},
	}

	metadata := ExtractPageMeta(response, base, document)
	if metadata.Title != "Example" {
		t.Fatalf("expected title Example, got %q", metadata.Title)
	}
	if metadata.Lang != "en-us" {
		t.Fatalf("expected normalized language, got %q", metadata.Lang)
	}
	if len(metadata.Emails) != 1 || metadata.Emails[0] != "hello@example.com" {
		t.Fatalf("unexpected emails: %#v", metadata.Emails)
	}
	if len(metadata.PhoneNumbers) != 1 || metadata.PhoneNumbers[0] != "+1-202-555-0100" {
		t.Fatalf("unexpected phone numbers: %#v", metadata.PhoneNumbers)
	}
}
