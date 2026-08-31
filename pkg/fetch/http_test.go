package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet_RateLimitsAcrossDistinctURLs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	defer srv.Close()

	f := New(Options{MinInterval: 50 * time.Millisecond})
	ctx := context.Background()

	start := time.Now()
	for _, path := range []string{"/a", "/b", "/c"} {
		body, err := f.Get(ctx, srv.URL+path)
		require.NoError(t, err)
		assert.Equal(t, path, string(body))
	}
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 2*50*time.Millisecond, "3 distinct URLs should be spaced by at least 2 intervals")
}

func TestGet_CachesResponsesWithoutReRequesting(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte("body"))
	}))
	defer srv.Close()

	f := New(Options{MinInterval: 200 * time.Millisecond})
	ctx := context.Background()

	_, err := f.Get(ctx, srv.URL)
	require.NoError(t, err)

	start := time.Now()
	_, err = f.Get(ctx, srv.URL)
	elapsed := time.Since(start)
	require.NoError(t, err)

	assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "second Get should be served from cache")
	assert.Less(t, elapsed, 200*time.Millisecond, "cache hit should not wait for the rate limiter")
}

func TestGet_CachesNegativeResults(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := New(Options{MinInterval: time.Millisecond})
	ctx := context.Background()

	_, err1 := f.Get(ctx, srv.URL)
	_, err2 := f.Get(ctx, srv.URL)

	require.Error(t, err1)
	require.Error(t, err2)
	assert.True(t, IsNotFound(err1))
	assert.True(t, IsNotFound(err2))
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "404 should be cached too")
}

func TestGet_FollowsRedirects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/target", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("final"))
	}))
	defer srv.Close()

	f := New(Options{MinInterval: time.Millisecond})
	body, err := f.Get(context.Background(), srv.URL+"/redirect")
	require.NoError(t, err)
	assert.Equal(t, "final", string(body))
}

func TestGet_NonOKStatusReturnsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	f := New(Options{MinInterval: time.Millisecond})
	_, err := f.Get(context.Background(), srv.URL)
	require.Error(t, err)

	var httpErr *HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusInternalServerError, httpErr.StatusCode)
	assert.False(t, IsNotFound(err))
}

func TestGet_SetsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
	}))
	defer srv.Close()

	f := New(Options{MinInterval: time.Millisecond, UserAgent: "image-tool-test"})
	_, err := f.Get(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "image-tool-test", gotUA)
}

func TestGet_ContextCancellation(t *testing.T) {
	f := New(Options{MinInterval: time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.Get(ctx, "http://127.0.0.1:0/unreachable")
	require.Error(t, err)
}
