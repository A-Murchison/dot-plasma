package profile

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"dot-plasma/internal/allowlist"
)

type ImportOptions struct {
	SourceDir string
	Profile   string
	Out       string
	DryRun    bool
}

type ImportResult struct {
	SourceDir  string
	ProfileDir string
	Actions    []Action
}

func ImportProfile(ctx context.Context, opts ImportOptions) (ImportResult, error) {
	if err := ctx.Err(); err != nil {
		return ImportResult{}, fmt.Errorf("import profile: %w", err)
	}
	if opts.SourceDir == "" {
		return ImportResult{}, fmt.Errorf("source directory is empty")
	}
	if err := validateProfileName(opts.Profile); err != nil {
		return ImportResult{}, err
	}
	if opts.Out == "" {
		opts.Out = "."
	}

	sourceDir := filepath.Clean(opts.SourceDir)
	info, err := os.Stat(sourceDir)
	if err != nil {
		return ImportResult{}, fmt.Errorf("read source profile directory %s: %w", sourceDir, err)
	}
	if !info.IsDir() {
		return ImportResult{}, fmt.Errorf("source profile %s is not a directory", sourceDir)
	}

	if err := rejectSymlinks(sourceDir); err != nil {
		return ImportResult{}, err
	}

	targetDir, err := safeProfileDir(opts.Out, opts.Profile)
	if err != nil {
		return ImportResult{}, err
	}
	if _, err := os.Stat(targetDir); err == nil {
		return ImportResult{}, fmt.Errorf("profile %s already exists at %s", opts.Profile, targetDir)
	} else if !os.IsNotExist(err) {
		return ImportResult{}, fmt.Errorf("check target profile %s: %w", targetDir, err)
	}

	manifest, err := readImportManifest(filepath.Join(sourceDir, "manifests", "files.toml"))
	if err != nil {
		return ImportResult{}, err
	}
	if len(manifest) == 0 {
		return ImportResult{}, fmt.Errorf("source manifest contains no files")
	}

	copyPlan, err := buildImportCopyPlan(sourceDir, targetDir, manifest)
	if err != nil {
		return ImportResult{}, err
	}

	result := ImportResult{SourceDir: sourceDir, ProfileDir: targetDir, Actions: make([]Action, 0, len(copyPlan))}
	for _, item := range copyPlan {
		if err := ctx.Err(); err != nil {
			return ImportResult{}, fmt.Errorf("import profile: %w", err)
		}
		status := "create"
		if opts.DryRun {
			status = "would-create"
		} else {
			if err := os.MkdirAll(filepath.Dir(item.target), 0o700); err != nil {
				return ImportResult{}, fmt.Errorf("create import target directory %s: %w", filepath.Dir(item.target), err)
			}
			if err := os.WriteFile(item.target, item.data, 0o600); err != nil {
				return ImportResult{}, fmt.Errorf("write imported file %s: %w", item.target, err)
			}
		}
		result.Actions = append(result.Actions, Action{Status: status, Source: item.source, Target: item.target})
	}
	return result, nil
}

type importManifestFile struct {
	Root   string
	Path   string
	SHA256 string
	Bytes  int
}

type importCopyItem struct {
	source string
	target string
	data   []byte
}

func buildImportCopyPlan(sourceDir, targetDir string, manifest []importManifestFile) ([]importCopyItem, error) {
	metadata := []struct {
		rel      string
		required bool
	}{
		{rel: "profile.toml", required: true},
		{rel: "README.md", required: true},
		{rel: filepath.Join("manifests", "files.toml"), required: true},
		{rel: filepath.Join("manifests", "ignored.toml")},
		{rel: filepath.Join("manifests", "environment.toml")},
		{rel: filepath.Join("manifests", "privacy.md")},
	}
	items := make([]importCopyItem, 0, len(metadata)+len(manifest))
	for _, meta := range metadata {
		item, err := importCopyItemForRel(sourceDir, targetDir, meta.rel, meta.required)
		if err != nil {
			return nil, err
		}
		if item != nil {
			items = append(items, *item)
		}
	}
	for _, file := range manifest {
		sourcePath, err := safeTargetPath(sourceDir, allowlist.Root(file.Root), file.Path)
		if err != nil {
			return nil, fmt.Errorf("validate manifest path %s/%s: %w", file.Root, file.Path, err)
		}
		targetPath, err := safeTargetPath(targetDir, allowlist.Root(file.Root), file.Path)
		if err != nil {
			return nil, fmt.Errorf("validate target manifest path %s/%s: %w", file.Root, file.Path, err)
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("read manifest file %s: %w", sourcePath, err)
		}
		if len(data) != file.Bytes {
			return nil, fmt.Errorf("manifest file %s byte count mismatch: got %d want %d", sourcePath, len(data), file.Bytes)
		}
		if got := sha256Hex(data); got != file.SHA256 {
			return nil, fmt.Errorf("manifest file %s sha256 mismatch", sourcePath)
		}
		items = append(items, importCopyItem{source: sourcePath, target: targetPath, data: data})
	}
	return items, nil
}

func importCopyItemForRel(sourceDir, targetDir, rel string, required bool) (*importCopyItem, error) {
	if filepath.IsAbs(rel) || filepath.Clean(rel) != rel || strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("import metadata path %q is unsafe", rel)
	}
	sourcePath := filepath.Join(sourceDir, rel)
	targetPath := filepath.Join(targetDir, rel)
	if !isWithin(sourceDir, sourcePath) || !isWithin(targetDir, targetPath) {
		return nil, fmt.Errorf("import metadata path %q escapes expected directory", rel)
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) && !required {
			return nil, nil
		}
		return nil, fmt.Errorf("read import metadata %s: %w", sourcePath, err)
	}
	return &importCopyItem{source: sourcePath, target: targetPath, data: data}, nil
}

func readImportManifest(path string) ([]importManifestFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read source manifest %s: %w", path, err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	var files []importManifestFile
	var current *importManifestFile
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "[[files]]" {
			if current != nil {
				files = append(files, *current)
			}
			current = &importManifestFile{}
			continue
		}
		if current == nil {
			return nil, fmt.Errorf("parse source manifest line %d: expected [[files]]", lineNum)
		}
		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("parse source manifest line %d: expected key/value", lineNum)
		}
		key = strings.TrimSpace(key)
		rawValue = strings.TrimSpace(rawValue)
		switch key {
		case "root":
			value, err := strconv.Unquote(rawValue)
			if err != nil {
				return nil, fmt.Errorf("parse source manifest line %d root: %w", lineNum, err)
			}
			current.Root = value
		case "path":
			value, err := strconv.Unquote(rawValue)
			if err != nil {
				return nil, fmt.Errorf("parse source manifest line %d path: %w", lineNum, err)
			}
			current.Path = value
		case "sha256":
			value, err := strconv.Unquote(rawValue)
			if err != nil {
				return nil, fmt.Errorf("parse source manifest line %d sha256: %w", lineNum, err)
			}
			current.SHA256 = value
		case "bytes":
			value, err := strconv.Atoi(rawValue)
			if err != nil {
				return nil, fmt.Errorf("parse source manifest line %d bytes: %w", lineNum, err)
			}
			current.Bytes = value
		case "parser":
			// Parser is informational for import. Validation happens before diff/apply.
		default:
			return nil, fmt.Errorf("parse source manifest line %d: unknown key %q", lineNum, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan source manifest: %w", err)
	}
	if current != nil {
		files = append(files, *current)
	}
	for idx, file := range files {
		if file.Root != string(allowlist.RootConfig) && file.Root != string(allowlist.RootLocalShare) {
			return nil, fmt.Errorf("source manifest file %d has invalid root %q", idx+1, file.Root)
		}
		if file.Path == "" || file.SHA256 == "" || file.Bytes < 0 {
			return nil, fmt.Errorf("source manifest file %d is incomplete", idx+1)
		}
		if _, err := hex.DecodeString(file.SHA256); err != nil || len(file.SHA256) != sha256.Size*2 {
			return nil, fmt.Errorf("source manifest file %d has invalid sha256", idx+1)
		}
	}
	return files, nil
}

func rejectSymlinks(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk import source %s: %w", path, err)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("import source contains symlink %s", path)
		}
		return nil
	})
}
