# action-releaser

Flexible GitHub Action for versioned releases powered by git-cliff.
Go binary baked into a Docker image, published to GHCR.

## Build

```bash
go build -o action-releaser ./cmd/action-releaser
```

## Test

```bash
go test -v -race ./...
```

## Architecture

- `cmd/action-releaser/` — CLI entrypoint
- `internal/strategy/` — VersionStrategy interface + semver, calver
- `internal/release/` — GitHub release creation via go-github SDK
- `internal/changelog/` — git-cliff changelog generation
- `internal/config/` — .release.yml parsing + action input merging
- `internal/gitutil/` — git operations (tag listing, shallow-clone guard)
- `internal/output/` — GitHub Actions output writer

## Release

Uses release-please (org standard). Push to main triggers release-please to create
a release PR. Merging the PR builds and pushes the Docker image to GHCR.
