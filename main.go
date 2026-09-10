package main

import (
	"github.com/spf13/cobra"

	"github.com/bk201/image-tool/cmd"
	"github.com/bk201/image-tool/cmd/createlist"
	_ "github.com/bk201/image-tool/cmd/version"
	"github.com/bk201/image-tool/pkg/subsystem"
	"github.com/bk201/image-tool/pkg/subsystem/rancher"
	"github.com/bk201/image-tool/pkg/subsystem/ranchercharts"
)

// registerSubsystems explicitly wires up every known subsystem. Add a new
// subsystem here — there is no implicit init()-based self-registration.
func registerSubsystems() {
	subsystem.Register(rancher.New())
	subsystem.Register(ranchercharts.NewMonitoring())
	subsystem.Register(ranchercharts.NewLogging())
}

func main() {
	registerSubsystems()

	// All subsystems are registered by now, so it's safe to wire up their flags.
	createlist.RegisterSubsystemFlags()

	cobra.CheckErr(cmd.RootCmd.Execute())
}
