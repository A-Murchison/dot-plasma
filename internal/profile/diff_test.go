package profile

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/A-Murchison/dot-plasma/internal/paths"
)

func TestDiffLiveReportsKConfigKeyChanges(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=BreezeDark\n")

	roots := paths.LiveRoots{Config: configRoot, LocalShare: t.TempDir()}
	_, err := Save(context.Background(), SaveOptions{Profile: "work", Out: out, Roots: roots, Now: fixedNow})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}
	writeFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=BreezeLight\n")

	result, err := DiffLive(context.Background(), DiffOptions{Profile: "work", Out: out, Roots: roots})
	if err != nil {
		t.Fatalf("DiffLive returned error: %s", err.Error())
	}
	if len(result.Differences) != 1 {
		t.Fatalf("len(Differences) = %d, want 1: %#v", len(result.Differences), result.Differences)
	}
	diff := result.Differences[0]
	if diff.Kind != "key" || diff.Status != "changed" || diff.Path != "kdeglobals" || diff.Key != "ColorScheme" {
		t.Fatalf("unexpected diff: %#v", diff)
	}
	if diff.OldValue != "BreezeDark" || diff.NewValue != "BreezeLight" {
		t.Fatalf("unexpected values: %#v", diff)
	}
}

func TestDiffLiveReportsPlasmaShellPanelViewChanges(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(configRoot, "plasmashellrc"), "[PlasmaViews][Panel 1]\nfloating=0\n")

	roots := paths.LiveRoots{Config: configRoot, LocalShare: t.TempDir()}
	_, err := Save(context.Background(), SaveOptions{Profile: "work", Out: out, Roots: roots, Now: fixedNow})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}
	writeFile(t, filepath.Join(configRoot, "plasmashellrc"), "[PlasmaViews][Panel 1]\nfloating=1\n")

	result, err := DiffLive(context.Background(), DiffOptions{Profile: "work", Out: out, Roots: roots})
	if err != nil {
		t.Fatalf("DiffLive returned error: %s", err.Error())
	}
	if len(result.Differences) != 1 {
		t.Fatalf("len(Differences) = %d, want 1: %#v", len(result.Differences), result.Differences)
	}
	diff := result.Differences[0]
	if diff.Kind != "key" || diff.Status != "changed" || diff.Path != "plasmashellrc" || diff.Key != "floating" {
		t.Fatalf("unexpected diff: %#v", diff)
	}
	if diff.OldValue != "0" || diff.NewValue != "1" {
		t.Fatalf("unexpected values: %#v", diff)
	}
}

func TestDiffLiveAutoSelectsSingleProfile(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	out := t.TempDir()
	writeFile(t, filepath.Join(configRoot, "kdeglobals"), "[General]\nColorScheme=BreezeDark\n")
	roots := paths.LiveRoots{Config: configRoot, LocalShare: t.TempDir()}
	_, err := Save(context.Background(), SaveOptions{Profile: "only", Out: out, Roots: roots, Now: fixedNow})
	if err != nil {
		t.Fatalf("Save returned error: %s", err.Error())
	}

	result, err := DiffLive(context.Background(), DiffOptions{Out: out, Roots: roots})
	if err != nil {
		t.Fatalf("DiffLive returned error: %s", err.Error())
	}
	if result.Profile != "only" {
		t.Fatalf("Profile = %q, want only", result.Profile)
	}
	if len(result.Differences) != 0 {
		t.Fatalf("unexpected differences: %#v", result.Differences)
	}
}

func TestDiffLiveRejectsMultipleProfilesWithoutExplicitName(t *testing.T) {
	t.Parallel()

	out := t.TempDir()
	writeFile(t, filepath.Join(out, "profiles", "a", "profile.toml"), "profile = \"a\"\n")
	writeFile(t, filepath.Join(out, "profiles", "b", "profile.toml"), "profile = \"b\"\n")

	_, err := DiffLive(context.Background(), DiffOptions{Out: out, Roots: paths.LiveRoots{Config: t.TempDir(), LocalShare: t.TempDir()}})
	if err == nil {
		t.Fatal("DiffLive returned nil error")
	}
}
