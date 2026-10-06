package profile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/A-Murchison/dot-plasma/internal/allowlist"
	"github.com/A-Murchison/dot-plasma/internal/kconfig"
	"github.com/A-Murchison/dot-plasma/internal/paths"
)

var ErrDifferencesFound = errors.New("differences found")

type DiffOptions struct {
	Profile string
	Out     string
	Roots   paths.LiveRoots
}

type DiffResult struct {
	Profile     string
	ProfileDir  string
	Differences []Diff
}

type Diff struct {
	Root     string
	Path     string
	Status   string
	Kind     string
	Group    []string
	Key      string
	Locale   string
	OldValue string
	NewValue string
}

func DiffLive(ctx context.Context, opts DiffOptions) (DiffResult, error) {
	if err := ctx.Err(); err != nil {
		return DiffResult{}, fmt.Errorf("diff profile: %w", err)
	}
	if opts.Out == "" {
		opts.Out = "."
	}
	if opts.Roots.Config == "" && opts.Roots.LocalShare == "" {
		roots, err := paths.DefaultLiveRoots()
		if err != nil {
			return DiffResult{}, fmt.Errorf("resolve live roots: %w", err)
		}
		opts.Roots = roots
	}

	profileName, err := resolveProfileName(opts.Out, opts.Profile)
	if err != nil {
		return DiffResult{}, err
	}
	if err := validateProfileName(profileName); err != nil {
		return DiffResult{}, err
	}
	profileDir, err := safeProfileDir(opts.Out, profileName)
	if err != nil {
		return DiffResult{}, err
	}
	if info, err := os.Stat(profileDir); err != nil {
		return DiffResult{}, fmt.Errorf("read profile %s: %w", profileName, err)
	} else if !info.IsDir() {
		return DiffResult{}, fmt.Errorf("profile %s is not a directory", profileName)
	}

	list, err := allowlist.LoadDefault()
	if err != nil {
		return DiffResult{}, fmt.Errorf("load allowlist: %w", err)
	}
	if err := allowlist.Validate(list); err != nil {
		return DiffResult{}, fmt.Errorf("validate allowlist: %w", err)
	}

	result := DiffResult{Profile: profileName, ProfileDir: profileDir}
	for _, file := range allowlist.SortedFiles(list.Files) {
		if err := ctx.Err(); err != nil {
			return DiffResult{}, fmt.Errorf("diff profile: %w", err)
		}
		diffs, err := diffFile(opts.Roots, profileDir, file)
		if err != nil {
			return DiffResult{}, err
		}
		result.Differences = append(result.Differences, diffs...)
	}
	return result, nil
}

func resolveProfileName(out, profileName string) (string, error) {
	if profileName != "" {
		return profileName, nil
	}
	profilesDir := filepath.Join(filepath.Clean(out), "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return "", fmt.Errorf("read profiles directory %s: %w", profilesDir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	switch len(names) {
	case 0:
		return "", fmt.Errorf("no profiles found in %s", profilesDir)
	case 1:
		return names[0], nil
	default:
		return "", fmt.Errorf("multiple profiles found; specify one: %s", strings.Join(names, ", "))
	}
}

func diffFile(roots paths.LiveRoots, profileDir string, file allowlist.File) ([]Diff, error) {
	livePath, err := roots.Join(file.Root, file.Path)
	if err != nil {
		return nil, fmt.Errorf("resolve live path %s/%s: %w", file.Root, file.Path, err)
	}
	savedPath, err := safeTargetPath(profileDir, file.Root, file.Path)
	if err != nil {
		return nil, err
	}

	savedData, savedErr := os.ReadFile(savedPath)
	liveData, liveErr := os.ReadFile(livePath)
	savedMissing := os.IsNotExist(savedErr)
	liveMissing := os.IsNotExist(liveErr)
	if savedErr != nil && !savedMissing {
		return nil, fmt.Errorf("read saved file %s: %w", savedPath, savedErr)
	}
	if liveErr != nil && !liveMissing {
		return nil, fmt.Errorf("read live file %s: %w", livePath, liveErr)
	}

	base := Diff{Root: string(file.Root), Path: file.Path, Kind: "file"}
	switch {
	case savedMissing && liveMissing:
		return nil, nil
	case savedMissing && !liveMissing:
		base.Status = "added"
		return []Diff{base}, nil
	case !savedMissing && liveMissing:
		base.Status = "removed"
		return []Diff{base}, nil
	}

	liveNormalized, err := normalizeFile(file, liveData)
	if err != nil {
		return nil, fmt.Errorf("normalize live file %s: %w", livePath, err)
	}
	if file.Parser == allowlist.ParserRawCopy {
		if bytes.Equal(savedData, liveNormalized) {
			return nil, nil
		}
		base.Status = "changed"
		return []Diff{base}, nil
	}

	savedDoc, err := kconfig.Parse(bytes.NewReader(savedData))
	if err != nil {
		return nil, fmt.Errorf("parse saved file %s: %w", savedPath, err)
	}
	liveDoc, err := kconfig.Parse(bytes.NewReader(liveNormalized))
	if err != nil {
		return nil, fmt.Errorf("parse live file %s: %w", livePath, err)
	}
	keyDiffs := kconfig.Diff(savedDoc, liveDoc)
	result := make([]Diff, 0, len(keyDiffs))
	for _, keyDiff := range keyDiffs {
		result = append(result, Diff{
			Root:     string(file.Root),
			Path:     file.Path,
			Status:   string(keyDiff.Status),
			Kind:     "key",
			Group:    append([]string(nil), keyDiff.Group...),
			Key:      keyDiff.Name,
			Locale:   keyDiff.Locale,
			OldValue: keyDiff.OldValue,
			NewValue: keyDiff.NewValue,
		})
	}
	return result, nil
}
