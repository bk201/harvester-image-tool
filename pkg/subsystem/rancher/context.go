package rancher

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"go.yaml.in/yaml/v4"

	"github.com/bk201/image-tool/pkg/fetch"
)

const (
	defaultRancherRawBase = "https://raw.githubusercontent.com/rancher/rancher"
	defaultChartsRawBase  = "https://raw.githubusercontent.com/rancher/charts"
)

// ErrVersionNotFound is returned when the requested Rancher version does not
// exist upstream (a 404 on build.yaml doubles as the existence check).
var ErrVersionNotFound = errors.New("rancher: version not found")

// Context carries everything a Component needs to fetch and parse Rancher's
// upstream sources for one version.
type Context struct {
	Fetcher fetch.Fetcher
	Version string // e.g. v2.15.1

	// ChartsBranch is the rancher/charts branch to read from. If empty, it
	// is derived from Version as "release-v<major>.<minor>".
	ChartsBranch string

	// RancherRawBase and ChartsRawBase override the raw.githubusercontent
	// bases for rancher/rancher and rancher/charts respectively. Tests point
	// these at an httptest server; production leaves them empty to use the
	// real upstream hosts.
	RancherRawBase string
	ChartsRawBase  string

	// Component is the name of the component currently calling into this
	// Context. It is set by rancherSubsystem.Collect before each Component's
	// Images call and is used only to tag log messages.
	Component string
}

// Log returns a logrus entry tagged with the currently running component,
// for components and Context methods to report which file and field they
// are working with.
func (c *Context) Log() *logrus.Entry {
	return logrus.WithField("component", c.Component)
}

// BuildYAML is the subset of rancher/rancher's build.yaml that determines
// component versions.
type BuildYAML struct {
	WebhookVersion      string `yaml:"webhookVersion"`
	TurtlesVersion      string `yaml:"turtlesVersion"`
	FleetVersion        string `yaml:"fleetVersion"`
	DefaultShellVersion string `yaml:"defaultShellVersion"`
}

func (c *Context) rancherRawBase() string {
	if c.RancherRawBase != "" {
		return c.RancherRawBase
	}
	return defaultRancherRawBase
}

func (c *Context) chartsRawBase() string {
	if c.ChartsRawBase != "" {
		return c.ChartsRawBase
	}
	return defaultChartsRawBase
}

func (c *Context) chartsBranch() (string, error) {
	if c.ChartsBranch != "" {
		return c.ChartsBranch, nil
	}
	return ChartsBranchFor(c.Version)
}

// ChartsBranchFor derives the rancher/charts branch for a Rancher version,
// e.g. "v2.15.1" -> "release-v2.15". This matches package/Dockerfile's own
// ARG CHART_DEFAULT_BRANCH default for the corresponding release line.
func ChartsBranchFor(version string) (string, error) {
	v := strings.TrimPrefix(version, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("cannot derive charts branch from version %q", version)
	}
	return fmt.Sprintf("release-v%s.%s", parts[0], parts[1]), nil
}

func (c *Context) rancherFile(ctx context.Context, relPath string) ([]byte, error) {
	u, err := url.Parse(c.rancherRawBase())
	if err != nil {
		return nil, fmt.Errorf("parse rancher raw base %q: %w", c.rancherRawBase(), err)
	}
	u = u.JoinPath(c.Version, relPath)

	c.Log().WithField("file", "rancher/rancher/"+relPath).Infof("reading %s", relPath)

	body, err := c.Fetcher.Get(ctx, u.String())
	if err != nil {
		if fetch.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrVersionNotFound, c.Version)
		}
		return nil, err
	}
	return body, nil
}

// BuildYAML fetches and parses rancher/rancher's build.yaml for c.Version.
func (c *Context) BuildYAML(ctx context.Context) (*BuildYAML, error) {
	body, err := c.rancherFile(ctx, "build.yaml")
	if err != nil {
		return nil, err
	}

	var b BuildYAML
	if err := yaml.Unmarshal(body, &b); err != nil {
		return nil, fmt.Errorf("parse build.yaml: %w", err)
	}
	return &b, nil
}

// Dockerfile fetches rancher/rancher's package/Dockerfile for c.Version.
func (c *Context) Dockerfile(ctx context.Context) ([]byte, error) {
	return c.rancherFile(ctx, "package/Dockerfile")
}

// ChartFile fetches relPath from rancher/charts at
// <charts-branch>/charts/<chart>/<chartVersion>/<relPath>.
//
// chartVersion arrives decoded (e.g. "110.0.1+up0.16.1"); url.JoinPath
// escapes each path segment exactly once, so it must never be pre-escaped
// before being passed here.
func (c *Context) ChartFile(ctx context.Context, chartName, chartVersion, relPath string) ([]byte, error) {
	branch, err := c.chartsBranch()
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(c.chartsRawBase())
	if err != nil {
		return nil, fmt.Errorf("parse charts raw base %q: %w", c.chartsRawBase(), err)
	}
	u = u.JoinPath(branch, "charts", chartName, chartVersion, relPath)

	file := fmt.Sprintf("charts/%s/%s/%s", chartName, chartVersion, relPath)
	c.Log().WithField("file", file).Infof("reading %s", file)

	return c.Fetcher.Get(ctx, u.String())
}

// ChartValues fetches and parses <chart>/<chartVersion>/values.yaml into out.
func (c *Context) ChartValues(ctx context.Context, chartName, chartVersion string, out any) error {
	body, err := c.ChartFile(ctx, chartName, chartVersion, "values.yaml")
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse %s/%s values.yaml: %w", chartName, chartVersion, err)
	}
	return nil
}
