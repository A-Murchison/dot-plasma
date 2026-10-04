package kconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalKDEFixturesParseAndRoundTrip(t *testing.T) {
	t.Parallel()

	fixtures := []string{
		"kdeglobals",
		"kwinrc",
		"plasmarc",
		"plasma-org.kde.plasma.desktop-appletsrc",
	}

	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join("testdata", "local", fixture)
			original, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					t.Skipf("fixture missing: %s", path)
				}
				t.Fatalf("read fixture %s: %v", path, err)
			}

			doc, err := Parse(strings.NewReader(string(original)))
			if err != nil {
				t.Fatalf("Parse returned error for %s: %v", path, err)
			}
			if len(original) > 0 && len(doc.Lines) == 0 {
				t.Fatalf("parsed zero lines for non-empty fixture %s", path)
			}

			if got := doc.String(); got != string(original) {
				t.Fatalf("round trip mismatch for %s", path)
			}
		})
	}
}
