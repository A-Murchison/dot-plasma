package profile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type ListOptions struct {
	Out string
}

type ListResult struct {
	ProfilesDir string
	Profiles    []ProfileSummary
}

type ProfileSummary struct {
	Name          string
	SavedAt       string
	PlasmaVersion string
	Distro        string
	TrackedFiles  int
}

func List(ctx context.Context, opts ListOptions) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, fmt.Errorf("list profiles: %w", err)
	}
	if opts.Out == "" {
		opts.Out = "."
	}

	profilesDir := filepath.Join(filepath.Clean(opts.Out), "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return ListResult{ProfilesDir: profilesDir}, nil
		}
		return ListResult{}, fmt.Errorf("read profiles directory %s: %w", profilesDir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	result := ListResult{ProfilesDir: profilesDir, Profiles: make([]ProfileSummary, 0, len(names))}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return ListResult{}, fmt.Errorf("list profiles: %w", err)
		}
		profileDir := filepath.Join(profilesDir, name)
		summary, err := profileSummary(profileDir, name)
		if err != nil {
			return ListResult{}, err
		}
		result.Profiles = append(result.Profiles, summary)
	}
	return result, nil
}

func profileSummary(profileDir, name string) (ProfileSummary, error) {
	metadataPath := filepath.Join(profileDir, "profile.toml")
	environmentPath := filepath.Join(profileDir, "manifests", "environment.toml")
	plasmaVersion := firstProfileValue(metadataPath, "plasma_version")
	if plasmaVersion == "" {
		plasmaVersion = firstProfileValue(environmentPath, "plasma_version")
	}
	distro := firstProfileValue(metadataPath, "distro")
	if distro == "" {
		distro = firstProfileValue(environmentPath, "distro")
	}
	summary := ProfileSummary{
		Name:          name,
		SavedAt:       valueOrUnknown(firstProfileValue(metadataPath, "saved_at")),
		PlasmaVersion: valueOrUnknown(plasmaVersion),
		Distro:        valueOrUnknown(distro),
		TrackedFiles:  profileTrackedFiles(metadataPath, filepath.Join(profileDir, "manifests", "files.toml")),
	}
	return summary, nil
}

func profileTrackedFiles(metadataPath, manifestPath string) int {
	if value := firstRawProfileValue(metadataPath, "tracked_files"); value != "" {
		tracked, err := strconv.Atoi(value)
		if err == nil && tracked >= 0 {
			return tracked
		}
	}
	manifest, err := readImportManifest(manifestPath)
	if err != nil {
		return 0
	}
	return len(manifest)
}

func firstRawProfileValue(path, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	prefix := key + " = "
	for _, line := range strings.Split(string(data), "\n") {
		value, ok := strings.CutPrefix(strings.TrimSpace(line), prefix)
		if ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
