package rancher

import (
	"context"
	"fmt"
)

type shellComponent struct{}

func (shellComponent) Name() string { return "shell" }

func (shellComponent) Images(ctx context.Context, rc *Context) ([]string, error) {
	build, err := rc.BuildYAML(ctx)
	if err != nil {
		return nil, err
	}
	if build.DefaultShellVersion == "" {
		return nil, fmt.Errorf("shell: defaultShellVersion is empty in build.yaml")
	}
	rc.Log().WithFields(logFields("build.yaml", "defaultShellVersion")).Infof("defaultShellVersion = %s", build.DefaultShellVersion)

	// defaultShellVersion is already a full image reference (e.g.
	// "rancher/shell:v0.8.1"); do not append a tag to it.
	return []string{build.DefaultShellVersion}, nil
}
