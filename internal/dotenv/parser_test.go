package dotenv

import (
	"strings"
	"testing"
)

func TestParseBasic(t *testing.T) {
	input := strings.NewReader("FOO=bar\nexport BAZ=\"x y\"\nQUX='z'\n")
	vals, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if vals["FOO"] != "bar" {
		t.Fatalf("FOO = %q, want %q", vals["FOO"], "bar")
	}
	if vals["BAZ"] != "x y" {
		t.Fatalf("BAZ = %q, want %q", vals["BAZ"], "x y")
	}
	if vals["QUX"] != "z" {
		t.Fatalf("QUX = %q, want %q", vals["QUX"], "z")
	}
}

func TestParseInvalidKey(t *testing.T) {
	_, err := Parse(strings.NewReader("1BAD=value\n"))
	if err == nil {
		t.Fatal("expected parse error")
	}
}
