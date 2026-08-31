package rancher

import (
	"context"
	"fmt"

	"github.com/bk201/image-tool/pkg/chart"
)

const webhookChartName = "rancher-webhook"

type webhookComponent struct{}

func (webhookComponent) Name() string { return "rancher-webhook" }

func (webhookComponent) Images(ctx context.Context, rc *Context) ([]string, error) {
	build, err := rc.BuildYAML(ctx)
	if err != nil {
		return nil, err
	}
	if build.WebhookVersion == "" {
		return nil, fmt.Errorf("rancher-webhook: webhookVersion is empty in build.yaml")
	}
	rc.Log().WithFields(logFields("build.yaml", "webhookVersion")).Infof("webhookVersion = %s", build.WebhookVersion)

	var values struct {
		Image chart.Image `yaml:"image"`
	}
	if err := rc.ChartValues(ctx, webhookChartName, build.WebhookVersion, &values); err != nil {
		return nil, err
	}

	valuesFile := fmt.Sprintf("charts/%s/%s/values.yaml", webhookChartName, build.WebhookVersion)
	rc.Log().WithFields(logFields(valuesFile, "image.repository,image.tag")).
		Infof("image.repository = %s, image.tag = %s", values.Image.Repository, values.Image.Tag)

	ref, err := values.Image.Ref()
	if err != nil {
		return nil, fmt.Errorf("rancher-webhook: %w", err)
	}
	return []string{ref}, nil
}
