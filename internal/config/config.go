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
	Draft           bool      `yaml:"draft"`
	Prerelease      bool      `yaml:"prerelease"`
	DryRun          bool      `yaml:"-"`
	GithubToken     string    `yaml:"-"`
	Packages        []Package `yaml:"packages"`

	// Set per-package during monorepo iteration.
	CurrentPackage *Package `yaml:"-"`
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
	if v := os.Getenv("INPUT_VERSION_STRATEGY"); v != "" {
		cfg.VersionStrategy = v
	}
	if v := os.Getenv("INPUT_TAG_PREFIX"); v != "" {
		cfg.TagPrefix = v
	}
	if v := os.Getenv("INPUT_CLIFF_CONFIG"); v != "" {
		cfg.CliffConfig = v
	}
	if v := os.Getenv("INPUT_DRAFT"); strings.EqualFold(v, "true") {
		cfg.Draft = true
	}
	if v := os.Getenv("INPUT_PRERELEASE"); strings.EqualFold(v, "true") {
		cfg.Prerelease = true
	}
	if v := os.Getenv("INPUT_DRY_RUN"); strings.EqualFold(v, "true") {
		cfg.DryRun = true
	}
	cfg.GithubToken = os.Getenv("INPUT_GITHUB_TOKEN")
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

	return cfg, nil
}
