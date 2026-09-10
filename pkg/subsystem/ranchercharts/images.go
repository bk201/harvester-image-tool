package ranchercharts

import (
	"fmt"
	"strings"

	"github.com/bk201/image-tool/pkg/chart"
	"github.com/bk201/image-tool/pkg/imagelist"
)

// imagePolicy records the specific upstream template's rules. Different charts
// disagree on registry precedence, appVersion prefixes, and digest syntax.
type imagePolicy struct {
	fallback      bool
	versionPrefix string
	registry      string // cattle, grafana, monitoring, operator, exporter, webhook
	digest        string // sha256, sha (already qualified), digest (already qualified)
	digestOnly    bool
}

var (
	cattleImage     = imagePolicy{registry: "cattle"}
	grafanaImage    = imagePolicy{registry: "grafana", digest: "sha256"}
	monitoringImage = imagePolicy{registry: "monitoring", digest: "sha256", digestOnly: true}
	operatorImage   = imagePolicy{registry: "operator", digest: "sha256", fallback: true}
)

func imageRef(v chart.Values, field, appVersion string, p imagePolicy) (string, error) {
	if _, ok := v.Get(field).(chart.Values); !ok {
		return "", fmt.Errorf("%s: expected image mapping", field)
	}
	fields := []string{"repository", "tag", "registry", "sha", "digest"}
	parts := make(map[string]string, len(fields))
	for _, k := range fields {
		s, err := v.String(field + "." + k)
		if err != nil {
			return "", err
		}
		parts[k] = s
	}
	repo, tag := parts["repository"], parts["tag"]
	if repo == "" {
		return "", fmt.Errorf("%s.repository: image repository is empty", field)
	}
	if tag == "" && p.fallback && appVersion != "" {
		tag = p.versionPrefix + appVersion
	}
	cattle, err := v.String("global.cattle.systemDefaultRegistry")
	if err != nil {
		return "", err
	}
	global, err := v.String("global.imageRegistry")
	if err != nil {
		return "", err
	}
	registry := cattle
	switch p.registry {
	case "webhook":
		registry = global
		if registry == "" {
			registry = parts["registry"]
		}
	case "grafana":
		if registry == "" {
			registry = parts["registry"]
		}
	case "monitoring", "exporter":
		if registry == "" {
			registry = global
		}
		if registry == "" {
			registry = parts["registry"]
		}
	case "operator":
		if registry == "" {
			registry = global
		}
		if parts["registry"] != "" {
			registry = parts["registry"]
		}
	}
	digest := ""
	switch p.digest {
	case "sha256":
		if parts["sha"] != "" {
			digest = "sha256:" + parts["sha"]
		}
	case "sha":
		digest = parts["sha"]
	case "digest":
		if parts["sha"] != "" {
			return "", fmt.Errorf("%s.sha: forbidden by chart; use digest", field)
		}
		digest = parts["digest"]
	}
	if tag == "" && (!p.digestOnly || digest == "") {
		return "", fmt.Errorf("%s.tag: image tag is empty (appVersion %q)", field, appVersion)
	}
	ref := repo
	if registry != "" {
		ref = strings.TrimSuffix(registry, "/") + "/" + ref
	}
	if tag != "" {
		ref += ":" + tag
	}
	if digest != "" {
		ref += "@" + digest
	}
	return imagelist.NormalizeRef(ref), nil
}
