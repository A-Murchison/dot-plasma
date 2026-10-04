package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			name: "save flag plumbing",
			args: []string{"save", "work", "--out", "/tmp/profiles", "--dry-run"},
			wantOutput: []string{
				"save is not implemented yet",
				"profile: work",
				"out: /tmp/profiles",
				"dry-run: true",
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
			name: "diff default profile and format flag",
			args: []string{"diff", "--out", "/tmp/profiles", "--format", "json"},
			wantOutput: []string{
				"diff is not implemented yet",
				"profile: <auto>",
				"out: /tmp/profiles",
				"format: json",
			},
		},
		{
			name: "diff explicit profile",
			args: []string{"diff", "work"},
			wantOutput: []string{
				"profile: work",
				"format: text",
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
