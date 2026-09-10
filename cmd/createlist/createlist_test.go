package createlist

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	rootcmd "github.com/bk201/image-tool/cmd"
	"github.com/bk201/image-tool/pkg/subsystem/ranchercharts"
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
	chartBranch = ""
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

// fixtureTransport exercises run's real HTTP fetcher and flag forwarding while
// serving only committed fixtures. Unexpected URLs fail without network access.
type fixtureTransport struct {
	calls []string
}

func (f *fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.calls = append(f.calls, r.URL.String())
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/rancher/charts/test-branch/charts/"), "/", 3)
	if len(parts) != 3 || !strings.HasPrefix(r.URL.Path, "/rancher/charts/test-branch/charts/") {
		return nil, assert.AnError
	}
	p := filepath.Join("../../pkg/subsystem/ranchercharts/testdata", parts[0], parts[2])
	body, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{}, Request: r}, nil
}

func TestChartSubsystemCLI(t *testing.T) {
	resetFlags()
	t.Cleanup(resetFlags)
	transport := &fixtureTransport{}
	oldTransport, oldInterval := http.DefaultTransport, rootcmd.MinRequestInterval
	http.DefaultTransport = transport
	rootcmd.MinRequestInterval = time.Nanosecond
	t.Cleanup(func() { http.DefaultTransport = oldTransport; rootcmd.MinRequestInterval = oldInterval })
	subsystem.Register(ranchercharts.NewLogging())
	subsystem.Register(ranchercharts.NewMonitoring())
	RegisterSubsystemFlags()
	require.NotNil(t, command.Flags().Lookup("chart-branch"))
	assert.Contains(t, command.Long, "rancher-logging")
	assert.Contains(t, command.Long, "rancher-monitoring")
	for _, tc := range []struct {
		name, version string
		count         int
	}{
		{"rancher-monitoring", "109.0.3+up80.9.1-rancher.14", 16},
		{"rancher-logging", "109.0.0+up4.10.0-rancher.23", 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetFlags()
			outputPath = filepath.Join(t.TempDir(), "images.txt")
			require.NoError(t, os.WriteFile(outputPath, []byte("existing\n"), 0600))
			var out bytes.Buffer
			err := run(newTestCmd(&out), []string{tc.name, tc.version})
			require.ErrorContains(t, err, "--chart-branch")
			body, err := os.ReadFile(outputPath)
			require.NoError(t, err)
			assert.Equal(t, "existing\n", string(body))
			require.NoError(t, command.Flags().Set("chart-branch", "test-branch"))
			noHeader = true
			require.NoError(t, run(newTestCmd(&out), []string{tc.name, tc.version}))
			body, err = os.ReadFile(outputPath)
			require.NoError(t, err)
			lines := strings.Split(strings.TrimSpace(string(body)), "\n")
			assert.Len(t, lines, tc.count)
			assert.IsIncreasing(t, lines)
			assert.Empty(t, out.String())
		})
	}
	for _, u := range transport.calls {
		assert.NotContains(t, u, ".tgz")
		assert.NotContains(t, u, "/assets/")
	}
}
