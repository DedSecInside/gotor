package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/mgutz/ansi"
	progressbar "github.com/schollz/progressbar/v3"
	"golang.org/x/net/html"
)

// TableWriter can write to a table and display the output using the flush method
type TableWriter interface {
	Write([]byte) (int, error)
	Flush() error
}

// TableConfig ...
type TableConfig struct {
	MinWidth int
	TabWidth int
	Padding  int
}

// Table ...
type Table struct {
	Writer TableWriter
	Rows   [][]string
}

func newTable(c *TableConfig) *Table {
	writer := tabwriter.NewWriter(os.Stdout, c.MinWidth, c.TabWidth, c.Padding, byte('\t'), tabwriter.Debug)
	return &Table{
		Writer: writer,
		Rows:   make([][]string, 0),
	}
}

// AddRow ...
func (t *Table) AddRow(cellValues []string) error {
	t.Rows = append(t.Rows, cellValues)
	return nil
}

func (t *Table) formatRow(row []string) string {
	formattedRow := ""
	for _, cell := range row {
		formattedRow += cell + "\t"
	}
	return formattedRow
}

// Display ...
func (t *Table) Display() error {
	for _, row := range t.Rows {
		formattedRow := t.formatRow(row)
		fmt.Fprintln(t.Writer, formattedRow)
	}
	return t.Writer.Flush()
}

// create a simple tor client, this can be modified to allow the user to
// set their address and port at some point
func createTorClient() (*http.Client, error) {
	proxyStr := "socks5://127.0.0.1:9050"
	proxyUrl, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyUrl),
	}
	return &http.Client{
		Transport: transport,
	}, nil
}

// parses the links from a reader
func parseLinks(r io.Reader) []string {
	links := make([]string, 0)
	tokenizer := html.NewTokenizer(r)

	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			return links
		case html.StartTagToken:
			token := tokenizer.Token()
			for _, attr := range token.Attr {
				if attr.Key == "href" {
					if u, err := url.ParseRequestURI(attr.Val); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
						links = append(links, attr.Val)
					}
				}
			}
		}
	}
}

func main() {
	var link string
	var depthInput string
	flag.StringVar(&link, "l", "", "Root used for searching. Requred.")
	flag.StringVar(&depthInput, "d", "1", "Depth of search. Defaults to 1.")
	flag.Parse()
	if link == "" {
		log.Fatal("-l (link) argument is required.")
		return
	}

	depth, err := strconv.Atoi(depthInput)
	if err != nil {
		log.Fatal("Invalid depth found. Depth: ", depth)
		return
	}

	client, err := createTorClient()
	if err != nil {
		log.Fatal(err)
		return
	}
	resp, err := client.Get(link)
	if err != nil {
		log.Fatal(err)
		return
	}

	links := parseLinks(resp.Body)
	bar := progressbar.NewOptions(int(len(links)), progressbar.OptionSetDescription("processing..."))
	markError := ansi.ColorFunc("red")
	markSuccess := ansi.ColorFunc("green")
	t := newTable(&TableConfig{
		MinWidth: 0,
		TabWidth: 0,
		Padding:  0,
	})
	t.AddRow([]string{"Link", "Status"})
	for _, link := range links {
		bar.Describe(fmt.Sprintf("processing %s", link))
		resp, err = client.Get(link)
		if err != nil {
			log.Fatal(err)
			return
		}
		status := fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		if resp.StatusCode != 200 {
			t.AddRow([]string{link, markError(status)})
		} else {
			t.AddRow([]string{link, markSuccess(status)})
		}
		bar.Add(1)
	}
	bar.IsFinished()
	bar.Clear()
	t.Display()
}
