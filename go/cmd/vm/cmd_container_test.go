package vm

import (
	"testing"

	core "dappco.re/go"
)

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

func TestCmdContainer_parsePublish_Good(t *testing.T) {
	r := parsePublish([]string{"8080:80", "127.0.0.1:5432:5432/tcp"})
	if !r.OK {
		t.Fatal(r.Error())
	}
	got := core.MustCast[map[int]int](r)
	if got[8080] != 80 || got[5432] != 5432 {
		t.Fatalf("parsePublish => %v, want 8080->80, 5432->5432", got)
	}
}

func TestCmdContainer_parsePublish_Bad(t *testing.T) {
	if parsePublish([]string{"8080"}).OK {
		t.Fatal("expected error for missing colon")
	}
	if parsePublish([]string{"http:80"}).OK {
		t.Fatal("expected error for non-numeric host port")
	}
}

func TestCmdContainer_parseVolumes_Good(t *testing.T) {
	r := parseVolumes([]string{"/data:/app", "./cfg:/etc/app"})
	if !r.OK {
		t.Fatal(r.Error())
	}
	got := core.MustCast[map[string]string](r)
	if got["/data"] != "/app" || got["./cfg"] != "/etc/app" {
		t.Fatalf("parseVolumes => %v", got)
	}
}

func TestCmdContainer_parseVolumes_Bad(t *testing.T) {
	if parseVolumes([]string{"/data"}).OK {
		t.Fatal("expected error for missing colon")
	}
}

func TestCmdContainer_parseEnv_Good(t *testing.T) {
	r := parseEnv([]string{"FOO=bar", "URL=https://x?a=b", "EMPTY="})
	if !r.OK {
		t.Fatal(r.Error())
	}
	got := core.MustCast[[]string](r)
	if len(got) != 3 || got[0] != "FOO=bar" || got[2] != "EMPTY=" {
		t.Fatalf("parseEnv => %v", got)
	}
}

func TestCmdContainer_parseEnv_Bad(t *testing.T) {
	if parseEnv([]string{"NOEQUALS"}).OK {
		t.Fatal("expected error for missing '='")
	}
}

func TestCmdContainer_killContainer_Bad(t *testing.T) {
	if killContainer("").OK {
		t.Fatal("expected error for empty id")
	}
}

func TestCmdContainer_removeContainer_Bad(t *testing.T) {
	if removeContainer("").OK {
		t.Fatal("expected error for empty id")
	}
}

func TestCmdContainer_inspectContainer_Bad(t *testing.T) {
	if inspectContainer("").OK {
		t.Fatal("expected error for empty id")
	}
}

func TestCmdContainer_shellContainer_Bad(t *testing.T) {
	if shellContainer("", nil).OK {
		t.Fatal("expected error for empty id")
	}
}

func TestCmdContainer_wantInteractive_Good(t *testing.T) {
	// Any of -i/--interactive/-t/--tty routes exec through the TTY path; the
	// short forms parse to single-char keys (i, t), the long forms to full names.
	for _, key := range []string{"i", "interactive", "t", "tty"} {
		on := core.NewOptions(core.Option{Key: key, Value: true})
		if !wantInteractive(on) {
			t.Fatalf("flag %q should request the interactive path", key)
		}
	}
	if wantInteractive(core.NewOptions()) {
		t.Fatal("no flags should not request the interactive path")
	}
}
