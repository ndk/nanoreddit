package chi_utils

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"testing"

	"github.com/go-chi/render"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"

	"nanoreddit/pkg/protocol"
)

func TestErrResponse(t *testing.T) {
	Convey("Test ErrResponse", t, func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/blablabla", nil)
		r = r.WithContext(context.Background())

		er := &errResponse{
			HTTPStatusCode: http.StatusBadRequest,
			ErrorResponse: protocol.ErrorResponse{
				Errors: []protocol.Error{
					{
						Code:        http.StatusBadRequest,
						Description: "my error",
					},
				},
			},
		}

		err := er.Render(w, r)

		So(err, ShouldBeNil)
		status, ok := r.Context().Value(render.StatusCtxKey).(int)
		So(ok, ShouldBeTrue)
		So(status, ShouldEqual, http.StatusBadRequest)
		result := w.Result()
		defer result.Body.Close()
		b, err := ioutil.ReadAll(result.Body)
		So(err, ShouldBeNil)
		So(b, ShouldBeEmpty)
	})
}

func TestResponseRender(t *testing.T) {
	Convey("Test responseRender", t, func() {
		m := &mock.Mock{}

		rr := &responseRender{
			r: func(w http.ResponseWriter, r *http.Request, v render.Renderer) error {
				args := m.MethodCalled("Render", w, r, v)
				return args.Error(0)
			},
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/blablabla", nil)
		r = r.WithContext(context.Background())

		Convey("It doesn't render anything if the render fails", func() {
			m.On("Render", w, r, nil).Return(errors.New("render error"))

			rr.render(w, r, nil)

			So(m.AssertExpectations(t), ShouldBeTrue)
			result := w.Result()
			defer result.Body.Close()
			b, err := ioutil.ReadAll(result.Body)
			So(err, ShouldBeNil)
			So(b, ShouldBeEmpty)
		})

		Convey("InvalidRequest", func() {
			er := &errResponse{
				HTTPStatusCode: http.StatusBadRequest,
				ErrorResponse: protocol.ErrorResponse{
					Errors: []protocol.Error{
						{
							Code:        http.StatusBadRequest,
							Description: "my error",
						},
					},
				},
			}
			m.On("Render", w, r, er).Return(nil).Run(func(args mock.Arguments) { w.WriteHeader(er.HTTPStatusCode) })

			rr.InvalidRequest(w, r, errors.New("my error"))

			So(m.AssertExpectations(t), ShouldBeTrue)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
			result := w.Result()
			defer result.Body.Close()
			b, err := ioutil.ReadAll(result.Body)
			So(err, ShouldBeNil)
			So(b, ShouldBeEmpty)
		})

		Convey("InternalServerError", func() {
			er := &errResponse{
				HTTPStatusCode: http.StatusInternalServerError,
				ErrorResponse: protocol.ErrorResponse{
					Errors: []protocol.Error{
						{
							Code:        http.StatusInternalServerError,
							Description: http.StatusText(http.StatusInternalServerError),
						},
					},
				},
			}
			m.On("Render", w, r, er).Return(nil).Run(func(args mock.Arguments) { w.WriteHeader(er.HTTPStatusCode) })

			rr.InternalServerError(w, r, errors.New("my error"))

			So(m.AssertExpectations(t), ShouldBeTrue)
			So(w.Code, ShouldEqual, http.StatusInternalServerError)
			result := w.Result()
			defer result.Body.Close()
			b, err := ioutil.ReadAll(result.Body)
			So(err, ShouldBeNil)
			So(b, ShouldBeEmpty)
		})
	})
}

func TestNewRender(t *testing.T) {
	Convey("Test NewRender", t, func() {
		rr := NewRender()

		So(rr, ShouldNotBeNil)
		So(runtime.FuncForPC(reflect.ValueOf(rr.r).Pointer()).Name(), ShouldEqual, runtime.FuncForPC(reflect.ValueOf(render.Render).Pointer()).Name())
	})
}
