package chart

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/bk201/image-tool/pkg/fetch"
)

func TestValuesMerge(t *testing.T) {
	var defaults, overrides Values
	require.NoError(t, yaml.Unmarshal([]byte("image: {repository: example/image, tag: 1.0}\nlist: [a, b]\nenabled: true\nnullable: value"), &defaults))
	require.NoError(t, yaml.Unmarshal([]byte("image: {tag: 2.0}\nlist: [c]\nnullable: null"), &overrides))
	result := Merge(defaults, overrides)
	assert.Equal(t, "example/image", result.Get("image.repository"))
	assert.Equal(t, "2.0", result.Get("image.tag"))
	assert.Equal(t, "1.0", defaults.Get("image.tag"))
	assert.Equal(t, []any{"c"}, result.Get("list"))
	assert.Nil(t, result.Get("nullable"))
	enabled, err := result.Enabled("enabled")
	require.NoError(t, err)
	assert.True(t, enabled)
	_, err = result.Enabled("absent")
	require.ErrorContains(t, err, "expected boolean")
}

func TestInvalidValues(t *testing.T) {
	for _, body := range []string{"[a,b]", "key: one\nkey: two", "key: ["} {
		var v Values
		require.Error(t, yaml.Unmarshal([]byte(body), &v))
	}
}

func TestSourceErrors(t *testing.T) {
	const u = "https://raw.githubusercontent.com/rancher/charts/test/charts/test/1.0+up1/values.yaml"
	f := &fetch.Fake{Errs: map[string]error{u: assert.AnError}}
	s := Source{Fetcher: f, Branch: "test", Name: "test", Version: "1.0+up1"}
	var v Values
	err := s.Read(context.Background(), "values.yaml", &v)
	require.ErrorIs(t, err, assert.AnError)
	require.ErrorContains(t, err, "chart test/1.0+up1 file values.yaml")
	assert.Equal(t, []string{u}, f.Calls)
}
