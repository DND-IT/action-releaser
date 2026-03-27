// Package releasepr manages the release PR lifecycle:
// create/update a release PR on push, detect merge and trigger release.
//
// Flow:
//   push to main → CreateOrUpdate() → release PR with changelog in body
//   PR merged    → DetectMerge()    → returns manifest for release creation
//   post-release → Cleanup()        → swap labels, delete branch
package releasepr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v68/github"
)

const (
	LabelPending = "autorelease: pending"
	LabelTagged  = "autorelease: tagged"
	BranchPrefix = "release/"
	ManifestFile = ".release-pending.json"
)

// Manifest is the JSON payload committed to the release branch.
type Manifest struct {
	Version  string `json:"version"`
	Strategy string `json:"strategy"`
	Tag      string `json:"tag"`
}

// Client wraps go-github for release PR operations.
type Client struct {
	gh    *github.Client
	owner string
	repo  string
}

// NewClient creates a release PR client.
func NewClient(gh *github.Client, owner, repo string) *Client {
	return &Client{gh: gh, owner: owner, repo: repo}
}

// CreateOrUpdate creates a new release PR or updates an existing one.
// The PR body contains the changelog preview. A .release-pending.json
// manifest is committed to the release branch.
func (c *Client) CreateOrUpdate(ctx context.Context, version, tag, changelog, baseBranch string) (prURL string, err error) {
	branchName := BranchPrefix + tag

	// Search for existing release PR by label.
	existing, err := c.findPendingPR(ctx)
	if err != nil {
		return "", fmt.Errorf("search release PRs: %w", err)
	}

	manifest := Manifest{
		Version:  version,
		Strategy: "", // filled by caller if needed
		Tag:      tag,
	}
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")

	if existing != nil {
		log.Printf("found existing release PR #%d, updating", existing.GetNumber())

		// Force-update the release branch to current HEAD.
		if err := c.updateReleaseBranch(ctx, branchName, baseBranch, manifestJSON); err != nil {
			return "", fmt.Errorf("update release branch: %w", err)
		}

		// Update PR title and body.
		title := fmt.Sprintf("chore: release %s", tag)
		body := formatPRBody(version, changelog)
		_, _, err := c.gh.PullRequests.Edit(ctx, c.owner, c.repo, existing.GetNumber(), &github.PullRequest{
			Title: github.Ptr(title),
			Body:  github.Ptr(body),
		})
		if err != nil {
			return "", fmt.Errorf("update PR #%d: %w", existing.GetNumber(), err)
		}
		log.Printf("release PR #%d updated", existing.GetNumber())
		return existing.GetHTMLURL(), nil
	}

	// No existing PR — create release branch and PR.
	log.Printf("creating release branch %s", branchName)
	if err := c.createReleaseBranch(ctx, branchName, baseBranch, manifestJSON); err != nil {
		return "", fmt.Errorf("create release branch: %w", err)
	}

	title := fmt.Sprintf("chore: release %s", tag)
	body := formatPRBody(version, changelog)

	pr, _, err := c.gh.PullRequests.Create(ctx, c.owner, c.repo, &github.NewPullRequest{
		Title: github.Ptr(title),
		Body:  github.Ptr(body),
		Head:  github.Ptr(branchName),
		Base:  github.Ptr(baseBranch),
	})
	if err != nil {
		return "", fmt.Errorf("create PR: %w", err)
	}

	// Add pending label.
	if err := c.ensureLabel(ctx, LabelPending, "fbca04"); err != nil {
		log.Printf("warning: could not create label %q: %v", LabelPending, err)
	}
	_, _, err = c.gh.Issues.AddLabelsToIssue(ctx, c.owner, c.repo, pr.GetNumber(), []string{LabelPending})
	if err != nil {
		log.Printf("warning: could not add label to PR #%d: %v", pr.GetNumber(), err)
	}

	log.Printf("release PR #%d created: %s", pr.GetNumber(), pr.GetHTMLURL())
	return pr.GetHTMLURL(), nil
}

// DetectMerge checks if the current event is a release PR merge.
// Returns the manifest if it is, nil otherwise.
func DetectMerge() (*Manifest, error) {
	eventName := os.Getenv("GITHUB_EVENT_NAME")
	if eventName != "pull_request" {
		return nil, nil
	}

	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(eventPath)
	if err != nil {
		return nil, fmt.Errorf("read event file: %w", err)
	}

	var event struct {
		Action      string `json:"action"`
		PullRequest struct {
			Merged bool `json:"merged"`
			Head   struct {
				Ref string `json:"ref"`
			} `json:"head"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("parse event: %w", err)
	}

	// Must be a closed+merged PR.
	if event.Action != "closed" || !event.PullRequest.Merged {
		return nil, nil
	}

	// Must have the pending label.
	hasPendingLabel := false
	for _, l := range event.PullRequest.Labels {
		if l.Name == LabelPending {
			hasPendingLabel = true
			break
		}
	}
	if !hasPendingLabel {
		return nil, nil
	}

	// Must be from a release branch.
	if !strings.HasPrefix(event.PullRequest.Head.Ref, BranchPrefix) {
		return nil, nil
	}

	// Try to read the manifest from the workspace.
	manifest, err := readManifestFromWorkspace()
	if err != nil {
		log.Printf("warning: could not read manifest, will recalculate: %v", err)
		// Return a marker manifest — caller should recalculate version.
		tag := strings.TrimPrefix(event.PullRequest.Head.Ref, BranchPrefix)
		return &Manifest{Tag: tag}, nil
	}

	return manifest, nil
}

// Cleanup swaps labels and deletes the release branch after a successful release.
func (c *Client) Cleanup(ctx context.Context, prNumber int, branchName string) {
	// Remove pending label, add tagged label.
	_, err := c.gh.Issues.RemoveLabelForIssue(ctx, c.owner, c.repo, prNumber, LabelPending)
	if err != nil {
		log.Printf("warning: could not remove label %q from PR #%d: %v", LabelPending, prNumber, err)
	}

	if err := c.ensureLabel(ctx, LabelTagged, "0e8a16"); err != nil {
		log.Printf("warning: could not create label %q: %v", LabelTagged, err)
	}
	_, _, err = c.gh.Issues.AddLabelsToIssue(ctx, c.owner, c.repo, prNumber, []string{LabelTagged})
	if err != nil {
		log.Printf("warning: could not add label %q to PR #%d: %v", LabelTagged, prNumber, err)
	}

	// Delete the release branch.
	_, err = c.gh.Git.DeleteRef(ctx, c.owner, c.repo, "heads/"+branchName)
	if err != nil {
		log.Printf("warning: could not delete branch %s: %v", branchName, err)
	} else {
		log.Printf("deleted release branch %s", branchName)
	}
}

// --- internal helpers ---

func (c *Client) findPendingPR(ctx context.Context) (*github.PullRequest, error) {
	prs, _, err := c.gh.PullRequests.List(ctx, c.owner, c.repo, &github.PullRequestListOptions{
		State: "open",
	})
	if err != nil {
		return nil, err
	}
	for _, pr := range prs {
		for _, l := range pr.Labels {
			if l.GetName() == LabelPending {
				return pr, nil
			}
		}
	}
	return nil, nil
}

func (c *Client) createReleaseBranch(ctx context.Context, branchName, baseBranch string, manifestJSON []byte) error {
	// Get the SHA of the base branch.
	baseRef, _, err := c.gh.Git.GetRef(ctx, c.owner, c.repo, "refs/heads/"+baseBranch)
	if err != nil {
		return fmt.Errorf("get base ref %s: %w", baseBranch, err)
	}
	baseSHA := baseRef.GetObject().GetSHA()

	// Delete stale branch if it exists.
	c.gh.Git.DeleteRef(ctx, c.owner, c.repo, "refs/heads/"+branchName) //nolint:errcheck

	// Create the release branch at the same commit as base.
	_, _, err = c.gh.Git.CreateRef(ctx, c.owner, c.repo, &github.Reference{
		Ref:    github.Ptr("refs/heads/" + branchName),
		Object: &github.GitObject{SHA: github.Ptr(baseSHA)},
	})
	if err != nil {
		return fmt.Errorf("create branch %s: %w", branchName, err)
	}

	// Commit the manifest file to the release branch.
	return c.commitManifest(ctx, branchName, manifestJSON)
}

func (c *Client) updateReleaseBranch(ctx context.Context, branchName, baseBranch string, manifestJSON []byte) error {
	// Force-update the branch to match the base.
	baseRef, _, err := c.gh.Git.GetRef(ctx, c.owner, c.repo, "refs/heads/"+baseBranch)
	if err != nil {
		return fmt.Errorf("get base ref %s: %w", baseBranch, err)
	}
	baseSHA := baseRef.GetObject().GetSHA()

	// Delete and recreate to force-update.
	c.gh.Git.DeleteRef(ctx, c.owner, c.repo, "refs/heads/"+branchName) //nolint:errcheck
	_, _, err = c.gh.Git.CreateRef(ctx, c.owner, c.repo, &github.Reference{
		Ref:    github.Ptr("refs/heads/" + branchName),
		Object: &github.GitObject{SHA: github.Ptr(baseSHA)},
	})
	if err != nil {
		return fmt.Errorf("recreate branch %s: %w", branchName, err)
	}

	return c.commitManifest(ctx, branchName, manifestJSON)
}

func (c *Client) commitManifest(ctx context.Context, branchName string, manifestJSON []byte) error {
	// Get current commit on the branch.
	ref, _, err := c.gh.Git.GetRef(ctx, c.owner, c.repo, "refs/heads/"+branchName)
	if err != nil {
		return fmt.Errorf("get branch ref: %w", err)
	}
	parentSHA := ref.GetObject().GetSHA()

	// Get the tree of the parent commit.
	parentCommit, _, err := c.gh.Git.GetCommit(ctx, c.owner, c.repo, parentSHA)
	if err != nil {
		return fmt.Errorf("get parent commit: %w", err)
	}

	// Create a blob for the manifest.
	blob, _, err := c.gh.Git.CreateBlob(ctx, c.owner, c.repo, &github.Blob{
		Content:  github.Ptr(string(manifestJSON)),
		Encoding: github.Ptr("utf-8"),
	})
	if err != nil {
		return fmt.Errorf("create blob: %w", err)
	}

	// Create a tree with the manifest file.
	tree, _, err := c.gh.Git.CreateTree(ctx, c.owner, c.repo, parentCommit.GetTree().GetSHA(), []*github.TreeEntry{
		{
			Path: github.Ptr(ManifestFile),
			Mode: github.Ptr("100644"),
			Type: github.Ptr("blob"),
			SHA:  blob.SHA,
		},
	})
	if err != nil {
		return fmt.Errorf("create tree: %w", err)
	}

	// Create a commit.
	commit, _, err := c.gh.Git.CreateCommit(ctx, c.owner, c.repo, &github.Commit{
		Message: github.Ptr("chore: prepare release"),
		Tree:    tree,
		Parents: []*github.Commit{{SHA: github.Ptr(parentSHA)}},
	}, nil)
	if err != nil {
		return fmt.Errorf("create commit: %w", err)
	}

	// Update the branch ref.
	_, _, err = c.gh.Git.UpdateRef(ctx, c.owner, c.repo, &github.Reference{
		Ref:    github.Ptr("refs/heads/" + branchName),
		Object: &github.GitObject{SHA: commit.SHA},
	}, true)
	if err != nil {
		return fmt.Errorf("update branch ref: %w", err)
	}

	return nil
}

func (c *Client) ensureLabel(ctx context.Context, name, color string) error {
	_, _, err := c.gh.Issues.CreateLabel(ctx, c.owner, c.repo, &github.Label{
		Name:  github.Ptr(name),
		Color: github.Ptr(color),
	})
	if err != nil && !strings.Contains(err.Error(), "already_exists") {
		return err
	}
	return nil
}

func readManifestFromWorkspace() (*Manifest, error) {
	data, err := os.ReadFile(ManifestFile)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

func formatPRBody(version, changelog string) string {
	var b strings.Builder
	b.WriteString("## Release `")
	b.WriteString(version)
	b.WriteString("`\n\n")
	b.WriteString("This PR was created automatically by [action-releaser](https://github.com/dnd-it/action-releaser).\n")
	b.WriteString("Merge this PR to create the release.\n\n")
	if changelog != "" {
		b.WriteString("### Changelog\n\n")
		b.WriteString(changelog)
	} else {
		b.WriteString("*No changelog entries.*\n")
	}
	return b.String()
}
