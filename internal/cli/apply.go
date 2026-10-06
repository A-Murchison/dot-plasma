package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/A-Murchison/dot-plasma/internal/profile"
)

func runApply(ctx context.Context, stdout io.Writer, opts profile.ApplyOptions, reloadValue string) error {
	reload, err := parseApplyReloadMode(reloadValue)
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout, "Warning: applying a profile can overwrite tracked live Plasma configuration files.")
	if reload == applyReloadPlasmashell {
		fmt.Fprintln(stdout, "Warning: --reload plasmashell stops KDE Plasma Shell before applying and starts it again afterward.")
		fmt.Fprintln(stdout, "Warning: panels, desktop widgets, wallpaper, and the command bar may briefly disappear during reload.")
		if opts.DryRun {
			fmt.Fprintln(stdout, "Dry run: would stop plasmashell before applying and start it afterward.")
		} else {
			preflight := opts
			preflight.DryRun = true
			if _, err := profile.Apply(ctx, preflight); err != nil {
				return fmt.Errorf("preflight apply before stopping plasmashell: %w", err)
			}
			fmt.Fprintln(stdout, "Reload: stopping plasmashell.")
			if err := stopPlasmashellForApply(ctx); err != nil {
				return err
			}
		}
	}

	result, applyErr := profile.Apply(ctx, opts)
	restartErr := error(nil)
	if reload == applyReloadPlasmashell && !opts.DryRun {
		fmt.Fprintln(stdout, "Reload: starting plasmashell.")
		restartErr = startPlasmashellAfterApply(ctx)
	}
	if applyErr != nil || restartErr != nil {
		if applyErr != nil && restartErr != nil {
			return errors.Join(applyErr, fmt.Errorf("restart plasmashell after apply: %w", restartErr))
		}
		if applyErr != nil {
			return applyErr
		}
		return fmt.Errorf("restart plasmashell after apply: %w", restartErr)
	}

	if opts.DryRun {
		fmt.Fprintln(stdout, "Dry run: no files written; no backups created.")
	}
	fmt.Fprintf(stdout, "Profile: %s\n", result.Profile)
	fmt.Fprintf(stdout, "Profile directory: %s\n", result.ProfileDir)
	fmt.Fprintf(stdout, "Backup directory: %s\n", result.BackupDir)
	for _, warning := range result.Warnings {
		fmt.Fprintf(stdout, "Warning: %s\n", warning)
	}
	return writeActionTable(stdout, "Changes", "Saved file", "Live file", result.Actions)
}
