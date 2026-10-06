package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/A-Murchison/dot-plasma/internal/profile"
)

func runDiff(ctx context.Context, stdout io.Writer, opts profile.DiffOptions, format string, colorMode string, verbose bool) error {
	if format != "text" {
		return fmt.Errorf("unsupported diff format %q: only text is currently implemented", format)
	}
	color, err := shouldUseColor(stdout, colorMode)
	if err != nil {
		return err
	}
	result, err := profile.DiffLive(ctx, opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Profile: %s\n", result.Profile)
	if len(result.Differences) == 0 {
		fmt.Fprintln(stdout, "No differences found.")
		return nil
	}
	fmt.Fprintf(stdout, "Differences: %d\n", len(result.Differences))
	writeDiffSummary(stdout, result.Differences, color)
	if verbose {
		writeVerboseDiffDetails(stdout, result.Differences, color)
	} else {
		fmt.Fprintln(stdout, "\nUse --verbose to show changed settings and values.")
	}
	return ExitError{Code: 1, Err: profile.ErrDifferencesFound}
}

func writeDiffSummary(stdout io.Writer, differences []profile.Diff, color bool) {
	fmt.Fprintln(stdout, "\nChanged files:")
	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "File\t|\tKey changes\t|\tFile status")
	for _, group := range groupDiffsByFile(differences) {
		fileStatus := "settings differ"
		if group.fileStatus != "" {
			fileStatus = describeFileStatus(group.fileStatus, color)
		}
		fmt.Fprintf(writer, "%s\t|\t%d\t|\t%s\n", group.file, group.keyChanges, fileStatus)
	}
	_ = writer.Flush()
}

func writeVerboseDiffDetails(stdout io.Writer, differences []profile.Diff, color bool) {
	fmt.Fprintln(stdout, "\nDetails:")
	for _, group := range groupDiffsByFile(differences) {
		fmt.Fprintf(stdout, "\nFile: %s\n", group.file)
		for _, diff := range group.differences {
			fmt.Fprintf(stdout, "\n  %s %s:\n", describeStatus(diff.Status, color), strings.ToLower(describeKind(diff.Kind)))
			fmt.Fprintf(stdout, "    %s\n", describeSetting(diff))
			writeVerboseValue(stdout, "Saved profile", ansiYellow, diff.OldValue, describeSavedValue(diff), diff.Kind == "file" || diff.Status == "added", color)
			writeVerboseValue(stdout, "Live system", ansiGreen, diff.NewValue, describeLiveValue(diff), diff.Kind == "file" || diff.Status == "removed", color)
		}
	}
}

func writeVerboseValue(stdout io.Writer, label, labelColor, value, fallback string, useFallback bool, color bool) {
	fmt.Fprintf(stdout, "\n    %s:\n", colorize(label, labelColor, color))
	if useFallback {
		fmt.Fprintf(stdout, "      %s\n", fallback)
		return
	}
	for _, line := range formatVerboseValue(value) {
		fmt.Fprintf(stdout, "      %s\n", line)
	}
}

func formatVerboseValue(value string) []string {
	if strings.Contains(value, ",") && len(value) > 80 {
		parts := strings.Split(value, ",")
		lines := make([]string, 0, len(parts))
		for _, part := range parts {
			lines = append(lines, strings.TrimSpace(part))
		}
		return lines
	}
	return []string{fmt.Sprintf("%q", value)}
}

type diffFileGroup struct {
	file        string
	keyChanges  int
	fileStatus  string
	differences []profile.Diff
}

func groupDiffsByFile(differences []profile.Diff) []diffFileGroup {
	groups := make([]diffFileGroup, 0)
	for _, diff := range differences {
		file := diff.Root + "/" + diff.Path
		groupIndex := -1
		for i := range groups {
			if groups[i].file == file {
				groupIndex = i
				break
			}
		}
		if groupIndex == -1 {
			groups = append(groups, diffFileGroup{file: file})
			groupIndex = len(groups) - 1
		}
		groups[groupIndex].differences = append(groups[groupIndex].differences, diff)
		if diff.Kind == "file" {
			groups[groupIndex].fileStatus = diff.Status
			continue
		}
		groups[groupIndex].keyChanges++
	}
	return groups
}

func describeStatus(status string, color bool) string {
	switch status {
	case "added":
		return colorize("Added", ansiGreen, color)
	case "changed":
		return "Changed"
	case "removed":
		return colorize("Removed", ansiRed, color)
	default:
		return status
	}
}

func describeFileStatus(status string, color bool) string {
	switch status {
	case "added":
		return colorize("added in live system", ansiGreen, color)
	case "changed":
		return colorize("contents differ", ansiYellow, color)
	case "removed":
		return colorize("missing from live system", ansiRed, color)
	default:
		return status
	}
}

func describeKind(kind string) string {
	if kind == "file" {
		return "Whole file"
	}
	return "Key"
}

func describeSetting(diff profile.Diff) string {
	if diff.Kind == "file" {
		return "whole file"
	}
	return formatKeyPath(diff.Group, diff.Key, diff.Locale)
}

func describeSavedValue(diff profile.Diff) string {
	if diff.Kind == "file" {
		switch diff.Status {
		case "added":
			return "missing"
		case "changed", "removed":
			return "present"
		default:
			return "-"
		}
	}
	if diff.Status == "added" {
		return "missing"
	}
	return fmt.Sprintf("%q", diff.OldValue)
}

func describeLiveValue(diff profile.Diff) string {
	if diff.Kind == "file" {
		switch diff.Status {
		case "added", "changed":
			return "present"
		case "removed":
			return "missing"
		default:
			return "-"
		}
	}
	if diff.Status == "removed" {
		return "missing"
	}
	return fmt.Sprintf("%q", diff.NewValue)
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
