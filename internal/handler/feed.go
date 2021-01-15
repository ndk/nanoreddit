package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/render"
	"github.com/rs/zerolog"
)

func (h *handler) Feed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var page int
	{
		pageVal := r.FormValue("page")
		if pageVal != "" {
			v, err := strconv.Atoi(pageVal)
			if err != nil {
				h.render.InvalidRequest(w, r, fmt.Errorf("Couldn't recognize the page number: %w", err))
				return
			}
			page = v
		}
	}

	feed, err := h.storage.GetFeed(ctx, page)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("Couldn't fetch a feed")
		h.render.InternalServerError(w, r, err)
		return
	}

	render.Respond(w, r, feed)
}
