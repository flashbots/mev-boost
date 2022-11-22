package rcp

import (
	"errors"
	"fmt"
)

type Error struct {
	Cause   error
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e Error) Unwrap() error {
	return e.Cause
}

func WrapErr(err error) Error {
	return Error{
		Cause:   err,
		Message: fmt.Sprintf("%v", ErrCannotFetchConfig),
	}
}

var (
	ErrCannotFetchConfig     = errors.New("cannot fetch relay config")
	ErrHTTPRequestFailed     = errors.New("http request failed")
	ErrMalformedProviderURL  = errors.New("malformed relay config provider url")
	ErrMalformedResponseBody = errors.New("malformed response body")
)

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e APIError) Error() string {
	return fmt.Sprintf("api error: %d: %s: %v", e.Code, e.Message, ErrCannotFetchConfig)
}

func (e APIError) Unwrap() error {
	return ErrCannotFetchConfig
}
