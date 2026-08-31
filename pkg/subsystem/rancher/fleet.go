package rancher

import (
	"context"
	"fmt"

	"github.com/bk201/image-tool/pkg/chart"
)

const fleetChartName = "fleet"

type fleetComponent struct{}

func (fleetComponent) Name() string { return "fleet" }

func (fleetComponent) Images(ctx context.Context, rc *Context) ([]string, error) {
	build, err := rc.BuildYAML(ctx)
	if err != nil {
		return nil, err
	}
	if build.FleetVersion == "" {
		return nil, fmt.Errorf("fleet: fleetVersion is empty in build.yaml")
	}
	rc.Log().WithFields(logFields("build.yaml", "fleetVersion")).Infof("fleetVersion = %s", build.FleetVersion)

	var values struct {
		Image      chart.Image `yaml:"image"`
		AgentImage chart.Image `yaml:"agentImage"`
	}
	if err := rc.ChartValues(ctx, fleetChartName, build.FleetVersion, &values); err != nil {
		return nil, err
	}

	valuesFile := fmt.Sprintf("charts/%s/%s/values.yaml", fleetChartName, build.FleetVersion)
	rc.Log().WithFields(logFields(valuesFile, "image.repository,image.tag")).
		Infof("image.repository = %s, image.tag = %s", values.Image.Repository, values.Image.Tag)
	rc.Log().WithFields(logFields(valuesFile, "agentImage.repository,agentImage.tag")).
		Infof("agentImage.repository = %s, agentImage.tag = %s", values.AgentImage.Repository, values.AgentImage.Tag)

	controllerRef, err := values.Image.Ref()
	if err != nil {
		return nil, fmt.Errorf("fleet: controller image: %w", err)
	}
	agentRef, err := values.AgentImage.Ref()
	if err != nil {
		return nil, fmt.Errorf("fleet: agent image: %w", err)
	}

	return []string{controllerRef, agentRef}, nil
}
