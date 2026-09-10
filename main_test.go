package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bk201/image-tool/cmd"
	"github.com/bk201/image-tool/cmd/createlist"
	"github.com/bk201/image-tool/pkg/subsystem"
)

func TestSubsystemRegistration(t *testing.T) {
	registerSubsystems()
	createlist.RegisterSubsystemFlags()
	assert.Equal(t, []string{"rancher", "rancher-logging", "rancher-monitoring"}, subsystem.Names())
	c, _, err := cmd.RootCmd.Find([]string{"create-list"})
	require.NoError(t, err)
	require.NotNil(t, c.Flags().Lookup("chart-branch"))
	require.NotNil(t, c.Flags().Lookup("rancher-charts-branch"))
	for _, name := range subsystem.Names() {
		assert.Contains(t, c.Long, name)
	}
}
