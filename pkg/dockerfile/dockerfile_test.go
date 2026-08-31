package dockerfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvValue(t *testing.T) {
	content := []byte(`
FROM registry.suse.com/bci/bci-base:latest
ARG CHART_DEFAULT_BRANCH
ENV CATTLE_SYSTEM_UPGRADE_CONTROLLER_CHART_VERSION=110.0.0
ENV SOME_KEY "quoted value"
ENV OTHER_KEY 'single quoted'
ENV SPACE_FORM plain value here
ENV DUPLICATE first
ENV DUPLICATE second
`)

	tests := []struct {
		key       string
		wantValue string
		wantFound bool
	}{
		{"CATTLE_SYSTEM_UPGRADE_CONTROLLER_CHART_VERSION", "110.0.0", true},
		{"SOME_KEY", "quoted value", true},
		{"OTHER_KEY", "single quoted", true},
		{"SPACE_FORM", "plain value here", true},
		{"DUPLICATE", "second", true},
		{"MISSING", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, found := EnvValue(content, tt.key)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantValue, got)
		})
	}
}
