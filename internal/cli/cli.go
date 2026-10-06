package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	appconfig "github.com/A-Murchison/dot-plasma/internal/config"
	"github.com/A-Murchison/dot-plasma/internal/profile"

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
	var configPath string

	cmd := &cobra.Command{
		Use:           "dotplasma",
		Short:         "Snapshot, compare, and restore KDE Plasma configuration",
		Long:          "dotplasma snapshots, compares, and restores KDE Plasma configuration.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "config file (default: ${XDG_CONFIG_HOME:-~/.config}/dotplasma/config.toml)")

	cmd.AddCommand(newSaveCommand(stdout, &configPath))
	cmd.AddCommand(newDiffCommand(stdout, &configPath))
	cmd.AddCommand(newListCommand(stdout, &configPath))
	cmd.AddCommand(newDoctorCommand(stdout, &configPath))
	cmd.AddCommand(newInspectLiveCommand(stdout))
	cmd.AddCommand(newApplyCommand(stdout, &configPath))
	cmd.AddCommand(newImportCommand(stdout, &configPath))
	cmd.AddCommand(newVersionCommand(stdout))

	return cmd
}

func newSaveCommand(stdout io.Writer, configPath *string) *cobra.Command {
	var out string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "save <profile>",
		Short: "Save allowlisted live Plasma config into a profile",
		Args:  requireArgs("profile"),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedOut, err := resolveOutputDir(*configPath, out)
			if err != nil {
				return err
			}
			return runSave(cmd.Context(), stdout, profile.SaveOptions{Profile: args[0], Out: resolvedOut, DryRun: dryRun})
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output directory (default: ${XDG_CONFIG_HOME:-~/.config}/dotplasma; overrides config file)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be written without writing")
	return cmd
}

func newDiffCommand(stdout io.Writer, configPath *string) *cobra.Command {
	var out string
	var format string
	var color string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "diff [profile]",
		Short: "Compare a saved profile against the live system",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := ""
			if len(args) == 1 {
				profileName = args[0]
			}
			resolvedOut, err := resolveOutputDir(*configPath, out)
			if err != nil {
				return err
			}
			return runDiff(cmd.Context(), stdout, profile.DiffOptions{Profile: profileName, Out: resolvedOut}, format, color, verbose)
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output directory (default: ${XDG_CONFIG_HOME:-~/.config}/dotplasma; overrides config file)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().StringVar(&color, "color", "auto", "colorize diff output: auto, always, or never")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show per-setting diff details")
	return cmd
}

func newListCommand(stdout io.Writer, configPath *string) *cobra.Command {
	var out string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedOut, err := resolveOutputDir(*configPath, out)
			if err != nil {
				return err
			}
			return runList(cmd.Context(), stdout, profile.ListOptions{Out: resolvedOut})
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output directory (default: ${XDG_CONFIG_HOME:-~/.config}/dotplasma; overrides config file)")
	return cmd
}

func newDoctorCommand(stdout io.Writer, configPath *string) *cobra.Command {
	var out string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check local environment and project assumptions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedOut, err := resolveOutputDir(*configPath, out)
			if err != nil {
				return err
			}
			return runDoctor(cmd.Context(), stdout, resolvedOut)
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output directory (default: ${XDG_CONFIG_HOME:-~/.config}/dotplasma; overrides config file)")
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

func newImportCommand(stdout io.Writer, configPath *string) *cobra.Command {
	var out string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "import <source-dir> <profile>",
		Short: "Import a received profile without applying it",
		Args:  requireArgs("source-dir", "profile"),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedOut, err := resolveOutputDir(*configPath, out)
			if err != nil {
				return err
			}
			return runImport(cmd.Context(), stdout, profile.ImportOptions{SourceDir: args[0], Profile: args[1], Out: resolvedOut, DryRun: dryRun})
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output directory (default: ${XDG_CONFIG_HOME:-~/.config}/dotplasma; overrides config file)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be written without writing")
	return cmd
}

func newApplyCommand(stdout io.Writer, configPath *string) *cobra.Command {
	var out string
	var dryRun bool
	var reload string

	cmd := &cobra.Command{
		Use:   "apply <profile>",
		Short: "Restore a saved profile to the live system",
		Args:  requireArgs("profile"),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedOut, err := resolveOutputDir(*configPath, out)
			if err != nil {
				return err
			}
			return runApply(cmd.Context(), stdout, profile.ApplyOptions{Profile: args[0], Out: resolvedOut, DryRun: dryRun}, reload)
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output directory (default: ${XDG_CONFIG_HOME:-~/.config}/dotplasma; overrides config file)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be written without writing")
	cmd.Flags().StringVar(&reload, "reload", "none", "reload after applying: none or plasmashell")
	return cmd
}

func requireArgs(names ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == len(names) {
			return nil
		}

		usage := conciseUsage(cmd)
		if len(args) > len(names) {
			return fmt.Errorf("too many arguments\nusage: %s", usage)
		}

		missing := names[len(args):]
		if len(missing) == 1 {
			return fmt.Errorf("missing %s argument\nusage: %s", missing[0], usage)
		}
		return fmt.Errorf("missing %s arguments\nusage: %s", strings.Join(missing, " and "), usage)
	}
}

func conciseUsage(cmd *cobra.Command) string {
	use := cmd.Use
	if idx := strings.IndexByte(use, ' '); idx >= 0 {
		return cmd.CommandPath() + use[idx:]
	}
	return cmd.CommandPath()
}

func resolveOutputDir(configPath, flagOut string) (string, error) {
	if flagOut != "" {
		return flagOut, nil
	}
	cfg, _, err := appconfig.LoadOptional(configPath)
	if err != nil {
		return "", err
	}
	if cfg.OutputDir != "" {
		return cfg.OutputDir, nil
	}
	defaultOut, err := appconfig.DefaultOutputDir()
	if err != nil {
		return "", err
	}
	return defaultOut, nil
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
