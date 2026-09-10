// Package ranchercharts discovers upstream monitoring and logging chart images
// for a built-in Harvester component selection. Sources are individual files;
// collection never downloads chart archives or executes Helm.
package ranchercharts

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/bk201/image-tool/pkg/chart"
	"github.com/bk201/image-tool/pkg/subsystem"
)

type chartSubsystem struct{ name string }

// NewMonitoring returns the standalone monitoring chart subsystem.
func NewMonitoring() subsystem.Subsystem { return chartSubsystem{name: "rancher-monitoring"} }

// NewLogging returns the standalone logging chart subsystem.
func NewLogging() subsystem.Subsystem { return chartSubsystem{name: "rancher-logging"} }
func (s chartSubsystem) Name() string { return s.name }

func (s chartSubsystem) Collect(ctx context.Context, o subsystem.Options) ([]subsystem.ComponentResult, error) {
	if strings.TrimSpace(o.ChartBranch) == "" {
		return nil, fmt.Errorf("%s requires --chart-branch", s.name)
	}
	if strings.TrimSpace(o.Version) == "" {
		return nil, fmt.Errorf("%s requires a chart version", s.name)
	}
	source := chart.Source{Fetcher: o.Fetcher, Branch: o.ChartBranch, Name: s.name, Version: o.Version}
	main, err := loadChart(ctx, source, "", s.name, o.Version, true)
	if err != nil {
		return nil, err
	}
	source.Name += "-crd"
	crd, err := loadChart(ctx, source, "", source.Name, o.Version, s.name == "rancher-monitoring")
	if err != nil {
		return nil, err
	}
	c := collection{}
	if s.name == "rancher-monitoring" {
		err = c.monitoring(ctx, main, crd)
	} else {
		err = c.logging(main, crd)
	}
	if err != nil {
		return nil, err
	}
	return c.results, nil
}

type chartData struct {
	source   chart.Source
	prefix   string
	metadata chart.Metadata
	values   chart.Values
}

func loadChart(ctx context.Context, s chart.Source, prefix, name, version string, values bool) (*chartData, error) {
	m, err := s.Metadata(ctx, prefix, name, version)
	if err != nil {
		return nil, err
	}
	d := &chartData{source: s, prefix: prefix, metadata: m}
	if values {
		if err = s.Read(ctx, prefix+"values.yaml", &d.values); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (d *chartData) child(ctx context.Context, name string) (*chartData, error) {
	version := ""
	for _, dep := range d.metadata.Dependencies {
		if dep.Name == name {
			version = dep.Version
			break
		}
	}
	if version == "" {
		return nil, fmt.Errorf("chart %s/%s file Chart.yaml: missing dependency %s", d.source.Name, d.source.Version, name)
	}
	child, err := loadChart(ctx, d.source, "charts/"+name+"/", name, version, true)
	if err != nil {
		return nil, err
	}
	overrides, ok := d.values.Get(name).(chart.Values)
	if !ok && d.values.Get(name) != nil {
		return nil, d.fieldError(name, fmt.Errorf("expected subchart overrides mapping"))
	}
	child.values = chart.Merge(child.values, overrides)
	// Helm propagates parent globals into subcharts, with parent values winning.
	parentGlobal, _ := d.values.Get("global").(chart.Values)
	childGlobal, _ := child.values.Get("global").(chart.Values)
	child.values["global"] = chart.Merge(childGlobal, parentGlobal)
	return child, nil
}

func (d *chartData) enabled(path string) (bool, error) {
	b, err := d.values.Enabled(path)
	if err != nil {
		return false, d.fieldError(path, err)
	}
	return b, nil
}

func (d *chartData) fieldError(field string, err error) error {
	return fmt.Errorf("chart %s/%s file %svalues.yaml field %s: %w", d.source.Name, d.source.Version, d.prefix, field, err)
}

type collection struct{ results []subsystem.ComponentResult }
type imageField struct {
	path   string
	policy imagePolicy
}

func (c *collection) add(component string, d *chartData, fields ...imageField) error {
	images := make([]string, 0, len(fields))
	for _, f := range fields {
		ref, err := imageRef(d.values, f.path, d.metadata.AppVersion, f.policy)
		if err != nil {
			return fmt.Errorf("component %s: %w", component, d.fieldError(f.path, err))
		}
		logrus.WithFields(logrus.Fields{"component": component, "chart": d.source.Name, "version": d.source.Version, "file": d.prefix + "values.yaml", "field": f.path, "image": ref}).Info("discovered image")
		images = append(images, ref)
	}
	c.results = append(c.results, subsystem.ComponentResult{Component: component, Images: images})
	return nil
}

func (c *collection) logging(main, crd *chartData) error {
	operator := cattleImage
	operator.fallback = true
	for _, item := range []struct {
		name, path string
		policy     imagePolicy
	}{
		{"logging-operator", "image", operator},
		{"fluentd", "images.fluentd", cattleImage},
		{"fluentbit", "images.fluentbit", cattleImage},
		{"config-reloader", "images.config_reloader", cattleImage},
	} {
		if err := c.add(item.name, main, imageField{item.path, item.policy}); err != nil {
			return err
		}
	}
	logrus.WithField("component", crd.source.Name).Info("CRD definitions only; no runtime images")
	return c.add(crd.source.Name, crd)
}
