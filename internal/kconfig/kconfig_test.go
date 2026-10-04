package kconfig

import (
	"strings"
	"testing"
)

func TestParseRoundTripNestedGroupsLocalesAndFlags(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		"# comment",
		"",
		"[Containments][1][Applets][2][$i]",
		"Name[en_US]=Application Launcher",
		"Immutable[$i]=true",
		"escaped[$e]=line\\nvalue",
		"",
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if got := doc.String(); got != input {
		t.Fatalf("round trip mismatch\n--- got ---\n%q\n--- want ---\n%q", got, input)
	}
}

func TestEntriesDuplicateKeyLastValueWins(t *testing.T) {
	t.Parallel()

	doc, err := Parse(strings.NewReader("[General]\nTheme=one\nTheme=two\n"))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	entries := doc.Entries()
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Value != "two" {
		t.Fatalf("entry value = %q, want %q", entries[0].Value, "two")
	}
}

func TestDiff(t *testing.T) {
	t.Parallel()

	oldDoc := mustParse(t, strings.Join([]string{
		"[General]",
		"Theme=one",
		"Removed=yes",
		"Same=value",
		"",
	}, "\n"))
	newDoc := mustParse(t, strings.Join([]string{
		"[General]",
		"Theme=two",
		"Added=yes",
		"Same=value",
		"",
	}, "\n"))

	diffs := Diff(oldDoc, newDoc)
	if len(diffs) != 3 {
		t.Fatalf("len(diffs) = %d, want 3: %#v", len(diffs), diffs)
	}

	assertDiff(t, diffs[0], DiffAdded, "Added", "", "yes")
	assertDiff(t, diffs[1], DiffRemoved, "Removed", "yes", "")
	assertDiff(t, diffs[2], DiffChanged, "Theme", "one", "two")
}

func TestParseMalformedGroupReportsLine(t *testing.T) {
	t.Parallel()

	_, err := Parse(strings.NewReader("[General\nTheme=one\n"))
	if err == nil {
		t.Fatal("Parse returned nil error")
	}
	if !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("error missing line number: %v", err)
	}
}

func TestParseRejectsMultipleLocales(t *testing.T) {
	t.Parallel()

	_, err := Parse(strings.NewReader("Name[en_US][de_DE]=Value\n"))
	if err == nil {
		t.Fatal("Parse returned nil error")
	}
	if !strings.Contains(err.Error(), "multiple locale tokens") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustParse(t *testing.T, input string) *Document {
	t.Helper()
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	return doc
}

func assertDiff(t *testing.T, diff Difference, status DiffStatus, name, oldValue, newValue string) {
	t.Helper()
	if diff.Status != status || diff.Name != name || diff.OldValue != oldValue || diff.NewValue != newValue {
		t.Fatalf("diff = %#v, want status=%s name=%s old=%q new=%q", diff, status, name, oldValue, newValue)
	}
}
