package handler

import (
	"net/http"

	"github.com/go-chi/render"

	"nanoreddit/pkg/protocol"
)

func (h *handler) Feed(w http.ResponseWriter, r *http.Request) {
	var feed []protocol.Post
	render.Respond(w, r, feed)
}
