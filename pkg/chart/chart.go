// Package chart parses the small pieces of Helm chart values.yaml files that
// image-tool cares about: repository/tag image references.
package chart

import "fmt"

// Image is a Helm chart's conventional { repository, tag } image reference.
type Image struct {
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
}

// Ref returns "<repository>:<tag>", or an error if either field is empty.
func (i Image) Ref() (string, error) {
	if i.Repository == "" {
		return "", fmt.Errorf("image repository is empty")
	}
	if i.Tag == "" {
		return "", fmt.Errorf("image tag is empty for repository %s", i.Repository)
	}
	return i.Repository + ":" + i.Tag, nil
}
