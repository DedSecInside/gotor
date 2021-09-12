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
			<body><strong>Random IP Address</strong>
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
