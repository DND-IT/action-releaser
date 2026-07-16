package output

import (
	"fmt"
	"os"
	"strings"
)

// Set writes a GitHub Actions output variable.
// Uses the $GITHUB_OUTPUT file mechanism.
func Set(name, value string) error {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		// Not running in GitHub Actions — print to stdout for local testing.
		fmt.Printf("::set-output name=%s::%s\n", name, value)
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open GITHUB_OUTPUT: %w", err)
	}

	if strings.Contains(value, "\n") {
		// Multi-line: use a heredoc delimiter. The delimiter must not appear as
		// a line in the value, otherwise GitHub's parser ends the heredoc early
		// and mis-reads everything after it. Extend the delimiter until it is
		// unique so arbitrary changelog content can never break parsing.
		delim := heredocDelim(value)
		_, err = fmt.Fprintf(f, "%s<<%s\n%s\n%s\n", name, delim, value, delim)
	} else {
		_, err = fmt.Fprintf(f, "%s=%s\n", name, value)
	}
	if err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// heredocDelim returns a delimiter guaranteed not to appear as a line in value,
// so the heredoc written to $GITHUB_OUTPUT is always well-formed.
func heredocDelim(value string) string {
	base := fmt.Sprintf("RELEASER_EOF_%d", os.Getpid())
	delim := base
	for i := 0; containsLine(value, delim); i++ {
		delim = fmt.Sprintf("%s_%d", base, i)
	}
	return delim
}

// containsLine reports whether s has a line exactly equal to line.
func containsLine(s, line string) bool {
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimRight(l, "\r") == line {
			return true
		}
	}
	return false
}
