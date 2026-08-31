// Package fetch provides a polite, cacheable HTTP fetcher for reading files
// from public sources such as raw.githubusercontent.com.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Fetcher retrieves the raw bytes of a URL.
type Fetcher interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// HTTPError is returned when a request completes with a non-2xx status code.
type HTTPError struct {
	URL        string
	StatusCode int
	Status     string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("fetch %s: %s", e.URL, e.Status)
}

// IsNotFound reports whether err represents a 404 response.
func IsNotFound(err error) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == http.StatusNotFound
	}
	return false
}
