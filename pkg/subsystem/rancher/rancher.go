// Package rancher implements the "rancher" subsystem: discovering the
// container images for a Rancher release from its build.yaml, package
// Dockerfile, and rancher/charts values files.
package rancher

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/bk201/image-tool/pkg/imagelist"
	"github.com/bk201/image-tool/pkg/subsystem"
)

const releaseImagesURLTemplate = "https://github.com/rancher/rancher/releases/download/%s/rancher-images.txt"

var chartsBranchOverride string

// New returns the rancher subsystem. Callers register it explicitly with
// subsystem.Register — this package does not self-register via init().
func New() subsystem.Subsystem {
	return rancherSubsystem{}
}

type rancherSubsystem struct{}

func (rancherSubsystem) Name() string { return "rancher" }

func (rancherSubsystem) RegisterFlags(fs *pflag.FlagSet) {
	fs.StringVar(&chartsBranchOverride, "rancher-charts-branch", "", "override the rancher/charts branch to read from (default: derived from the version, e.g. release-v2.15)")
}

func (rancherSubsystem) Collect(ctx context.Context, o subsystem.Options) ([]subsystem.ComponentResult, error) {
	rc := &Context{
		Fetcher:      o.Fetcher,
		Version:      o.Version,
		ChartsBranch: chartsBranchOverride,
	}

	results := make([]subsystem.ComponentResult, 0, len(components))
	for _, c := range components {
		rc.Component = c.Name()
		logrus.WithField("component", c.Name()).Info("discovering images")

		images, err := c.Images(ctx, rc)
		if err != nil {
			return nil, fmt.Errorf("component %s: %w", c.Name(), err)
		}
		logrus.WithFields(logrus.Fields{"component": c.Name(), "images": images}).Info("discovered images")

		results = append(results, subsystem.ComponentResult{Component: c.Name(), Images: images})
	}
	return results, nil
}

// Verify cross-checks images against the official release image list, which
// contains every image in a Rancher release (unsorted, uncategorized) at
// https://github.com/rancher/rancher/releases/download/<version>/rancher-images.txt.
func (rancherSubsystem) Verify(ctx context.Context, o subsystem.Options, images []string) ([]string, error) {
	releaseURL := fmt.Sprintf(releaseImagesURLTemplate, o.Version)
	body, err := o.Fetcher.Get(ctx, releaseURL)
	if err != nil {
		return nil, fmt.Errorf("fetch official release image list: %w", err)
	}

	official := make(map[string]struct{})
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		official[imagelist.NormalizeRef(line)] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read official release image list: %w", err)
	}

	var missing []string
	for _, image := range images {
		if _, ok := official[imagelist.NormalizeRef(image)]; !ok {
			missing = append(missing, image)
		}
	}
	return missing, nil
}
