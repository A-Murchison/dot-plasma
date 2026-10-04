// Package kconfig parses KDE's KConfig/INI-like configuration files.
package kconfig

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Document is a parsed KConfig document. Lines preserve file order.
type Document struct {
	Lines []Line
}

// LineKind describes the kind of parsed line.
type LineKind int

const (
	LineBlank LineKind = iota
	LineComment
	LineGroup
	LineKey
)

// Line is one parsed source line.
type Line struct {
	Kind    LineKind
	Raw     string
	Group   Group
	Key     Key
	LineNum int
}

// Group is a KConfig group header, such as [Containments][1][Applets][2][$i].
type Group struct {
	Path  []string
	Flags []string
}

// Key is a KConfig key/value line, such as Name[en_US][$i]=Value.
type Key struct {
	Group  []string
	Name   string
	Locale string
	Flags  []string
	Value  string
}

// Entry is the effective value of a key in a group.
type Entry struct {
	Group  []string
	Name   string
	Locale string
	Flags  []string
	Value  string
}

// Difference is a key-level difference between two documents.
type Difference struct {
	Group    []string
	Name     string
	Locale   string
	OldValue string
	NewValue string
	Status   DiffStatus
}

// DiffStatus describes the type of document difference.
type DiffStatus string

const (
	DiffAdded   DiffStatus = "added"
	DiffRemoved DiffStatus = "removed"
	DiffChanged DiffStatus = "changed"
)

// Parse reads a KConfig document from r.
func Parse(r io.Reader) (*Document, error) {
	scanner := bufio.NewScanner(r)
	var lines []Line
	var currentGroup []string
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		line := Line{Raw: raw, LineNum: lineNum}
		switch {
		case trimmed == "":
			line.Kind = LineBlank
		case strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";"):
			line.Kind = LineComment
		case strings.HasPrefix(trimmed, "["):
			group, err := parseGroup(trimmed)
			if err != nil {
				return nil, fmt.Errorf("parse line %d group: %w", lineNum, err)
			}
			line.Kind = LineGroup
			line.Group = group
			currentGroup = append([]string(nil), group.Path...)
		case strings.Contains(raw, "="):
			key, err := parseKey(raw, currentGroup)
			if err != nil {
				return nil, fmt.Errorf("parse line %d key: %w", lineNum, err)
			}
			line.Kind = LineKey
			line.Key = key
		default:
			return nil, fmt.Errorf("parse line %d: expected blank, comment, group, or key", lineNum)
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan kconfig: %w", err)
	}

	return &Document{Lines: lines}, nil
}

// String serializes the document in a stable KConfig representation.
func (d *Document) String() string {
	var b strings.Builder
	for _, line := range d.Lines {
		switch line.Kind {
		case LineBlank, LineComment:
			b.WriteString(line.Raw)
		case LineGroup:
			b.WriteString(formatGroup(line.Group))
		case LineKey:
			b.WriteString(formatKey(line.Key))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Entries returns effective key entries. Duplicate keys are allowed; the last
// value for an identical group/name/locale tuple wins.
func (d *Document) Entries() []Entry {
	byID := make(map[string]Entry)
	for _, line := range d.Lines {
		if line.Kind != LineKey {
			continue
		}
		entry := Entry{
			Group:  append([]string(nil), line.Key.Group...),
			Name:   line.Key.Name,
			Locale: line.Key.Locale,
			Flags:  append([]string(nil), line.Key.Flags...),
			Value:  line.Key.Value,
		}
		byID[entryID(entry.Group, entry.Name, entry.Locale)] = entry
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	entries := make([]Entry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, byID[id])
	}
	return entries
}

// Diff compares oldDoc to newDoc and returns deterministic key-level changes.
func Diff(oldDoc, newDoc *Document) []Difference {
	oldEntries := entriesByID(oldDoc)
	newEntries := entriesByID(newDoc)

	idSet := make(map[string]struct{})
	for id := range oldEntries {
		idSet[id] = struct{}{}
	}
	for id := range newEntries {
		idSet[id] = struct{}{}
	}

	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	diffs := make([]Difference, 0)
	for _, id := range ids {
		oldEntry, oldOK := oldEntries[id]
		newEntry, newOK := newEntries[id]
		switch {
		case !oldOK && newOK:
			diffs = append(diffs, Difference{Group: newEntry.Group, Name: newEntry.Name, Locale: newEntry.Locale, NewValue: newEntry.Value, Status: DiffAdded})
		case oldOK && !newOK:
			diffs = append(diffs, Difference{Group: oldEntry.Group, Name: oldEntry.Name, Locale: oldEntry.Locale, OldValue: oldEntry.Value, Status: DiffRemoved})
		case oldOK && newOK && oldEntry.Value != newEntry.Value:
			diffs = append(diffs, Difference{Group: newEntry.Group, Name: newEntry.Name, Locale: newEntry.Locale, OldValue: oldEntry.Value, NewValue: newEntry.Value, Status: DiffChanged})
		}
	}
	return diffs
}

func entriesByID(doc *Document) map[string]Entry {
	entries := make(map[string]Entry)
	for _, entry := range doc.Entries() {
		entries[entryID(entry.Group, entry.Name, entry.Locale)] = entry
	}
	return entries
}

func parseGroup(s string) (Group, error) {
	tokens, err := parseBracketTokens(s)
	if err != nil {
		return Group{}, err
	}
	if len(tokens) == 0 {
		return Group{}, fmt.Errorf("empty group")
	}

	var group Group
	for _, token := range tokens {
		if strings.HasPrefix(token, "$") {
			group.Flags = append(group.Flags, token)
			continue
		}
		if token == "" {
			return Group{}, fmt.Errorf("empty group segment")
		}
		group.Path = append(group.Path, token)
	}
	if len(group.Path) == 0 {
		return Group{}, fmt.Errorf("group has no path segments")
	}
	return group, nil
}

func parseKey(s string, group []string) (Key, error) {
	idx := strings.Index(s, "=")
	if idx < 0 {
		return Key{}, fmt.Errorf("missing equals")
	}
	left := strings.TrimSpace(s[:idx])
	if left == "" {
		return Key{}, fmt.Errorf("empty key")
	}

	name, tokens, err := splitNameAndBracketTokens(left)
	if err != nil {
		return Key{}, err
	}
	if name == "" {
		return Key{}, fmt.Errorf("empty key name")
	}

	key := Key{Group: append([]string(nil), group...), Name: name, Value: s[idx+1:]}
	for _, token := range tokens {
		if strings.HasPrefix(token, "$") {
			key.Flags = append(key.Flags, token)
			continue
		}
		if key.Locale != "" {
			return Key{}, fmt.Errorf("multiple locale tokens")
		}
		key.Locale = token
	}
	return key, nil
}

func parseBracketTokens(s string) ([]string, error) {
	var tokens []string
	for s != "" {
		if !strings.HasPrefix(s, "[") {
			return nil, fmt.Errorf("expected '[' near %q", s)
		}
		end := strings.Index(s, "]")
		if end < 0 {
			return nil, fmt.Errorf("missing closing ']'")
		}
		tokens = append(tokens, s[1:end])
		s = s[end+1:]
	}
	return tokens, nil
}

func splitNameAndBracketTokens(s string) (string, []string, error) {
	firstBracket := strings.Index(s, "[")
	if firstBracket < 0 {
		return s, nil, nil
	}
	name := s[:firstBracket]
	tokens, err := parseBracketTokens(s[firstBracket:])
	if err != nil {
		return "", nil, err
	}
	return name, tokens, nil
}

func formatGroup(group Group) string {
	var b strings.Builder
	for _, part := range group.Path {
		b.WriteString("[")
		b.WriteString(part)
		b.WriteString("]")
	}
	for _, flag := range group.Flags {
		b.WriteString("[")
		b.WriteString(flag)
		b.WriteString("]")
	}
	return b.String()
}

func formatKey(key Key) string {
	var b strings.Builder
	b.WriteString(key.Name)
	if key.Locale != "" {
		b.WriteString("[")
		b.WriteString(key.Locale)
		b.WriteString("]")
	}
	for _, flag := range key.Flags {
		b.WriteString("[")
		b.WriteString(flag)
		b.WriteString("]")
	}
	b.WriteString("=")
	b.WriteString(key.Value)
	return b.String()
}

func entryID(group []string, name, locale string) string {
	return strings.Join(group, "\x00") + "\x01" + name + "\x01" + locale
}
