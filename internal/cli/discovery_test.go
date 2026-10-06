package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/A-Murchison/dot-plasma/internal/allowlist"
	"github.com/A-Murchison/dot-plasma/internal/paths"
)

func TestCheckAllowlistedFiles(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	localShareRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(configRoot, "presentrc"), []byte("[General]\n"), 0o600); err != nil {
		t.Fatalf("write present fixture: %s", err.Error())
	}
	if err := os.Mkdir(filepath.Join(configRoot, "directoryrc"), 0o700); err != nil {
		t.Fatalf("create directory fixture: %s", err.Error())
	}

	list := &allowlist.Allowlist{
		Version: 1,
		Files: []allowlist.File{
			{Root: allowlist.RootConfig, Path: "presentrc", Required: true, Parser: allowlist.ParserKConfig},
			{Root: allowlist.RootConfig, Path: "optional-missingrc", Required: false, Parser: allowlist.ParserKConfig},
			{Root: allowlist.RootLocalShare, Path: "required-missingrc", Required: true, Parser: allowlist.ParserRawCopy},
			{Root: allowlist.RootConfig, Path: "directoryrc", Required: true, Parser: allowlist.ParserKConfig},
		},
	}
	roots := paths.LiveRoots{Config: configRoot, LocalShare: localShareRoot}

	got, err := checkAllowlistedFiles(list, roots)
	if err != nil {
		t.Fatalf("checkAllowlistedFiles returned error: %s", err.Error())
	}
	want := allowlistedFilesCheck{Total: 4, Present: 1, MissingOptional: 1, MissingRequired: 1, Errors: 1}
	if got != want {
		t.Fatalf("checkAllowlistedFiles = %#v, want %#v", got, want)
	}
}
