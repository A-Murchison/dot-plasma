package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"dot-plasma/internal/allowlist"
	"dot-plasma/internal/paths"
	"dot-plasma/internal/plasma"
)

func runDoctor(ctx context.Context, stdout io.Writer, out string) error {
	list, err := allowlist.LoadDefault()
	if err != nil {
		return fmt.Errorf("load allowlist: %w", err)
	}
	if err := allowlist.Validate(list); err != nil {
		return fmt.Errorf("validate allowlist: %w", err)
	}

	roots, err := paths.DefaultLiveRoots()
	if err != nil {
		return fmt.Errorf("resolve live roots: %w", err)
	}

	fmt.Fprintln(stdout, "dotplasma doctor")
	fmt.Fprintf(stdout, "allowlist: ok (version %d, %d files)\n", list.Version, len(list.Files))
	healthy := true
	healthy = printPathCheck(stdout, "config root", roots.Config, false) && healthy
	healthy = printPathCheck(stdout, "local-share root", roots.LocalShare, false) && healthy
	healthy = printPathCheck(stdout, "output dir", out, true) && healthy
	printPlasmaVersion(ctx, stdout)
	if !healthy {
		return fmt.Errorf("doctor found problems")
	}
	return nil
}

func runInspectLive(stdout io.Writer) error {
	list, err := allowlist.LoadDefault()
	if err != nil {
		return fmt.Errorf("load allowlist: %w", err)
	}
	if err := allowlist.Validate(list); err != nil {
		return fmt.Errorf("validate allowlist: %w", err)
	}
	roots, err := paths.DefaultLiveRoots()
	if err != nil {
		return fmt.Errorf("resolve live roots: %w", err)
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ROOT\tPATH\tSTATUS\tREQUIRED\tPARSER")
	for _, file := range allowlist.SortedFiles(list.Files) {
		livePath, err := roots.Join(file.Root, file.Path)
		if err != nil {
			return fmt.Errorf("resolve %s/%s: %w", file.Root, file.Path, err)
		}
		status := fileStatus(livePath)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%t\t%s\n", file.Root, file.Path, status, file.Required, file.Parser)
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flush inspect-live output: %w", err)
	}
	return nil
}

func printPlasmaVersion(ctx context.Context, stdout io.Writer) {
	versionCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	version, err := plasma.DetectVersion(versionCtx)
	if err == nil {
		fmt.Fprintf(stdout, "plasma version: %s\n", version)
		return
	}
	if errors.Is(err, plasma.ErrVersionUnavailable) {
		fmt.Fprintln(stdout, "plasma version: unknown")
		return
	}
	fmt.Fprintf(stdout, "plasma version: unknown (%s)\n", err.Error())
}

func printPathCheck(stdout io.Writer, label, path string, requireWritable bool) bool {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stdout, "%s: missing (%s)\n", label, path)
			return false
		}
		fmt.Fprintf(stdout, "%s: error (%s): %s\n", label, path, err.Error())
		return false
	}
	if !info.IsDir() {
		fmt.Fprintf(stdout, "%s: not a directory (%s)\n", label, path)
		return false
	}
	if requireWritable {
		file, err := os.CreateTemp(path, ".dotplasma-doctor-*")
		if err != nil {
			fmt.Fprintf(stdout, "%s: not writable (%s): %s\n", label, path, err.Error())
			return false
		}
		name := file.Name()
		if err := file.Close(); err != nil {
			fmt.Fprintf(stdout, "%s: close temp file failed (%s): %s\n", label, path, err.Error())
			return false
		}
		if err := os.Remove(name); err != nil {
			fmt.Fprintf(stdout, "%s: remove temp file failed (%s): %s\n", label, path, err.Error())
			return false
		}
	}
	fmt.Fprintf(stdout, "%s: ok (%s)\n", label, path)
	return true
}

func fileStatus(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "missing"
		}
		return "error"
	}
	if info.IsDir() {
		return "directory"
	}
	return "present"
}
