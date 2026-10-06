package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/A-Murchison/dot-plasma/internal/profile"
)

func runImport(ctx context.Context, stdout io.Writer, opts profile.ImportOptions) error {
	fmt.Fprintln(stdout, "Privacy: imported profiles may contain personal data; review before sharing or applying.")
	fmt.Fprintln(stdout, "Security: imported profiles are validated before they are copied.")
	result, err := profile.ImportProfile(ctx, opts)
	if err != nil {
		return err
	}
	if opts.DryRun {
		fmt.Fprintln(stdout, "Dry run: no files written.")
	}
	fmt.Fprintf(stdout, "Source profile: %s\n", result.SourceDir)
	fmt.Fprintf(stdout, "Imported as: %s\n", opts.Profile)
	fmt.Fprintf(stdout, "Profile directory: %s\n", result.ProfileDir)
	return writeActionTable(stdout, "Changes", "Source", "Target", result.Actions)
}
