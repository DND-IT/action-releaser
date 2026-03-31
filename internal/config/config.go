package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the merged configuration from .release.yml and action inputs.
type Config struct {
	VersionStrategy string    `yaml:"version-strategy"`
	TagPrefix       string    `yaml:"tag-prefix"`
	CliffConfig     string    `yaml:"cliff-config"`
	ReleaseMode     string    `yaml:"release-mode"`
	Draft           bool      `yaml:"draft"`
	Prerelease      bool      `yaml:"prerelease"`
	DryRun          bool      `yaml:"-"`
	GithubToken     string    `yaml:"-"`
	IncludePath     string    `yaml:"-"`
	Packages        []Package `yaml:"packages"`

	// Set per-package during monorepo iteration.
	CurrentPackage *Package `yaml:"-"`

	// EffectiveTagPattern is a regex for git-cliff --tag-pattern, computed at
	// runtime from TagPrefix + VersionStrategy. Scopes git-cliff to only see
	// tags belonging to this service.
	EffectiveTagPattern string `yaml:"-"`
}

// Package defines a monorepo package with its own release scope.
type Package struct {
	Name       string `yaml:"name"`
	Path       string `yaml:"path"`
	TagPattern string `yaml:"tag-pattern"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		VersionStrategy: "semver",
		TagPrefix:       "v",
		ReleaseMode:     "direct",
	}
}

// Load reads .release.yml (if present), then overlays action inputs (env vars).
// Action inputs always win when explicitly set.
//
// Priority: action inputs > .release.yml > defaults
func Load() (Config, error) {
	cfg := DefaultConfig()

	// Read .release.yml if it exists.
	data, err := os.ReadFile(".release.yml")
	if err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse .release.yml: %w", err)
		}
		// Strict mode: re-decode to check for unknown fields.
		strict := struct {
			VersionStrategy string    `yaml:"version-strategy"`
			TagPrefix       string    `yaml:"tag-prefix"`
			CliffConfig     string    `yaml:"cliff-config"`
			ReleaseMode     string    `yaml:"release-mode"`
			Draft           bool      `yaml:"draft"`
			Prerelease      bool      `yaml:"prerelease"`
			Packages        []Package `yaml:"packages"`
		}{}
		dec := yaml.NewDecoder(strings.NewReader(string(data)))
		dec.KnownFields(true)
		if err := dec.Decode(&strict); err != nil {
			return Config{}, fmt.Errorf("parse .release.yml: unknown field: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("read .release.yml: %w", err)
	}

	// Overlay action inputs from environment variables.
	// Docker actions receive INPUT_VERSION-STRATEGY (hyphens), not INPUT_VERSION_STRATEGY (underscores).
	// Check both forms for compatibility.
	if v := getInput("VERSION-STRATEGY", "VERSION_STRATEGY"); v != "" {
		cfg.VersionStrategy = v
	}
	if v := getInput("TAG-PREFIX", "TAG_PREFIX"); v != "" {
		cfg.TagPrefix = v
	}
	if v := getInput("CLIFF-CONFIG", "CLIFF_CONFIG"); v != "" {
		cfg.CliffConfig = v
	}
	if v := getInput("DRAFT"); strings.EqualFold(v, "true") {
		cfg.Draft = true
	}
	if v := getInput("PRERELEASE"); strings.EqualFold(v, "true") {
		cfg.Prerelease = true
	}
	if v := getInput("RELEASE-MODE", "RELEASE_MODE"); v != "" {
		cfg.ReleaseMode = v
	}
	if v := getInput("DRY-RUN", "DRY_RUN"); strings.EqualFold(v, "true") {
		cfg.DryRun = true
	}
	if v := getInput("INCLUDE-PATH", "INCLUDE_PATH"); v != "" {
		cfg.IncludePath = v
	}
	cfg.GithubToken = getInput("GITHUB-TOKEN", "GITHUB_TOKEN")
	if cfg.GithubToken == "" {
		cfg.GithubToken = os.Getenv("GITHUB_TOKEN")
	}

	// Validate strategy.
	switch cfg.VersionStrategy {
	case "semver", "date-rolling", "numeric-rolling":
		// ok
	default:
		return Config{}, fmt.Errorf("unknown version-strategy %q: use semver, date-rolling, or numeric-rolling", cfg.VersionStrategy)
	}

	// Validate release mode.
	switch cfg.ReleaseMode {
	case "direct", "pr":
		// ok
	default:
		return Config{}, fmt.Errorf("unknown release-mode %q: use direct or pr", cfg.ReleaseMode)
	}

	return cfg, nil
}

// getInput reads a GitHub Actions input, checking multiple env var forms.
// Docker actions receive INPUT_VERSION-STRATEGY (hyphens preserved),
// while JS actions receive INPUT_VERSION_STRATEGY (hyphens→underscores).
func getInput(names ...string) string {
	for _, name := range names {
		if v := os.Getenv("INPUT_" + name); v != "" {
			return v
		}
	}
	return ""
}
