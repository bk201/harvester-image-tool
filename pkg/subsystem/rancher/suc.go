package rancher

import (
	"context"
	"fmt"

	"github.com/bk201/image-tool/pkg/chart"
	"github.com/bk201/image-tool/pkg/dockerfile"
)

const (
	sucChartName          = "system-upgrade-controller"
	sucChartVersionEnvKey = "CATTLE_SYSTEM_UPGRADE_CONTROLLER_CHART_VERSION"
)

type sucComponent struct{}

func (sucComponent) Name() string { return "system-upgrade-controller" }

func (sucComponent) Images(ctx context.Context, rc *Context) ([]string, error) {
	dockerfileContent, err := rc.Dockerfile(ctx)
	if err != nil {
		return nil, err
	}

	chartVersion, ok := dockerfile.EnvValue(dockerfileContent, sucChartVersionEnvKey)
	if !ok || chartVersion == "" {
		return nil, fmt.Errorf("system-upgrade-controller: %s not found in package/Dockerfile", sucChartVersionEnvKey)
	}
	rc.Log().WithFields(logFields("package/Dockerfile", sucChartVersionEnvKey)).Infof("%s = %s", sucChartVersionEnvKey, chartVersion)

	var values struct {
		SystemUpgradeController struct {
			Image chart.Image `yaml:"image"`
		} `yaml:"systemUpgradeController"`
		Kubectl struct {
			Image chart.Image `yaml:"image"`
		} `yaml:"kubectl"`
	}
	if err := rc.ChartValues(ctx, sucChartName, chartVersion, &values); err != nil {
		return nil, err
	}

	valuesFile := fmt.Sprintf("charts/%s/%s/values.yaml", sucChartName, chartVersion)
	rc.Log().WithFields(logFields(valuesFile, "systemUpgradeController.image.repository,.tag")).
		Infof("systemUpgradeController.image = %s:%s", values.SystemUpgradeController.Image.Repository, values.SystemUpgradeController.Image.Tag)
	rc.Log().WithFields(logFields(valuesFile, "kubectl.image.repository,.tag")).
		Infof("kubectl.image = %s:%s", values.Kubectl.Image.Repository, values.Kubectl.Image.Tag)

	sucRef, err := values.SystemUpgradeController.Image.Ref()
	if err != nil {
		return nil, fmt.Errorf("system-upgrade-controller: controller image: %w", err)
	}
	kubectlRef, err := values.Kubectl.Image.Ref()
	if err != nil {
		return nil, fmt.Errorf("system-upgrade-controller: kubectl image: %w", err)
	}

	return []string{sucRef, kubectlRef}, nil
}
