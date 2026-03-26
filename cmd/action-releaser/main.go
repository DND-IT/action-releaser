package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/dnd-it/action-releaser/internal/changelog"
	"github.com/dnd-it/action-releaser/internal/config"
	"github.com/dnd-it/action-releaser/internal/gitutil"
	"github.com/dnd-it/action-releaser/internal/output"
	"github.com/dnd-it/action-releaser/internal/release"
	"github.com/dnd-it/action-releaser/internal/strategy"
)

func main() {
	log.SetPrefix("[releaser] ")
	log.SetFlags(0)

	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	// 1. Load config.
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log.Printf("strategy=%s tag-prefix=%q dry-run=%v", cfg.VersionStrategy, cfg.TagPrefix, cfg.DryRun)

	// 2. Shallow-clone guard.
	if err := gitutil.CheckShallowClone(); err != nil {
		return err
	}

	// 3. Resolve strategy.
	strat, err := strategy.New(cfg.VersionStrategy)
	if err != nil {
		return err
	}

	// 4. Determine packages to release.
	packages := cfg.Packages
	if len(packages) == 0 {
		// Single-repo mode: one implicit package.
		packages = []config.Package{{Name: ""}}
	}

	// 5. Process each package.
	var (
		hadFailure bool
		hadRelease bool
	)
	for _, pkg := range packages {
		pkgCfg := cfg
		if pkg.Name != "" {
			log.Printf("--- package: %s (path=%s)", pkg.Name, pkg.Path)
			pkgCfg.CurrentPackage = &pkg
		}

		if err := releasePackage(pkgCfg, strat, pkg); err != nil {
			if pkg.Name != "" {
				log.Printf("error releasing package %s: %v", pkg.Name, err)
				hadFailure = true
				continue
			}
			return err // Single-repo: fail immediately.
		}
		hadRelease = true
	}

	if hadFailure {
		if hadRelease {
			log.Printf("partial failure: some packages released, some failed")
			os.Exit(2) // Non-zero but distinct from fatal error (1).
		}
		return fmt.Errorf("all packages failed to release")
	}

	return nil
}

func releasePackage(cfg config.Config, strat strategy.VersionStrategy, pkg config.Package) error {
	// List tags.
	prefix := cfg.TagPrefix
	if pkg.TagPattern != "" {
		prefix = pkg.TagPattern
	}
	tags, err := gitutil.ListTags(prefix)
	if err != nil {
		return err
	}
	log.Printf("found %d tags matching prefix %q", len(tags), prefix)

	// Calculate next version.
	result, err := strat.NextVersion(tags, cfg)
	if err != nil {
		return fmt.Errorf("calculate version: %w", err)
	}

	if result.Skipped {
		log.Printf("skipped: no release needed")
		return setOutputs("", "", "", "", result.PreviousVersion, true, cfg.DryRun)
	}

	tag := cfg.TagPrefix + result.Version
	if pkg.Name != "" {
		tag = pkg.Name + "/" + tag
	}
	log.Printf("next version: %s (tag: %s, previous: %s)", result.Version, tag, result.PreviousVersion)

	// Generate changelog.
	cl, err := changelog.Generate(cfg)
	if err != nil {
		return fmt.Errorf("generate changelog: %w", err)
	}
	log.Printf("changelog: %d bytes", len(cl))

	// Dry-run: output version and changelog, skip tag/release.
	if cfg.DryRun {
		log.Printf("dry-run: skipping tag creation and release")
		return setOutputs(result.Version, cl, "", "", result.PreviousVersion, false, true)
	}

	// Check for tag conflict.
	exists, err := gitutil.TagExists(tag)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("tag %s already exists: a concurrent release may have created it", tag)
	}

	// Create and push tag.
	if err := gitutil.CreateTag(tag, fmt.Sprintf("Release %s", tag)); err != nil {
		return err
	}
	if err := gitutil.PushTag(tag); err != nil {
		return err
	}
	log.Printf("tag %s created and pushed", tag)

	// Create GitHub release.
	owner, repo, err := release.OwnerRepoFromEnv()
	if err != nil {
		return err
	}

	releaseName := tag
	if pkg.Name != "" {
		releaseName = fmt.Sprintf("%s %s", pkg.Name, result.Version)
	}

	res, err := release.Create(context.Background(), release.Params{
		Owner:      owner,
		Repo:       repo,
		Tag:        tag,
		Name:       releaseName,
		Body:       cl,
		Draft:      cfg.Draft,
		Prerelease: cfg.Prerelease,
		Token:      cfg.GithubToken,
	})
	if err != nil {
		return fmt.Errorf("create release: %w", err)
	}
	log.Printf("release created: %s", res.URL)

	return setOutputs(result.Version, cl, res.Tag, res.URL, result.PreviousVersion, false, false)
}

func setOutputs(version, changelogText, tag, releaseURL, previousVersion string, skipped, dryRun bool) error {
	pairs := []struct{ name, value string }{
		{"version", version},
		{"changelog", changelogText},
		{"tag", tag},
		{"release-url", releaseURL},
		{"previous-version", previousVersion},
		{"skipped", boolStr(skipped)},
		{"dry-run", boolStr(dryRun)},
	}
	for _, p := range pairs {
		if err := output.Set(p.name, p.value); err != nil {
			return fmt.Errorf("set output %s: %w", p.name, err)
		}
	}
	return nil
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
