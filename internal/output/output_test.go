package output

import (
	"os"
	"strings"
	"testing"
)

func TestSet_SingleLine(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "github-output-*")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", f.Name())

	if err := Set("version", "1.2.3"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(f.Name())
	if got := strings.TrimSpace(string(data)); got != "version=1.2.3" {
		t.Errorf("got %q, want version=1.2.3", got)
	}
}

func TestSet_MultiLine(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "github-output-*")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", f.Name())

	if err := Set("changelog", "line1\nline2\nline3"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(f.Name())
	content := string(data)
	if !strings.Contains(content, "changelog<<RELEASER_EOF_") {
		t.Errorf("expected heredoc delimiter, got %q", content)
	}
	if !strings.Contains(content, "line1\nline2\nline3") {
		t.Errorf("expected multi-line content, got %q", content)
	}
}

func TestSet_NoGithubOutput(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", "")

	// Should not error — falls back to stdout.
	if err := Set("version", "1.0.0"); err != nil {
		t.Fatal(err)
	}
}
