package chart

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestImage_Ref(t *testing.T) {
	tests := []struct {
		name    string
		image   Image
		want    string
		wantErr bool
	}{
		{"ok", Image{Repository: "rancher/fleet", Tag: "v0.16.1"}, "rancher/fleet:v0.16.1", false},
		{"empty repository", Image{Tag: "v0.16.1"}, "", true},
		{"empty tag", Image{Repository: "rancher/fleet"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.image.Ref()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestImage_YAMLTrailingComment(t *testing.T) {
	// A trailing "# comment" after a scalar must not leak into the parsed value.
	data := []byte("repository: rancher/kuberlr-kubectl\ntag: v8.1.1 # For compatibility, see https://example.com\n")

	var img Image
	require.NoError(t, yaml.Unmarshal(data, &img))

	ref, err := img.Ref()
	require.NoError(t, err)
	assert.Equal(t, "rancher/kuberlr-kubectl:v8.1.1", ref)
}
