package rancher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bk201/image-tool/pkg/fetch"
)

func TestChartsBranchFor(t *testing.T) {
	tests := []struct {
		version string
		want    string
		wantErr bool
	}{
		{"v2.15.1", "release-v2.15", false},
		{"v2.15.0", "release-v2.15", false},
		{"2.15.1", "release-v2.15", false},
		{"v2", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got, err := ChartsBranchFor(tt.version)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContext_ChartFile_URLConstruction(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := &Context{
		Fetcher:       fetch.New(fetch.Options{MinInterval: time.Microsecond}),
		Version:       "v2.15.1",
		ChartsBranch:  "release-v2.15",
		ChartsRawBase: srv.URL,
	}

	tests := []struct {
		name         string
		chart        string
		chartVersion string
		relPath      string
		wantPath     string
	}{
		{
			name:         "chart version containing a literal +",
			chart:        "fleet",
			chartVersion: "110.0.1+up0.16.1",
			relPath:      "values.yaml",
			wantPath:     "/release-v2.15/charts/fleet/110.0.1+up0.16.1/values.yaml",
		},
		{
			name:         "nested relPath",
			chart:        "rancher-turtles",
			chartVersion: "110.0.1+up0.27.1",
			relPath:      "templates/core-provider-configmap.yaml",
			wantPath:     "/release-v2.15/charts/rancher-turtles/110.0.1+up0.27.1/templates/core-provider-configmap.yaml",
		},
		{
			name:         "chart version without a +",
			chart:        "system-upgrade-controller",
			chartVersion: "110.0.0",
			relPath:      "values.yaml",
			wantPath:     "/release-v2.15/charts/system-upgrade-controller/110.0.0/values.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.ChartFile(context.Background(), tt.chart, tt.chartVersion, tt.relPath)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, gotPath)
			assert.NotContains(t, gotPath, "%25", "must never double-escape")
		})
	}
}

func TestContext_ChartFile_DerivesBranchFromVersionWhenUnset(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := &Context{
		Fetcher:       fetch.New(fetch.Options{MinInterval: time.Microsecond}),
		Version:       "v2.15.1",
		ChartsRawBase: srv.URL,
	}

	_, err := c.ChartFile(context.Background(), "fleet", "110.0.1+up0.16.1", "values.yaml")
	require.NoError(t, err)
	assert.Equal(t, "/release-v2.15/charts/fleet/110.0.1+up0.16.1/values.yaml", gotPath)
}

func TestContext_BuildYAML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2.15.1/build.yaml" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, err := os.ReadFile("testdata/build.yaml")
		require.NoError(t, err)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := &Context{
		Fetcher:        fetch.New(fetch.Options{MinInterval: time.Microsecond}),
		Version:        "v2.15.1",
		RancherRawBase: srv.URL,
	}

	b, err := c.BuildYAML(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "110.0.2+up0.11.1", b.WebhookVersion)
	assert.Equal(t, "110.0.1+up0.27.1", b.TurtlesVersion)
	assert.Equal(t, "110.0.1+up0.16.1", b.FleetVersion)
	assert.Equal(t, "rancher/shell:v0.8.1", b.DefaultShellVersion)
}

func TestContext_BuildYAML_VersionNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &Context{
		Fetcher:        fetch.New(fetch.Options{MinInterval: time.Microsecond}),
		Version:        "v9.99.9",
		RancherRawBase: srv.URL,
	}

	_, err := c.BuildYAML(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVersionNotFound)
}

func TestContext_Dockerfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2.15.1/package/Dockerfile", r.URL.Path)
		_, _ = w.Write([]byte("ENV FOO=bar"))
	}))
	defer srv.Close()

	c := &Context{
		Fetcher:        fetch.New(fetch.Options{MinInterval: time.Microsecond}),
		Version:        "v2.15.1",
		RancherRawBase: srv.URL,
	}

	body, err := c.Dockerfile(context.Background())
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(body), "FOO=bar"))
}

func TestContext_ChartValues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := os.ReadFile("testdata/charts/fleet/values.yaml")
		require.NoError(t, err)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	c := &Context{
		Fetcher:       fetch.New(fetch.Options{MinInterval: time.Microsecond}),
		Version:       "v2.15.1",
		ChartsBranch:  "release-v2.15",
		ChartsRawBase: srv.URL,
	}

	var values struct {
		Image struct {
			Repository string `yaml:"repository"`
			Tag        string `yaml:"tag"`
		} `yaml:"image"`
	}
	require.NoError(t, c.ChartValues(context.Background(), "fleet", "110.0.1+up0.16.1", &values))
	assert.Equal(t, "rancher/fleet", values.Image.Repository)
	assert.Equal(t, "v0.16.1", values.Image.Tag)
}
