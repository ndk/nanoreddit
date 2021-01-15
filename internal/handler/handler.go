package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"

	"nanoreddit/internal/chi_utils"
	"nanoreddit/internal/validation"
)

type requestBinder interface {
	Bind(w http.ResponseWriter, r *http.Request, request render.Binder) error
}

type responseRender interface {
	InvalidRequest(w http.ResponseWriter, r *http.Request, err error)
	InternalServerError(w http.ResponseWriter, r *http.Request, err error)
}

///////////////////////////////////////////////////////////////////////////////

type handler struct {
	binder  requestBinder
	render  responseRender
}

func NewHandler() (*handler, error) {
	validateStruct, err := validation.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("Couldn't create a validator: %w", err)
	}
	render := chi_utils.NewRender()
	binder := chi_utils.NewBinder(validateStruct, render.InvalidRequest)

	return &handler{
		render:  render,
		binder:  binder,
	}, nil
}
