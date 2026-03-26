package app

import (
	"errors"
	"testing"
)

func TestShellNameFromCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
		ok      bool
	}{
		{name: "bash path", command: "/bin/bash", want: "bash", ok: true},
		{name: "zsh login", command: "-zsh", want: "zsh", ok: true},
		{name: "fish plain", command: "fish", want: "fish", ok: true},
		{name: "unknown", command: "python", want: "", ok: false},
		{name: "empty", command: "", want: "", ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := shellNameFromCommand(tc.command)
			if ok != tc.ok {
				t.Fatalf("ok = %t, want %t", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("shell = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseShellName(t *testing.T) {
	if got, ok := parseShellName("bash"); !ok || got != "bash" {
		t.Fatalf("parseShellName(bash) = (%q, %t)", got, ok)
	}
	if _, ok := parseShellName("pwsh"); ok {
		t.Fatal("expected unsupported shell to fail")
	}
}

func TestParsePSProcessInfo(t *testing.T) {
	ppid, cmd, err := parsePSProcessInfo("  123 /bin/zsh\n")
	if err != nil {
		t.Fatalf("parsePSProcessInfo() error = %v", err)
	}
	if ppid != 123 {
		t.Fatalf("ppid = %d, want 123", ppid)
	}
	if cmd != "/bin/zsh" {
		t.Fatalf("cmd = %q, want /bin/zsh", cmd)
	}
}

func TestDetectShellFromProcessChainFindsAncestorShell(t *testing.T) {
	lookup := func(pid int) (int, string, error) {
		switch pid {
		case 400:
			return 300, "starship", nil
		case 300:
			return 200, "/bin/bash", nil
		default:
			return 1, "", nil
		}
	}

	shell, err := detectShellFromProcessChain(400, lookup, 10)
	if err != nil {
		t.Fatalf("detectShellFromProcessChain() error = %v", err)
	}
	if shell != "bash" {
		t.Fatalf("shell = %q, want bash", shell)
	}
}

func TestDetectShellFromProcessChainErrorsWhenNotFound(t *testing.T) {
	lookup := func(pid int) (int, string, error) {
		switch pid {
		case 400:
			return 300, "python", nil
		case 300:
			return 1, "launchd", nil
		default:
			return 1, "", nil
		}
	}

	if _, err := detectShellFromProcessChain(400, lookup, 10); err == nil {
		t.Fatal("expected error when no shell found")
	}
}

func TestDetectShellFromProcessChainPropagatesLookupError(t *testing.T) {
	lookup := func(int) (int, string, error) {
		return 0, "", errors.New("boom")
	}

	if _, err := detectShellFromProcessChain(400, lookup, 10); err == nil {
		t.Fatal("expected lookup error")
	}
}
