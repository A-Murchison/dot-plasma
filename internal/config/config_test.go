package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    string
		want    Config
		wantErr string
	}{
		{
			name: "output dir",
			data: "output_dir = \"/tmp/dotplasma\"\n",
			want: Config{OutputDir: "/tmp/dotplasma"},
		},
		{
			name: "comments and inline comment",
			data: "# config\noutput_dir = \"/tmp/dot#plasma\" # comment\n",
			want: Config{OutputDir: "/tmp/dot#plasma"},
		},
		{
			name:    "unknown key",
			data:    "profiles_dir = \"/tmp/profiles\"\n",
			wantErr: "unknown key",
		},
		{
			name:    "empty output dir",
			data:    "output_dir = \" \"\n",
			wantErr: "output_dir is empty",
		},
		{
			name:    "sections rejected",
			data:    "[paths]\noutput_dir = \"/tmp/dotplasma\"\n",
			wantErr: "sections are not supported",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Parse(tt.data)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Parse() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDefaultPathsUseXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	wantDir := filepath.Join(configHome, "dotplasma")
	gotDir, err := DefaultDir()
	if err != nil {
		t.Fatalf("DefaultDir() error = %v", err)
	}
	if gotDir != wantDir {
		t.Fatalf("DefaultDir() = %q, want %q", gotDir, wantDir)
	}

	gotPath, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}
	wantPath := filepath.Join(wantDir, "config.toml")
	if gotPath != wantPath {
		t.Fatalf("DefaultPath() = %q, want %q", gotPath, wantPath)
	}

	gotOut, err := DefaultOutputDir()
	if err != nil {
		t.Fatalf("DefaultOutputDir() error = %v", err)
	}
	if gotOut != wantDir {
		t.Fatalf("DefaultOutputDir() = %q, want %q", gotOut, wantDir)
	}
}

func TestLoadOptional(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("output_dir = \"/tmp/dotplasma\"\n"), 0o600); err != nil {
		t.Fatalf("write config: %s", err.Error())
	}

	got, usedPath, err := LoadOptional(configPath)
	if err != nil {
		t.Fatalf("LoadOptional() error = %v", err)
	}
	if usedPath != configPath {
		t.Fatalf("LoadOptional() usedPath = %q, want %q", usedPath, configPath)
	}
	want := Config{OutputDir: "/tmp/dotplasma"}
	if got != want {
		t.Fatalf("LoadOptional() = %+v, want %+v", got, want)
	}
}
