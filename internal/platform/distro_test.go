package platform

import "testing"

func TestParseOSReleaseDistro(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "pretty name preferred",
			input: `NAME="Fedora Linux"
VERSION_ID=42
PRETTY_NAME="Fedora Linux 42 (KDE Plasma)"
`,
			want: "Fedora Linux 42 (KDE Plasma)",
		},
		{
			name: "name and version fallback",
			input: `NAME="Arch Linux"
VERSION_ID=rolling
`,
			want: "Arch Linux rolling",
		},
		{
			name: "name only fallback",
			input: `NAME=Debian
`,
			want: "Debian",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseOSReleaseDistro(tt.input)
			if got != tt.want {
				t.Fatalf("ParseOSReleaseDistro() = %q, want %q", got, tt.want)
			}
		})
	}
}
