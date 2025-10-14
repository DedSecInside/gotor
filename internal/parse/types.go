package parse

import (
	"encoding/json"
)

// PageMeta holds metadata extracted from a web page.
type PageMeta struct {
	URL           string
	Canonical     string
	Title         string
	Description   string
	H1            string
	Status        int
	ContentType   string
	ContentLength int64

	Robots     []string // merged X-Robots-Tag + meta robots, split by comma
	Lang       string
	Hreflang   map[string]string // lang -> URL
	Pagination map[string]string // "next"/"prev" -> URL

	OG      map[string]string // og:title, og:desc, og:image, og:type, og:url
	Twitter map[string]string // twitter:card/title/desc/image
	JSONLD  []string          // raw JSON-LD blobs

	PrimaryImage string   // pick best from og:image/images
	Feeds        []string // rss/atom links
	Favicons     []string
}

func (pm PageMeta) String() string {
	b, _ := json.MarshalIndent(pm, "", "  ")
	return string(b)
}
