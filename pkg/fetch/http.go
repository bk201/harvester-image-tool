package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

const (
	defaultMinInterval = 800 * time.Millisecond
	defaultTimeout     = 30 * time.Second
	defaultUserAgent   = "image-tool"
	maxBodySize        = 32 << 20 // 32MiB
)

// Options configures a Fetcher returned by New.
type Options struct {
	// MinInterval is the minimum time between two requests. Defaults to 800ms.
	MinInterval time.Duration
	// Timeout is the per-request timeout. Defaults to 30s.
	Timeout time.Duration
	// UserAgent is sent as the HTTP User-Agent header.
	UserAgent string
}

type cacheEntry struct {
	body []byte
	err  error
}

type httpFetcher struct {
	client    *http.Client
	limiter   *rate.Limiter
	userAgent string

	mu    sync.Mutex
	cache map[string]cacheEntry
}

// New returns a Fetcher that issues plain HTTP GET requests, rate-limited to
// at most one request per MinInterval and caching every response (including
// errors) so repeated lookups of the same URL never re-issue a request.
func New(o Options) Fetcher {
	if o.MinInterval <= 0 {
		o.MinInterval = defaultMinInterval
	}
	if o.Timeout <= 0 {
		o.Timeout = defaultTimeout
	}
	if o.UserAgent == "" {
		o.UserAgent = defaultUserAgent
	}

	return &httpFetcher{
		client:    &http.Client{Timeout: o.Timeout},
		limiter:   rate.NewLimiter(rate.Every(o.MinInterval), 1),
		userAgent: o.UserAgent,
		cache:     make(map[string]cacheEntry),
	}
}

func (f *httpFetcher) Get(ctx context.Context, rawURL string) ([]byte, error) {
	f.mu.Lock()
	if entry, ok := f.cache[rawURL]; ok {
		f.mu.Unlock()
		logrus.WithField("url", rawURL).Debug("http cache hit")
		return entry.body, entry.err
	}
	f.mu.Unlock()

	if err := f.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	start := time.Now()
	logrus.WithField("url", rawURL).Info("http request: GET")

	body, err := f.get(ctx, rawURL)

	log := logrus.WithFields(logrus.Fields{"url": rawURL, "elapsed": time.Since(start).Round(time.Millisecond)})
	if err != nil {
		log.WithError(err).Info("http request: failed")
	} else {
		log.WithField("bytes", len(body)).Info("http request: ok")
	}

	f.mu.Lock()
	f.cache[rawURL] = cacheEntry{body: body, err: err}
	f.mu.Unlock()

	return body, err
}

func (f *httpFetcher) get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", rawURL, err)
	}
	req.Header.Set("User-Agent", f.userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", rawURL, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{URL: rawURL, StatusCode: resp.StatusCode, Status: resp.Status}
	}

	return body, nil
}
