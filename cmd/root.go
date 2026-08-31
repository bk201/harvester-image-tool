package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/bk201/image-tool/pkg/version"
)

var (
	Debug              bool
	HTTPTimeout        time.Duration
	MinRequestInterval time.Duration
)

var RootCmd = &cobra.Command{
	Use:     "image-tool",
	Short:   "A CLI for generating container image lists for subsystems",
	Long:    "image-tool derives container image lists for subsystems from their upstream sources",
	Version: fmt.Sprintf("%s (%s)", version.Version, version.GitCommit),
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		logrus.SetOutput(os.Stderr)
		if Debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
	},
}

func init() {
	RootCmd.PersistentFlags().BoolVar(&Debug, "debug", false, "set logging level to debug")
	RootCmd.PersistentFlags().DurationVar(&HTTPTimeout, "http-timeout", 30*time.Second, "timeout for a single HTTP request")
	RootCmd.PersistentFlags().DurationVar(&MinRequestInterval, "min-request-interval", 800*time.Millisecond, "minimum interval between requests to the same host")
}
