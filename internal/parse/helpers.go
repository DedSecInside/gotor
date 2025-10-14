package parse

import (
	"bytes"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func parseHTML(body io.Reader) (*html.Node, error) {
	return html.Parse(body)
}

func getAttr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func eachNode(n *html.Node, fn func(*html.Node)) {
	var w func(*html.Node)
	w = func(c *html.Node) {
		fn(c)
		for k := c.FirstChild; k != nil; k = k.NextSibling {
			w(k)
		}
	}
	w(n)
}

func textOf(n *html.Node) string {
	var b bytes.Buffer
	var w func(*html.Node)
	w = func(c *html.Node) {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
		for k := c.FirstChild; k != nil; k = k.NextSibling {
			w(k)
		}
	}
	w(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

func resolve(base *url.URL, href string) string {
	u, err := base.Parse(strings.TrimSpace(href))
	if err != nil {
		return ""
	}
	u.Fragment = ""
	return u.String()
}
