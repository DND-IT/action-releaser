//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binary returns the path to the built action-releaser binary.
// Build it before running: go build -o action-releaser ./cmd/action-releaser
func binary(t *testing.T) string {
	t.Helper()
	// Check repo root first, then PATH.
	candidates := []string{
		filepath.Join(repoRoot(t), "action-releaser"),
		"action-releaser",
	}
	for _, p := range candidates {
		if _, err := exec.LookPath(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	t.Fatal("action-releaser binary not found — run: go build -o action-releaser ./cmd/action-releaser")
	return ""
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// integration/ is one level deep.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(dir)
}

// setupRepo creates a temp git repo and returns its path.
func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init", "-q")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	return dir
}

func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %s\n%s", name, args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func commit(t *testing.T, dir, msg string) {
	t.Helper()
	run(t, dir, "git", "commit", "--allow-empty", "-m", msg)
}

func tag(t *testing.T, dir, name string) {
	t.Helper()
	run(t, dir, "git", "tag", "-a", name, "-m", name)
}

// runReleaser runs the binary in the given dir with env vars, returns stdout+stderr.
func runReleaser(t *testing.T, dir string, env map[string]string) (string, error) {
	t.Helper()
	outputFile := filepath.Join(t.TempDir(), "github-output")
	os.WriteFile(outputFile, nil, 0644)

	bin := binary(t)
	cmd := exec.Command(bin)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GITHUB_OUTPUT="+outputFile,
		"CLIFF_TEMPLATES_DIR="+filepath.Join(repoRoot(t), "cliff-templates"),
	)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	out, err := cmd.CombinedOutput()
	return string(out), err
}

func readOutput(t *testing.T, dir string, env map[string]string) map[string]string {
	t.Helper()
	outputFile := filepath.Join(t.TempDir(), "github-output")
	os.WriteFile(outputFile, nil, 0644)

	bin := binary(t)
	cmd := exec.Command(bin)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GITHUB_OUTPUT="+outputFile,
		"CLIFF_TEMPLATES_DIR="+filepath.Join(repoRoot(t), "cliff-templates"),
	)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	cmd.CombinedOutput() // ignore error — we read outputs regardless

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	outputs := map[string]string{}
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if idx := strings.Index(line, "="); idx > 0 {
			outputs[line[:idx]] = line[idx+1:]
		} else if idx := strings.Index(line, "<<"); idx > 0 {
			// Heredoc multi-line value.
			key := line[:idx]
			delim := line[idx+2:]
			var val []string
			for i++; i < len(lines); i++ {
				if lines[i] == delim {
					break
				}
				val = append(val, lines[i])
			}
			outputs[key] = strings.Join(val, "\n")
		}
	}
	return outputs
}

func TestSemverRelease(t *testing.T) {
	dir := setupRepo(t)
	commit(t, dir, "feat: initial feature")
	tag(t, dir, "v0.1.0")
	commit(t, dir, "feat: add new feature")
	commit(t, dir, "fix: correct a bug")

	outputs := readOutput(t, dir, map[string]string{
		"INPUT_VERSION_STRATEGY": "semver",
		"INPUT_TAG_PREFIX":       "v",
		"INPUT_DRY_RUN":          "true",
	})

	if outputs["skipped"] != "false" {
		t.Fatalf("expected skipped=false, got %q", outputs["skipped"])
	}
	if outputs["version"] == "" {
		t.Fatal("expected non-empty version")
	}
	if outputs["previous-version"] != "v0.1.0" {
		t.Errorf("previous-version = %q, want v0.1.0", outputs["previous-version"])
	}
	if outputs["changelog"] == "" {
		t.Error("expected non-empty changelog")
	}
}

func TestDateRollingRelease(t *testing.T) {
	dir := setupRepo(t)
	commit(t, dir, "some work")

	outputs := readOutput(t, dir, map[string]string{
		"INPUT_VERSION_STRATEGY": "date-rolling",
		"INPUT_TAG_PREFIX":       "v",
		"INPUT_DRY_RUN":          "true",
	})

	if outputs["skipped"] != "false" {
		t.Fatalf("expected skipped=false, got %q", outputs["skipped"])
	}
	if !strings.Contains(outputs["version"], ".") {
		t.Errorf("expected date format YYYY.MM.DD, got %q", outputs["version"])
	}
}

func TestNumericRollingRelease(t *testing.T) {
	dir := setupRepo(t)
	commit(t, dir, "init")
	tag(t, dir, "v1")
	commit(t, dir, "more work")

	outputs := readOutput(t, dir, map[string]string{
		"INPUT_VERSION_STRATEGY": "numeric-rolling",
		"INPUT_TAG_PREFIX":       "v",
		"INPUT_DRY_RUN":          "true",
	})

	if outputs["version"] != "2" {
		t.Errorf("version = %q, want 2", outputs["version"])
	}
	if outputs["previous-version"] != "v1" {
		t.Errorf("previous-version = %q, want v1", outputs["previous-version"])
	}
}

func TestDryRun(t *testing.T) {
	dir := setupRepo(t)
	commit(t, dir, "feat: initial")
	tag(t, dir, "v1.0.0")
	commit(t, dir, "feat: new thing")

	outputs := readOutput(t, dir, map[string]string{
		"INPUT_VERSION_STRATEGY": "semver",
		"INPUT_TAG_PREFIX":       "v",
		"INPUT_DRY_RUN":          "true",
	})

	if outputs["dry-run"] != "true" {
		t.Errorf("dry-run = %q, want true", outputs["dry-run"])
	}
	// Tag and release-url should be empty in dry-run.
	if outputs["tag"] != "" {
		t.Errorf("tag should be empty in dry-run, got %q", outputs["tag"])
	}
	if outputs["release-url"] != "" {
		t.Errorf("release-url should be empty in dry-run, got %q", outputs["release-url"])
	}
	// But version and changelog should be populated.
	if outputs["version"] == "" {
		t.Error("expected non-empty version in dry-run")
	}
}

func TestBootstrap(t *testing.T) {
	dir := setupRepo(t)
	commit(t, dir, "feat: first feature ever")

	outputs := readOutput(t, dir, map[string]string{
		"INPUT_VERSION_STRATEGY": "semver",
		"INPUT_TAG_PREFIX":       "v",
		"INPUT_DRY_RUN":          "true",
	})

	if outputs["version"] != "0.1.0" {
		t.Errorf("version = %q, want 0.1.0", outputs["version"])
	}
	if outputs["previous-version"] != "" {
		t.Errorf("previous-version should be empty on bootstrap, got %q", outputs["previous-version"])
	}
}

func TestSkipNoConventionalCommits(t *testing.T) {
	dir := setupRepo(t)
	commit(t, dir, "feat: initial")
	tag(t, dir, "v1.0.0")
	commit(t, dir, "wip: not conventional")

	outputs := readOutput(t, dir, map[string]string{
		"INPUT_VERSION_STRATEGY": "semver",
		"INPUT_TAG_PREFIX":       "v",
		"INPUT_DRY_RUN":          "true",
	})

	if outputs["skipped"] != "true" {
		t.Fatalf("expected skipped=true, got %q", outputs["skipped"])
	}
}

func TestShallowClone(t *testing.T) {
	// Create an origin repo.
	origin := setupRepo(t)
	commit(t, origin, "feat: init")

	// Shallow clone it.
	shallow := filepath.Join(t.TempDir(), "shallow")
	cmd := exec.Command("git", "clone", "--depth", "1", "file://"+origin, shallow)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("shallow clone failed: %s\n%s", err, out)
	}

	_, err := runReleaser(t, shallow, map[string]string{
		"INPUT_VERSION_STRATEGY": "semver",
		"INPUT_TAG_PREFIX":       "v",
		"INPUT_DRY_RUN":          "true",
	})
	if err == nil {
		t.Fatal("expected error for shallow clone")
	}
}

func TestConfigFile(t *testing.T) {
	dir := setupRepo(t)
	os.WriteFile(filepath.Join(dir, ".release.yml"), []byte(`
version-strategy: numeric-rolling
tag-prefix: "build-"
`), 0644)
	commit(t, dir, "init")

	outputs := readOutput(t, dir, map[string]string{
		"INPUT_DRY_RUN": "true",
	})

	if outputs["version"] != "1" {
		t.Errorf("version = %q, want 1 (numeric-rolling from config)", outputs["version"])
	}
}
