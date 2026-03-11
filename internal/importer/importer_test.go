package importer

import "testing"

func TestParseAssignmentsAllAcceptsExportLines(t *testing.T) {
	content := "export FOO=bar\nBAR='baz'\n"
	values, issues, err := ParseAssignmentsAll(content)
	if err != nil {
		t.Fatalf("ParseAssignmentsAll() error = %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %d, want 0", len(issues))
	}
	if values["FOO"] != "bar" {
		t.Fatalf("FOO = %q", values["FOO"])
	}
	if values["BAR"] != "baz" {
		t.Fatalf("BAR = %q", values["BAR"])
	}
}

func TestParseAssignmentsAllReportsAllIssues(t *testing.T) {
	content := "export GOOD=yes\nlayout go\nif true; then\nBAD-KEY=x\n"
	values, issues, err := ParseAssignmentsAll(content)
	if err != nil {
		t.Fatalf("ParseAssignmentsAll() error = %v", err)
	}
	if values != nil {
		t.Fatal("expected nil values when issues are present")
	}
	if len(issues) != 3 {
		t.Fatalf("issues = %d, want 3", len(issues))
	}
	if issues[0].Line != 2 {
		t.Fatalf("issues[0].Line = %d, want 2", issues[0].Line)
	}
	if issues[1].Line != 3 {
		t.Fatalf("issues[1].Line = %d, want 3", issues[1].Line)
	}
	if issues[2].Line != 4 {
		t.Fatalf("issues[2].Line = %d, want 4", issues[2].Line)
	}
}
