package api

import (
	"errors"
	"fmt"
)

// ErrUnauthenticated indicates the server rejected the credentials.
// CLI handlers map this to "Error: not logged in. Run 'ntzh login'.".
var ErrUnauthenticated = errors.New("not authenticated")

// Error is a typed API error returned from the server envelope or a non-2xx response.
type Error struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *Error) Error() string {
	if e.HTTPStatus != 0 && e.Code == "http_error" {
		return fmt.Sprintf("http_error: status %d", e.HTTPStatus)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
