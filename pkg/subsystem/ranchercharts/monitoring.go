package ranchercharts

import (
	"context"

	"github.com/bk201/image-tool/pkg/chart"
)

func (c *collection) monitoring(ctx context.Context, main, crd *chartData) error {
	// These are the image-relevant Harvester addon overrides. Chart versions
	// remain the sole source of image repositories and tags.
	main.values = chart.Merge(main.values, chart.Values{
		"rancherMonitoring": chart.Values{"enabled": "false"},
		"grafana":           chart.Values{"persistence": chart.Values{"enabled": "true", "type": "pvc"}},
	})
	for _, group := range []struct {
		name, condition string
		fields          []imageField
	}{
		{"prometheus", "prometheus.enabled", []imageField{{"prometheus.prometheusSpec.image", monitoringImage}, {"prometheus.prometheusSpec.proxy.image", cattleImage}}},
		{"alertmanager", "alertmanager.enabled", []imageField{{"alertmanager.alertmanagerSpec.image", monitoringImage}}},
		{"prometheus-operator", "prometheusOperator.enabled", []imageField{{"prometheusOperator.image", operatorImage}, {"prometheusOperator.prometheusConfigReloader.image", operatorImage}}},
		{"monitoring-upgrade", "upgrade.enabled", []imageField{{"upgrade.image", cattleImage}}},
	} {
		enabled, err := main.enabled(group.condition)
		if err != nil {
			return err
		}
		if enabled {
			if err = c.add(group.name, main, group.fields...); err != nil {
				return err
			}
		}
	}
	// Include the standalone webhook image in the inventory even when its
	// deployment is disabled. Its registry rules differ from the operator's.
	webhook := imagePolicy{registry: "webhook", digest: "sha256", fallback: true}
	if err := c.add("admission-webhook", main, imageField{"prometheusOperator.admissionWebhooks.deployment.image", webhook}); err != nil {
		return err
	}
	if err := c.admissionPatch(main); err != nil {
		return err
	}
	crdUpgrade, err := main.enabled("crds.upgradeJob.enabled")
	if err != nil {
		return err
	}
	crdsEnabled, err := main.enabled("crds.enabled")
	if err != nil {
		return err
	}
	// Keep the chart-defined BusyBox image in the inventory even when the
	// CRD upgrade job is disabled. Kubectl still follows the job's switches.
	crdUpgradeImages := []imageField{{"crds.upgradeJob.image.busybox", cattleImage}}
	if crdUpgrade && crdsEnabled {
		crdUpgradeImages = append(crdUpgradeImages, imageField{"crds.upgradeJob.image.kubectl", cattleImage})
	}
	if err = c.add("monitoring-crd-upgrade", main, crdUpgradeImages...); err != nil {
		return err
	}
	for _, sub := range []struct {
		name, condition string
		policy          imagePolicy
	}{
		{"grafana", "grafana.enabled", grafanaImage},
		{"kube-state-metrics", "kubeStateMetrics.enabled", imagePolicy{registry: "exporter", fallback: true, versionPrefix: "v", digest: "sha"}},
		{"prometheus-node-exporter", "nodeExporter.enabled", imagePolicy{registry: "exporter", fallback: true, versionPrefix: "v", digest: "digest"}},
		{"prometheus-adapter", "prometheus-adapter.enabled", imagePolicy{registry: "cattle", fallback: true}},
	} {
		enabled, err := main.enabled(sub.condition)
		if err != nil {
			return err
		}
		if !enabled {
			continue
		}
		child, err := main.child(ctx, sub.name)
		if err != nil {
			return err
		}
		if sub.name == "grafana" {
			err = c.grafana(child)
		} else {
			err = c.add(sub.name, child, imageField{"image", sub.policy})
		}
		if err != nil {
			return err
		}
	}
	return c.add(crd.source.Name, crd, imageField{"image", cattleImage})
}

func (c *collection) admissionPatch(main *chartData) error {
	for _, path := range []string{"prometheusOperator.enabled", "prometheusOperator.admissionWebhooks.enabled", "prometheusOperator.admissionWebhooks.patch.enabled"} {
		enabled, err := main.enabled(path)
		if err != nil {
			return err
		}
		if !enabled {
			return nil
		}
	}
	certManager, err := main.enabled("prometheusOperator.admissionWebhooks.certManager.enabled")
	if err != nil {
		return err
	}
	if certManager {
		return nil
	}
	patch := monitoringImage
	patch.digestOnly = false
	return c.add("admission-patch", main, imageField{"prometheusOperator.admissionWebhooks.patch.image", patch})
}

func (c *collection) grafana(d *chartData) error {
	mainImage := grafanaImage
	mainImage.fallback = true
	fields := []imageField{{"image", mainImage}, {"proxy.image", cattleImage}}
	dashboards, err := d.enabled("sidecar.dashboards.enabled")
	if err != nil {
		return err
	}
	datasources, err := d.enabled("sidecar.datasources.enabled")
	if err != nil {
		return err
	}
	if dashboards || datasources {
		fields = append(fields, imageField{"sidecar.image", grafanaImage})
	}
	init, err := d.enabled("initChownData.enabled")
	if err != nil {
		return err
	}
	if init {
		fields = append(fields, imageField{"initChownData.image", grafanaImage})
	}
	return c.add("grafana", d, fields...)
}
