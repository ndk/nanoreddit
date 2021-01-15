package handler

import (
	"net/http"

	"nanoreddit/pkg/protocol"
)

func (h *handler) Submit(w http.ResponseWriter, r *http.Request) {
	var request protocol.SubmitRequest
	if err := h.binder.Bind(w, r, &request); err != nil {
		return
	}
}
