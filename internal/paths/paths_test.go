package paths

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/A-Murchison/dot-plasma/internal/allowlist"
)

func TestJoin(t *testing.T) {
	t.Parallel()

	configRoot := filepath.Join(string(filepath.Separator), "home", "me", ".config")
	localShareRoot := filepath.Join(string(filepath.Separator), "home", "me", ".local", "share")
	roots := LiveRoots{Config: configRoot, LocalShare: localShareRoot}

	tests := []struct {
		name         string
		root         allowlist.Root
		relativePath string
		want         string
		wantErr      string
	}{
		{
			name:         "config root",
			root:         allowlist.RootConfig,
			relativePath: "kwinrc",
			want:         filepath.Join(configRoot, "kwinrc"),
		},
		{
			name:         "local share root",
			root:         allowlist.RootLocalShare,
			relativePath: filepath.Join("plasma", "look-and-feel", "theme"),
			want:         filepath.Join(localShareRoot, "plasma", "look-and-feel", "theme"),
		},
		{
			name:         "unknown root",
			root:         allowlist.Root("cache"),
			relativePath: "kwinrc",
			wantErr:      "unknown root",
		},
		{
			name:         "empty relative path",
			root:         allowlist.RootConfig,
			relativePath: "",
			wantErr:      "relative path is empty",
		},
		{
			name:         "absolute relative path",
			root:         allowlist.RootConfig,
			relativePath: filepath.Join(string(filepath.Separator), "tmp", "kwinrc"),
			wantErr:      "must be relative",
		},
		{
			name:         "traversal path",
			root:         allowlist.RootConfig,
			relativePath: filepath.Join("..", "kwinrc"),
			wantErr:      "unsafe",
		},
		{
			name:         "unclean path",
			root:         allowlist.RootConfig,
			relativePath: filepath.FromSlash("foo/../kwinrc"),
			wantErr:      "unsafe",
		},
		{
			name:         "current directory path",
			root:         allowlist.RootConfig,
			relativePath: ".",
			wantErr:      "unsafe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := roots.Join(tt.root, tt.relativePath)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("Join returned nil error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Join error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Join returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("Join = %q, want %q", got, tt.want)
			}
		})
	}
}
