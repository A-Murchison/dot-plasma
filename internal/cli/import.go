package cli

import (
	"context"
	"fmt"
	"io"

	"dot-plasma/internal/profile"
)

func runImport(ctx context.Context, stdout io.Writer, opts profile.ImportOptions) error {
	fmt.Fprintln(stdout, "privacy: imported profiles may contain personal data; review before sharing or applying")
	fmt.Fprintln(stdout, "security: imported profiles are treated as untrusted input and are only copied after validation")
	result, err := profile.ImportProfile(ctx, opts)
	if err != nil {
		return err
	}
	if opts.DryRun {
		fmt.Fprintln(stdout, "dry run: no files written")
	}
	fmt.Fprintf(stdout, "source dir: %s\n", result.SourceDir)
	fmt.Fprintf(stdout, "profile: %s\n", opts.Profile)
	fmt.Fprintf(stdout, "profile dir: %s\n", result.ProfileDir)
	for _, action := range result.Actions {
		fmt.Fprintf(stdout, "%s %s -> %s\n", action.Status, action.Source, action.Target)
	}
	return nil
}
