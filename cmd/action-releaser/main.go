package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/google/go-github/v68/github"

	"github.com/dnd-it/action-releaser/internal/changelog"
	"github.com/dnd-it/action-releaser/internal/config"
	"github.com/dnd-it/action-releaser/internal/gitutil"
	"github.com/dnd-it/action-releaser/internal/output"
	"github.com/dnd-it/action-releaser/internal/release"
	"github.com/dnd-it/action-releaser/internal/releasepr"
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
	log.Printf("strategy=%s tag-prefix=%q release-mode=%s dry-run=%v",
		cfg.VersionStrategy, cfg.TagPrefix, cfg.ReleaseMode, cfg.DryRun)

	// 2. Configure git auth for push operations.
	gitutil.ConfigureAuth(cfg.GithubToken)

	// 3. If release-mode is "pr", check for a merged release PR via the API.
	//    This works on any event type (push, pull_request, workflow_dispatch),
	//    matching how release-please operates.
	if cfg.ReleaseMode == "pr" {
		owner, repo, err := release.OwnerRepoFromEnv()
		if err != nil {
			return err
		}
		ghClient := github.NewClient(nil).WithAuthToken(cfg.GithubToken)
		prClient := releasepr.NewClient(ghClient, owner, repo)

		baseBranch := os.Getenv("GITHUB_REF_NAME")
		if baseBranch == "" {
			baseBranch = "main"
		}

		result, err := prClient.DetectMerge(context.Background(), baseBranch)
		if err != nil {
			return fmt.Errorf("detect merge: %w", err)
		}
		if result != nil {
			log.Printf("release PR merge detected: tag=%s version=%s", result.Manifest.Tag, result.Manifest.Version)
			return handleReleasePRMerge(cfg, prClient, result)
		}
		// No merged release PR found — fall through to create/update PR.
	}

	// 3. Shallow-clone guard.
	if err := gitutil.CheckShallowClone(); err != nil {
		return err
	}

	// 4. Resolve strategy.
	strat, err := strategy.New(cfg.VersionStrategy)
	if err != nil {
		return err
	}

	// 5. Determine packages to release.
	packages := cfg.Packages
	if len(packages) == 0 {
		packages = []config.Package{{Name: ""}}
	}

	// 6. Process each package.
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

		if err := processPackage(pkgCfg, strat, pkg); err != nil {
			if pkg.Name != "" {
				log.Printf("error releasing package %s: %v", pkg.Name, err)
				hadFailure = true
				continue
			}
			return err
		}
		hadRelease = true
	}

	if hadFailure {
		if hadRelease {
			log.Printf("partial failure: some packages released, some failed")
			os.Exit(2)
		}
		return fmt.Errorf("all packages failed to release")
	}

	return nil
}

func processPackage(cfg config.Config, strat strategy.VersionStrategy, pkg config.Package) error {
	// List tags.
	prefix := cfg.TagPrefix
	if pkg.TagPattern != "" {
		prefix = pkg.TagPattern
	}
	tags, err := gitutil.ListTags(prefix)
	if err != nil {
		return err
	}

	// Filter tags: after stripping the prefix, the remainder must be a valid
	// version for the strategy. This prevents cross-contamination in monorepos
	// where bad tags (e.g. python-api-vgo-service-v1.13.0) match the glob but
	// have garbage version suffixes.
	tags = strategy.FilterTags(tags, cfg.TagPrefix, cfg.VersionStrategy)
	log.Printf("found %d valid tags matching prefix %q", len(tags), prefix)

	// Compute effective tag pattern for git-cliff scoping.
	if pkg.TagPattern == "" && cfg.TagPrefix != "" {
		cfg.EffectiveTagPattern = strategy.TagPatternRegex(cfg.TagPrefix, cfg.VersionStrategy)
	}

	// Calculate next version.
	result, err := strat.NextVersion(tags, cfg)
	if err != nil {
		return fmt.Errorf("calculate version: %w", err)
	}

	if result.Skipped {
		log.Printf("skipped: no release needed")
		return setOutputs(actionOutputs{previousVersion: result.PreviousVersion, releaseMode: cfg.ReleaseMode, skipped: true, dryRun: cfg.DryRun})
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
		return setOutputs(actionOutputs{version: result.Version, changelog: cl, tag: tag, previousVersion: result.PreviousVersion, releaseMode: cfg.ReleaseMode, dryRun: true})
	}

	// Release PR mode: create/update PR instead of releasing directly.
	if cfg.ReleaseMode == "pr" {
		return createOrUpdateReleasePR(cfg, result.Version, tag, cl)
	}

	// Direct mode: create tag and release.
	return directRelease(cfg, result, tag, cl, pkg)
}

func createOrUpdateReleasePR(cfg config.Config, version, tag, cl string) error {
	owner, repo, err := release.OwnerRepoFromEnv()
	if err != nil {
		return err
	}

	ghClient := github.NewClient(nil).WithAuthToken(cfg.GithubToken)
	client := releasepr.NewClient(ghClient, owner, repo)

	baseBranch := os.Getenv("GITHUB_REF_NAME")
	if baseBranch == "" {
		baseBranch = "main"
	}

	prURL, prNumber, created, err := client.CreateOrUpdate(context.Background(), version, tag, cl, baseBranch, cfg.ServicePath())
	if err != nil {
		return fmt.Errorf("create/update release PR: %w", err)
	}

	return setOutputs(actionOutputs{
		version:         version,
		changelog:       cl,
		tag:             tag, // proposed tag (not yet created)
		prURL:           prURL,
		releaseMode:     "pr",
		releasePRNumber: prNumber,
		prCreated:       created,
	})
}

func handleReleasePRMerge(cfg config.Config, prClient *releasepr.Client, result *releasepr.MergeResult) error {
	tag := result.Manifest.Tag
	version := result.Manifest.Version

	// If manifest didn't have version, recalculate.
	if version == "" {
		log.Printf("manifest missing version, recalculating")
		if err := gitutil.CheckShallowClone(); err != nil {
			return err
		}
		strat, err := strategy.New(cfg.VersionStrategy)
		if err != nil {
			return err
		}
		tags, err := gitutil.ListTags(cfg.TagPrefix)
		if err != nil {
			return err
		}
		vResult, err := strat.NextVersion(tags, cfg)
		if err != nil {
			return err
		}
		if vResult.Skipped {
			log.Printf("skipped after recalculation")
			return nil
		}
		version = vResult.Version
		tag = cfg.TagPrefix + version
	}

	// Generate changelog for the release body.
	cl, err := changelog.Generate(cfg)
	if err != nil {
		log.Printf("warning: changelog generation failed: %v", err)
		cl = ""
	}

	// Check for tag conflict.
	exists, err := gitutil.TagExists(tag)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("tag %s already exists", tag)
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

	res, err := release.Create(context.Background(), release.Params{
		Owner:      owner,
		Repo:       repo,
		Tag:        tag,
		Name:       tag,
		Body:       cl,
		Draft:      cfg.Draft,
		Prerelease: cfg.Prerelease,
		Token:      cfg.GithubToken,
	})
	if err != nil {
		return fmt.Errorf("create release: %w", err)
	}
	log.Printf("release created: %s", res.URL)

	// Cleanup: swap labels, delete branch.
	branchName := releasepr.ReleaseBranchName(tag, version)
	if result.PRNumber > 0 {
		prClient.Cleanup(context.Background(), result.PRNumber, branchName)
	}

	return setOutputs(actionOutputs{
		version:          version,
		changelog:        cl,
		tag:              res.Tag,
		releaseURL:       res.URL,
		releaseMode:      "pr",
		releasePublished: true,
	})
}

func directRelease(cfg config.Config, result strategy.Result, tag, cl string, pkg config.Package) error {
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

	return setOutputs(actionOutputs{
		version:          result.Version,
		changelog:        cl,
		tag:              res.Tag,
		releaseURL:       res.URL,
		previousVersion:  result.PreviousVersion,
		releaseMode:      "direct",
		releasePublished: true,
	})
}

// actionOutputs holds all GitHub Actions output values for a single run.
type actionOutputs struct {
	version          string
	changelog        string
	tag              string // created tag (direct/merged) or proposed tag (pr mode, pending merge)
	releaseURL       string
	prURL            string
	previousVersion  string
	skipped          bool
	dryRun           bool
	releaseMode      string // "direct" or "pr"
	releasePublished bool   // true when a GitHub Release was actually created
	releasePRNumber  int    // non-zero when a release PR is open
	prCreated        bool   // true when a new PR was opened (vs updated)
}

func setOutputs(o actionOutputs) error {
	prNumberStr := ""
	if o.releasePRNumber > 0 {
		prNumberStr = strconv.Itoa(o.releasePRNumber)
	}
	pairs := []struct{ name, value string }{
		{"version", o.version},
		{"changelog", o.changelog},
		{"tag", o.tag},
		{"release-url", o.releaseURL},
		{"pr-url", o.prURL},
		{"previous-version", o.previousVersion},
		{"skipped", boolStr(o.skipped)},
		{"dry-run", boolStr(o.dryRun)},
		{"release-mode", o.releaseMode},
		{"release-published", boolStr(o.releasePublished)},
		{"release-pr-url", o.prURL},
		{"release-pr-number", prNumberStr},
		{"pr-created", boolStr(o.prCreated)},
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
