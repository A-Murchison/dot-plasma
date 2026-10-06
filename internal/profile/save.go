package profile

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/A-Murchison/dot-plasma/internal/allowlist"
	"github.com/A-Murchison/dot-plasma/internal/kconfig"
	"github.com/A-Murchison/dot-plasma/internal/paths"
	"github.com/A-Murchison/dot-plasma/internal/plasma"
	"github.com/A-Murchison/dot-plasma/internal/platform"
)

type SaveOptions struct {
	Profile string
	Out     string
	DryRun  bool
	Roots   paths.LiveRoots
	Now     func() time.Time
}

type SaveResult struct {
	ProfileDir string
	Actions    []Action
}

type Action struct {
	Status string
	Source string
	Target string
}

func Save(ctx context.Context, opts SaveOptions) (SaveResult, error) {
	if err := validateProfileName(opts.Profile); err != nil {
		return SaveResult{}, err
	}
	if opts.Out == "" {
		opts.Out = "."
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Roots.Config == "" && opts.Roots.LocalShare == "" {
		roots, err := paths.DefaultLiveRoots()
		if err != nil {
			return SaveResult{}, fmt.Errorf("resolve live roots: %w", err)
		}
		opts.Roots = roots
	}

	list, err := allowlist.LoadDefault()
	if err != nil {
		return SaveResult{}, fmt.Errorf("load allowlist: %w", err)
	}
	if err := allowlist.Validate(list); err != nil {
		return SaveResult{}, fmt.Errorf("validate allowlist: %w", err)
	}

	profileDir, err := safeProfileDir(opts.Out, opts.Profile)
	if err != nil {
		return SaveResult{}, err
	}

	result := SaveResult{ProfileDir: profileDir}
	manifestFiles := make([]manifestFile, 0, len(list.Files))

	for _, file := range allowlist.SortedFiles(list.Files) {
		action, manifest, err := saveFile(opts, profileDir, file)
		if err != nil {
			return SaveResult{}, err
		}
		result.Actions = append(result.Actions, action)
		if manifest != nil {
			manifestFiles = append(manifestFiles, *manifest)
		}
	}

	metadataActions, err := saveMetadata(ctx, opts, profileDir, list, manifestFiles)
	if err != nil {
		return SaveResult{}, err
	}
	result.Actions = append(result.Actions, metadataActions...)
	return result, nil
}

func saveFile(opts SaveOptions, profileDir string, file allowlist.File) (Action, *manifestFile, error) {
	livePath, err := opts.Roots.Join(file.Root, file.Path)
	if err != nil {
		return Action{}, nil, fmt.Errorf("resolve live path %s/%s: %w", file.Root, file.Path, err)
	}
	targetPath, err := safeTargetPath(profileDir, file.Root, file.Path)
	if err != nil {
		return Action{}, nil, err
	}

	data, err := os.ReadFile(livePath)
	if err != nil {
		if os.IsNotExist(err) && !file.Required {
			return Action{Status: "missing", Source: livePath, Target: targetPath}, nil, nil
		}
		return Action{}, nil, fmt.Errorf("read live file %s: %w", livePath, err)
	}

	stored, err := normalizeFile(file, data)
	if err != nil {
		return Action{}, nil, fmt.Errorf("normalize %s/%s: %w", file.Root, file.Path, err)
	}

	status, err := plannedStatus(targetPath, stored)
	if err != nil {
		return Action{}, nil, err
	}
	if !opts.DryRun && status != "unchanged" {
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
			return Action{}, nil, fmt.Errorf("create target directory %s: %w", filepath.Dir(targetPath), err)
		}
		if err := os.WriteFile(targetPath, stored, 0o600); err != nil {
			return Action{}, nil, fmt.Errorf("write target file %s: %w", targetPath, err)
		}
	}

	manifest := manifestFile{
		Root:   string(file.Root),
		Path:   file.Path,
		Parser: string(file.Parser),
		SHA256: sha256Hex(stored),
		Bytes:  len(stored),
	}
	return Action{Status: status, Source: livePath, Target: targetPath}, &manifest, nil
}

func normalizeFile(file allowlist.File, data []byte) ([]byte, error) {
	switch file.Parser {
	case allowlist.ParserKConfig:
		if _, err := kconfig.Parse(bytes.NewReader(data)); err != nil {
			return nil, err
		}
		return append([]byte(nil), data...), nil
	case allowlist.ParserRawCopy:
		return append([]byte(nil), data...), nil
	default:
		return nil, fmt.Errorf("unsupported parser %q", file.Parser)
	}
}

func plannedStatus(path string, data []byte) (string, error) {
	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "create", nil
		}
		return "", fmt.Errorf("read existing target %s: %w", path, err)
	}
	if bytes.Equal(existing, data) {
		return "unchanged", nil
	}
	return "update", nil
}

type manifestFile struct {
	Root   string
	Path   string
	Parser string
	SHA256 string
	Bytes  int
}

func saveMetadata(ctx context.Context, opts SaveOptions, profileDir string, list *allowlist.Allowlist, files []manifestFile) ([]Action, error) {
	filesContent := filesManifest(files)
	savedAt := opts.Now().UTC().Format(time.RFC3339)
	filesManifestPath := filepath.Join(profileDir, "manifests", "files.toml")
	filesManifestStatus, err := plannedStatus(filesManifestPath, []byte(filesContent))
	if err != nil {
		return nil, err
	}
	if filesManifestStatus == "unchanged" {
		if existingSavedAt, ok := existingSavedAt(filepath.Join(profileDir, "profile.toml")); ok {
			savedAt = existingSavedAt
		}
	}

	profileContent := profileMetadata(ctx, opts, list, len(files), savedAt)
	readmeContent := profileReadme(opts.Profile)
	environmentContent := environmentMetadata(ctx)
	metadata := []struct {
		path string
		data []byte
	}{
		{path: filepath.Join(profileDir, "profile.toml"), data: []byte(profileContent)},
		{path: filepath.Join(profileDir, "README.md"), data: []byte(readmeContent)},
		{path: filepath.Join(profileDir, "manifests", "files.toml"), data: []byte(filesContent)},
		{path: filepath.Join(profileDir, "manifests", "ignored.toml"), data: []byte("# Reserved for future ignored files or keys. Saved files are not stripped.\n")},
		{path: filepath.Join(profileDir, "manifests", "environment.toml"), data: []byte(environmentContent)},
		{path: filepath.Join(profileDir, "manifests", "privacy.md"), data: []byte(privacyReview())},
	}

	actions := make([]Action, 0, len(metadata))
	for _, item := range metadata {
		status, err := plannedStatus(item.path, item.data)
		if err != nil {
			return nil, err
		}
		if !opts.DryRun && status != "unchanged" {
			if err := os.MkdirAll(filepath.Dir(item.path), 0o700); err != nil {
				return nil, fmt.Errorf("create metadata directory %s: %w", filepath.Dir(item.path), err)
			}
			if err := os.WriteFile(item.path, item.data, 0o600); err != nil {
				return nil, fmt.Errorf("write metadata %s: %w", item.path, err)
			}
		}
		actions = append(actions, Action{Status: status, Target: item.path})
	}
	return actions, nil
}

func profileMetadata(ctx context.Context, opts SaveOptions, list *allowlist.Allowlist, trackedFiles int, savedAt string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "profile = %q\n", opts.Profile)
	fmt.Fprintf(&b, "saved_at = %q\n", savedAt)
	fmt.Fprintf(&b, "dotplasma_version = %q\n", "dev")
	fmt.Fprintf(&b, "allowlist_version = %d\n", list.Version)
	fmt.Fprintf(&b, "tracked_files = %d\n", trackedFiles)
	if version, err := plasma.DetectVersion(ctx); err == nil {
		fmt.Fprintf(&b, "plasma_version = %q\n", version)
	}
	if distro, err := platform.DetectDistro(); err == nil {
		fmt.Fprintf(&b, "distro = %q\n", distro)
	}
	return b.String()
}

func existingSavedAt(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		value, ok := strings.CutPrefix(line, "saved_at = ")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", false
		}
		return unquoted, true
	}
	return "", false
}

func environmentMetadata(ctx context.Context) string {
	var b strings.Builder
	if version, err := plasma.DetectVersion(ctx); err == nil {
		fmt.Fprintf(&b, "plasma_version = %q\n", version)
	}
	if distro, err := platform.DetectDistro(); err == nil {
		fmt.Fprintf(&b, "distro = %q\n", distro)
	}
	return b.String()
}

func privacyReview() string {
	return strings.Join([]string{
		"# Privacy review",
		"",
		"Treat this profile as private by default.",
		"",
		"Saved Plasma configuration may contain personal or machine-specific data, including:",
		"",
		"- absolute paths under your home directory",
		"- wallpaper, launcher, widget, panel, and activity configuration",
		"- monitor or screen layout details",
		"- application-specific settings copied from allowlisted KDE config files",
		"- other identifiers or local state present in KDE configuration",
		"",
		"Review the saved files before committing this profile to a public repository.",
		"",
	}, "\n")
}

func filesManifest(files []manifestFile) string {
	files = append([]manifestFile(nil), files...)
	sort.Slice(files, func(i, j int) bool {
		left := files[i].Root + "/" + files[i].Path
		right := files[j].Root + "/" + files[j].Path
		return left < right
	})

	var b strings.Builder
	for _, file := range files {
		fmt.Fprintln(&b, "[[files]]")
		fmt.Fprintf(&b, "root = %q\n", file.Root)
		fmt.Fprintf(&b, "path = %q\n", file.Path)
		fmt.Fprintf(&b, "parser = %q\n", file.Parser)
		fmt.Fprintf(&b, "sha256 = %q\n", file.SHA256)
		fmt.Fprintf(&b, "bytes = %d\n\n", file.Bytes)
	}
	return b.String()
}

func profileReadme(profile string) string {
	return fmt.Sprintf("# dotplasma profile %s\n\nThis directory was generated by dotplasma.\n", profile)
}

func safeProfileDir(out, profile string) (string, error) {
	cleanOut := filepath.Clean(out)
	profileDir := filepath.Join(cleanOut, "profiles", profile)
	if !isWithin(cleanOut, profileDir) {
		return "", fmt.Errorf("profile path escapes output directory")
	}
	return profileDir, nil
}

func safeTargetPath(profileDir string, root allowlist.Root, relativePath string) (string, error) {
	rootDir, err := profileRootDir(profileDir, root)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("target path %q must be relative", relativePath)
	}
	clean := filepath.Clean(relativePath)
	if clean != relativePath || clean == "." || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("target path %q is unsafe", relativePath)
	}
	target := filepath.Join(rootDir, clean)
	if !isWithin(rootDir, target) {
		return "", fmt.Errorf("target path %q escapes profile root", relativePath)
	}
	return target, nil
}

func profileRootDir(profileDir string, root allowlist.Root) (string, error) {
	switch root {
	case allowlist.RootConfig:
		return filepath.Join(profileDir, "files", "config"), nil
	case allowlist.RootLocalShare:
		return filepath.Join(profileDir, "files", "local-share"), nil
	default:
		return "", fmt.Errorf("unknown root %q", root)
	}
}

func validateProfileName(profile string) error {
	if profile == "" {
		return fmt.Errorf("profile name is empty")
	}
	if filepath.IsAbs(profile) || filepath.Clean(profile) != profile || profile == "." || strings.HasPrefix(profile, "..") {
		return fmt.Errorf("profile name %q is unsafe", profile)
	}
	if strings.ContainsAny(profile, `/\`) || strings.Contains(profile, "\x00") {
		return fmt.Errorf("profile name %q is unsafe", profile)
	}
	return nil
}

func isWithin(base, candidate string) bool {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(baseAbs, candidateAbs)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
