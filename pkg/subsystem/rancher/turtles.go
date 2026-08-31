package rancher

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"go.yaml.in/yaml/v4"

	"github.com/bk201/image-tool/pkg/chart"
)

const (
	turtlesChartName        = "rancher-turtles"
	coreProviderConfigMap   = "templates/core-provider-configmap.yaml"
	capiVersionLabel        = "provider.cluster.x-k8s.io/version"
	clusterAPIControllerImg = "rancher/cluster-api-controller"
)

type turtlesComponent struct{}

func (turtlesComponent) Name() string { return "turtles" }

func (turtlesComponent) Images(ctx context.Context, rc *Context) ([]string, error) {
	build, err := rc.BuildYAML(ctx)
	if err != nil {
		return nil, err
	}
	if build.TurtlesVersion == "" {
		return nil, fmt.Errorf("turtles: turtlesVersion is empty in build.yaml")
	}
	rc.Log().WithFields(logFields("build.yaml", "turtlesVersion")).Infof("turtlesVersion = %s", build.TurtlesVersion)

	var values struct {
		Image      chart.Image `yaml:"image"`
		ShellImage struct {
			Image chart.Image `yaml:"image"`
		} `yaml:"shellImage"`
	}
	if err := rc.ChartValues(ctx, turtlesChartName, build.TurtlesVersion, &values); err != nil {
		return nil, err
	}

	valuesFile := fmt.Sprintf("charts/%s/%s/values.yaml", turtlesChartName, build.TurtlesVersion)
	rc.Log().WithFields(logFields(valuesFile, "image.repository,image.tag")).
		Infof("image.repository = %s, image.tag = %s", values.Image.Repository, values.Image.Tag)
	rc.Log().WithFields(logFields(valuesFile, "shellImage.image.repository,.tag")).
		Infof("shellImage.image.repository = %s, shellImage.image.tag = %s", values.ShellImage.Image.Repository, values.ShellImage.Image.Tag)

	turtlesRef, err := values.Image.Ref()
	if err != nil {
		return nil, fmt.Errorf("turtles: image: %w", err)
	}
	shellRef, err := values.ShellImage.Image.Ref()
	if err != nil {
		return nil, fmt.Errorf("turtles: shellImage.image: %w", err)
	}

	capiVersion, err := capiControllerVersion(ctx, rc, build.TurtlesVersion)
	if err != nil {
		return nil, err
	}

	return []string{turtlesRef, shellRef, clusterAPIControllerImg + ":" + capiVersion}, nil
}

// capiControllerVersion extracts the cluster-api-controller version from the
// turtles chart's core-provider-configmap.yaml, which carries it as a
// metadata label rather than a values.yaml field.
func capiControllerVersion(ctx context.Context, rc *Context, turtlesChartVersion string) (string, error) {
	body, err := rc.ChartFile(ctx, turtlesChartName, turtlesChartVersion, coreProviderConfigMap)
	if err != nil {
		return "", fmt.Errorf("turtles: core-provider-configmap.yaml: %w", err)
	}
	if bytes.Contains(body, []byte("{{")) {
		return "", fmt.Errorf("turtles: core-provider-configmap.yaml appears to be Helm-templated, not plain YAML")
	}

	type configMap struct {
		Kind     string `yaml:"kind"`
		Metadata struct {
			Labels map[string]string `yaml:"labels"`
		} `yaml:"metadata"`
	}

	dec := yaml.NewDecoder(bytes.NewReader(body))
	for {
		var cm configMap
		if err := dec.Decode(&cm); err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("turtles: parse core-provider-configmap.yaml: %w", err)
		}
		if version := cm.Metadata.Labels[capiVersionLabel]; version != "" {
			configMapFile := fmt.Sprintf("charts/%s/%s/%s", turtlesChartName, turtlesChartVersion, coreProviderConfigMap)
			rc.Log().WithFields(logFields(configMapFile, "metadata.labels[\""+capiVersionLabel+"\"]")).
				Infof("metadata.labels[%q] = %s", capiVersionLabel, version)
			return version, nil
		}
	}

	return "", fmt.Errorf("turtles: label %s not found in core-provider-configmap.yaml", capiVersionLabel)
}
