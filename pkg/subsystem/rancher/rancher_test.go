package rancher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bk201/image-tool/pkg/fetch"
	"github.com/bk201/image-tool/pkg/imagelist"
	"github.com/bk201/image-tool/pkg/subsystem"
)

const releaseImagesURL = "https://github.com/rancher/rancher/releases/download/v2.15.1/rancher-images.txt"

func newFullFixtureFake(t *testing.T) *fetch.Fake {
	t.Helper()
	fk := newFixtureFake(t)
	release := fetch.FakeFromFiles(t, map[string]string{releaseImagesURL: "testdata/rancher-images.txt"})
	fk.Bodies[releaseImagesURL] = release.Bodies[releaseImagesURL]
	return fk
}

func TestRancherSubsystem_Collect_GoldenOutput(t *testing.T) {
	chartsBranchOverride = "release-v2.15"
	t.Cleanup(func() { chartsBranchOverride = "" })

	fk := newFullFixtureFake(t)
	s := rancherSubsystem{}

	results, err := s.Collect(context.Background(), subsystem.Options{Fetcher: fk, Version: "v2.15.1"})
	require.NoError(t, err)
	require.Len(t, results, len(components))

	var all []string
	for _, r := range results {
		all = append(all, r.Images...)
	}
	require.Len(t, all, 12, "12 discovered refs before dedupe")

	got := imagelist.Normalize(all)
	want := []string{
		"rancher/cluster-api-controller:v1.13.3",
		"rancher/fleet-agent:v0.16.1",
		"rancher/fleet:v0.16.1",
		"rancher/kuberlr-kubectl:v8.1.1",
		"rancher/rancher-agent:v2.15.1",
		"rancher/rancher-webhook:v0.11.1",
		"rancher/rancher:v2.15.1",
		"rancher/shell:v0.8.1",
		"rancher/system-agent:v0.15.1-suc",
		"rancher/system-upgrade-controller:v0.20.1",
		"rancher/turtles:v0.27.1",
	}
	assert.Equal(t, want, got)

	missing, err := s.Verify(context.Background(), subsystem.Options{Fetcher: fk, Version: "v2.15.1"}, got)
	require.NoError(t, err)
	assert.Empty(t, missing, "every generated image must appear in the official release list")
}

func TestRancherSubsystem_Verify_ReportsMissing(t *testing.T) {
	fk := newFullFixtureFake(t)
	s := rancherSubsystem{}

	missing, err := s.Verify(context.Background(), subsystem.Options{Fetcher: fk, Version: "v2.15.1"}, []string{
		"rancher/rancher:v2.15.1",
		"rancher/does-not-exist:v0.0.0",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"rancher/does-not-exist:v0.0.0"}, missing)
}

func TestRancherSubsystem_Collect_UnknownVersion(t *testing.T) {
	fk := &fetch.Fake{}
	s := rancherSubsystem{}

	_, err := s.Collect(context.Background(), subsystem.Options{Fetcher: fk, Version: "v9.99.9"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVersionNotFound)
}

func TestNew_ImplementsSubsystemInterfaces(t *testing.T) {
	s := New()
	assert.Equal(t, "rancher", s.Name())

	_, ok := s.(subsystem.Verifier)
	assert.True(t, ok, "rancher subsystem should implement subsystem.Verifier")
	_, ok = s.(subsystem.FlagRegistrar)
	assert.True(t, ok, "rancher subsystem should implement subsystem.FlagRegistrar")
}
