package fetch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
)

// Fake is a Fetcher backed by an in-memory map, for use in tests of packages
// that depend on Fetcher. An unmapped URL returns an error rather than empty
// bytes, so a wrong URL fails loudly instead of silently.
type Fake struct {
	Bodies map[string][]byte
	Errs   map[string]error

	// Calls records every URL passed to Get, in order.
	Calls []string
}

func (f *Fake) Get(_ context.Context, rawURL string) ([]byte, error) {
	f.Calls = append(f.Calls, rawURL)

	if err, ok := f.Errs[rawURL]; ok {
		return nil, err
	}
	if body, ok := f.Bodies[rawURL]; ok {
		return body, nil
	}
	return nil, &HTTPError{URL: rawURL, StatusCode: http.StatusNotFound, Status: fmt.Sprintf("no fake body registered for %s", rawURL)}
}

// FakeFromFiles builds a Fake whose Bodies are read from the given
// URL-to-file-path mapping.
func FakeFromFiles(t *testing.T, m map[string]string) *Fake {
	t.Helper()

	bodies := make(map[string][]byte, len(m))
	for rawURL, path := range m {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read fixture %s for %s: %v", path, rawURL, err)
		}
		bodies[rawURL] = body
	}

	return &Fake{Bodies: bodies}
}
