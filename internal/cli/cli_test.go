package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/A-Murchison/dot-plasma/internal/paths"
	"github.com/A-Murchison/dot-plasma/internal/profile"
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
			wantErr: "missing profile argument\nusage: dotplasma save <profile>",
		},
		{
			name:    "apply requires profile",
			args:    []string{"apply"},
			wantErr: "missing profile argument\nusage: dotplasma apply <profile>",
		},
		{
			name:    "import requires profile",
			args:    []string{"import", "/tmp/source"},
			wantErr: "missing profile argument\nusage: dotplasma import <source-dir> <profile>",
		},
		{
			name:    "import requires source and profile",
			args:    []string{"import"},
			wantErr: "missing source-dir and profile arguments\nusage: dotplasma import <source-dir> <profile>",
		},
		{
			name:    "apply rejects unknown reload mode",
			args:    []string{"apply", "work", "--reload", "session"},
			wantErr: `unsupported reload mode "session"`,
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
		{
			name:    "diff rejects unknown color mode",
			args:    []string{"diff", "--color", "sparkle"},
			wantErr: `unsupported color mode "sparkle"`,
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
				"Allowlist: OK (version 1, 6 files)",
				"Config root:",
				"Local data root:",
				"Output directory: OK",
				"Tracked live files:",
				"Plasma version:",
			},
		},
		{
			name: "inspect-live",
			args: []string{"inspect-live"},
			wantOutput: []string{
				"Tracked live Plasma files",
				"Root",
				"Path",
				"Status",
				"Required",
				"Parser",
				"config",
				"kdeglobals",
				"kwinrc",
				"plasma-org.kde.plasma.desktop-appletsrc",
				"plasmashellrc",
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
	if !strings.Contains(got, "Output directory: not a directory") {
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
	err = runDiff(context.Background(), &stdout, profile.DiffOptions{Profile: "work", Out: out, Roots: roots}, "text", "never", false)
	if err == nil {
		t.Fatal("runDiff returned nil error")
	}
	if ExitCode(err) != 1 {
		t.Fatalf("ExitCode = %d, want 1", ExitCode(err))
	}
	got := stdout.String()
	for _, want := range []string{
		"Profile: work",
		"Differences: 1",
		"Changed files:",
		"File               |  Key changes  |  File status",
		"config/kdeglobals  |  1            |  settings differ",
		"Use --verbose to show changed settings and values.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("runDiff output = %q, want containing %q", got, want)
		}
	}
}

func TestRunDiffVerboseReportsVerticalDetails(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	out := t.TempDir()
	roots := paths.LiveRoots{Config: configRoot, LocalShare: t.TempDir()}
	writeCLITestFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nFavoriteApps=applications:systemsettings.desktop,preferred://filemanager,preferred://browser,applications:dev.zed.Zed.desktop\n")
	_, err := profile.Save(context.Background(), profile.SaveOptions{Profile: "work", Out: out, Roots: roots})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}
	writeCLITestFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nFavoriteApps=applications:org.kde.konsole.desktop,applications:systemsettings.desktop,preferred://filemanager,preferred://browser,applications:dev.zed.Zed.desktop\n")

	var stdout bytes.Buffer
	err = runDiff(context.Background(), &stdout, profile.DiffOptions{Profile: "work", Out: out, Roots: roots}, "text", "never", true)
	if err == nil {
		t.Fatal("runDiff returned nil error")
	}
	got := stdout.String()
	for _, want := range []string{
		"Changed files:",
		"Details:",
		"File: config/kdeglobals",
		"Changed key:",
		"General/FavoriteApps",
		"Saved profile:",
		"      applications:systemsettings.desktop",
		"Live system:",
		"      applications:org.kde.konsole.desktop",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("runDiff output = %q, want containing %q", got, want)
		}
	}
	if strings.Contains(got, "Status  |  Type") {
		t.Fatalf("runDiff output = %q, want vertical details without table header", got)
	}
}

func TestRunDiffColorAlways(t *testing.T) {
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
	err = runDiff(context.Background(), &stdout, profile.DiffOptions{Profile: "work", Out: out, Roots: roots}, "text", "always", true)
	if err == nil {
		t.Fatal("runDiff returned nil error")
	}
	got := stdout.String()
	for _, want := range []string{
		"Changed key:",
		"\x1b[33mSaved profile\x1b[0m:",
		"\x1b[32mLive system\x1b[0m:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("runDiff output = %q, want containing %q", got, want)
		}
	}
	if strings.Contains(got, "\x1b[33mChanged\x1b[0m") {
		t.Fatalf("runDiff output = %q, want Changed status without color", got)
	}
}

func TestRunListProfiles(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	out := t.TempDir()
	roots := paths.LiveRoots{Config: configRoot, LocalShare: t.TempDir()}
	writeCLITestFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=BreezeDark\n")
	_, err := profile.Save(context.Background(), profile.SaveOptions{Profile: "work", Out: out, Roots: roots})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = Run(context.Background(), []string{"list", "--out", out}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %s", err.Error())
	}
	got := stdout.String()
	for _, want := range []string{"Profiles directory: " + filepath.Join(out, "profiles"), "Profile", "Saved at", "Plasma", "Distro", "work"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Run output = %q, want containing %q", got, want)
		}
	}
	if strings.Contains(got, "Files") {
		t.Fatalf("Run output = %q, want no Files column", got)
	}
	if !strings.Contains(got, "Profile  |  Plasma  |  Distro") {
		t.Fatalf("Run output = %q, want Distro column before Saved at", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "Profile  |") && !strings.HasSuffix(line, "Saved at") {
			t.Fatalf("header line = %q, want Saved at column at end", line)
		}
	}
}

func TestRunImportDryRun(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	sourceOut := t.TempDir()
	targetOut := t.TempDir()
	roots := paths.LiveRoots{Config: configRoot, LocalShare: t.TempDir()}
	writeCLITestFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=BreezeDark\n")
	_, err := profile.Save(context.Background(), profile.SaveOptions{Profile: "shared", Out: sourceOut, Roots: roots})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = Run(context.Background(), []string{"import", filepath.Join(sourceOut, "profiles", "shared"), "friend", "--out", targetOut, "--dry-run"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %s", err.Error())
	}
	got := stdout.String()
	for _, want := range []string{"Privacy: imported profiles may contain personal data", "Security: imported profiles are validated", "Dry run: no files written.", "Imported as: friend", "Would Create"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Run output = %q, want containing %q", got, want)
		}
	}
}

func TestRunUsesConfigOutputDirectory(t *testing.T) {
	t.Parallel()

	out := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.toml")
	writeCLITestFile(t, configPath, "output_dir = "+strconv.Quote(out)+"\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), []string{"save", "work", "--config", configPath, "--dry-run"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := stdout.String()
	want := "Profile directory: " + filepath.Join(out, "profiles", "work")
	if !strings.Contains(got, want) {
		t.Fatalf("Run output = %q, want containing %q", got, want)
	}
}

func TestRunOutFlagOverridesConfigOutputDirectory(t *testing.T) {
	t.Parallel()

	configOut := t.TempDir()
	flagOut := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.toml")
	writeCLITestFile(t, configPath, "output_dir = "+strconv.Quote(configOut)+"\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), []string{"list", "--config", configPath, "--out", flagOut}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := stdout.String()
	want := "Profiles directory: " + filepath.Join(flagOut, "profiles")
	if !strings.Contains(got, want) {
		t.Fatalf("Run output = %q, want containing %q", got, want)
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

func TestRunApplyDryRun(t *testing.T) {
	home := t.TempDir()
	out := t.TempDir()
	configRoot := filepath.Join(home, ".config")
	localShareRoot := filepath.Join(home, ".local", "share")
	t.Setenv("HOME", home)
	writeCLITestFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=Saved\n")
	_, err := profile.Save(context.Background(), profile.SaveOptions{
		Profile: "work",
		Out:     out,
		Roots:   paths.LiveRoots{Config: configRoot, LocalShare: localShareRoot},
	})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}
	writeCLITestFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=Live\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = Run(context.Background(), []string{"apply", "work", "--out", out, "--dry-run"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"Warning: applying a profile can overwrite tracked live Plasma configuration files.",
		"Dry run: no files written; no backups created.",
		"Profile: work",
		"Would Update",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Run output = %q, want containing %q", got, want)
		}
	}
}

func TestRunApplyUsesDefaultBackupDirectory(t *testing.T) {
	home := t.TempDir()
	configHome := t.TempDir()
	out := filepath.Join(configHome, "dotplasma")
	configRoot := filepath.Join(home, ".config")
	localShareRoot := filepath.Join(home, ".local", "share")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeCLITestFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=Saved\n")
	_, err := profile.Save(context.Background(), profile.SaveOptions{
		Profile: "work",
		Out:     out,
		Roots:   paths.LiveRoots{Config: configRoot, LocalShare: localShareRoot},
	})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}
	writeCLITestFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=Live\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = Run(context.Background(), []string{"apply", "work", "--dry-run"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := stdout.String()
	wantProfileDir := "Profile directory: " + filepath.Join(out, "profiles", "work")
	if !strings.Contains(got, wantProfileDir) {
		t.Fatalf("Run output = %q, want containing %q", got, wantProfileDir)
	}
	wantBackupPrefix := "Backup directory: " + filepath.Join(out, "backups", "work") + string(filepath.Separator)
	if !strings.Contains(got, wantBackupPrefix) {
		t.Fatalf("Run output = %q, want containing %q", got, wantBackupPrefix)
	}
}

func TestRunApplyDryRunWithPlasmashellReloadPlansReload(t *testing.T) {
	home := t.TempDir()
	out := t.TempDir()
	configRoot := filepath.Join(home, ".config")
	localShareRoot := filepath.Join(home, ".local", "share")
	t.Setenv("HOME", home)
	writeCLITestFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=Saved\n")
	_, err := profile.Save(context.Background(), profile.SaveOptions{
		Profile: "work",
		Out:     out,
		Roots:   paths.LiveRoots{Config: configRoot, LocalShare: localShareRoot},
	})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}
	writeCLITestFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=Live\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = Run(context.Background(), []string{"apply", "work", "--out", out, "--dry-run", "--reload", "plasmashell"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"Warning: --reload plasmashell stops KDE Plasma Shell before applying and starts it again afterward.",
		"Dry run: would stop plasmashell before applying and start it afterward.",
		"Would Update",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Run output = %q, want containing %q", got, want)
		}
	}
}

func TestResolveOutputDirDefaultsToUserConfigDir(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	got, err := resolveOutputDir("", "")
	if err != nil {
		t.Fatalf("resolveOutputDir() error = %v", err)
	}
	want := filepath.Join(configHome, "dotplasma")
	if got != want {
		t.Fatalf("resolveOutputDir() = %q, want %q", got, want)
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
				"Privacy: saved profiles may contain personal or machine-specific data",
				"Dry run: no files written.",
				"Profile: work",
				"Profile directory: /tmp/profiles/profiles/work",
				"kdeglobals",
			},
		},
		{
			name: "list flag plumbing",
			args: []string{"list", "--out", "/tmp/profiles"},
			wantOutput: []string{
				"Profiles directory: /tmp/profiles/profiles",
				"No profiles found.",
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
