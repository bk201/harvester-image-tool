package createlist

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bk201/image-tool/pkg/subsystem"
)

type fakeSubsystem struct {
	name         string
	images       []string
	collectErr   error
	missing      []string
	verifyErr    error
	verifyCalled bool
	verifyImages []string
}

func (f *fakeSubsystem) Name() string { return f.name }

func (f *fakeSubsystem) Collect(_ context.Context, _ subsystem.Options) ([]subsystem.ComponentResult, error) {
	if f.collectErr != nil {
		return nil, f.collectErr
	}
	return []subsystem.ComponentResult{{Component: "only", Images: f.images}}, nil
}

func (f *fakeSubsystem) Verify(_ context.Context, _ subsystem.Options, images []string) ([]string, error) {
	f.verifyCalled = true
	f.verifyImages = images
	return f.missing, f.verifyErr
}

func resetFlags() {
	outputPath = ""
	noVerify = false
	strict = false
	noHeader = false
}

func newTestCmd(out *bytes.Buffer) *cobra.Command {
	c := &cobra.Command{}
	c.SetOut(out)
	c.SetContext(context.Background())
	return c
}

func TestRun_WritesSortedDedupedListWithHeader(t *testing.T) {
	resetFlags()
	t.Cleanup(resetFlags)

	fake := &fakeSubsystem{name: "cmdtest-ok", images: []string{"b/img:v1", "a/img:v1", "b/img:v1"}}
	subsystem.Register(fake)

	var out bytes.Buffer
	err := run(newTestCmd(&out), []string{"cmdtest-ok", "v1.0.0"})
	require.NoError(t, err)

	want := "# subsystem: cmdtest-ok\n# version: v1.0.0\na/img:v1\nb/img:v1\n"
	assert.Equal(t, want, out.String())
	assert.True(t, fake.verifyCalled)
	assert.Equal(t, []string{"a/img:v1", "b/img:v1"}, fake.verifyImages)
}

func TestRun_NoHeader(t *testing.T) {
	resetFlags()
	t.Cleanup(resetFlags)
	noHeader = true

	fake := &fakeSubsystem{name: "cmdtest-noheader", images: []string{"a/img:v1"}}
	subsystem.Register(fake)

	var out bytes.Buffer
	err := run(newTestCmd(&out), []string{"cmdtest-noheader", "v1.0.0"})
	require.NoError(t, err)
	assert.Equal(t, "a/img:v1\n", out.String())
}

func TestRun_UnknownSubsystem(t *testing.T) {
	resetFlags()
	t.Cleanup(resetFlags)

	var out bytes.Buffer
	err := run(newTestCmd(&out), []string{"does-not-exist", "v1.0.0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown subsystem")
}

func TestRun_CollectError(t *testing.T) {
	resetFlags()
	t.Cleanup(resetFlags)

	fake := &fakeSubsystem{name: "cmdtest-collecterr", collectErr: assert.AnError}
	subsystem.Register(fake)

	var out bytes.Buffer
	err := run(newTestCmd(&out), []string{"cmdtest-collecterr", "v1.0.0"})
	require.Error(t, err)
}

func TestRun_NoVerify_SkipsVerification(t *testing.T) {
	resetFlags()
	t.Cleanup(resetFlags)
	noVerify = true

	fake := &fakeSubsystem{name: "cmdtest-noverify", images: []string{"a/img:v1"}}
	subsystem.Register(fake)

	var out bytes.Buffer
	err := run(newTestCmd(&out), []string{"cmdtest-noverify", "v1.0.0"})
	require.NoError(t, err)
	assert.False(t, fake.verifyCalled)
}

func TestRun_MissingImages_WarnsButSucceeds(t *testing.T) {
	resetFlags()
	t.Cleanup(resetFlags)

	fake := &fakeSubsystem{name: "cmdtest-missing", images: []string{"a/img:v1"}, missing: []string{"a/img:v1"}}
	subsystem.Register(fake)

	var out bytes.Buffer
	err := run(newTestCmd(&out), []string{"cmdtest-missing", "v1.0.0"})
	require.NoError(t, err)
}

func TestRun_MissingImages_StrictFails(t *testing.T) {
	resetFlags()
	t.Cleanup(resetFlags)
	strict = true

	fake := &fakeSubsystem{name: "cmdtest-strict", images: []string{"a/img:v1"}, missing: []string{"a/img:v1"}}
	subsystem.Register(fake)

	var out bytes.Buffer
	err := run(newTestCmd(&out), []string{"cmdtest-strict", "v1.0.0"})
	require.Error(t, err)
}

func TestRun_WritesToOutputFile(t *testing.T) {
	resetFlags()
	t.Cleanup(resetFlags)

	dir := t.TempDir()
	outputPath = filepath.Join(dir, "images.txt")

	fake := &fakeSubsystem{name: "cmdtest-outfile", images: []string{"a/img:v1"}}
	subsystem.Register(fake)

	var out bytes.Buffer
	err := run(newTestCmd(&out), []string{"cmdtest-outfile", "v1.0.0"})
	require.NoError(t, err)
	assert.Empty(t, out.String(), "nothing should be written to stdout when -o is set")

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, "# subsystem: cmdtest-outfile\n# version: v1.0.0\na/img:v1\n", string(content))
}
