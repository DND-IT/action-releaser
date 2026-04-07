# action-releaser

Flexible GitHub Action for versioned releases powered by [git-cliff](https://git-cliff.org).

Supports multiple versioning strategies, monorepo releases, and per-repo configuration — designed for orgs where teams have different release needs.

## Features

- **3 versioning strategies:** semver (conventional commits), date-rolling (`2026.03.27`), numeric-rolling (`1`, `2`, `3`)
- **Monorepo support:** release each package independently with path-based filtering
- **Dry-run mode:** preview the next version and changelog without creating a release
- **Per-repo config:** `.release.yml` checked into your repo, overridable via action inputs
- **Changelog generation:** powered by git-cliff with customizable templates
- **Self-contained:** Go binary + git-cliff baked into a single Docker image

## Quick Start

```yaml
name: Release
on:
  push:
    branches: [main]

concurrency:
  group: release-${{ github.ref }}
  cancel-in-progress: false

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: dnd-it/action-releaser@v0
        with:
          version-strategy: semver
```

> **Important:** `fetch-depth: 0` is required. The action will fail with a clear error if it detects a shallow clone.

## Versioning Strategies

### Semver (default)

Parses [conventional commits](https://www.conventionalcommits.org/) to determine the next version. Only creates a release when there are releasable commits (`feat:`, `fix:`, etc.).

```yaml
- uses: dnd-it/action-releaser@v0
  with:
    version-strategy: semver
```

- `feat:` → minor bump
- `fix:` → patch bump
- `feat!:` or `BREAKING CHANGE:` → major bump
- No conventional commits since last tag → skip (no release)
- First release (no tags) → `0.1.0`

### Date-rolling

Version is the current UTC date. Multiple releases per day get a suffix.

```yaml
- uses: dnd-it/action-releaser@v0
  with:
    version-strategy: date-rolling
```

- First release of the day → `2026.03.27`
- Second release same day → `2026.03.27.2`
- Always releases (no commit-type gating)

### Numeric-rolling

Simple incrementing number.

```yaml
- uses: dnd-it/action-releaser@v0
  with:
    version-strategy: numeric-rolling
```

- Increments the highest numeric tag by 1
- First release → `1`
- Always releases (no commit-type gating)

## Inputs

| Input | Description | Default |
|-------|-------------|---------|
| `version-strategy` | `semver`, `date-rolling`, or `numeric-rolling` | `semver` |
| `tag-prefix` | Prefix for git tags (e.g. `v`, `release-`) | `""` |
| `cliff-config` | Path to custom cliff.toml | auto-detect |
| `draft` | Create release as draft | `false` |
| `prerelease` | Mark release as prerelease | `false` |
| `dry-run` | Calculate version + changelog without creating a release | `false` |
| `github-token` | Token for release creation (needs `contents: write`) | `${{ github.token }}` |

## Outputs

| Output | Description |
|--------|-------------|
| `version` | Calculated version string (empty if skipped) |
| `changelog` | Generated changelog text (empty if skipped) |
| `tag` | Created git tag (empty if skipped or dry-run) |
| `release-url` | GitHub release URL (empty if skipped or dry-run) |
| `previous-version` | Previous version tag (empty on first release) |
| `skipped` | `true` if no release was created |
| `dry-run` | `true` if dry-run mode was active |

## Per-Repo Configuration

Create a `.release.yml` in your repo root:

```yaml
version-strategy: semver
tag-prefix: "v"
cliff-config: ""       # path to custom cliff.toml (optional)
draft: false
prerelease: false
```

**Priority:** action inputs > `.release.yml` > defaults.

## Monorepo

Define packages in `.release.yml`:

```yaml
version-strategy: semver
packages:
  - name: api
    path: services/api
    tag-pattern: "api/v*"
  - name: web
    path: services/web
    tag-pattern: "web/v*"
```

Each package gets its own version, changelog, and GitHub release. Tags are namespaced (e.g. `api/v1.2.0`, `web/v0.3.1`). If one package fails, others still release (partial failure with non-zero exit).

## Dry-Run

Preview what would happen without creating a release:

```yaml
- uses: dnd-it/action-releaser@v0
  id: preview
  with:
    dry-run: true
- run: |
    echo "Next version: ${{ steps.preview.outputs.version }}"
    echo "Changelog: ${{ steps.preview.outputs.changelog }}"
```

## Per-PR Release

```yaml
name: Release
on:
  pull_request:
    types: [closed]

permissions:
  contents: write

jobs:
  release:
    if: github.event.pull_request.merged == true
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: dnd-it/action-releaser@v0
```

## Custom Changelog Template

Provide your own [cliff.toml](https://git-cliff.org/docs/configuration):

```yaml
- uses: dnd-it/action-releaser@v0
  with:
    cliff-config: .cliff.toml
```

## How It Works

The action runs a Go binary inside a Docker container with git-cliff pre-installed:

1. Load config from `.release.yml` + action inputs
2. Guard against shallow clones
3. List existing tags, calculate next version via the selected strategy
4. Generate changelog with git-cliff
5. Create git tag + GitHub release (unless dry-run)
6. Write outputs to `$GITHUB_OUTPUT`

## Development

```bash
# Build
go build -o action-releaser ./cmd/action-releaser

# Unit tests
go test -v -race ./...

# Integration tests (requires git-cliff installed)
go test -v -tags=integration ./integration/...

# Local e2e test
./scripts/test-local.sh

# Docker build
docker build -t action-releaser:test .
```

## License

MIT
