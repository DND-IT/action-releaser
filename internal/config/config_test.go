package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.VersionStrategy != "semver" {
		t.Errorf("default strategy = %q, want semver", cfg.VersionStrategy)
	}
	if cfg.TagPrefix != "v" {
		t.Errorf("default prefix = %q, want v", cfg.TagPrefix)
	}
}

func TestLoad_NoFile(t *testing.T) {
	// Run in a temp dir with no .release.yml.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	// Clear env.
	for _, k := range []string{"INPUT_VERSION_STRATEGY", "INPUT_TAG_PREFIX", "INPUT_CLIFF_CONFIG", "INPUT_DRAFT", "INPUT_PRERELEASE", "INPUT_DRY_RUN", "INPUT_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.VersionStrategy != "semver" {
		t.Errorf("strategy = %q, want semver", cfg.VersionStrategy)
	}
}

func TestLoad_WithFile(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, ".release.yml"), []byte(`
version-strategy: date-rolling
tag-prefix: "release-"
draft: true
`), 0644)

	for _, k := range []string{"INPUT_VERSION_STRATEGY", "INPUT_TAG_PREFIX", "INPUT_CLIFF_CONFIG", "INPUT_DRAFT", "INPUT_PRERELEASE", "INPUT_DRY_RUN", "INPUT_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.VersionStrategy != "date-rolling" {
		t.Errorf("strategy = %q, want date-rolling", cfg.VersionStrategy)
	}
	if cfg.TagPrefix != "release-" {
		t.Errorf("prefix = %q, want release-", cfg.TagPrefix)
	}
	if !cfg.Draft {
		t.Error("draft should be true")
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, ".release.yml"), []byte(`
version-strategy: date-rolling
tag-prefix: "release-"
`), 0644)

	t.Setenv("INPUT_VERSION_STRATEGY", "numeric-rolling")
	t.Setenv("INPUT_TAG_PREFIX", "build-")
	for _, k := range []string{"INPUT_CLIFF_CONFIG", "INPUT_DRAFT", "INPUT_PRERELEASE", "INPUT_DRY_RUN", "INPUT_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.VersionStrategy != "numeric-rolling" {
		t.Errorf("strategy = %q, want numeric-rolling (env override)", cfg.VersionStrategy)
	}
	if cfg.TagPrefix != "build-" {
		t.Errorf("prefix = %q, want build- (env override)", cfg.TagPrefix)
	}
}

func TestLoad_InvalidStrategy(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	t.Setenv("INPUT_VERSION_STRATEGY", "invalid")
	for _, k := range []string{"INPUT_TAG_PREFIX", "INPUT_CLIFF_CONFIG", "INPUT_DRAFT", "INPUT_PRERELEASE", "INPUT_DRY_RUN", "INPUT_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(k, "")
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid strategy")
	}
}

func TestLoad_UnknownFields(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, ".release.yml"), []byte(`
version-strategy: semver
unknown-field: "oops"
`), 0644)

	for _, k := range []string{"INPUT_VERSION_STRATEGY", "INPUT_TAG_PREFIX", "INPUT_CLIFF_CONFIG", "INPUT_DRAFT", "INPUT_PRERELEASE", "INPUT_DRY_RUN", "INPUT_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(k, "")
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for unknown field (strict mode)")
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, ".release.yml"), []byte(`{{{not yaml`), 0644)

	for _, k := range []string{"INPUT_VERSION_STRATEGY", "INPUT_TAG_PREFIX", "INPUT_CLIFF_CONFIG", "INPUT_DRAFT", "INPUT_PRERELEASE", "INPUT_DRY_RUN", "INPUT_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(k, "")
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestLoad_DryRun(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	t.Setenv("INPUT_DRY_RUN", "true")
	for _, k := range []string{"INPUT_VERSION_STRATEGY", "INPUT_TAG_PREFIX", "INPUT_CLIFF_CONFIG", "INPUT_DRAFT", "INPUT_PRERELEASE", "INPUT_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.DryRun {
		t.Error("dry-run should be true")
	}
}

func TestLoad_Packages(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	os.WriteFile(filepath.Join(dir, ".release.yml"), []byte(`
version-strategy: semver
packages:
  - name: api
    path: services/api
    tag-pattern: "api/v*"
  - name: web
    path: services/web
    tag-pattern: "web/v*"
`), 0644)

	for _, k := range []string{"INPUT_VERSION_STRATEGY", "INPUT_TAG_PREFIX", "INPUT_CLIFF_CONFIG", "INPUT_DRAFT", "INPUT_PRERELEASE", "INPUT_DRY_RUN", "INPUT_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Packages) != 2 {
		t.Fatalf("packages = %d, want 2", len(cfg.Packages))
	}
	if cfg.Packages[0].Name != "api" {
		t.Errorf("packages[0].Name = %q, want api", cfg.Packages[0].Name)
	}
	if cfg.Packages[1].Path != "services/web" {
		t.Errorf("packages[1].Path = %q, want services/web", cfg.Packages[1].Path)
	}
}
