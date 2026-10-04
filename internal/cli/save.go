package cli

import (
	"context"
	"fmt"
	"io"

	"dot-plasma/internal/profile"
)

func runSave(ctx context.Context, stdout io.Writer, opts profile.SaveOptions) error {
	fmt.Fprintln(stdout, "privacy: saved Plasma configuration may contain personal or machine-specific data; review before committing publicly")
	result, err := profile.Save(ctx, opts)
	if err != nil {
		return err
	}
	if opts.DryRun {
		fmt.Fprintln(stdout, "dry run: no files written")
	}
	fmt.Fprintf(stdout, "profile: %s\n", opts.Profile)
	fmt.Fprintf(stdout, "profile dir: %s\n", result.ProfileDir)
	for _, action := range result.Actions {
		if action.Source == "" {
			fmt.Fprintf(stdout, "%s %s\n", action.Status, action.Target)
			continue
		}
		fmt.Fprintf(stdout, "%s %s -> %s\n", action.Status, action.Source, action.Target)
	}
	return nil
}
