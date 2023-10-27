package linktree

import (
	"io"
	"log"
	"net/http"

	"github.com/DedSecInside/gotor/internal/logger"
	"golang.org/x/net/html"
)

// tokenFilter determines which tokens will be filtered from a stream,
// 1. There are zero to many attributes per tag.
// if the tag is included then those tags will be used (e.g. all anchor tags)
// if the attribute is included then those attributes will be used (e.g. all href attributes)
// if both are specified then the combination will be used (e.g. all href attributes within anchor tags only)
// if neither is specified then all tokens will be used (e.g. all tags found)
type tokenFilter struct {
	tags       map[string]bool
	attributes map[string]bool
}

// streams start tag tokens found within HTML content at the given link
func streamTokens(client *http.Client, link string) chan html.Token {
	TOKEN_CHAN_SIZE := 100
	logger.Debug("streaming tokens",
		"link", link,
		"channel size", TOKEN_CHAN_SIZE,
	)
	tokenStream := make(chan html.Token, TOKEN_CHAN_SIZE)
	go func() {
		defer close(tokenStream)
		resp, err := client.Get(link)
		if err != nil {
			logger.Info("unable to get html to tokenize", "link", link, "error", err)
			return
		}
		defer resp.Body.Close()

		tokenizer := html.NewTokenizer(resp.Body)
		for {
			tokenType := tokenizer.Next()
			switch tokenType {
			case html.ErrorToken:
				err := tokenizer.Err()
				if err != io.EOF {
					log.Println(err)
				}
				return
			case html.StartTagToken:
				token := tokenizer.Token()
				logger.Debug("queue token stream",
					"token", token,
				)
				tokenStream <- token
			}
		}
	}()
	return tokenStream
}

// filters tokens from the stream that do not pass the given tokenFilter
func filterTokens(tokenStream chan html.Token, filter *tokenFilter) chan string {
	FILTER_CHAN_SIZE := 10
	logger.Debug("Filtering tokens",
		"filter", filter,
		"channel size", FILTER_CHAN_SIZE,
	)
	filterStream := make(chan string, FILTER_CHAN_SIZE)

	filterAttributes := func(token html.Token) {
		// check if token passes filter
		for _, attr := range token.Attr {
			if _, foundAttribute := filter.attributes[attr.Key]; foundAttribute {
				logger.Debug("queue filter stream",
					"data", attr.Val,
				)
				filterStream <- attr.Val
			}
		}
	}

	go func() {
		defer close(filterStream)
		for token := range tokenStream {
			logger.Debug("dequeue token stream",
				"token", token,
			)
			if len(filter.tags) == 0 {
				logger.Debug("queue filter stream",
					"data", token.Data,
				)
				filterStream <- token.Data
			}

			// check if token passes tag filter or tag filter is empty
			if _, foundTag := filter.tags[token.Data]; foundTag {
				// emit attributes if there is a filter, otherwise emit token
				if len(filter.attributes) > 0 {
					filterAttributes(token)
				} else {
					logger.Debug("queue filter stream",
						"data", token.Data,
					)
					filterStream <- token.Data
				}
			}
		}
	}()

	return filterStream
}
