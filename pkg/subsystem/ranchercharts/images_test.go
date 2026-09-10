package ranchercharts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/bk201/image-tool/pkg/chart"
)

func TestImageTemplateRules(t *testing.T) {
	for _, tc := range []struct {
		name, input, appVersion string
		policy                  imagePolicy
		want, err               string
	}{
		{name: "numeric spelling", input: "image: {repository: example/image, tag: 1.0}", policy: cattleImage, want: "example/image:1.0"},
		{name: "operator appVersion", input: "image: {repository: example/image, tag: ''}", appVersion: "v2.3", policy: operatorImage, want: "example/image:v2.3"},
		{name: "no invented fallback", input: "image: {repository: example/image}", appVersion: "v2.3", policy: cattleImage, err: "image.tag"},
		{name: "no appVersion", input: "image: {repository: example/image}", policy: operatorImage, err: "image.tag"},
		{name: "exporter version prefix", input: "image: {repository: example/image}", appVersion: "2.3", policy: imagePolicy{registry: "exporter", fallback: true, versionPrefix: "v"}, want: "example/image:v2.3"},
		{name: "sha and tag", input: "image: {repository: example/image, tag: v1, sha: abc}", policy: monitoringImage, want: "example/image:v1@sha256:abc"},
		{name: "digest only", input: "image: {repository: example/image, sha: abc}", policy: monitoringImage, want: "example/image@sha256:abc"},
		{name: "qualified sha", input: "image: {repository: example/image, tag: v1, sha: 'sha256:abc'}", policy: imagePolicy{digest: "sha"}, want: "example/image:v1@sha256:abc"},
		{name: "qualified digest", input: "image: {repository: example/image, tag: v1, digest: 'sha256:abc'}", policy: imagePolicy{digest: "digest"}, want: "example/image:v1@sha256:abc"},
		{name: "node exporter rejects sha", input: "image: {repository: example/image, tag: v1, sha: abc}", policy: imagePolicy{digest: "digest"}, err: "forbidden"},
		{name: "missing repository", input: "image: {tag: v1}", policy: cattleImage, err: "repository"},
		{name: "invalid tag", input: "image: {repository: example/image, tag: [v1]}", policy: cattleImage, err: "expected scalar"},
		{name: "missing mapping", input: "image: hello", policy: cattleImage, err: "image mapping"},
		{name: "cattle priority", input: "global: {cattle: {systemDefaultRegistry: cattle.example}, imageRegistry: global.example}\nimage: {repository: example/image, tag: v1, registry: local.example}", policy: monitoringImage, want: "cattle.example/example/image:v1"},
		{name: "operator image registry priority", input: "global: {cattle: {systemDefaultRegistry: cattle.example}}\nimage: {repository: example/image, tag: v1, registry: local.example}", policy: operatorImage, want: "local.example/example/image:v1"},
		{name: "monitoring global priority", input: "global: {imageRegistry: global.example}\nimage: {repository: example/image, tag: v1, registry: local.example}", policy: monitoringImage, want: "global.example/example/image:v1"},
		{name: "monitoring image registry fallback", input: "image: {repository: example/image, tag: v1, registry: local.example}", policy: monitoringImage, want: "local.example/example/image:v1"},
		{name: "grafana ignores global imageRegistry", input: "global: {imageRegistry: global.example}\nimage: {repository: example/image, tag: v1}", policy: grafanaImage, want: "example/image:v1"},
		{name: "docker hub normalization", input: "global: {imageRegistry: docker.io}\nimage: {repository: example/image, tag: v1}", policy: monitoringImage, want: "example/image:v1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var v chart.Values
			require.NoError(t, yaml.Unmarshal([]byte(tc.input), &v))
			got, err := imageRef(v, "image", tc.appVersion, tc.policy)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
