package rancher

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bk201/image-tool/pkg/fetch"
)

const (
	buildYAMLURL  = "https://raw.githubusercontent.com/rancher/rancher/v2.15.1/build.yaml"
	dockerfileURL = "https://raw.githubusercontent.com/rancher/rancher/v2.15.1/package/Dockerfile"
	webhookURL    = "https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-webhook/110.0.2+up0.11.1/values.yaml"
	fleetURL      = "https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/fleet/110.0.1+up0.16.1/values.yaml"
	turtlesURL    = "https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-turtles/110.0.1+up0.27.1/values.yaml"
	configMapURL  = "https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-turtles/110.0.1+up0.27.1/templates/core-provider-configmap.yaml"
	sucValuesURL  = "https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/system-upgrade-controller/110.0.0/values.yaml"
)

func newFixtureFake(t *testing.T) *fetch.Fake {
	t.Helper()
	return fetch.FakeFromFiles(t, map[string]string{
		buildYAMLURL:  "testdata/build.yaml",
		dockerfileURL: "testdata/package-Dockerfile",
		webhookURL:    "testdata/charts/rancher-webhook/values.yaml",
		fleetURL:      "testdata/charts/fleet/values.yaml",
		turtlesURL:    "testdata/charts/rancher-turtles/values.yaml",
		configMapURL:  "testdata/charts/rancher-turtles/templates/core-provider-configmap.yaml",
		sucValuesURL:  "testdata/charts/system-upgrade-controller/values.yaml",
	})
}

func newFixtureContext(fk fetch.Fetcher) *Context {
	return &Context{
		Fetcher:      fk,
		Version:      "v2.15.1",
		ChartsBranch: "release-v2.15",
	}
}

func TestComponents_Images(t *testing.T) {
	tests := []struct {
		component Component
		want      []string
		wantURLs  []string
	}{
		{
			component: managerComponent{},
			want:      []string{"rancher/rancher:v2.15.1", "rancher/rancher-agent:v2.15.1"},
			wantURLs:  nil,
		},
		{
			component: shellComponent{},
			want:      []string{"rancher/shell:v0.8.1"},
			wantURLs:  []string{buildYAMLURL},
		},
		{
			component: webhookComponent{},
			want:      []string{"rancher/rancher-webhook:v0.11.1"},
			wantURLs:  []string{buildYAMLURL, webhookURL},
		},
		{
			component: fleetComponent{},
			want:      []string{"rancher/fleet:v0.16.1", "rancher/fleet-agent:v0.16.1"},
			wantURLs:  []string{buildYAMLURL, fleetURL},
		},
		{
			component: sucComponent{},
			want:      []string{"rancher/system-upgrade-controller:v0.20.1", "rancher/kuberlr-kubectl:v8.1.1"},
			wantURLs:  []string{dockerfileURL, sucValuesURL},
		},
		{
			component: turtlesComponent{},
			want:      []string{"rancher/turtles:v0.27.1", "rancher/kuberlr-kubectl:v8.1.1", "rancher/cluster-api-controller:v1.13.3"},
			wantURLs:  []string{buildYAMLURL, turtlesURL, configMapURL},
		},
		{
			component: systemAgentComponent{},
			want:      []string{"rancher/system-agent:v0.15.1-suc"},
			wantURLs:  []string{dockerfileURL},
		},
	}

	for _, tt := range tests {
		t.Run(tt.component.Name(), func(t *testing.T) {
			fk := newFixtureFake(t)
			rc := newFixtureContext(fk)

			got, err := tt.component.Images(context.Background(), rc)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			if tt.wantURLs != nil {
				assert.Equal(t, tt.wantURLs, fk.Calls)
			}
		})
	}
}

func TestComponents_RegisteredInPackageRegistry(t *testing.T) {
	names := make(map[string]bool, len(components))
	for _, c := range components {
		names[c.Name()] = true
	}
	for _, want := range []string{"rancher-manager", "rancher-webhook", "fleet", "shell", "turtles", "system-upgrade-controller", "system-agent"} {
		assert.True(t, names[want], "expected %s to be registered", want)
	}
}

func TestFleetComponent_EmptyTagIsAnError(t *testing.T) {
	notag := fetch.FakeFromFiles(t, map[string]string{
		buildYAMLURL: "testdata/build.yaml",
		fleetURL:     "testdata/charts/fleet/values-notag.yaml",
	})
	rc := newFixtureContext(notag)

	_, err := fleetComponent{}.Images(context.Background(), rc)
	require.Error(t, err)
}

func TestWebhookComponent_VersionNotFound(t *testing.T) {
	fk := &fetch.Fake{
		Errs: map[string]error{
			buildYAMLURL: &fetch.HTTPError{URL: buildYAMLURL, StatusCode: http.StatusNotFound, Status: "404 Not Found"},
		},
	}
	rc := newFixtureContext(fk)

	_, err := webhookComponent{}.Images(context.Background(), rc)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVersionNotFound)
}
