package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"dot-plasma/internal/profile"
)

func runDiff(ctx context.Context, stdout io.Writer, opts profile.DiffOptions, format string) error {
	if format != "text" {
		return fmt.Errorf("unsupported diff format %q: only text is currently implemented", format)
	}
	result, err := profile.DiffLive(ctx, opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "profile: %s\n", result.Profile)
	fmt.Fprintf(stdout, "profile dir: %s\n", result.ProfileDir)
	if len(result.Differences) == 0 {
		fmt.Fprintln(stdout, "no differences")
		return nil
	}
	fmt.Fprintf(stdout, "differences: %d\n", len(result.Differences))
	for _, diff := range result.Differences {
		if diff.Kind == "file" {
			fmt.Fprintf(stdout, "%s file %s/%s\n", diff.Status, diff.Root, diff.Path)
			continue
		}
		fmt.Fprintf(stdout, "%s key %s/%s %s", diff.Status, diff.Root, diff.Path, formatKeyPath(diff.Group, diff.Key, diff.Locale))
		switch diff.Status {
		case "changed":
			fmt.Fprintf(stdout, " %q -> %q", diff.OldValue, diff.NewValue)
		case "added":
			fmt.Fprintf(stdout, " = %q", diff.NewValue)
		case "removed":
			fmt.Fprintf(stdout, " was %q", diff.OldValue)
		}
		fmt.Fprintln(stdout)
	}
	return ExitError{Code: 1, Err: profile.ErrDifferencesFound}
}

func formatKeyPath(group []string, key, locale string) string {
	parts := append([]string(nil), group...)
	name := key
	if locale != "" {
		name += "[" + locale + "]"
	}
	parts = append(parts, name)
	return strings.Join(parts, "/")
}
