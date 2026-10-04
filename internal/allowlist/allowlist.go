package allowlist

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

//go:embed allowlist.v1.toml
var embedded embed.FS

const defaultPath = "allowlist.v1.toml"

type Root string

const (
	RootConfig     Root = "config"
	RootLocalShare Root = "local-share"
)

type Parser string

const (
	ParserKConfig Parser = "kconfig"
	ParserRawCopy Parser = "raw-copy"
)

type Allowlist struct {
	Version int
	Files   []File
}

type File struct {
	Root     Root
	Path     string
	Required bool
	Parser   Parser
	Notes    string
	Volatile []string
}

func LoadDefault() (*Allowlist, error) {
	data, err := embedded.ReadFile(defaultPath)
	if err != nil {
		return nil, fmt.Errorf("read embedded allowlist: %w", err)
	}
	allowlist, err := Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse embedded allowlist: %w", err)
	}
	return allowlist, nil
}

func Parse(input string) (*Allowlist, error) {
	var allowlist Allowlist
	var current *File
	var multilineKey string
	var multilineValues []string

	commitMultiline := func(lineNum int) error {
		if multilineKey == "" {
			return nil
		}
		if current == nil {
			return fmt.Errorf("line %d: multiline %s outside file entry", lineNum, multilineKey)
		}
		switch multilineKey {
		case "volatile":
			current.Volatile = append([]string(nil), multilineValues...)
		default:
			return fmt.Errorf("line %d: unsupported multiline key %q", lineNum, multilineKey)
		}
		multilineKey = ""
		multilineValues = nil
		return nil
	}

	lines := strings.Split(input, "\n")
	for idx, raw := range lines {
		lineNum := idx + 1
		line := stripComment(strings.TrimSpace(raw))
		if line == "" {
			continue
		}

		if multilineKey != "" {
			if line == "]" {
				if err := commitMultiline(lineNum); err != nil {
					return nil, err
				}
				continue
			}
			value := strings.TrimSuffix(line, ",")
			parsed, err := parseString(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: parse multiline value: %w", lineNum, err)
			}
			multilineValues = append(multilineValues, parsed)
			continue
		}

		if line == "[[files]]" {
			allowlist.Files = append(allowlist.Files, File{})
			current = &allowlist.Files[len(allowlist.Files)-1]
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected key=value", lineNum)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if value == "[" {
			multilineKey = key
			continue
		}

		if current == nil {
			switch key {
			case "version":
				version, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("line %d: parse version: %w", lineNum, err)
				}
				allowlist.Version = version
			default:
				return nil, fmt.Errorf("line %d: unsupported top-level key %q", lineNum, key)
			}
			continue
		}

		if err := parseFileField(current, key, value); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}
	if multilineKey != "" {
		return nil, fmt.Errorf("unterminated multiline array %q", multilineKey)
	}
	return &allowlist, nil
}

func Validate(allowlist *Allowlist) error {
	if allowlist == nil {
		return fmt.Errorf("allowlist is nil")
	}
	if allowlist.Version <= 0 {
		return fmt.Errorf("allowlist version must be positive")
	}
	seen := make(map[string]struct{})
	for idx, file := range allowlist.Files {
		if err := validateFile(file); err != nil {
			return fmt.Errorf("file %d: %w", idx, err)
		}
		id := string(file.Root) + ":" + file.Path
		if _, ok := seen[id]; ok {
			return fmt.Errorf("file %d: duplicate entry %s", idx, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func SortedFiles(files []File) []File {
	out := append([]File(nil), files...)
	sort.Slice(out, func(i, j int) bool {
		left := string(out[i].Root) + "/" + out[i].Path
		right := string(out[j].Root) + "/" + out[j].Path
		return left < right
	})
	return out
}

func validateFile(file File) error {
	switch file.Root {
	case RootConfig, RootLocalShare:
	default:
		return fmt.Errorf("invalid root %q", file.Root)
	}
	if file.Path == "" {
		return fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(file.Path) {
		return fmt.Errorf("path %q must be relative", file.Path)
	}
	clean := filepath.Clean(file.Path)
	if clean == "." || clean != file.Path || strings.HasPrefix(clean, "..") || strings.Contains(clean, string(filepath.Separator)+".."+string(filepath.Separator)) {
		return fmt.Errorf("path %q is unsafe", file.Path)
	}
	if strings.Contains(file.Path, "\x00") {
		return fmt.Errorf("path contains NUL")
	}
	switch file.Parser {
	case ParserKConfig, ParserRawCopy:
	default:
		return fmt.Errorf("invalid parser %q", file.Parser)
	}
	return nil
}

func parseFileField(file *File, key, value string) error {
	switch key {
	case "root":
		parsed, err := parseString(value)
		if err != nil {
			return fmt.Errorf("parse root: %w", err)
		}
		file.Root = Root(parsed)
	case "path":
		parsed, err := parseString(value)
		if err != nil {
			return fmt.Errorf("parse path: %w", err)
		}
		file.Path = parsed
	case "required":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse required: %w", err)
		}
		file.Required = parsed
	case "parser":
		parsed, err := parseString(value)
		if err != nil {
			return fmt.Errorf("parse parser: %w", err)
		}
		file.Parser = Parser(parsed)
	case "notes":
		parsed, err := parseString(value)
		if err != nil {
			return fmt.Errorf("parse notes: %w", err)
		}
		file.Notes = parsed
	case "volatile":
		parsed, err := parseStringArray(value)
		if err != nil {
			return fmt.Errorf("parse volatile: %w", err)
		}
		file.Volatile = parsed
	default:
		return fmt.Errorf("unsupported file key %q", key)
	}
	return nil
}

func parseStringArray(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "[]" {
		return nil, nil
	}
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil, fmt.Errorf("expected array")
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	if inner == "" {
		return nil, nil
	}
	parts := strings.Split(inner, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		parsed, err := parseString(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		values = append(values, parsed)
	}
	return values, nil
}

func parseString(value string) (string, error) {
	parsed, err := strconv.Unquote(value)
	if err != nil {
		return "", fmt.Errorf("expected quoted string: %w", err)
	}
	return parsed, nil
}

func stripComment(line string) string {
	inString := false
	escaped := false
	for idx, r := range line {
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
			return strings.TrimSpace(line[:idx])
		}
	}
	return strings.TrimSpace(line)
}
