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
)

func TestSubmit(t *testing.T) {
	Convey("Test Submit", t, func() {
		m := &mock.Mock{}

		w := httptest.NewRecorder()
		w.Body = bytes.NewBuffer(nil)

		const body = `{
			"title": "title 1",
			"author": "t2_abcdefg2",
			"link": "https://reddit.com/3",
			"subreddit": "subreddit 4",
			"score": 123,
			"promoted": false,
			"nsfw": false
		}`
		req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewBufferString(body))
		req.Header.Add("Content-Type", "application/json")

		handler, err := mockHandler(m)
		So(err, ShouldBeNil)

		Convey("It fails if binding has been failed", func() {
			handler.binder = &mockBinder{m: m}

			m.
				On("Bind", mock.Anything, mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					w := args.Get(0).(http.ResponseWriter)
					w.WriteHeader(http.StatusBadRequest)
					_, err := w.Write([]byte(`{"message": "hello"}`))
					So(err, ShouldBeNil)
				}).
				Return(errors.New("binding error"))

			handler.Submit(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			So(resp.StatusCode, ShouldEqual, http.StatusBadRequest)
			So(m.AssertExpectations(t), ShouldBeTrue)
			resBbody, err := ioutil.ReadAll(resp.Body)
			So(err, ShouldBeNil)
			So(string(resBbody), assertions.ShouldEqualJSON, `{"message":"hello"}`)
		})

		Convey("It fails if an storage has been failed", func() {
			m.
				On("AddPost", mock.Anything, mock.Anything).Return(errors.New("storage error"))

			handler.Submit(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			So(resp.StatusCode, ShouldEqual, http.StatusInternalServerError)
			So(m.AssertExpectations(t), ShouldBeTrue)
			resBbody, err := ioutil.ReadAll(resp.Body)
			So(err, ShouldBeNil)
			So(string(resBbody), assertions.ShouldEqualJSON, `{"errors":[{"description":"Internal Server Error","code":500}]}`)
		})

		Convey("Successful story", func() {
			m.
				On("AddPost", mock.Anything, mock.Anything).Return(nil)

			handler.Submit(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			So(resp.StatusCode, ShouldEqual, http.StatusOK)
			So(m.AssertExpectations(t), ShouldBeTrue)
			resBbody, err := ioutil.ReadAll(resp.Body)
			So(err, ShouldBeNil)
			So(resBbody, ShouldBeEmpty)
		})
	})
}
