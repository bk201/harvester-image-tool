package rancher

import (
	"context"
	"fmt"

	"github.com/bk201/image-tool/pkg/dockerfile"
)

const systemAgentVersionEnvKey = "CATTLE_SYSTEM_AGENT_VERSION"

type systemAgentComponent struct{}

func (systemAgentComponent) Name() string { return "system-agent" }

func (systemAgentComponent) Images(ctx context.Context, rc *Context) ([]string, error) {
	dockerfileContent, err := rc.Dockerfile(ctx)
	if err != nil {
		return nil, err
	}

	version, ok := dockerfile.EnvValue(dockerfileContent, systemAgentVersionEnvKey)
	if !ok || version == "" {
		return nil, fmt.Errorf("system-agent: %s not found in package/Dockerfile", systemAgentVersionEnvKey)
	}
	rc.Log().WithFields(logFields("package/Dockerfile", systemAgentVersionEnvKey)).Infof("%s = %s", systemAgentVersionEnvKey, version)

	return []string{"rancher/system-agent:" + version + "-suc"}, nil
}
