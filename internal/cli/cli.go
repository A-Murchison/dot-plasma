package cli

import (
	"context"
	"fmt"
	"io"

	"dot-plasma/internal/profile"

	"github.com/spf13/cobra"
)

const version = "dev"

// Run executes the dotplasma CLI.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cmd := NewRootCommand(stdout, stderr)
	cmd.SetContext(ctx)
	cmd.SetArgs(args)
	return cmd.Execute()
}

// NewRootCommand builds the root command for tests and main.
func NewRootCommand(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "dotplasma",
		Short:         "Snapshot, compare, and restore KDE Plasma configuration",
		Long:          "dotplasma snapshots, compares, and restores KDE Plasma configuration.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	cmd.AddCommand(newSaveCommand(stdout))
	cmd.AddCommand(newDiffCommand(stdout))
	cmd.AddCommand(newListCommand(stdout))
	cmd.AddCommand(newDoctorCommand(stdout))
	cmd.AddCommand(newInspectLiveCommand(stdout))
	cmd.AddCommand(newApplyCommand(stdout))
	cmd.AddCommand(newVersionCommand(stdout))

	return cmd
}

func newSaveCommand(stdout io.Writer) *cobra.Command {
	var out string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "save <profile>",
		Short: "Save allowlisted live Plasma config into a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSave(cmd.Context(), stdout, profile.SaveOptions{Profile: args[0], Out: out, DryRun: dryRun})
		},
	}
	cmd.Flags().StringVar(&out, "out", ".", "output directory")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be written without writing")
	return cmd
}

func newDiffCommand(stdout io.Writer) *cobra.Command {
	var out string
	var format string

	cmd := &cobra.Command{
		Use:   "diff [profile]",
		Short: "Compare a saved profile against the live system",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := ""
			if len(args) == 1 {
				profileName = args[0]
			}
			return runDiff(cmd.Context(), stdout, profile.DiffOptions{Profile: profileName, Out: out}, format)
		},
	}
	cmd.Flags().StringVar(&out, "out", ".", "output directory")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}

func newListCommand(stdout io.Writer) *cobra.Command {
	var out string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(stdout, "list is not implemented yet\nout: %s\n", out)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", ".", "output directory")
	return cmd
}

func newDoctorCommand(stdout io.Writer) *cobra.Command {
	var out string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check local environment and project assumptions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd.Context(), stdout, out)
		},
	}
	cmd.Flags().StringVar(&out, "out", ".", "output directory")
	return cmd
}

func newInspectLiveCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect-live",
		Short: "Show known live config files and whether they exist",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspectLive(stdout)
		},
	}
}

func newApplyCommand(stdout io.Writer) *cobra.Command {
	var out string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "apply <profile>",
		Short: "Restore a saved profile to the live system",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(stdout, "apply is not implemented yet\nprofile: %s\nout: %s\ndry-run: %t\n", args[0], out, dryRun)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", ".", "output directory")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be written without writing")
	return cmd
}

func newVersionCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the dotplasma version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(stdout, "dotplasma %s\n", version)
			return nil
		},
	}
}
