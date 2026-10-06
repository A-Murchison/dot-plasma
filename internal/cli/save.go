package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/A-Murchison/dot-plasma/internal/profile"
)

func runSave(ctx context.Context, stdout io.Writer, opts profile.SaveOptions) error {
	fmt.Fprintln(stdout, "Privacy: saved profiles may contain personal or machine-specific data; review before sharing or committing.")
	result, err := profile.Save(ctx, opts)
	if err != nil {
		return err
	}
	if opts.DryRun {
		fmt.Fprintln(stdout, "Dry run: no files written.")
	}
	fmt.Fprintf(stdout, "Profile: %s\n", opts.Profile)
	fmt.Fprintf(stdout, "Profile directory: %s\n", result.ProfileDir)
	return writeActionTable(stdout, "Changes", "Live file", "Saved file", result.Actions)
}
