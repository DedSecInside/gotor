package parse

import (
	"net/http"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

// extractLinks processes a <link> HTML node and updates the PageMeta struct accordingly.
func extractLinks(n *html.Node, pm *PageMeta, base *url.URL) {
	rel, _ := getAttr(n, "rel")
	href, _ := getAttr(n, "href")
	if href == "" {
		return
	}
	abs := resolve(base, href)
	switch strings.ToLower(rel) {
	case "canonical":
		if pm.Canonical == "" {
			pm.Canonical = abs
		}
	case "alternate":
		// feeds or hreflang
		t, _ := getAttr(n, "type")
		if strings.Contains(strings.ToLower(t), "rss") || strings.Contains(strings.ToLower(t), "atom") {
			pm.Feeds = append(pm.Feeds, abs)
		}
		if hl, ok := getAttr(n, "hreflang"); ok {
			pm.Hreflang[strings.ToLower(hl)] = abs
		}
	case "next", "prev":
		pm.Pagination[strings.ToLower(rel)] = abs
	case "icon", "shortcut icon":
		pm.Favicons = append(pm.Favicons, abs)
	}
}

// extractMeta processes a <meta> HTML node and updates the PageMeta struct accordingly.
func extractMeta(n *html.Node, pm *PageMeta) {
	name, _ := getAttr(n, "name")
	prop, _ := getAttr(n, "property")
	content, _ := getAttr(n, "content")
	if content == "" {
		return
	}

	switch strings.ToLower(name) {
	case "description":
		if pm.Description == "" {
			pm.Description = content
		}
	case "robots", "googlebot":
		for _, t := range strings.Split(content, ",") {
			pm.Robots = append(pm.Robots, strings.TrimSpace(strings.ToLower(t)))
		}
	case "twitter:card", "twitter:title", "twitter:description", "twitter:image":
		pm.Twitter[strings.ToLower(name)] = content
	}

	if strings.HasPrefix(strings.ToLower(prop), "og:") {
		pm.OG[strings.ToLower(prop)] = content
	}
}

// walkDOM traverses the HTML node tree and extracts metadata into the PageMeta struct.
func walkDOM(doc *html.Node, pm *PageMeta, base *url.URL) {
	var h1Found bool
	emails := map[string]struct{}{}
	phoneNumbers := map[string]struct{}{}

	eachNode(doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}

		switch n.Data {
		case "html":
			if pm.Lang == "" {
				if lang, ok := getAttr(n, "lang"); ok {
					pm.Lang = strings.ToLower(strings.TrimSpace(lang))
				}
			}

		case "a":
			href, _ := getAttr(n, "href")
			href = strings.TrimSpace(href)
			lowerHref := strings.ToLower(href)
			switch {
			case strings.HasPrefix(lowerHref, "mailto:"):
				value := strings.TrimSpace(href[len("mailto:"):])
				if query := strings.IndexByte(value, '?'); query >= 0 {
					value = value[:query]
				}
				if value != "" {
					emails[value] = struct{}{}
				}
			case strings.HasPrefix(lowerHref, "tel:"):
				value := strings.TrimSpace(href[len("tel:"):])
				if value != "" {
					phoneNumbers[value] = struct{}{}
				}
			}

		case "title":
			if pm.Title == "" {
				pm.Title = strings.TrimSpace(textOf(n))
			}

		case "h1":
			if !h1Found {
				pm.H1 = strings.TrimSpace(textOf(n))
				h1Found = true
			}

		case "meta":
			extractMeta(n, pm)
		case "link":
			extractLinks(n, pm, base)
		case "script":
			t, _ := getAttr(n, "type")
			if strings.EqualFold(t, "application/ld+json") && n.FirstChild != nil {
				raw := strings.TrimSpace(n.FirstChild.Data)
				if raw != "" {
					pm.JSONLD = append(pm.JSONLD, raw)
				}
			}
		}
	})

	for email := range emails {
		pm.Emails = append(pm.Emails, email)
	}
	for phoneNumber := range phoneNumbers {
		pm.PhoneNumbers = append(pm.PhoneNumbers, phoneNumber)
	}
	sort.Strings(pm.Emails)
	sort.Strings(pm.PhoneNumbers)
}

// ExtractPageMeta extracts metadata from the HTTP response and parsed HTML document.
func ExtractPageMeta(resp *http.Response, base *url.URL, doc *html.Node) PageMeta {
	pm := PageMeta{
		URL:           base.String(),
		Status:        resp.StatusCode,
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: resp.ContentLength,
		OG:            map[string]string{},
		Twitter:       map[string]string{},
		Hreflang:      map[string]string{},
		Pagination:    map[string]string{},
	}

	// Headers first (robots)
	if xr := resp.Header.Get("X-Robots-Tag"); xr != "" {
		for _, t := range strings.Split(xr, ",") {
			pm.Robots = append(pm.Robots, strings.TrimSpace(strings.ToLower(t)))
		}
	}

	// walks DOM and fills PageMeta
	walkDOM(doc, &pm, base)

	// Fallback canonical
	if pm.Canonical == "" {
		if ogu := pm.OG["og:url"]; ogu != "" {
			pm.Canonical = resolve(base, ogu)
		}
	}
	// Choose primary image
	if pm.OG["og:image"] != "" {
		pm.PrimaryImage = resolve(base, pm.OG["og:image"])
	} else if pm.Twitter["twitter:image"] != "" {
		pm.PrimaryImage = resolve(base, pm.Twitter["twitter:image"])
	}
	return pm
}
