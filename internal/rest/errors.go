package rest

import (
	"errors"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
)

var ErrNotFound = errors.New("resource not found")

type NotFoundError struct {
	Message string
}

func (e NotFoundError) Error() string {
	return e.Message
}

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

type HTTPError struct {
	StatusCode int `json:"-"`

	Message string `json:"message"`
}

func (e HTTPError) Error() string {
	return e.Message
}

func NewHTTPError(err error) HTTPError {
	switch err := err.(type) {
	case NotFoundError:
		return HTTPError{
			StatusCode: http.StatusNotFound,
			Message:    err.Message,
		}
	case ValidationError:
		return HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    err.Message,
		}
	}

	switch err {
	case ErrNotFound:
		return HTTPError{
			StatusCode: http.StatusNotFound,
			Message:    "resource not found",
		}
	case io.EOF:
		return HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "EOF reading HTTP request body",
		}
	}

	return HTTPError{
		StatusCode: http.StatusInternalServerError,
		Message:    "Internal server error",
	}
}

func errorHandler(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		err := next(w, req)
		switch err := err.(type) {
		case nil:
			// no error
		case HTTPError: // already a HTTPError
			w.WriteHeader(err.StatusCode)
			_ = JSON(w, err.StatusCode, err)
		default:
			httpErr := NewHTTPError(err)
			_ = JSON(w, httpErr.StatusCode, httpErr)
		}

		return err // return the err in case there other middlewares
	}
}
