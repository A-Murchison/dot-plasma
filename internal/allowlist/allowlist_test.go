package allowlist

import (
	"slices"
	"strings"
	"testing"
)

func TestLoadDefault(t *testing.T) {
	t.Parallel()

	list, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault returned error: %v", err)
	}
	if err := Validate(list); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if list.Version != 1 {
		t.Fatalf("Version = %d, want 1", list.Version)
	}
	if len(list.Files) == 0 {
		t.Fatal("default allowlist has no files")
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		"# comment before version",
		"version = 1 # trailing comment",
		"",
		"[[files]]",
		`root = "config"`,
		`path = "plasma#theme" # comment after quoted hash`,
		"required = true",
		`parser = "kconfig"`,
		`notes = "hash # inside quotes is preserved"`,
		"volatile = [",
		`  "Containments/*/lastScreen",`,
		`  "Containments/*/Applets/*/Configuration/PreloadWeight",`,
		"]",
		"",
	}, "\n")

	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if list.Version != 1 {
		t.Fatalf("Version = %d, want 1", list.Version)
	}
	if len(list.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(list.Files))
	}

	got := list.Files[0]
	if got.Root != RootConfig {
		t.Errorf("Root = %q, want %q", got.Root, RootConfig)
	}
	if got.Path != "plasma#theme" {
		t.Errorf("Path = %q, want %q", got.Path, "plasma#theme")
	}
	if !got.Required {
		t.Error("Required = false, want true")
	}
	if got.Parser != ParserKConfig {
		t.Errorf("Parser = %q, want %q", got.Parser, ParserKConfig)
	}
	if got.Notes != "hash # inside quotes is preserved" {
		t.Errorf("Notes = %q, want %q", got.Notes, "hash # inside quotes is preserved")
	}
	wantVolatile := []string{
		"Containments/*/lastScreen",
		"Containments/*/Applets/*/Configuration/PreloadWeight",
	}
	if !slices.Equal(got.Volatile, wantVolatile) {
		t.Errorf("Volatile = %#v, want %#v", got.Volatile, wantVolatile)
	}
}

func TestParseRejectsMalformedInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "unsupported top level key",
			input:   `name = "dotplasma"`,
			wantErr: "unsupported top-level key",
		},
		{
			name: "unsupported file key",
			input: strings.Join([]string{
				"version = 1",
				"[[files]]",
				`root = "config"`,
				`unexpected = "value"`,
			}, "\n"),
			wantErr: "unsupported file key",
		},
		{
			name: "unterminated multiline array",
			input: strings.Join([]string{
				"version = 1",
				"[[files]]",
				"volatile = [",
				`  "one",`,
			}, "\n"),
			wantErr: "unterminated multiline array",
		},
		{
			name: "bad quoted string",
			input: strings.Join([]string{
				"version = 1",
				"[[files]]",
				`root = config`,
			}, "\n"),
			wantErr: "expected quoted string",
		},
		{
			name: "bad required bool",
			input: strings.Join([]string{
				"version = 1",
				"[[files]]",
				`required = sometimes`,
			}, "\n"),
			wantErr: "parse required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Parse(tt.input)
			if err == nil {
				t.Fatal("Parse returned nil error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Parse error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRejectsInvalidAllowlists(t *testing.T) {
	t.Parallel()

	validFile := File{Root: RootConfig, Path: "kwinrc", Parser: ParserKConfig}
	tests := []struct {
		name      string
		allowlist *Allowlist
		wantErr   string
	}{
		{
			name:      "nil allowlist",
			allowlist: nil,
			wantErr:   "allowlist is nil",
		},
		{
			name:      "non-positive version",
			allowlist: &Allowlist{Version: 0, Files: []File{validFile}},
			wantErr:   "version must be positive",
		},
		{
			name:      "invalid root",
			allowlist: &Allowlist{Version: 1, Files: []File{{Root: Root("cache"), Path: "kwinrc", Parser: ParserKConfig}}},
			wantErr:   "invalid root",
		},
		{
			name:      "empty path",
			allowlist: &Allowlist{Version: 1, Files: []File{{Root: RootConfig, Path: "", Parser: ParserKConfig}}},
			wantErr:   "path is empty",
		},
		{
			name:      "absolute path",
			allowlist: &Allowlist{Version: 1, Files: []File{{Root: RootConfig, Path: "/tmp/kwinrc", Parser: ParserKConfig}}},
			wantErr:   "must be relative",
		},
		{
			name:      "traversal path",
			allowlist: &Allowlist{Version: 1, Files: []File{{Root: RootConfig, Path: "../kwinrc", Parser: ParserKConfig}}},
			wantErr:   "unsafe",
		},
		{
			name:      "unclean path",
			allowlist: &Allowlist{Version: 1, Files: []File{{Root: RootConfig, Path: "foo/../kwinrc", Parser: ParserKConfig}}},
			wantErr:   "unsafe",
		},
		{
			name:      "nul path",
			allowlist: &Allowlist{Version: 1, Files: []File{{Root: RootConfig, Path: "kwinrc\x00", Parser: ParserKConfig}}},
			wantErr:   "NUL",
		},
		{
			name:      "invalid parser",
			allowlist: &Allowlist{Version: 1, Files: []File{{Root: RootConfig, Path: "kwinrc", Parser: Parser("json")}}},
			wantErr:   "invalid parser",
		},
		{
			name: "duplicate file",
			allowlist: &Allowlist{Version: 1, Files: []File{
				validFile,
				validFile,
			}},
			wantErr: "duplicate entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Validate(tt.allowlist)
			if err == nil {
				t.Fatal("Validate returned nil error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
