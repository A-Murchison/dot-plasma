package profile

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dot-plasma/internal/paths"
)

func TestSaveCreatesProfileFiles(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	localShareRoot := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=BreezeDark\n")
	writeFile(t, filepath.Join(configRoot, "kwinrc"), "[Windows]\nPlacement=Smart\n")
	writeFile(t, filepath.Join(configRoot, "plasmarc"), "[Theme]\nname=breeze\n")
	writeFile(t, filepath.Join(configRoot, "plasma-org.kde.plasma.desktop-appletsrc"), strings.Join([]string{
		"[Containments][1]",
		"lastScreen=0",
		"plugin=org.kde.plasma.folder",
		"",
	}, "\n"))
	writeFile(t, filepath.Join(configRoot, "kscreenlockerrc"), "[Daemon]\nAutolock=false\n")

	result, err := Save(context.Background(), SaveOptions{
		Profile: "work",
		Out:     out,
		Roots:   paths.LiveRoots{Config: configRoot, LocalShare: localShareRoot},
		Now:     fixedNow,
	})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}
	if result.ProfileDir != filepath.Join(out, "profiles", "work") {
		t.Fatalf("ProfileDir = %q", result.ProfileDir)
	}

	wantFiles := []string{
		"profile.toml",
		"README.md",
		filepath.Join("manifests", "files.toml"),
		filepath.Join("manifests", "ignored.toml"),
		filepath.Join("manifests", "environment.toml"),
		filepath.Join("manifests", "privacy.md"),
		filepath.Join("files", "config", "kdeglobals"),
		filepath.Join("files", "config", "kwinrc"),
		filepath.Join("files", "config", "plasmarc"),
		filepath.Join("files", "config", "plasma-org.kde.plasma.desktop-appletsrc"),
		filepath.Join("files", "config", "kscreenlockerrc"),
	}
	for _, rel := range wantFiles {
		path := filepath.Join(result.ProfileDir, rel)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected saved file %s: %s", path, err.Error())
		}
	}

	plasmaConfig := readFile(t, filepath.Join(result.ProfileDir, "files", "config", "plasma-org.kde.plasma.desktop-appletsrc"))
	if strings.Contains(plasmaConfig, "lastScreen=0") {
		t.Fatalf("volatile key was not stripped: %q", plasmaConfig)
	}
	if !strings.Contains(plasmaConfig, "plugin=org.kde.plasma.folder") {
		t.Fatalf("non-volatile key missing: %q", plasmaConfig)
	}

	manifest := readFile(t, filepath.Join(result.ProfileDir, "manifests", "files.toml"))
	if !strings.Contains(manifest, `path = "kdeglobals"`) || !strings.Contains(manifest, `sha256 = "`) {
		t.Fatalf("manifest missing expected content: %q", manifest)
	}
	privacy := readFile(t, filepath.Join(result.ProfileDir, "manifests", "privacy.md"))
	if !strings.Contains(privacy, "Treat this profile as private by default") {
		t.Fatalf("privacy review missing expected warning: %q", privacy)
	}
	profileMetadata := readFile(t, filepath.Join(result.ProfileDir, "profile.toml"))
	if strings.Contains(profileMetadata, "hostname") || strings.Contains(profileMetadata, "username") {
		t.Fatalf("profile metadata contains machine-identifying fields: %q", profileMetadata)
	}
}

func TestSaveSecondRunIsStable(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=BreezeDark\n")

	_, err := Save(context.Background(), SaveOptions{
		Profile: "work",
		Out:     out,
		Roots:   paths.LiveRoots{Config: configRoot, LocalShare: t.TempDir()},
		Now:     fixedNow,
	})
	if err != nil {
		t.Fatalf("first Save returned error: %s", err.Error())
	}

	later := func() time.Time {
		return fixedNow().Add(24 * time.Hour)
	}
	result, err := Save(context.Background(), SaveOptions{
		Profile: "work",
		Out:     out,
		Roots:   paths.LiveRoots{Config: configRoot, LocalShare: t.TempDir()},
		Now:     later,
	})
	if err != nil {
		t.Fatalf("second Save returned error: %s", err.Error())
	}
	for _, action := range result.Actions {
		if action.Status == "create" || action.Status == "update" {
			t.Fatalf("second save was not stable: %#v", result.Actions)
		}
	}

	profileMetadata := readFile(t, filepath.Join(result.ProfileDir, "profile.toml"))
	if !strings.Contains(profileMetadata, `saved_at = "2026-01-02T03:04:05Z"`) {
		t.Fatalf("saved_at was not preserved: %q", profileMetadata)
	}
}

func TestSaveDryRunDoesNotWrite(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=BreezeDark\n")

	_, err := Save(context.Background(), SaveOptions{
		Profile: "work",
		Out:     out,
		DryRun:  true,
		Roots:   paths.LiveRoots{Config: configRoot, LocalShare: t.TempDir()},
		Now:     fixedNow,
	})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}
	profileDir := filepath.Join(out, "profiles", "work")
	if _, err := os.Stat(profileDir); !os.IsNotExist(err) {
		t.Fatalf("dry run created profile dir or unexpected stat error: %v", err)
	}
}

func TestSaveRejectsUnsafeProfileName(t *testing.T) {
	t.Parallel()

	_, err := Save(context.Background(), SaveOptions{Profile: "../bad", Out: t.TempDir(), Roots: paths.LiveRoots{Config: t.TempDir(), LocalShare: t.TempDir()}, Now: fixedNow})
	if err == nil {
		t.Fatal("Save returned nil error")
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create fixture dir: %s", err.Error())
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture %s: %s", path, err.Error())
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %s", path, err.Error())
	}
	return string(data)
}
