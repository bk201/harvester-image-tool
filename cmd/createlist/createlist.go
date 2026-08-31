// Package createlist implements the "create-list" subcommand, which
// aggregates a subsystem's per-component image discovery into a single,
// deduped, sorted image list.
package createlist

import (
	"context"
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	rootcmd "github.com/bk201/image-tool/cmd"
	"github.com/bk201/image-tool/pkg/fetch"
	"github.com/bk201/image-tool/pkg/imagelist"
	"github.com/bk201/image-tool/pkg/subsystem"
)

var (
	outputPath string
	noVerify   bool
	strict     bool
	noHeader   bool
)

var command = &cobra.Command{
	Use:   "create-list <subsystem> <version>",
	Short: "Create an image list for a subsystem",
	Long: "Create-list discovers the container images that make up one version of a subsystem " +
		"and writes them, deduped and sorted, one per line.",
	Args: cobra.ExactArgs(2),
	RunE: run,
}

func subsystemNamesForHelp() string {
	names := subsystem.Names()
	if len(names) == 0 {
		return "(none registered)"
	}
	out := names[0]
	for _, n := range names[1:] {
		out += ", " + n
	}
	return out
}

func init() {
	command.Flags().StringVarP(&outputPath, "output", "o", "", "write the image list to this file (default: stdout)")
	command.Flags().BoolVar(&noVerify, "no-verify", false, "skip cross-checking images against the subsystem's official release list")
	command.Flags().BoolVar(&strict, "strict", false, "exit non-zero if any image is missing from the official release list")
	command.Flags().BoolVar(&noHeader, "no-header", false, "omit the leading '# subsystem: ...' / '# version: ...' header comments")

	rootcmd.RootCmd.AddCommand(command)
}

// RegisterSubsystemFlags wires up each registered subsystem's own flags and
// appends the list of known subsystems to the command's help text.
//
// Subsystems self-register via their own package init() functions, typically
// via a blank import in main. Go does not guarantee those run before this
// package's init(), so this must be called explicitly from main(), after all
// blank imports have run and before cmd.RootCmd.Execute() — by which point
// every package's init() is guaranteed to have completed.
func RegisterSubsystemFlags() {
	command.Long += "\n\nKnown subsystems: " + subsystemNamesForHelp()

	for _, name := range subsystem.Names() {
		s, _ := subsystem.Get(name)
		if r, ok := s.(subsystem.FlagRegistrar); ok {
			r.RegisterFlags(command.Flags())
		}
	}
}

func run(cmd *cobra.Command, args []string) error {
	name, version := args[0], args[1]

	s, ok := subsystem.Get(name)
	if !ok {
		return fmt.Errorf("unknown subsystem %q; known subsystems: %s", name, subsystemNamesForHelp())
	}

	fetcher := fetch.New(fetch.Options{
		MinInterval: rootcmd.MinRequestInterval,
		Timeout:     rootcmd.HTTPTimeout,
	})
	opts := subsystem.Options{Fetcher: fetcher, Version: version}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	results, err := s.Collect(ctx, opts)
	if err != nil {
		return fmt.Errorf("collect images for %s %s: %w", name, version, err)
	}

	var all []string
	for _, r := range results {
		all = append(all, r.Images...)
	}
	images := imagelist.Normalize(all)

	if !noVerify {
		if verifier, ok := s.(subsystem.Verifier); ok {
			missing, verr := verifier.Verify(ctx, opts, images)
			if verr != nil {
				logrus.Warnf("could not verify images against the official release list: %v", verr)
			} else if len(missing) > 0 {
				sort.Strings(missing)
				logrus.Warnf("%d image(s) not found in the official release list:", len(missing))
				for _, m := range missing {
					logrus.Warnf("  %s", m)
				}
				if strict {
					return fmt.Errorf("%d image(s) missing from the official release list", len(missing))
				}
			}
		}
	}

	var header []string
	if !noHeader {
		header = []string{"subsystem: " + name, "version: " + version}
	}

	if outputPath == "" {
		return imagelist.Write(cmd.OutOrStdout(), images, header)
	}
	return imagelist.WriteFile(outputPath, images, header)
}
