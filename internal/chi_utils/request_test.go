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
)

type mockRequest struct {
	m *mock.Mock

	Email string `json:"email"`
	Phone string `json:"phone"`
}

func (m *mockRequest) Bind(r *http.Request) error {
	args := m.m.Called(r)
	return args.Error(0)
}

func TestRequestBinder(t *testing.T) {
	Convey("Test requestBinder", t, func() {
		m := &mock.Mock{}

		rb := &requestBinder{
			binder: func(r *http.Request, v render.Binder) error {
				args := m.MethodCalled("Bind", r, v)
				return args.Error(0)
			},
			validate: func(v interface{}) error {
				args := m.MethodCalled("Validate", v)
				return args.Error(0)
			},
			invalidRequest: func(w http.ResponseWriter, r *http.Request, err error) {
				m.MethodCalled("InvalidRequest", w, r, err)
			},
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/blablabla", nil)
		r = r.WithContext(context.Background())

		Convey("It fails if the binder fails", func() {
			m.
				On("Bind", r, nil).Return(errors.New("binder error")).
				On("InvalidRequest", w, r, errors.New("binder error")).Run(func(args mock.Arguments) { w.WriteHeader(http.StatusBadRequest) })

			err := rb.Bind(w, r, nil)

			So(err, ShouldBeError, "binder error")
			So(m.AssertExpectations(t), ShouldBeTrue)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
			result := w.Result()
			defer result.Body.Close()
			b, err := ioutil.ReadAll(result.Body)
			So(err, ShouldBeNil)
			So(b, ShouldBeEmpty)
		})

		Convey("It fails if the validation fails", func() {
			m.
				On("Bind", r, nil).Return(nil).
				On("Validate", nil).Return(errors.New("validation error")).
				On("InvalidRequest", w, r, errors.New("validation error")).Run(func(args mock.Arguments) { w.WriteHeader(http.StatusBadRequest) })

			err := rb.Bind(w, r, nil)

			So(err, ShouldBeError, "validation error")
			So(m.AssertExpectations(t), ShouldBeTrue)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
			result := w.Result()
			defer result.Body.Close()
			b, err := ioutil.ReadAll(result.Body)
			So(err, ShouldBeNil)
			So(b, ShouldBeEmpty)
		})

		Convey("Successful way", func() {
			request := mockRequest{}

			m.
				On("Bind", mock.Anything, &request).Return(nil).Run(
				func(args mock.Arguments) {
					request := args.Get(1).(*mockRequest)
					request.Email = "devnull@neverhood.net"
					request.Phone = "+79091112233"
				},
			).
				On("Validate", &request).Return(nil)

			err := rb.Bind(w, r, &request)

			So(err, ShouldBeNil)
			So(m.AssertExpectations(t), ShouldBeTrue)
			So(w.Code, ShouldEqual, http.StatusOK)
			result := w.Result()
			defer result.Body.Close()
			b, err := ioutil.ReadAll(result.Body)
			So(err, ShouldBeNil)
			So(b, ShouldBeEmpty)
			So(request.Email, ShouldEqual, "devnull@neverhood.net")
			So(request.Phone, ShouldEqual, "+79091112233")
		})
	})
}

func TestNewBinder(t *testing.T) {
	Convey("Test NewBinder", t, func() {
		validateStruct := func(s interface{}) error { return nil }
		invalidRequest := func(w http.ResponseWriter, r *http.Request, err error) {}

		rb := NewBinder(validateStruct, invalidRequest)

		So(rb, ShouldNotBeNil)
		So(rb.validate, ShouldNotBeNil)
		So(runtime.FuncForPC(reflect.ValueOf(rb.binder).Pointer()).Name(), ShouldEqual, runtime.FuncForPC(reflect.ValueOf(render.Bind).Pointer()).Name())
		So(runtime.FuncForPC(reflect.ValueOf(rb.validate).Pointer()).Name(), ShouldEqual, runtime.FuncForPC(reflect.ValueOf(validateStruct).Pointer()).Name())
		So(runtime.FuncForPC(reflect.ValueOf(rb.invalidRequest).Pointer()).Name(), ShouldEqual, runtime.FuncForPC(reflect.ValueOf(invalidRequest).Pointer()).Name())
	})
}
