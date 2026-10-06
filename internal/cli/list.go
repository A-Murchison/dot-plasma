package cli

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/A-Murchison/dot-plasma/internal/profile"
)

func runList(ctx context.Context, stdout io.Writer, opts profile.ListOptions) error {
	result, err := profile.List(ctx, opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Profiles directory: %s\n", result.ProfilesDir)
	if len(result.Profiles) == 0 {
		fmt.Fprintln(stdout, "No profiles found.")
		return nil
	}

	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "Profile\t|\tPlasma\t|\tDistro\t|\tSaved at")
	for _, item := range result.Profiles {
		fmt.Fprintf(writer, "%s\t|\t%s\t|\t%s\t|\t%s\n", item.Name, item.PlasmaVersion, item.Distro, item.SavedAt)
	}
	return writer.Flush()
}
