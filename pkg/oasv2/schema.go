package oasv2

import (
	"net/http/httptest"

	httpcodec "github.com/RussellLuo/kok/pkg/codec/httpv2"
)

type Response struct {
	StatusCode  int
	ContentType string
	Body        interface{}
}

type Schema interface {
	SuccessResponse(name string, statusCode int, body interface{}) Response
	FailureResponses(name string) []Response
}

type ResponseSchema struct {
	Codecs       httpcodec.Codecs
	FailuresFunc func(name string) []Response
}

func (rs *ResponseSchema) SuccessResponse(name string, statusCode int, body interface{}) Response {
	if rs.Codecs == nil {
		return Response{
			StatusCode:  statusCode,
			ContentType: "application/json",
			Body:        body,
		}
	}

	codec := rs.Codecs.EncodeDecoder(name)

	w := httptest.NewRecorder()
	_ = codec.EncodeSuccessResponse(w, statusCode, body)
	contentType := w.Result().Header.Get("Content-Type")

	if !isMediaFile(contentType) {
		body = codec.SuccessResponse(body)
	} else {
		body = nil
	}

	return Response{
		StatusCode:  statusCode,
		ContentType: contentType,
		Body:        body,
	}
}

func (rs *ResponseSchema) FailureResponses(name string) []Response {
	if rs.FailuresFunc == nil {
		return nil
	}
	return rs.FailuresFunc(name)
}
