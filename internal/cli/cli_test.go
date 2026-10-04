package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dot-plasma/internal/paths"
	"dot-plasma/internal/profile"
)

func TestRunHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(context.Background(), []string{"--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "dotplasma snapshots") {
		t.Fatalf("help output missing description: %q", got)
	}
	if !strings.Contains(got, "save") {
		t.Fatalf("help output missing save command: %q", got)
	}
}

func TestRunRejectsInvalidArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "unknown command",
			args:    []string{"wat"},
			wantErr: `unknown command "wat"`,
		},
		{
			name:    "save requires profile",
			args:    []string{"save"},
			wantErr: "accepts 1 arg(s), received 0",
		},
		{
			name:    "apply requires profile",
			args:    []string{"apply"},
			wantErr: "accepts 1 arg(s), received 0",
		},
		{
			name:    "list rejects args",
			args:    []string{"list", "extra"},
			wantErr: `unknown command "extra" for "dotplasma list"`,
		},
		{
			name:    "version rejects args",
			args:    []string{"version", "extra"},
			wantErr: `unknown command "extra" for "dotplasma version"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := Run(context.Background(), tt.args, &stdout, &stderr)
			if err == nil {
				t.Fatal("Run returned nil error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Run error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestRunDiscoveryCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantOutput []string
	}{
		{
			name: "doctor",
			args: []string{"doctor", "--out", t.TempDir()},
			wantOutput: []string{
				"dotplasma doctor",
				"allowlist: ok (version 1, 5 files)",
				"config root:",
				"local-share root:",
				"output dir: ok",
				"allowlisted files:",
				"plasma version:",
			},
		},
		{
			name: "inspect-live",
			args: []string{"inspect-live"},
			wantOutput: []string{
				"ROOT",
				"PATH",
				"STATUS",
				"REQUIRED",
				"PARSER",
				"config",
				"kdeglobals",
				"kwinrc",
				"plasma-org.kde.plasma.desktop-appletsrc",
				"kconfig",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := Run(context.Background(), tt.args, &stdout, &stderr)
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			got := stdout.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(got, want) {
					t.Fatalf("Run output = %q, want containing %q", got, want)
				}
			}
		})
	}
}

func TestDoctorReportsInvalidOutputDirectory(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write test output file: %s", err.Error())
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(context.Background(), []string{"doctor", "--out", path}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run returned nil error")
	}
	if !strings.Contains(err.Error(), "doctor found problems") {
		t.Fatalf("Run error = %s, want doctor found problems", err.Error())
	}
	got := stdout.String()
	if !strings.Contains(got, "output dir: not a directory") {
		t.Fatalf("Run output = %q, want output dir problem", got)
	}
}

func TestRunDiffReportsDifferences(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	out := t.TempDir()
	roots := paths.LiveRoots{Config: configRoot, LocalShare: t.TempDir()}
	writeCLITestFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=BreezeDark\n")
	_, err := profile.Save(context.Background(), profile.SaveOptions{Profile: "work", Out: out, Roots: roots})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}
	writeCLITestFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=BreezeLight\n")

	var stdout bytes.Buffer
	err = runDiff(context.Background(), &stdout, profile.DiffOptions{Profile: "work", Out: out, Roots: roots}, "text")
	if err == nil {
		t.Fatal("runDiff returned nil error")
	}
	if ExitCode(err) != 1 {
		t.Fatalf("ExitCode = %d, want 1", ExitCode(err))
	}
	got := stdout.String()
	for _, want := range []string{"profile: work", "differences: 1", "changed key config/kdeglobals General/ColorScheme"} {
		if !strings.Contains(got, want) {
			t.Fatalf("runDiff output = %q, want containing %q", got, want)
		}
	}
}

func TestRunDiffRejectsUnsupportedFormat(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(context.Background(), []string{"diff", "work", "--format", "json"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run returned nil error")
	}
	if !strings.Contains(err.Error(), `unsupported diff format "json"`) {
		t.Fatalf("Run error = %s, want unsupported format", err.Error())
	}
}

func writeCLITestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create test dir: %s", err.Error())
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file %s: %s", path, err.Error())
	}
}

func TestRunCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantOutput []string
	}{
		{
			name: "version",
			args: []string{"version"},
			wantOutput: []string{
				"dotplasma dev",
			},
		},
		{
			name: "save dry run",
			args: []string{"save", "work", "--out", "/tmp/profiles", "--dry-run"},
			wantOutput: []string{
				"privacy: saved Plasma configuration may contain personal or machine-specific data",
				"dry run: no files written",
				"profile: work",
				"profile dir: /tmp/profiles/profiles/work",
				"kdeglobals",
			},
		},
		{
			name: "apply flag plumbing",
			args: []string{"apply", "work", "--out", "/tmp/profiles", "--dry-run"},
			wantOutput: []string{
				"apply is not implemented yet",
				"profile: work",
				"out: /tmp/profiles",
				"dry-run: true",
			},
		},
		{
			name: "list flag plumbing",
			args: []string{"list", "--out", "/tmp/profiles"},
			wantOutput: []string{
				"list is not implemented yet",
				"out: /tmp/profiles",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := Run(context.Background(), tt.args, &stdout, &stderr)
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			got := stdout.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(got, want) {
					t.Fatalf("Run output = %q, want containing %q", got, want)
				}
			}
		})
	}
}
