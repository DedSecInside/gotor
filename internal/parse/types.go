package parse

import (
	"encoding/json"
)

// PageMeta holds metadata extracted from a web page.
type PageMeta struct {
	URL           string `json:"url"`
	Canonical     string `json:"canonical,omitempty"`
	Title         string `json:"title,omitempty"`
	Description   string `json:"description,omitempty"`
	H1            string `json:"h1,omitempty"`
	Status        int    `json:"status"`
	ContentType   string `json:"contentType,omitempty"`
	ContentLength int64  `json:"contentLength,omitempty"`

	Robots     []string          `json:"robots,omitempty"` // merged X-Robots-Tag + meta robots, split by comma
	Lang       string            `json:"lang,omitempty"`
	Hreflang   map[string]string `json:"hreflang,omitempty"`   // lang -> URL
	Pagination map[string]string `json:"pagination,omitempty"` // "next"/"prev" -> URL

	OG      map[string]string `json:"openGraph,omitempty"` // og:title, og:desc, og:image, og:type, og:url
	Twitter map[string]string `json:"twitter,omitempty"`   // twitter:card/title/desc/image
	JSONLD  []string          `json:"jsonLd,omitempty"`    // raw JSON-LD blobs

	PrimaryImage string   `json:"primaryImage,omitempty"` // pick best from og:image/images
	Feeds        []string `json:"feeds,omitempty"`        // rss/atom links
	Favicons     []string `json:"favicons,omitempty"`
	Emails       []string `json:"emails,omitempty"`
	PhoneNumbers []string `json:"phoneNumbers,omitempty"`
}

func (pm PageMeta) String() string {
	b, _ := json.MarshalIndent(pm, "", "  ")
	return string(b)
}
