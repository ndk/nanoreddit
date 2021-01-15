package middleware

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog"
)

func RequestBody(invalidRequest func(w http.ResponseWriter, r *http.Request, err error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			l := zerolog.Ctx(r.Context())

			body := r.Body
			var buf []byte
			if body != nil {
				b, err := ioutil.ReadAll(r.Body)
				if err != nil {
					l.Error().Err(err).Msg("Cannot read out a request body")
					invalidRequest(w, r, err)
					return
				}
				buf = b
				r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
			}

			{
				compactBody := &bytes.Buffer{}
				if err := json.Compact(compactBody, buf); err != nil {
					l.Trace().Str("method", r.Method).Str("url", r.URL.String()).Interface("headers", r.Header).Bytes("request_body", buf).Msg("The incoming request")
				} else {
					l.Trace().Str("method", r.Method).Str("url", r.URL.String()).Interface("headers", r.Header).Bytes("request_body", compactBody.Bytes()).Msg("The incoming request")
				}
			}

			next.ServeHTTP(w, r)

			r.Body = body
		}
		return http.HandlerFunc(fn)
	}
}
