package api

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"net/http"
)

func TestGetTorIP(t *testing.T) {
	myMockPage := `<!DOCTYPE html>
		<html lang=en-US class=no-js>
			<head><title>Tor Project</title></head>
			<body><strong>Random IP Address</strong></body>
		</html>`

	httpmock.Activate()
	httpmock.RegisterResponder("GET", "https://check.torproject.org/",
		httpmock.NewStringResponder(200, myMockPage))

	ip, err := getTorIP(http.DefaultClient)
	if err != nil {
		t.Error(err)
	}
	httpmock.DeactivateAndReset()

	if ip != "Random IP Address" {
		t.Error("The IP address was not succesfully extracted.")
	}
}

func TestGetEmails(t *testing.T) {
	myMockPage := `<!DOCTYPE html>
		<html lang=en-US class=no-js>
			<head><title>Tor Project</title></head>
			<body><a href="mailto:random@gmail.com">Email me</a></body>
		</html>`

	link := "https://random.com"
	httpmock.Activate()
	httpmock.RegisterResponder("GET", link,
		httpmock.NewStringResponder(200, myMockPage))

	emails := getEmails(http.DefaultClient, link)
	httpmock.DeactivateAndReset()

	if len(emails) != 1 || emails[0] != "random@gmail.com" {
		t.Error("The email address was not succesfully extracted.")
	}
}
