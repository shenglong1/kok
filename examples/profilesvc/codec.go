package profilesvc

import (
	"net/http"

	httpcodec "github.com/RussellLuo/kok/pkg/codec/httpv2"
	"github.com/RussellLuo/kok/pkg/oasv2"
)

type Codec struct {
	httpcodec.JSONCodec
}

func (c Codec) EncodeFailureResponse(w http.ResponseWriter, err error) error {
	return c.JSONCodec.EncodeSuccessResponse(w, codeFrom(err), toBody(err))
}

func toBody(err error) interface{} {
	return map[string]string{
		"error": err.Error(),
	}
}

func codeFrom(err error) int {
	switch err {
	case ErrNotFound:
		return http.StatusNotFound
	case ErrAlreadyExists, ErrInconsistentIDs:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func NewCodecs() httpcodec.Codecs {
	return httpcodec.CodecMap{
		Default: Codec{},
	}
}

func GetFailures(name string) []oasv2.Response {
	switch name {
	case "PostProfile":
		return []oasv2.Response{
			{
				StatusCode: codeFrom(ErrAlreadyExists),
				Body:       toBody(ErrAlreadyExists),
			},
		}
	case "GetProfile":
		return []oasv2.Response{
			{
				StatusCode: codeFrom(ErrNotFound),
				Body:       toBody(ErrNotFound),
			},
		}
	case "PutProfile":
		return []oasv2.Response{
			{
				StatusCode: codeFrom(ErrInconsistentIDs),
				Body:       toBody(ErrInconsistentIDs),
			},
		}
	case "PatchProfile":
		return []oasv2.Response{
			{
				StatusCode: codeFrom(ErrInconsistentIDs),
				Body:       toBody(ErrInconsistentIDs),
			},
			{
				StatusCode: codeFrom(ErrNotFound),
				Body:       toBody(ErrNotFound),
			},
		}
	case "DeleteProfile":
		return []oasv2.Response{
			{
				StatusCode: codeFrom(ErrNotFound),
				Body:       toBody(ErrNotFound),
			},
		}
	default:
		return nil
	}
}

func NewSchema() oasv2.Schema {
	return &oasv2.ResponseSchema{
		Codecs:       NewCodecs(),
		FailuresFunc: GetFailures,
	}
}
