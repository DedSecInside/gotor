package linktree

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func newPage(title, body string) string {
	baseHTML := `<!DOCTYPE html>
		<html lang=en-US class=no-js>
			<head><title>%s</title></head>
			<body>%s</body>
	</html>`
	return fmt.Sprintf(baseHTML, title, body)
}

func TestNewNode(t *testing.T) {
	httpmock.Activate()

	link := "https://www.random.com"
	n := NewNode(http.DefaultClient, link)
	assert.Equal(t, n.URL, link)
	// no response has been registered yet
	assert.Equal(t, n.Status, http.StatusText(http.StatusInternalServerError))
	assert.Equal(t, n.StatusCode, http.StatusInternalServerError)

	httpmock.RegisterResponder(http.MethodGet, link, httpmock.NewStringResponder(http.StatusOK, newPage("Random", "")))
	n.updateStatus() // update status now that link has been registered
	assert.Equal(t, n.URL, link)
	assert.Equal(t, n.Status, http.StatusText(http.StatusOK))
	assert.Equal(t, n.StatusCode, http.StatusOK)

	httpmock.DeactivateAndReset()
}

func TestIsValidURL(t *testing.T) {
	assert.True(t, isValidURL("https://www.example.com"))
	assert.True(t, isValidURL("http://www.example.com"))
	assert.True(t, isValidURL("https://www.example.org"))
	assert.False(t, isValidURL("htts://www.example.org"))
	assert.False(t, isValidURL("ws://ww.exampleg"))
	assert.False(t, isValidURL(""))
	assert.False(t, isValidURL("a"))
	assert.False(t, isValidURL("ddd"))
}

func TestLoadNode(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	link := "https://www.test.com"
	n := NewNode(http.DefaultClient, link)
	n.Load(1)
	assert.True(t, n.loaded)

	childLink := "https://www.child1.com"
	page := newPage("test", fmt.Sprintf(`<a href="%s">link to child</a>`, childLink))
	httpmock.RegisterResponder(http.MethodGet, link, httpmock.NewStringResponder(http.StatusOK, page))

	n = NewNode(http.DefaultClient, link)
	n.Load(1)
	assert.True(t, n.loaded)
	assert.Len(t, n.Children, 1)
	assert.Equal(t, n.Children[0].URL, childLink)

}

func TestCrawlNode(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	link := "https://www.test.com"
	page := newPage("test", `<a href="https://www.child1.com">link to child</a>`)
	httpmock.RegisterResponder(http.MethodGet, link, httpmock.NewStringResponder(http.StatusOK, page))

	n := NewNode(http.DefaultClient, link)
	n.Crawl(1, func(link string) {
		assert.Equal(t, link, "https://www.child1.com")
	})

	assert.Len(t, n.Children, 0) // nothing should be stored in memory
}
