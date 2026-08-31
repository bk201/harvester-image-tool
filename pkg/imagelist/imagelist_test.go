package imagelist

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name   string
		images []string
		want   []string
	}{
		{
			name:   "dedupes and sorts in byte order",
			images: []string{"rancher/fleet:v0.16.1", "rancher/fleet-agent:v0.16.1", "rancher/fleet:v0.16.1"},
			want:   []string{"rancher/fleet-agent:v0.16.1", "rancher/fleet:v0.16.1"},
		},
		{
			name:   "the real dedupe case: kuberlr-kubectl from two components",
			images: []string{"rancher/kuberlr-kubectl:v8.1.1", "rancher/turtles:v0.27.1", "rancher/kuberlr-kubectl:v8.1.1", "rancher/system-upgrade-controller:v0.20.1"},
			want:   []string{"rancher/kuberlr-kubectl:v8.1.1", "rancher/system-upgrade-controller:v0.20.1", "rancher/turtles:v0.27.1"},
		},
		{
			name:   "empty input",
			images: nil,
			want:   []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Normalize(tt.images))
		})
	}
}

func TestNormalizeRef(t *testing.T) {
	tests := []struct {
		ref  string
		want string
	}{
		{"docker.io/rancher/rancher:v2.15.1", "rancher/rancher:v2.15.1"},
		{"index.docker.io/rancher/rancher:v2.15.1", "rancher/rancher:v2.15.1"},
		{"rancher/rancher:v2.15.1", "rancher/rancher:v2.15.1"},
		{"quay.io/rancher/rancher:v2.15.1", "quay.io/rancher/rancher:v2.15.1"},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeRef(tt.ref))
		})
	}
}

func TestWrite(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, []string{"rancher/fleet:v0.16.1", "rancher/rancher:v2.15.1"}, []string{"subsystem: rancher", "version: v2.15.1"})
	require.NoError(t, err)

	want := "# subsystem: rancher\n# version: v2.15.1\nrancher/fleet:v0.16.1\nrancher/rancher:v2.15.1\n"
	assert.Equal(t, want, buf.String())
}

func TestWrite_NoHeader(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, []string{"rancher/fleet:v0.16.1"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "rancher/fleet:v0.16.1\n", buf.String())
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "images.txt")

	err := WriteFile(path, []string{"rancher/fleet:v0.16.1"}, []string{"subsystem: rancher"})
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "# subsystem: rancher\nrancher/fleet:v0.16.1\n", string(got))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())

	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no leftover temp file")
}
