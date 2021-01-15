package handler

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/smartystreets/assertions"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"

	"nanoreddit/pkg/protocol"
)

func TestFeed(t *testing.T) {
	Convey("Test Feed", t, func() {
		m := &mock.Mock{}

		w := httptest.NewRecorder()
		w.Body = bytes.NewBuffer(nil)

		handler, err := mockHandler(m)
		So(err, ShouldBeNil)

		Convey("It fails if a page number is invalid", func() {
			req := httptest.NewRequest(http.MethodPost, "/feed?page=a", nil)
			req.Header.Add("Content-Type", "application/json")

			handler.Feed(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			So(resp.StatusCode, ShouldEqual, http.StatusBadRequest)
			So(m.AssertExpectations(t), ShouldBeTrue)
			resBbody, err := ioutil.ReadAll(resp.Body)
			So(err, ShouldBeNil)
			So(string(resBbody), assertions.ShouldEqualJSON, `{"errors":[{"code":400,"description":"couldn't recognize the page number: strconv.Atoi: parsing \"a\": invalid syntax"}]}`)
		})

		Convey("It passes a page number to the storage", func() {
			Convey("A page number is assumed zero if it's not specified", func() {
				req := httptest.NewRequest(http.MethodPost, "/feed", nil)
				req.Header.Add("Content-Type", "application/json")

				m.
					On("GetFeed", mock.Anything, 0).Return([]protocol.Post(nil), nil)

				handler.Feed(w, req)

				resp := w.Result()
				defer resp.Body.Close()
				So(resp.StatusCode, ShouldEqual, http.StatusOK)
				So(m.AssertExpectations(t), ShouldBeTrue)
				resBbody, err := ioutil.ReadAll(resp.Body)
				So(err, ShouldBeNil)
				So(string(resBbody), assertions.ShouldEqualJSON, `null`)
			})
			Convey("Otherwise the specified page numer will be used", func() {
				req := httptest.NewRequest(http.MethodPost, "/feed?page=123", nil)
				req.Header.Add("Content-Type", "application/json")

				m.
					On("GetFeed", mock.Anything, 123).Return([]protocol.Post(nil), nil)

				handler.Feed(w, req)

				resp := w.Result()
				defer resp.Body.Close()
				So(resp.StatusCode, ShouldEqual, http.StatusOK)
				So(m.AssertExpectations(t), ShouldBeTrue)
				resBbody, err := ioutil.ReadAll(resp.Body)
				So(err, ShouldBeNil)
				So(string(resBbody), assertions.ShouldEqualJSON, `null`)
			})
		})

		Convey("It fails if an storage has been failed", func() {
			req := httptest.NewRequest(http.MethodPost, "/feed?page=123", nil)
			req.Header.Add("Content-Type", "application/json")

			m.
				On("GetFeed", mock.Anything, 123).Return([]protocol.Post(nil), errors.New("storage error"))

			handler.Feed(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			So(resp.StatusCode, ShouldEqual, http.StatusInternalServerError)
			So(m.AssertExpectations(t), ShouldBeTrue)
			resBbody, err := ioutil.ReadAll(resp.Body)
			So(err, ShouldBeNil)
			So(string(resBbody), assertions.ShouldEqualJSON, `{"errors":[{"description":"Internal Server Error","code":500}]}`)
		})

		Convey("Successful story", func() {
			req := httptest.NewRequest(http.MethodPost, "/feed?page=123", nil)
			req.Header.Add("Content-Type", "application/json")

			m.
				On("GetFeed", mock.Anything, 123).Return([]protocol.Post{}, nil)

			handler.Feed(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			So(resp.StatusCode, ShouldEqual, http.StatusOK)
			So(m.AssertExpectations(t), ShouldBeTrue)
			resBbody, err := ioutil.ReadAll(resp.Body)
			So(err, ShouldBeNil)
			So(string(resBbody), assertions.ShouldEqualJSON, `[]`)
		})
	})
}
