package protocol

import (
	"errors"
	"net/http"
)

type ErrorResponse struct {
	Errors []Error `json:"errors,omitempty"`
}

type GeneralResponse struct {
	Data struct{} `json:"data,omitempty"`
}

///////////////////////////////////////////////////////////////////////////////

type SubmitRequest struct {
	Post
}

func (sr *SubmitRequest) Bind(r *http.Request) error {
	if len(sr.Link) != 0 && len(sr.Content) != 0 {
		return errors.New("A post cannot have both a link and content populated")
	}
	return nil
}
