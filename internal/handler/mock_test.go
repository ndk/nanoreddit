package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/render"
	"github.com/stretchr/testify/mock"

	"nanoreddit/internal/chi_utils"
	"nanoreddit/internal/validation"
	"nanoreddit/pkg/protocol"
)

type mockBinder struct {
	m *mock.Mock
}

func (m *mockBinder) Bind(w http.ResponseWriter, r *http.Request, request render.Binder) error {
	args := m.m.Called(w, r, request)
	return args.Error(0)
}

///////////////////////////////////////////////////////////////////////////////

type mockRender struct {
	m *mock.Mock
}

func (m *mockRender) InvalidRequest(w http.ResponseWriter, r *http.Request, err error) {
	m.m.Called(w, r, err)
}

func (m *mockRender) InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	m.m.Called(w, r, err)
}

///////////////////////////////////////////////////////////////////////////////

type mockStorage struct {
	m *mock.Mock
}

func (m *mockStorage) AddPost(ctx context.Context, post *protocol.Post) error {
	args := m.m.Called(ctx, post)
	return args.Error(0)
}

///////////////////////////////////////////////////////////////////////////////

func mockHandler(m *mock.Mock) (*handler, error) {
	validateStruct, err := validation.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("Couldn't create a validator: %w", err)
	}

	render := chi_utils.NewRender()
	binder := chi_utils.NewBinder(validateStruct, render.InvalidRequest)

	return &handler{
		binder:  binder,
		render:  render,
		storage: &mockStorage{m: m},
	}, nil
}
