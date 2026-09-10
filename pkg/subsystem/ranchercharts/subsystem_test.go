package ranchercharts

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/bk201/image-tool/pkg/chart"
	"github.com/bk201/image-tool/pkg/fetch"
	"github.com/bk201/image-tool/pkg/imagelist"
	"github.com/bk201/image-tool/pkg/subsystem"
)

const monitoringVersion = "109.0.3+up80.9.1-rancher.14"
const loggingVersion = "109.0.0+up4.10.0-rancher.23"

func fixtureURL(name, file string) string {
	v := monitoringVersion
	if strings.HasPrefix(name, "rancher-logging") {
		v = loggingVersion
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/%s/%s/%s", name, v, file)
}

func fixtures(t *testing.T) *fetch.Fake {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir("testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		rel, err := filepath.Rel("testdata", path)
		if err != nil {
			return err
		}
		name, file, _ := strings.Cut(filepath.ToSlash(rel), "/")
		files[fixtureURL(name, file)] = path
		return nil
	})
	require.NoError(t, err)
	return fetch.FakeFromFiles(t, files)
}

func collect(t *testing.T, s subsystem.Subsystem, f *fetch.Fake) ([]subsystem.ComponentResult, error) {
	t.Helper()
	v := monitoringVersion
	if s.Name() == "rancher-logging" {
		v = loggingVersion
	}
	return s.Collect(context.Background(), subsystem.Options{Fetcher: f, ChartBranch: "release-v2.15", Version: v})
}

func flatten(results []subsystem.ComponentResult) []string {
	var out []string
	for _, r := range results {
		out = append(out, r.Images...)
	}
	return imagelist.Normalize(out)
}

func TestCollectFixtures(t *testing.T) {
	for _, tc := range []struct {
		s     subsystem.Subsystem
		want  map[string][]string
		calls int
	}{
		{NewMonitoring(), map[string][]string{
			"prometheus":               {"rancher/prom-prometheus:v3.8.1", "rancher/mirrored-library-nginx:1.29.1-alpine"},
			"alertmanager":             {"rancher/appco-alertmanager:0.30.0-12.16"},
			"prometheus-operator":      {"rancher/mirrored-prometheus-operator-prometheus-operator:v0.87.1", "rancher/mirrored-prometheus-operator-prometheus-config-reloader:v0.87.1"},
			"grafana":                  {"rancher/appco-grafana:12.3.1-1.12", "rancher/mirrored-library-nginx:1.29.1-alpine", "rancher/appco-k8s-sidecar:2.1.2-1.10", "rancher/mirrored-library-busybox:1.31.1"},
			"kube-state-metrics":       {"rancher/appco-kube-state-metrics:2.17.0-10.14"},
			"prometheus-node-exporter": {"rancher/appco-node-exporter:1.10.2-14.3"},
			"prometheus-adapter":       {"rancher/mirrored-prometheus-adapter-prometheus-adapter:v0.12.0"},
			"admission-patch":          {"rancher/mirrored-jkroepke-kube-webhook-certgen:1.7.4"},
			"admission-webhook":        {"rancher/mirrored-prometheus-operator-admission-webhook:v0.87.1"},
			"monitoring-upgrade":       {"rancher/kuberlr-kubectl:v7.1.0"},
			"monitoring-crd-upgrade":   {"rancher/mirrored-library-busybox:1.37.0"},
			"rancher-monitoring-crd":   {"rancher/kuberlr-kubectl:v7.1.1"},
		}, 12},
		{NewLogging(), map[string][]string{
			"logging-operator":    {"rancher/mirrored-kube-logging-logging-operator:4.10.0"},
			"fluentd":             {"rancher/mirrored-kube-logging-fluentd:v1.16-4.10-full"},
			"fluentbit":           {"rancher/mirrored-fluent-fluent-bit:3.1.8"},
			"config-reloader":     {"rancher/mirrored-kube-logging-config-reloader:v0.0.6"},
			"rancher-logging-crd": {},
		}, 3},
	} {
		t.Run(tc.s.Name(), func(t *testing.T) {
			f := fixtures(t)
			results, err := collect(t, tc.s, f)
			require.NoError(t, err)
			got := map[string][]string{}
			for _, r := range results {
				got[r.Component] = r.Images
			}
			assert.Equal(t, tc.want, got)
			require.Len(t, f.Calls, tc.calls)
			for _, u := range f.Calls {
				assert.Contains(t, u, "/release-v2.15/charts/")
				assert.NotContains(t, u, ".tgz")
				assert.NotContains(t, u, "/assets/")
				assert.Contains(t, u, "+up")
			}
			_, verifies := tc.s.(subsystem.Verifier)
			assert.False(t, verifies)
		})
	}
}

func modifyValues(t *testing.T, f *fetch.Fake, name, file string, fn func(chart.Values)) {
	t.Helper()
	u := fixtureURL(name, file)
	var v chart.Values
	require.NoError(t, yaml.Unmarshal(f.Bodies[u], &v))
	fn(v)
	b, err := yaml.Marshal(v)
	require.NoError(t, err)
	f.Bodies[u] = b
}

func TestParentOverridesAndChildFallback(t *testing.T) {
	f := fixtures(t)
	modifyValues(t, f, "rancher-monitoring", "values.yaml", func(v chart.Values) {
		v["grafana"].(chart.Values)["image"] = chart.Values{"repository": "example/grafana", "tag": ""}
		v["grafana"].(chart.Values)["sidecar"].(chart.Values)["image"] = chart.Values{"tag": "custom"}
		v["global"].(chart.Values)["cattle"].(chart.Values)["systemDefaultRegistry"] = "mirror.example"
	})
	results, err := collect(t, NewMonitoring(), f)
	require.NoError(t, err)
	images := flatten(results)
	assert.Contains(t, images, "mirror.example/example/grafana:12.3.1")
	assert.Contains(t, images, "mirror.example/rancher/appco-k8s-sidecar:custom")
	assert.Contains(t, images, "mirror.example/rancher/prom-prometheus:v3.8.1")
	// CRD chart is independent and does not inherit the main chart's values.
	assert.Contains(t, images, "rancher/kuberlr-kubectl:v7.1.1")
}

func TestDisabledComponentsAndCRDUpgrade(t *testing.T) {
	f := fixtures(t)
	modifyValues(t, f, "rancher-monitoring", "values.yaml", func(v chart.Values) {
		v["grafana"].(chart.Values)["enabled"] = "false"
		v["upgrade"].(chart.Values)["enabled"] = "false"
		v["prometheusOperator"].(chart.Values)["admissionWebhooks"].(chart.Values)["certManager"].(chart.Values)["enabled"] = "true"
		v["crds"].(chart.Values)["upgradeJob"].(chart.Values)["enabled"] = "true"
	})
	results, err := collect(t, NewMonitoring(), f)
	require.NoError(t, err)
	names := map[string]bool{}
	for _, r := range results {
		names[r.Component] = true
		if r.Component == "monitoring-crd-upgrade" {
			assert.Equal(t, []string{"rancher/mirrored-library-busybox:1.37.0", "rancher/kuberlr-kubectl:v7.1.0"}, r.Images)
		}
	}
	assert.False(t, names["grafana"])
	assert.False(t, names["admission-patch"])
	assert.False(t, names["monitoring-upgrade"])
	assert.True(t, names["monitoring-crd-upgrade"])
	assert.Contains(t, flatten(results), "rancher/mirrored-library-busybox:1.37.0")
	for _, u := range f.Calls {
		assert.NotContains(t, u, "/charts/grafana/")
	}
}

func TestFailures(t *testing.T) {
	for _, tc := range []struct{ name, file, body, want string }{
		{"missing repository", "values.yaml", "image: {tag: v1}", "image.repository"},
		{"missing tag", "values.yaml", "image: {repository: example/operator}\nimages: {fluentd: {repository: example/fluentd}}", "images.fluentd.tag"},
		{"invalid YAML", "values.yaml", "image: [", "values.yaml"},
		{"wrong name", "Chart.yaml", "name: other\nversion: " + loggingVersion, "expected name"},
		{"wrong version", "Chart.yaml", "name: rancher-logging\nversion: 0.0.0", "expected name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := fixtures(t)
			f.Bodies[fixtureURL("rancher-logging", tc.file)] = []byte(tc.body)
			results, err := collect(t, NewLogging(), f)
			require.ErrorContains(t, err, tc.want)
			assert.Nil(t, results)
		})
	}
	for _, name := range []string{"rancher-logging", "rancher-logging-crd"} {
		t.Run("missing "+name, func(t *testing.T) {
			f := fixtures(t)
			delete(f.Bodies, fixtureURL(name, "Chart.yaml"))
			results, err := collect(t, NewLogging(), f)
			require.ErrorContains(t, err, name+"/"+loggingVersion)
			assert.Nil(t, results)
			assert.True(t, fetch.IsNotFound(err))
		})
	}
}

func TestBranchAndCancellation(t *testing.T) {
	for _, s := range []subsystem.Subsystem{NewLogging(), NewMonitoring()} {
		f := fixtures(t)
		_, err := s.Collect(context.Background(), subsystem.Options{Fetcher: f, Version: loggingVersion})
		require.ErrorContains(t, err, "--chart-branch")
		assert.Empty(t, f.Calls)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = s.Collect(ctx, subsystem.Options{Fetcher: f, Version: loggingVersion, ChartBranch: "release-v2.15"})
		require.ErrorIs(t, err, context.Canceled)
		assert.Empty(t, f.Calls)
	}
	f := fixtures(t)
	custom := map[string][]byte{}
	for u, b := range f.Bodies {
		custom[strings.Replace(u, "release-v2.15", "custom-branch", 1)] = b
	}
	f.Bodies = custom
	_, err := NewLogging().Collect(context.Background(), subsystem.Options{Fetcher: f, Version: loggingVersion, ChartBranch: "custom-branch"})
	require.NoError(t, err)
	for _, u := range f.Calls {
		assert.Contains(t, u, "/custom-branch/charts/")
	}
}

func TestSubchartSchemaErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		modify func(*fetch.Fake)
		want   string
	}{
		{"missing values", func(f *fetch.Fake) { delete(f.Bodies, fixtureURL("rancher-monitoring", "charts/grafana/values.yaml")) }, "charts/grafana/values.yaml"},
		{"metadata mismatch", func(f *fetch.Fake) {
			f.Bodies[fixtureURL("rancher-monitoring", "charts/grafana/Chart.yaml")] = []byte("name: grafana\nversion: wrong")
		}, "charts/grafana/Chart.yaml"},
		{"missing dependency", func(f *fetch.Fake) {
			f.Bodies[fixtureURL("rancher-monitoring", "Chart.yaml")] = []byte("name: rancher-monitoring\nversion: " + monitoringVersion)
		}, "missing dependency grafana"},
		{"missing switch", func(f *fetch.Fake) {
			modifyValues(t, f, "rancher-monitoring", "values.yaml", func(v chart.Values) { delete(v["prometheus"].(chart.Values), "enabled") })
		}, "prometheus.enabled"},
		{"invalid switch", func(f *fetch.Fake) {
			modifyValues(t, f, "rancher-monitoring", "values.yaml", func(v chart.Values) { v["upgrade"].(chart.Values)["enabled"] = "maybe" })
		}, "upgrade.enabled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := fixtures(t)
			tc.modify(f)
			results, err := collect(t, NewMonitoring(), f)
			require.ErrorContains(t, err, tc.want)
			assert.Nil(t, results)
		})
	}
}

func TestAdmissionWebhookIncludedWhenDisabled(t *testing.T) {
	for _, globalRegistry := range []string{"", "global.example"} {
		t.Run(globalRegistry, func(t *testing.T) {
			f := fixtures(t)
			modifyValues(t, f, "rancher-monitoring", "values.yaml", func(v chart.Values) {
				operator := v["prometheusOperator"].(chart.Values)
				operator["enabled"] = "false"
				deployment := operator["admissionWebhooks"].(chart.Values)["deployment"].(chart.Values)
				deployment["enabled"] = "false"
				deployment["image"] = chart.Values{"repository": "example/webhook", "tag": "", "sha": "abc", "registry": "image.example"}
				global := v["global"].(chart.Values)
				global["imageRegistry"] = globalRegistry
				global["cattle"].(chart.Values)["systemDefaultRegistry"] = "ignored.example"
			})
			results, err := collect(t, NewMonitoring(), f)
			require.NoError(t, err)
			registry := globalRegistry
			if registry == "" {
				registry = "image.example"
			}
			assert.Contains(t, flatten(results), registry+"/example/webhook:v0.87.1@sha256:abc")
		})
	}
}
