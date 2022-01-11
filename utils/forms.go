package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	validator "gopkg.in/go-playground/validator.v9"
)

var validate = validator.New()

// ReadBody of a HTTP request up to limit bytes and make sure the Body is not consumed
func ReadBody(r *http.Request, limit int64) ([]byte, error) {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, limit))
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return body, err

}

// DecodeAndValidateJSON takes the passed in envelope and tries to unmarshal it from the body
// of the passed in request, then validating it
func DecodeAndValidateJSON(envelope interface{}, r *http.Request) error {
	body, err := ReadBody(r, 100000)
	if err != nil {
		return fmt.Errorf("unable to read request body: %s", err)
	}

	// try to decode our envelope
	if err = json.Unmarshal(body, envelope); err != nil {
		return fmt.Errorf("unable to parse request JSON: %s", err)
	}

	// check our input is valid
	err = validate.Struct(envelope)
	if err != nil {
		return fmt.Errorf("request JSON doesn't match required schema: %s", err)
	}

	return nil
}
