package profile

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/A-Murchison/dot-plasma/internal/paths"
)

func TestImportProfileCopiesValidatedProfile(t *testing.T) {
	t.Parallel()

	sourceProfile := savedImportSource(t)
	out := t.TempDir()

	result, err := ImportProfile(context.Background(), ImportOptions{SourceDir: sourceProfile, Profile: "friend", Out: out})
	if err != nil {
		t.Fatalf("ImportProfile returned error: %s", err.Error())
	}
	if result.ProfileDir != filepath.Join(out, "profiles", "friend") {
		t.Fatalf("ProfileDir = %q", result.ProfileDir)
	}
	for _, rel := range []string{
		"profile.toml",
		"README.md",
		filepath.Join("manifests", "files.toml"),
		filepath.Join("manifests", "privacy.md"),
		filepath.Join("files", "config", "kdeglobals"),
	} {
		if _, err := os.Stat(filepath.Join(result.ProfileDir, rel)); err != nil {
			t.Fatalf("expected imported %s: %s", rel, err.Error())
		}
	}
}

func TestImportProfileDryRunDoesNotWrite(t *testing.T) {
	t.Parallel()

	sourceProfile := savedImportSource(t)
	out := t.TempDir()

	_, err := ImportProfile(context.Background(), ImportOptions{SourceDir: sourceProfile, Profile: "friend", Out: out, DryRun: true})
	if err != nil {
		t.Fatalf("ImportProfile returned error: %s", err.Error())
	}
	if _, err := os.Stat(filepath.Join(out, "profiles", "friend")); !os.IsNotExist(err) {
		t.Fatalf("dry run created profile or unexpected stat error: %v", err)
	}
}

func TestImportProfileRejectsExistingProfile(t *testing.T) {
	t.Parallel()

	sourceProfile := savedImportSource(t)
	out := t.TempDir()
	if err := os.MkdirAll(filepath.Join(out, "profiles", "friend"), 0o700); err != nil {
		t.Fatalf("create existing profile: %s", err.Error())
	}

	_, err := ImportProfile(context.Background(), ImportOptions{SourceDir: sourceProfile, Profile: "friend", Out: out})
	if err == nil {
		t.Fatal("ImportProfile returned nil error")
	}
}

func TestImportProfileRejectsSymlink(t *testing.T) {
	t.Parallel()

	sourceProfile := savedImportSource(t)
	if err := os.Symlink(filepath.Join(sourceProfile, "profile.toml"), filepath.Join(sourceProfile, "link.toml")); err != nil {
		t.Skipf("symlink unavailable: %s", err.Error())
	}

	_, err := ImportProfile(context.Background(), ImportOptions{SourceDir: sourceProfile, Profile: "friend", Out: t.TempDir()})
	if err == nil {
		t.Fatal("ImportProfile returned nil error")
	}
}

func TestImportProfileRejectsTamperedManifestFile(t *testing.T) {
	t.Parallel()

	sourceProfile := savedImportSource(t)
	writeFile(t, filepath.Join(sourceProfile, "files", "config", "kdeglobals"), "[General]\nColorScheme=Tampered\n")

	_, err := ImportProfile(context.Background(), ImportOptions{SourceDir: sourceProfile, Profile: "friend", Out: t.TempDir()})
	if err == nil {
		t.Fatal("ImportProfile returned nil error")
	}
}

func savedImportSource(t *testing.T) string {
	t.Helper()
	configRoot := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=BreezeDark\n")
	_, err := Save(context.Background(), SaveOptions{
		Profile: "shared",
		Out:     out,
		Roots:   paths.LiveRoots{Config: configRoot, LocalShare: t.TempDir()},
		Now:     fixedNow,
	})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}
	return filepath.Join(out, "profiles", "shared")
}
