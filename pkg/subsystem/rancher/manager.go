package rancher

import "context"

type managerComponent struct{}

func (managerComponent) Name() string { return "rancher-manager" }

func (managerComponent) Images(_ context.Context, rc *Context) ([]string, error) {
	rc.Log().WithField("field", "version").Infof("version = %s", rc.Version)
	return []string{
		"rancher/rancher:" + rc.Version,
		"rancher/rancher-agent:" + rc.Version,
	}, nil
}
