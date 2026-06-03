package vm

import "testing"

func TestCmdContainer_shortID_Good(t *testing.T) {
	// Long ids truncate to 8 chars; short Apple-style names (the container id
	// is the user-chosen --name) pass through unharmed. A naive id[:8] would
	// panic on names shorter than 8 characters.
	cases := map[string]string{
		"0123456789abcdef": "01234567",
		"web":              "web",
		"exact888":         "exact888",
		"":                 "",
	}
	for in, want := range cases {
		if got := shortID(in); got != want {
			t.Fatalf("shortID(%q) = %q, want %q", in, got, want)
		}
	}
}
