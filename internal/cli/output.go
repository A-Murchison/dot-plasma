package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/A-Murchison/dot-plasma/internal/profile"
)

const (
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiReset  = "\x1b[0m"
)

func writeActionTable(stdout io.Writer, title, sourceHeader, targetHeader string, actions []profile.Action) error {
	if len(actions) == 0 {
		fmt.Fprintf(stdout, "%s: none\n", title)
		return nil
	}

	fmt.Fprintf(stdout, "%s:\n", title)
	writer := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "Status\t|\t%s\t|\t%s\n", sourceHeader, targetHeader)
	for _, action := range actions {
		source := action.Source
		if source == "" {
			source = "-"
		}
		fmt.Fprintf(writer, "%s\t|\t%s\t|\t%s\n", describeActionStatus(action.Status), source, action.Target)
	}
	return writer.Flush()
}

func shouldUseColor(stdout io.Writer, mode string) (bool, error) {
	switch mode {
	case "always":
		return true, nil
	case "never":
		return false, nil
	case "auto", "":
		if _, ok := os.LookupEnv("NO_COLOR"); ok {
			return false, nil
		}
		file, ok := stdout.(*os.File)
		if !ok {
			return false, nil
		}
		info, err := file.Stat()
		if err != nil {
			return false, fmt.Errorf("inspect output stream: %w", err)
		}
		return info.Mode()&os.ModeCharDevice != 0, nil
	default:
		return false, fmt.Errorf("unsupported color mode %q: use auto, always, or never", mode)
	}
}

func colorize(text, color string, enabled bool) string {
	if !enabled {
		return text
	}
	return color + text + ansiReset
}

func describeActionStatus(status string) string {
	words := strings.Split(status, "-")
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}
