package plasma

import "testing"

func TestParseVersionOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "plasma six", output: "plasmashell 6.2.5\n", want: "6.2.5"},
		{name: "kde prefix", output: "plasmashell 5.27.11", want: "5.27.11"},
		{name: "empty", output: "\n", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ParseVersionOutput(tt.output)
			if got != tt.want {
				t.Fatalf("ParseVersionOutput(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}
