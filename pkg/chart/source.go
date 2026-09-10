package chart

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"go.yaml.in/yaml/v4"

	"github.com/bk201/image-tool/pkg/fetch"
)

// Source reads individual extracted files from rancher/charts. Version is
// decoded (including any '+'); URL construction escapes it exactly once.
type Source struct {
	Fetcher               fetch.Fetcher
	Branch, Name, Version string
}

func (s Source) Read(ctx context.Context, file string, out any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	u, err := url.JoinPath("https://raw.githubusercontent.com/rancher/charts", s.Branch, "charts", s.Name, s.Version, file)
	if err != nil {
		return err
	}
	logrus.WithFields(logrus.Fields{"chart": s.Name, "version": s.Version, "file": file}).Info("reading chart source")
	body, err := s.Fetcher.Get(ctx, u)
	if err == nil {
		err = yaml.Unmarshal(body, out)
	}
	if err != nil {
		return fmt.Errorf("chart %s/%s file %s: %w", s.Name, s.Version, file, err)
	}
	return nil
}

// Metadata includes the dependency versions used to validate bundled charts.
type Metadata struct {
	Name         string `yaml:"name"`
	Version      string `yaml:"version"`
	AppVersion   string `yaml:"appVersion"`
	Dependencies []struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"dependencies"`
}

func (s Source) Metadata(ctx context.Context, prefix, name, version string) (Metadata, error) {
	var m Metadata
	file := prefix + "Chart.yaml"
	if err := s.Read(ctx, file, &m); err != nil {
		return m, err
	}
	if m.Name != name || m.Version != version {
		return m, fmt.Errorf("chart %s/%s file %s: expected name=%q version=%q, got name=%q version=%q", s.Name, s.Version, file, name, version, m.Name, m.Version)
	}
	return m, nil
}

// Values preserves YAML scalar spelling, particularly numeric-looking tags.
// It is intentionally a small values reader, not a Helm template interpreter.
type Values map[string]any

func (v *Values) UnmarshalYAML(n *yaml.Node) error {
	x, err := valueNode(n, 0)
	if err != nil {
		return err
	}
	m, ok := x.(Values)
	if !ok {
		return fmt.Errorf("expected values mapping")
	}
	*v = m
	return nil
}

func valueNode(n *yaml.Node, depth int) (any, error) {
	if depth > 100 {
		return nil, fmt.Errorf("values nesting exceeds 100 levels")
	}
	switch n.Kind {
	case yaml.AliasNode:
		return valueNode(n.Alias, depth+1)
	case yaml.MappingNode:
		m := Values{}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i].Value
			if _, ok := m[k]; ok {
				return nil, fmt.Errorf("duplicate values key %q", k)
			}
			x, err := valueNode(n.Content[i+1], depth+1)
			if err != nil {
				return nil, err
			}
			m[k] = x
		}
		return m, nil
	case yaml.SequenceNode:
		a := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			x, err := valueNode(c, depth+1)
			if err != nil {
				return nil, err
			}
			a = append(a, x)
		}
		return a, nil
	case yaml.ScalarNode:
		if n.Tag == "!!null" {
			return nil, nil
		}
		return n.Value, nil
	default:
		return nil, fmt.Errorf("unsupported YAML node %v", n.Kind)
	}
}

func (v Values) Get(path string) any {
	var x any = v
	for _, k := range strings.Split(path, ".") {
		m, ok := x.(Values)
		if !ok {
			return nil
		}
		x = m[k]
	}
	return x
}

func (v Values) String(path string) (string, error) {
	x := v.Get(path)
	if x == nil {
		return "", nil
	}
	s, ok := x.(string)
	if !ok {
		return "", fmt.Errorf("%s: expected scalar", path)
	}
	return s, nil
}

// Enabled requires the switch to exist so upstream schema drift fails loudly.
func (v Values) Enabled(path string) (bool, error) {
	s, err := v.String(path)
	if err != nil {
		return false, err
	}
	switch s {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%s: expected boolean, got %q", path, s)
	}
}

// Merge returns a fresh mapping; override scalars, sequences and nulls replace
// defaults, while nested mappings merge recursively.
func Merge(defaults, overrides Values) Values {
	out := Values{}
	for k, v := range defaults {
		if m, ok := v.(Values); ok {
			out[k] = Merge(m, nil)
		} else {
			out[k] = v
		}
	}
	for k, v := range overrides {
		if m, ok := v.(Values); ok {
			base, _ := out[k].(Values)
			out[k] = Merge(base, m)
		} else {
			out[k] = v
		}
	}
	return out
}
