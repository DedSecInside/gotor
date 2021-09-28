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
	assert.Equal(t, n.Status, "UNKNOWN")
	assert.Equal(t, n.StatusCode, http.StatusInternalServerError)

	page := newPage("Random", "")
	httpmock.RegisterResponder(http.MethodGet, link,
		httpmock.NewStringResponder(http.StatusOK, page))

	n.updateStatus()
	assert.Equal(t, n.URL, link)
	assert.Equal(t, n.Status, http.StatusText(http.StatusOK))
	assert.Equal(t, n.StatusCode, http.StatusOK)

	httpmock.DeactivateAndReset()
}

func TestLoadNode(t *testing.T) {
	httpmock.Activate()

	link := "https://www.test.com"
	n := NewNode(http.DefaultClient, link)
	n.Load(1)
	assert.True(t, n.Loaded)

	page := newPage("test", `<a href="https://www.child1.com">link to child</a>`)
	httpmock.RegisterResponder(http.MethodGet, link,
		httpmock.NewStringResponder(http.StatusOK, page))

	n = NewNode(http.DefaultClient, link)
	n.Load(1)
	assert.True(t, n.Loaded)
	assert.Len(t, n.Children, 1)

	httpmock.DeactivateAndReset()
}

func TestCrawlNode(t *testing.T) {
	httpmock.Activate()

	link := "https://www.test.com"
	page := newPage("test", `<a href="https://www.child1.com">link to child</a>`)
	httpmock.RegisterResponder(http.MethodGet, link,
		httpmock.NewStringResponder(http.StatusOK, page))

	n := NewNode(http.DefaultClient, link)
	n.Crawl(1, func(link string) {
		assert.Equal(t, link, "https://www.child1.com")
	})

	assert.Len(t, n.Children, 0) // nothing should be stored in memory

	httpmock.DeactivateAndReset()
}
