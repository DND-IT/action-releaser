package output

import (
	"fmt"
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

// TestSet_MultiLineContainingDelimiter guards the gated-release regression
// where a changelog line matching the heredoc delimiter would terminate the
// heredoc early, corrupting every output written afterwards (e.g.
// release-published). The delimiter must be extended so it never appears as a
// line in the value, keeping the heredoc well-formed.
func TestSet_MultiLineContainingDelimiter(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "github-output-*")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", f.Name())

	base := fmt.Sprintf("RELEASER_EOF_%d", os.Getpid())
	value := "line1\n" + base + "\nline3"
	if err := Set("changelog", value); err != nil {
		t.Fatal(err)
	}
	// A scalar written after the multiline value must remain parseable.
	if err := Set("release-published", "true"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(f.Name())
	outputs := parseGithubOutput(t, string(data))
	if outputs["changelog"] != value {
		t.Errorf("changelog round-trip mismatch:\n got %q\nwant %q", outputs["changelog"], value)
	}
	if outputs["release-published"] != "true" {
		t.Errorf("release-published = %q, want \"true\" (delimiter collision corrupted later outputs)", outputs["release-published"])
	}
}

// parseGithubOutput parses a $GITHUB_OUTPUT file the way the Actions runner
// does: `name=value` lines and `name<<DELIM ... DELIM` heredoc blocks.
func parseGithubOutput(t *testing.T, content string) map[string]string {
	t.Helper()
	out := map[string]string{}
	lines := strings.Split(content, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if idx := strings.Index(line, "<<"); idx > 0 && !strings.Contains(line[:idx], "=") {
			name := line[:idx]
			delim := line[idx+2:]
			var body []string
			i++
			for ; i < len(lines); i++ {
				if lines[i] == delim {
					break
				}
				body = append(body, lines[i])
			}
			if i == len(lines) {
				t.Fatalf("heredoc for %q never closed with delimiter %q", name, delim)
			}
			out[name] = strings.Join(body, "\n")
			continue
		}
		if eq := strings.Index(line, "="); eq > 0 {
			out[line[:eq]] = line[eq+1:]
		}
	}
	return out
}

func TestSet_NoGithubOutput(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", "")

	// Should not error — falls back to stdout.
	if err := Set("version", "1.0.0"); err != nil {
		t.Fatal(err)
	}
}
