package api

import (
	"fmt"
	"testing"

	"net/http"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func newPage(title string, body string) string {
	baseHTML := `<!DOCTYPE html>
		<html lang=en-US class=no-js>
			<head><title>%s</title></head>
			<body>%s</body>
	</html>`
	return fmt.Sprintf(baseHTML, title, body)
}

func TestGetTorIP(t *testing.T) {
	httpmock.Activate()
	page := newPage("Tor Project", "<strong>Random IP Address</strong>")
	httpmock.RegisterResponder("GET", "https://check.torproject.org/",
		httpmock.NewStringResponder(200, page))

	ip, err := getTorIP(http.DefaultClient)
	if err != nil {
		t.Error(err)
	}
	httpmock.DeactivateAndReset()

	if ip != "Random IP Address" {
		t.Error("The IP address was not successfully extracted.")
	}
}

func TestGetEmails(t *testing.T) {
	link := "https://random.com"
	httpmock.Activate()
	page := newPage("Random Site", `<a href="mailto:random@gmail.com">Email me</a>`)
	httpmock.RegisterResponder("GET", link,
		httpmock.NewStringResponder(200, page))

	emails := getEmails(http.DefaultClient, link)
	httpmock.DeactivateAndReset()

	if len(emails) != 1 || emails[0] != "random@gmail.com" {
		t.Error("The email address was not successfully extracted.")
	}

	httpmock.Activate()
	page = newPage("Random Site", `<a href="mailto:random@gmail.com">email me</a>
					<a href="mailto:random@yahoo.com">email me</a>
					<a href="mailto:random@protonmail.com">email me</a>
					<a href="mailto:random@outlook.com">email me</a>`)
	httpmock.RegisterResponder("GET", link,
		httpmock.NewStringResponder(200, page))

	emails = getEmails(http.DefaultClient, link)
	httpmock.DeactivateAndReset()

	assert.Contains(t, emails, "random@gmail.com", "Gmail address not parsed")
	assert.Contains(t, emails, "random@yahoo.com", "Yahoo address not parsed")
	assert.Contains(t, emails, "random@protonmail.com", "Protonmail address not parsed")
	assert.Contains(t, emails, "random@outlook.com", "Outlook address not parsed")
	if len(emails) != 4 {
		t.Error("The email address was not successfully extracted.")
	}
}
