package api

import (
	"fmt"
	"testing"

	"net/http"

	"github.com/KingAkeem/gotor/pkg/linktree"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func assertNode(t *testing.T, n linktree.Node, link string, numChildren int) {
	assert.Len(t, n.Children, numChildren, "There should be a single child.")
	assert.Equal(t, n.Status, "OK", "The status should be OK.")
	assert.Equal(t, n.StatusCode, 200, "The status code should be 200.")
	assert.Equal(t, n.URL, link, fmt.Sprintf("Node URL should be %s", link))
}

func newPage(title, body string) string {
	baseHTML := `
		<!DOCTYPE html>
		<html lang=en-US class=no-js>
			<head>
				<title>%s</title>
			</head>
			<body>%s</body>
		</html>
	`
	return fmt.Sprintf(baseHTML, title, body)
}

func TestGetTorIP(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	ip, err := getTorIP(http.DefaultClient)
	// an error is expected since nothing has been registered yet
	assert.NotNil(t, err)
	assert.Equal(t, ip, "", "There should be no IP address.")

	page := newPage("Tor Project", "<strong>Random IP Address</strong>")
	httpmock.RegisterResponder("GET", "https://check.torproject.org/", httpmock.NewStringResponder(200, page))
	ip, err = getTorIP(http.DefaultClient)
	assert.Nil(t, err)
	assert.Equal(t, ip, "Random IP Address", "The IP address was not successfully extracted.")

	page = newPage("Tor Project", "")
	httpmock.RegisterResponder("GET", "https://check.torproject.org/", httpmock.NewStringResponder(200, page))
	ip, err = getTorIP(http.DefaultClient)
	assert.Nil(t, err)
	assert.Equal(t, ip, "", "There should be no IP address.")
}

func TestGetEmails(t *testing.T) {
	link := "https://www.random.com"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	page := newPage("Random Site", `<a href="mailto:random@gmail.com">Email me</a>`)
	httpmock.RegisterResponder("GET", link, httpmock.NewStringResponder(200, page))

	emails := getEmails(http.DefaultClient, link)
	assert.Contains(t, emails, "random@gmail.com", "random@gmail.com should be in emails slice")
	assert.Len(t, emails, 1, "Found more than 1 email address.")

	page = newPage("Random Site", `<a href="mailto:random@gmail.com">email me</a>
					<a href="mailto:random@yahoo.com">email me</a>
					<a href="mailto:random@protonmail.com">email me</a>
					<a href="mailto:random@outlook.com">email me</a>`)
	httpmock.RegisterResponder("GET", link, httpmock.NewStringResponder(200, page))

	emails = getEmails(http.DefaultClient, link)
	assert.ElementsMatch(t, emails, []string{"random@gmail.com", "random@protonmail.com", "random@outlook.com", "random@yahoo.com"})
}

func TestGetPhoneNumbers(t *testing.T) {
	link := "https://www.random.com"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	page := newPage("Random Site", `<a href="tel:+1-555-555-5555">Call me</a>`)
	httpmock.RegisterResponder("GET", link, httpmock.NewStringResponder(200, page))
	number := getPhoneNumbers(http.DefaultClient, link)
	assert.Contains(t, number, "+1-555-555-5555", "The phone number should be in the slice.")
	assert.Len(t, number, 1, "There should be only one phone number.")

	page = newPage("Random Site", `<a href="tel:+1-555-555-5555">call me</a>
					<a href="tel:+1-555-555-5556">call me</a>
					<a href="tel:+1-555-555-5557">call me</a>
					<a href="tel:+1-555-555-5558">call me</a>`)
	httpmock.RegisterResponder("GET", link, httpmock.NewStringResponder(200, page))
	numbers := getPhoneNumbers(http.DefaultClient, link)
	assert.ElementsMatch(t, numbers, []string{"+1-555-555-5555", "+1-555-555-5556", "+1-555-555-5557", "+1-555-555-5558"})
}

func TestGetTree(t *testing.T) {
	// Test getting a tree of depth 1
	rootLink := "https://www.root.com"
	childLink := "https://www.child.com"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	page := newPage("Tree Site", fmt.Sprintf(`<a href="%s">Child Site</a>`, childLink))
	httpmock.RegisterResponder("GET", childLink, httpmock.NewStringResponder(200, newPage("Child Site", "")))
	httpmock.RegisterResponder("GET", rootLink, httpmock.NewStringResponder(200, page))

	node := linktree.NewNode(http.DefaultClient, rootLink)
	node.Load(1)
	assertNode(t, *node, rootLink, 1)

	// Test getting a tree of depth 2
	rootLink = "https://www.root.com"
	childLink = "https://www.child.com"
	subChildLink := "https://www.subchild.com"
	page = newPage("Tree Site", fmt.Sprintf(`<a href="%s">Child Site</a>`, childLink))
	childPage := newPage("Tree Site", fmt.Sprintf(`<a href="%s">Sub Child Site</a>`, subChildLink))
	httpmock.RegisterResponder("GET", subChildLink, httpmock.NewStringResponder(200, newPage("Sub Child Site", "")))
	httpmock.RegisterResponder("GET", childLink, httpmock.NewStringResponder(200, childPage))
	httpmock.RegisterResponder("GET", rootLink, httpmock.NewStringResponder(200, page))

	node = linktree.NewNode(http.DefaultClient, rootLink)
	node.Load(2)

	assertNode(t, *node, rootLink, 1)
	assertNode(t, *node.Children[0], childLink, 1)
	assertNode(t, *node.Children[0].Children[0], subChildLink, 0)
}

func TestGetWebsiteContent(t *testing.T) {
	link := "https://www.random.com"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	content, err := getWebsiteContent(http.DefaultClient, link)
	assert.NotNil(t, err)
	assert.Equal(t, "", content)

	httpmock.RegisterResponder("GET", link, httpmock.NewStringResponder(200, "Hello World"))

	content, err = getWebsiteContent(http.DefaultClient, link)
	assert.Nil(t, err)
	assert.Equal(t, "Hello World", content)
}
