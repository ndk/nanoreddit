package chi_utils

import (
	"net/http"

	"github.com/go-chi/render"
)

type requestBinder struct {
	binder         func(r *http.Request, v render.Binder) error
	validate       func(v interface{}) error
	invalidRequest func(w http.ResponseWriter, r *http.Request, err error)
}

func (rb *requestBinder) Bind(w http.ResponseWriter, r *http.Request, request render.Binder) error {
	if err := rb.binder(r, request); err != nil {
		rb.invalidRequest(w, r, err)
		return err
	}
	if err := rb.validate(request); err != nil {
		rb.invalidRequest(w, r, err)
		return err
	}
	return nil
}

func NewBinder(validateStruct func(s interface{}) error, invalidRequest func(w http.ResponseWriter, r *http.Request, err error)) *requestBinder {
	return &requestBinder{
		binder:         render.Bind,
		validate:       validateStruct,
		invalidRequest: invalidRequest,
	}
}
