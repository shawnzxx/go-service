package web

import (
	"encoding/json"
	"net/http"

	"github.com/dimfeld/httptreemux/v5"
)

type validator interface {
	Validate() error
}

// Param returns the web call parameters from the request.
func Param(r *http.Request, key string) string {
	m := httptreemux.ContextParams(r.Context())
	return m[key]
}

// Decode reads the body of an HTTP request looking for a JSON document. The
// body is decoded into the provided value.
// If the provided value is a struct then it is checked for validation tags.
// If the value implements a validate function execute it.
func Decode(r *http.Request, val any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	// check application layer model we didn't use in-build type like time.Time, email etc.
	// but all use normal string, or bool to represent json fields
	// because of default decoder didn't return failed reason when filed can not parse
	// walk around:
	// we use normal type for us to easily pass decode function first
	// then continue use validator to valid input value.
	if err := decoder.Decode(val); err != nil {
		return err
	}

	// If the value implements a validate function execute it.
	if v, ok := val.(validator); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}
