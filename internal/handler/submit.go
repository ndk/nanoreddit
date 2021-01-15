package handler

import (
	"net/http"

	"github.com/rs/zerolog"

	"nanoreddit/pkg/protocol"
)

func (h *handler) Submit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var request protocol.SubmitRequest
	if err := h.binder.Bind(w, r, &request); err != nil {
		return
	}

	if err := h.storage.AddPost(ctx, &request.Post); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("Couldn't publish a request")
		h.render.InternalServerError(w, r, err)
	}
}
