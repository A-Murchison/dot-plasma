// Package config loads dotplasma's user configuration file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config is the top-level dotplasma configuration.
type Config struct {
	OutputDir string
}

// DefaultDir returns the default dotplasma user configuration directory.
func DefaultDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(configDir, "dotplasma"), nil
}

// DefaultPath returns the default user configuration path.
func DefaultPath() (string, error) {
	defaultDir, err := DefaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(defaultDir, "config.toml"), nil
}

// DefaultOutputDir returns the default output directory containing profiles and backups.
func DefaultOutputDir() (string, error) {
	return DefaultDir()
}

// Load reads a dotplasma configuration file.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %s: %w", path, err)
	}
	cfg, err := Parse(string(data))
	if err != nil {
		return Config{}, fmt.Errorf("parse config file %s: %w", path, err)
	}
	return cfg, nil
}

// LoadOptional reads either an explicit config file or the default config file.
// Missing default config files are ignored. Missing explicit config files fail.
func LoadOptional(path string) (Config, string, error) {
	if path != "" {
		cfg, err := Load(path)
		if err != nil {
			return Config{}, path, err
		}
		return cfg, path, nil
	}

	defaultPath, err := DefaultPath()
	if err != nil {
		return Config{}, "", err
	}
	cfg, err := Load(defaultPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, defaultPath, nil
		}
		return Config{}, defaultPath, err
	}
	return cfg, defaultPath, nil
}

// Parse parses the supported config.toml subset.
func Parse(data string) (Config, error) {
	var cfg Config
	for lineNum, line := range strings.Split(data, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			return Config{}, fmt.Errorf("line %d: sections are not supported", lineNum+1)
		}
		key, rawValue, ok := strings.Cut(trimmed, "=")
		if !ok {
			return Config{}, fmt.Errorf("line %d: expected key/value", lineNum+1)
		}
		key = strings.TrimSpace(key)
		rawValue = stripInlineComment(strings.TrimSpace(rawValue))
		switch key {
		case "output_dir":
			value, err := strconv.Unquote(rawValue)
			if err != nil {
				return Config{}, fmt.Errorf("line %d: parse output_dir: %w", lineNum+1, err)
			}
			if strings.TrimSpace(value) == "" {
				return Config{}, fmt.Errorf("line %d: output_dir is empty", lineNum+1)
			}
			cfg.OutputDir = value
		default:
			return Config{}, fmt.Errorf("line %d: unknown key %q", lineNum+1, key)
		}
	}
	return cfg, nil
}

func stripInlineComment(value string) string {
	inString := false
	escaped := false
	for i, r := range value {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inString {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if r == '#' && !inString {
			return strings.TrimSpace(value[:i])
		}
	}
	return strings.TrimSpace(value)
}
