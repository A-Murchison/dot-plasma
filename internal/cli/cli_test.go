package cli

import (
	"bytes"
	"context"
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
