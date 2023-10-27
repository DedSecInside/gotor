package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/DedSecInside/gotor/internal/logger"
	"github.com/DedSecInside/gotor/pkg/linktree"
)

// GetTreeNode returns a LinkTree with the specified depth passed to the query parameter.
func (s Server) handleGetTreeNode(w http.ResponseWriter, r *http.Request) {
	depthInput := r.URL.Query().Get("depth")
	depth, err := strconv.Atoi(depthInput)
	if err != nil {
		msg := "invalid depth, must be an integer"
		logger.Error(msg, "error", err.Error())
		w.Write([]byte(msg))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	link := r.URL.Query().Get("link")
	if link == "" {
		logger.Error("found blank link")
		w.Write([]byte("Link cannot be blank."))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger.Info("attempting to build new tree from request",
		"root", link,
		"depth", depth,
	)
	node := linktree.NewNode(s.client, link)
	node.Load(depth)
	logger.Info("build successful",
		"node", node,
	)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(node)
	if err != nil {
		logger.Error("unable to marshal node",
			"error", err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
