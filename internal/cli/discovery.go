package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/A-Murchison/dot-plasma/internal/allowlist"
	"github.com/A-Murchison/dot-plasma/internal/paths"
	"github.com/A-Murchison/dot-plasma/internal/plasma"
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
	fmt.Fprintf(stdout, "Allowlist: OK (version %d, %d files)\n", list.Version, len(list.Files))
	healthy := true
	healthy = printPathCheck(stdout, "Config root", roots.Config, false) && healthy
	healthy = printPathCheck(stdout, "Local data root", roots.LocalShare, false) && healthy
	healthy = printPathCheck(stdout, "Output directory", out, true) && healthy
	healthy = printAllowlistedFilesCheck(stdout, list, roots) && healthy
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
	fmt.Fprintln(stdout, "Tracked live Plasma files")
	fmt.Fprintln(tw, "Root\tPath\tStatus\tRequired\tParser")
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

type allowlistedFilesCheck struct {
	Total           int
	Present         int
	MissingOptional int
	MissingRequired int
	Errors          int
}

func printAllowlistedFilesCheck(stdout io.Writer, list *allowlist.Allowlist, roots paths.LiveRoots) bool {
	check, err := checkAllowlistedFiles(list, roots)
	if err != nil {
		fmt.Fprintf(stdout, "Tracked live files: error (%s)\n", err.Error())
		return false
	}

	status := "OK"
	healthy := true
	if check.MissingRequired > 0 || check.Errors > 0 {
		status = "problems found"
		healthy = false
	}
	fmt.Fprintf(
		stdout,
		"Tracked live files: %s (%d total, %d present, %d optional missing, %d required missing, %d errors)\n",
		status,
		check.Total,
		check.Present,
		check.MissingOptional,
		check.MissingRequired,
		check.Errors,
	)
	return healthy
}

func checkAllowlistedFiles(list *allowlist.Allowlist, roots paths.LiveRoots) (allowlistedFilesCheck, error) {
	var check allowlistedFilesCheck
	for _, file := range allowlist.SortedFiles(list.Files) {
		check.Total++
		livePath, err := roots.Join(file.Root, file.Path)
		if err != nil {
			return check, fmt.Errorf("resolve %s/%s: %w", file.Root, file.Path, err)
		}

		info, err := os.Stat(livePath)
		if err != nil {
			if os.IsNotExist(err) {
				if file.Required {
					check.MissingRequired++
				} else {
					check.MissingOptional++
				}
				continue
			}
			check.Errors++
			continue
		}
		if info.IsDir() {
			check.Errors++
			continue
		}
		liveFile, err := os.Open(livePath)
		if err != nil {
			check.Errors++
			continue
		}
		if err := liveFile.Close(); err != nil {
			check.Errors++
			continue
		}
		check.Present++
	}
	return check, nil
}

func printPlasmaVersion(ctx context.Context, stdout io.Writer) {
	versionCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	version, err := plasma.DetectVersion(versionCtx)
	if err == nil {
		fmt.Fprintf(stdout, "Plasma version: %s\n", version)
		return
	}
	if errors.Is(err, plasma.ErrVersionUnavailable) {
		fmt.Fprintln(stdout, "Plasma version: unknown")
		return
	}
	fmt.Fprintf(stdout, "Plasma version: unknown (%s)\n", err.Error())
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
	fmt.Fprintf(stdout, "%s: OK (%s)\n", label, path)
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
