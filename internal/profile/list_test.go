package profile

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/A-Murchison/dot-plasma/internal/paths"
)

func TestListProfiles(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	out := t.TempDir()
	roots := paths.LiveRoots{Config: configRoot, LocalShare: t.TempDir()}
	writeFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=BreezeDark\n")
	_, err := Save(context.Background(), SaveOptions{Profile: "work", Out: out, Roots: roots, Now: fixedNow})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}
	_, err = Save(context.Background(), SaveOptions{Profile: "personal", Out: out, Roots: roots, Now: fixedNow})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}

	result, err := List(context.Background(), ListOptions{Out: out})
	if err != nil {
		t.Fatalf("List returned error: %s", err.Error())
	}
	if result.ProfilesDir != filepath.Join(out, "profiles") {
		t.Fatalf("ProfilesDir = %q, want %q", result.ProfilesDir, filepath.Join(out, "profiles"))
	}
	if len(result.Profiles) != 2 {
		t.Fatalf("len(Profiles) = %d, want 2", len(result.Profiles))
	}
	if result.Profiles[0].Name != "personal" || result.Profiles[1].Name != "work" {
		t.Fatalf("Profiles names = %q, %q; want sorted personal, work", result.Profiles[0].Name, result.Profiles[1].Name)
	}
	for _, item := range result.Profiles {
		if item.SavedAt != "2026-01-02T03:04:05Z" {
			t.Fatalf("SavedAt for %s = %q, want fixed time", item.Name, item.SavedAt)
		}
		if item.TrackedFiles != 1 {
			t.Fatalf("TrackedFiles for %s = %d, want 1", item.Name, item.TrackedFiles)
		}
	}
}

func TestListProfilesMissingDirectoryReturnsEmptyList(t *testing.T) {
	t.Parallel()

	out := t.TempDir()
	result, err := List(context.Background(), ListOptions{Out: out})
	if err != nil {
		t.Fatalf("List returned error: %s", err.Error())
	}
	if result.ProfilesDir != filepath.Join(out, "profiles") {
		t.Fatalf("ProfilesDir = %q, want %q", result.ProfilesDir, filepath.Join(out, "profiles"))
	}
	if len(result.Profiles) != 0 {
		t.Fatalf("len(Profiles) = %d, want 0", len(result.Profiles))
	}
}
