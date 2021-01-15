package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/render"

	"nanoreddit/internal/chi_utils"
	"nanoreddit/internal/validation"
	"nanoreddit/pkg/protocol"
)

type requestBinder interface {
	Bind(w http.ResponseWriter, r *http.Request, request render.Binder) error
}

type responseRender interface {
	InvalidRequest(w http.ResponseWriter, r *http.Request, err error)
	InternalServerError(w http.ResponseWriter, r *http.Request, err error)
}

type storage interface {
	AddPost(ctx context.Context, post *protocol.Post) error
	GetFeed(ctx context.Context, page int) ([]protocol.Post, error)
}

///////////////////////////////////////////////////////////////////////////////

type handler struct {
	binder  requestBinder
	render  responseRender
	storage storage
}

func NewHandler(storage storage) (*handler, error) {
	validateStruct, err := validation.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("couldn't create a validator: %w", err)
	}
	render := chi_utils.NewRender()
	binder := chi_utils.NewBinder(validateStruct, render.InvalidRequest)

	return &handler{
		render:  render,
		binder:  binder,
		storage: storage,
	}, nil
}
