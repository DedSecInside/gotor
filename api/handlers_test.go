package api

import (
	"fmt"
	"testing"

	"net/http"

	"github.com/KingAkeem/gotor/linktree"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func assertNode(t *testing.T, n *linktree.Node, link string, numChildren int) {
	assert.Len(t, n.Children, numChildren, "There should be a single child.")
	assert.Equal(t, n.Status, "OK", "The status should be OK.")
	assert.Equal(t, n.StatusCode, 200, "The status code should be 200.")
	assert.Equal(t, n.URL, link, fmt.Sprintf("Node URL should be %s", link))
}

func newPage(title, body string) string {
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
	assert.Equal(t, ip, "Random IP Address", "The IP address was not successfully extracted.")

	httpmock.Activate()
	page = newPage("Tor Project", "")
	httpmock.RegisterResponder("GET", "https://check.torproject.org/",
		httpmock.NewStringResponder(200, page))

	ip, err = getTorIP(http.DefaultClient)
	if err != nil {
		t.Error(err)
	}
	httpmock.DeactivateAndReset()
	assert.Equal(t, ip, "", "There should be no IP address.")
}

func TestGetEmails(t *testing.T) {
	link := "https://www.random.com"
	httpmock.Activate()
	page := newPage("Random Site", `<a href="mailto:random@gmail.com">Email me</a>`)
	httpmock.RegisterResponder("GET", link,
		httpmock.NewStringResponder(200, page))

	emails := getEmails(http.DefaultClient, link)
	httpmock.DeactivateAndReset()

	assert.Contains(t, emails, "random@gmail.com", "random@gmail.com should be in emails slice")
	assert.Len(t, emails, 1, "Found more than 1 email address.")

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
	assert.Len(t, emails, 4, "The email address was not successfully extracted.")
}

func TestGetTree(t *testing.T) {
	// Test getting a tree of depth 1
	rootLink := "https://www.root.com"
	childLink := "https://www.child.com"
	httpmock.Activate()
	page := newPage("Tree Site", fmt.Sprintf(`<a href="%s">Child Site</a>`, childLink))
	httpmock.RegisterResponder("GET", childLink,
		httpmock.NewStringResponder(200, newPage("Child Site", "")))
	httpmock.RegisterResponder("GET", rootLink,
		httpmock.NewStringResponder(200, page))

	manager := linktree.NewNodeManager(http.DefaultClient)
	node := manager.LoadNode(rootLink, 1)
	httpmock.DeactivateAndReset()

	assertNode(t, node, rootLink, 1)

	// Test getting a tree of depth 2
	rootLink = "https://www.root.com"
	childLink = "https://www.child.com"
	subChildLink := "https://www.subchild.com"
	httpmock.Activate()
	page = newPage("Tree Site", fmt.Sprintf(`<a href="%s">Child Site</a>`, childLink))
	childPage := newPage("Tree Site", fmt.Sprintf(`<a href="%s">Sub Child Site</a>`, subChildLink))
	httpmock.RegisterResponder("GET", subChildLink,
		httpmock.NewStringResponder(200, newPage("Sub Child Site", "")))
	httpmock.RegisterResponder("GET", childLink,
		httpmock.NewStringResponder(200, childPage))
	httpmock.RegisterResponder("GET", rootLink,
		httpmock.NewStringResponder(200, page))

	manager = linktree.NewNodeManager(http.DefaultClient)
	node = manager.LoadNode(rootLink, 2)
	httpmock.DeactivateAndReset()

	assertNode(t, node, rootLink, 1)
	assertNode(t, node.Children[0], childLink, 1)
	assertNode(t, node.Children[0].Children[0], subChildLink, 0)
}
