package chi_utils

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/rs/zerolog/hlog"

	"nanoreddit/pkg/protocol"
)

type errResponse struct {
	HTTPStatusCode int `json:"-"` // http response status code

	protocol.ErrorResponse
}

func (e *errResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

type responseRender struct {
	r func(w http.ResponseWriter, r *http.Request, v render.Renderer) error
}

func (rr *responseRender) render(w http.ResponseWriter, r *http.Request, response render.Renderer) {
	if err := rr.r(w, r, response); err != nil {
		hlog.FromRequest(r).Warn().Err(err).Msg("Couldn't render the response")
	}
}

func (rr *responseRender) InvalidRequest(w http.ResponseWriter, r *http.Request, err error) {
	rr.render(w, r, &errResponse{
		HTTPStatusCode: http.StatusBadRequest,
		ErrorResponse: protocol.ErrorResponse{
			Errors: []protocol.Error{
				{
					Code:        http.StatusBadRequest,
					Description: err.Error(),
				},
			},
		},
	})
}

func (rr *responseRender) InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	rr.render(w, r, &errResponse{
		HTTPStatusCode: http.StatusInternalServerError,
		ErrorResponse: protocol.ErrorResponse{
			Errors: []protocol.Error{
				{
					Code:        http.StatusInternalServerError,
					Description: http.StatusText(http.StatusInternalServerError),
				},
			},
		},
	})
}

func NewRender() *responseRender {
	return &responseRender{
		r: render.Render,
	}
}
