package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/DedSecInside/gotor/pkg/linktree"
)

// GetTreeNode returns a LinkTree with the specified depth passed to the query parameter.
func (s Server) handleGetTreeNode(w http.ResponseWriter, r *http.Request) {
	depthInput := strings.TrimSpace(r.URL.Query().Get("depth"))
	var depth int
	var err error
	if depthInput == "" {
		depth = 1
	} else {
		depth, err = strconv.Atoi(depthInput)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid depth, must be an integer. Depth %s."))
			log.Printf("Invalid depth, must be an integer. Depth %s. Error: %+v\n", depthInput, err)
			return
		}
	}

	link := strings.TrimSpace(r.URL.Query().Get("link"))
	if link == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Link cannot be blank."))
		log.Println("Found blank link")
		return
	}

	node := linktree.NewNode(s.client, link)
	node.Load(depth)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(node)
	if err != nil {
		log.Printf("Unable to marshal node. Error: %+v.\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
