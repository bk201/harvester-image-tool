package version

import (
	"fmt"

	"github.com/spf13/cobra"

	rootcmd "github.com/bk201/image-tool/cmd"
	pkgversion "github.com/bk201/image-tool/pkg/version"
)

var command = &cobra.Command{
	Use:   "version",
	Short: "Print the image-tool version",
	RunE: func(c *cobra.Command, _ []string) error {
		_, err := fmt.Fprintln(c.OutOrStdout(), pkgversion.FriendlyVersion())
		return err
	},
}

func init() {
	rootcmd.RootCmd.AddCommand(command)
}
