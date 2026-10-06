package profile

import (
	"bytes"
	"context"
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
)

type ApplyOptions struct {
	Profile string
	Out     string
	DryRun  bool
	Roots   paths.LiveRoots
	Now     func() time.Time
}

type ApplyResult struct {
	Profile    string
	ProfileDir string
	BackupDir  string
	Warnings   []string
	Actions    []Action
}

func Apply(ctx context.Context, opts ApplyOptions) (ApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return ApplyResult{}, fmt.Errorf("apply profile: %w", err)
	}
	if err := validateProfileName(opts.Profile); err != nil {
		return ApplyResult{}, err
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
			return ApplyResult{}, fmt.Errorf("resolve live roots: %w", err)
		}
		opts.Roots = roots
	}

	list, err := allowlist.LoadDefault()
	if err != nil {
		return ApplyResult{}, fmt.Errorf("load allowlist: %w", err)
	}
	if err := allowlist.Validate(list); err != nil {
		return ApplyResult{}, fmt.Errorf("validate allowlist: %w", err)
	}

	profileDir, err := safeProfileDir(opts.Out, opts.Profile)
	if err != nil {
		return ApplyResult{}, err
	}
	if info, err := os.Stat(profileDir); err != nil {
		return ApplyResult{}, fmt.Errorf("read profile %s: %w", opts.Profile, err)
	} else if !info.IsDir() {
		return ApplyResult{}, fmt.Errorf("profile %s is not a directory", opts.Profile)
	}

	manifest, err := readImportManifest(filepath.Join(profileDir, "manifests", "files.toml"))
	if err != nil {
		return ApplyResult{}, err
	}
	if len(manifest) == 0 {
		return ApplyResult{}, fmt.Errorf("profile manifest contains no files")
	}

	allowlisted := make(map[string]allowlist.File, len(list.Files))
	for _, file := range list.Files {
		allowlisted[string(file.Root)+":"+file.Path] = file
	}

	backupDir := filepath.Join(filepath.Clean(opts.Out), "backups", opts.Profile, opts.Now().UTC().Format("20060102T150405Z"))
	result := ApplyResult{Profile: opts.Profile, ProfileDir: profileDir, BackupDir: backupDir}
	result.Warnings = append(result.Warnings, applyVersionWarnings(ctx, profileDir)...)

	for _, item := range manifest {
		if err := ctx.Err(); err != nil {
			return ApplyResult{}, fmt.Errorf("apply profile: %w", err)
		}
		file, ok := allowlisted[item.Root+":"+item.Path]
		if !ok {
			return ApplyResult{}, fmt.Errorf("profile file %s/%s is not in the current allowlist", item.Root, item.Path)
		}
		action, err := applyFile(opts, profileDir, backupDir, file, item)
		if err != nil {
			return ApplyResult{}, err
		}
		result.Actions = append(result.Actions, action)
	}
	return result, nil
}

func applyFile(opts ApplyOptions, profileDir, backupDir string, file allowlist.File, manifest importManifestFile) (Action, error) {
	savedPath, err := safeTargetPath(profileDir, file.Root, file.Path)
	if err != nil {
		return Action{}, err
	}
	livePath, err := opts.Roots.Join(file.Root, file.Path)
	if err != nil {
		return Action{}, fmt.Errorf("resolve live path %s/%s: %w", file.Root, file.Path, err)
	}
	savedData, err := os.ReadFile(savedPath)
	if err != nil {
		return Action{}, fmt.Errorf("read saved file %s: %w", savedPath, err)
	}
	if len(savedData) != manifest.Bytes {
		return Action{}, fmt.Errorf("saved file %s byte count mismatch: got %d want %d", savedPath, len(savedData), manifest.Bytes)
	}
	if got := sha256Hex(savedData); got != manifest.SHA256 {
		return Action{}, fmt.Errorf("saved file %s sha256 mismatch", savedPath)
	}
	if err := validateSafeApplyFile(file, savedData); err != nil {
		return Action{}, fmt.Errorf("validate saved file %s: %w", savedPath, err)
	}

	liveData, liveErr := os.ReadFile(livePath)
	liveMissing := os.IsNotExist(liveErr)
	if liveErr != nil && !liveMissing {
		return Action{}, fmt.Errorf("read live file %s: %w", livePath, liveErr)
	}
	status := "update"
	if liveMissing {
		status = "create"
	} else if bytes.Equal(liveData, savedData) {
		status = "unchanged"
	}
	if opts.DryRun {
		if status == "create" {
			status = "would-create"
		} else if status == "update" {
			status = "would-update"
		}
		return Action{Status: status, Source: savedPath, Target: livePath}, nil
	}
	if status == "unchanged" {
		return Action{Status: status, Source: savedPath, Target: livePath}, nil
	}
	if !liveMissing {
		backupPath, err := safeTargetPath(backupDir, file.Root, file.Path)
		if err != nil {
			return Action{}, err
		}
		if err := os.MkdirAll(filepath.Dir(backupPath), 0o700); err != nil {
			return Action{}, fmt.Errorf("create backup directory %s: %w", filepath.Dir(backupPath), err)
		}
		if err := os.WriteFile(backupPath, liveData, 0o600); err != nil {
			return Action{}, fmt.Errorf("write backup file %s: %w", backupPath, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(livePath), 0o700); err != nil {
		return Action{}, fmt.Errorf("create live directory %s: %w", filepath.Dir(livePath), err)
	}
	if err := os.WriteFile(livePath, savedData, 0o600); err != nil {
		return Action{}, fmt.Errorf("write live file %s: %w", livePath, err)
	}
	return Action{Status: status, Source: savedPath, Target: livePath}, nil
}

func validateSafeApplyFile(file allowlist.File, data []byte) error {
	if file.Root != allowlist.RootConfig || file.Path != "plasma-org.kde.plasma.desktop-appletsrc" {
		return nil
	}
	doc, err := kconfig.Parse(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("parse Plasma Shell layout: %w", err)
	}
	containments := make(map[string]bool)
	lastScreens := make(map[string]bool)
	for _, line := range doc.Lines {
		switch line.Kind {
		case kconfig.LineGroup:
			if len(line.Group.Path) == 2 && line.Group.Path[0] == "Containments" {
				containments[line.Group.Path[1]] = true
			}
		case kconfig.LineKey:
			if len(line.Key.Group) == 2 && line.Key.Group[0] == "Containments" && line.Key.Name == "lastScreen" {
				lastScreens[line.Key.Group[1]] = true
			}
		}
	}
	if len(containments) == 0 {
		return nil
	}
	missing := make([]string, 0)
	for id := range containments {
		if !lastScreens[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("Plasma Shell layout is missing lastScreen for containments %s; re-save this profile with a current dotplasma before applying", strings.Join(missing, ", "))
}

func applyVersionWarnings(ctx context.Context, profileDir string) []string {
	saved := firstProfileValue(filepath.Join(profileDir, "profile.toml"), "plasma_version")
	if saved == "" {
		saved = firstProfileValue(filepath.Join(profileDir, "manifests", "environment.toml"), "plasma_version")
	}
	if saved == "" {
		return nil
	}
	current, err := plasma.DetectVersion(ctx)
	if err != nil || current == "" || current == saved {
		return nil
	}
	return []string{fmt.Sprintf("Plasma version mismatch: profile captured %s, current system reports %s", saved, current)}
}

func firstProfileValue(path, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	prefix := key + " = "
	for _, line := range strings.Split(string(data), "\n") {
		value, ok := strings.CutPrefix(strings.TrimSpace(line), prefix)
		if !ok {
			continue
		}
		unquoted, err := strconv.Unquote(strings.TrimSpace(value))
		if err != nil {
			return ""
		}
		return unquoted
	}
	return ""
}
