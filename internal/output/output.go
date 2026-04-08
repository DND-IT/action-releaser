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
		// Multi-line: use heredoc delimiter.
		delim := fmt.Sprintf("RELEASER_EOF_%d", os.Getpid())
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
