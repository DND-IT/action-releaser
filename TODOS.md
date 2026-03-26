# TODOS

## Align build/release pipeline with existing org patterns

**What:** Match the exact build/release setup used by action-config, action-yaml-update, and action-lock.

**Why:** All three existing dnd-it actions use an identical, proven pipeline. action-releaser should follow the same pattern for consistency, so the team maintains one pattern, not two.

**Existing org pattern (from action-config / action-yaml-update / action-lock):**

### Release management
- **Release-please v4** (`googleapis/release-please-action@v4`) — NOT GoReleaser
- Fission GitHub App for auth (`FISSION_GH_APP_ID` / `FISSION_GH_APP_PRIVATE_KEY`)
- `release-please-config.json` with changelog sections (feat, fix, perf, revert visible; docs, style, chore, refactor, test, build, ci hidden)
- `.release-please-manifest.json` tracks current version
- `release-type: "go"` with `include-v-in-tag: true`, `bump-minor-pre-major: true`

### Docker image
- Multi-stage Dockerfile: `golang:1.23-alpine` builder → `alpine:3.20` runtime
- `CGO_ENABLED=0 go build -ldflags="-s -w"` for static binary
- Runtime image includes `git` (via `apk add --no-cache git`)
- Published to GHCR: `ghcr.io/dnd-it/action-releaser`

### action.yml
- `runs.using: docker` with `image: docker://ghcr.io/dnd-it/action-releaser:X.Y.Z`
- Version marker comment: `# x-release-please-version` on the image line
- Release-please auto-updates this line during release PRs via `extra-files`

### Version tagging
- Exact tags: `vX.Y.Z` (created by release-please)
- Floating tags: `vX` and `vX.Y` (force-pushed by `update-version-aliases` job)
- `latest` Docker tag on release
- SHA-based Docker tag on every push

### CI/CD workflows
1. **test.yml** — on push/PR:
   - `golangci-lint-action@v6`
   - `go test -v -race -coverprofile=coverage.out ./...`
   - Integration tests (build binary, run against test fixtures)
   - PR Docker image: `ghcr.io/dnd-it/action-releaser:pr-{number}`
   - PR image cleanup on PR close
2. **release.yml** — on push to main:
   - Job 1: `release-please` → creates/updates release PR
   - Job 2: `build-and-push` → Docker build + push with semantic tags
   - Job 3: `update-version-aliases` → force-push `vX` and `vX.Y` git tags

### Key differences from current plan
- **Replace GoReleaser with release-please** — org standard, handles version bumping + changelog + release PR
- **Replace GoReleaser Docker build with docker/build-push-action** — org standard
- **Add `# x-release-please-version` marker** to action.yml
- **Add Fission GitHub App auth** to release workflow
- **Add PR Docker images** for testing PRs before merge
- **Add PR image cleanup** on PR close
- **Add golangci-lint** to CI

**Effort:** S (human: ~2h / CC: ~15 min) — copy-paste from action-config, adjust names
**Priority:** P1 — must be done before first release
**Depends on:** Basic Go binary + Dockerfile working

---

## Local binary end-to-end test

**What:** Create a shell script or Makefile target that spins up a temp git repo with conventional commits and tags, then runs the binary in dry-run mode against each strategy. Verifies version calculation, changelog generation, and output writing work end-to-end without needing GitHub Actions or Docker.

**Why:** Fastest feedback loop during development. Catches issues with git-cliff exec, config loading, and strategy logic before pushing.

**How:**
1. Create temp git repo with `git init`
2. Add a series of conventional commits (`feat:`, `fix:`, etc.)
3. Tag an initial version (`v0.1.0`)
4. Add more commits after the tag
5. Run binary with `INPUT_DRY_RUN=true` for each strategy (semver, date-rolling, numeric-rolling)
6. Assert outputs: version is non-empty, changelog is non-empty, skipped is false
7. Test skip case: no new commits after tag → semver should skip
8. Test monorepo: two packages with path-filtered commits

**Effort:** S (human: ~1h / CC: ~10 min)
**Priority:** P1 — needed now for development iteration
**Depends on:** git-cliff installed locally (`brew install git-cliff`)

---

## Docker build test

**What:** Verify the Docker image builds successfully and the baked-in binaries (action-releaser + git-cliff) work inside the container.

**Why:** The Docker image is the shipping artifact. If it doesn't build or git-cliff isn't properly installed, the action fails for every consumer.

**How:**
1. `docker build -t action-releaser:test .`
2. `docker run --rm action-releaser:test --help` (verify binary runs)
3. `docker run --rm --entrypoint git-cliff action-releaser:test --version` (verify git-cliff is baked in)
4. Mount a test git repo into the container and run a dry-run release

**Effort:** S (human: ~30 min / CC: ~5 min)
**Priority:** P1 — must pass before first push
**Depends on:** Docker installed locally, local binary test passing

---

## Go integration tests (temp git repos)

**What:** Go test files in `integration/` that programmatically create temp git repos, run the binary via `os/exec`, and assert on outputs. Tagged with `//go:build integration` so they don't run in `go test ./...` by default.

**Why:** Reproducible, CI-friendly tests that cover the full flow including git operations and git-cliff. These run in the `integration` job in test.yaml.

**How:**
- `TestSemverRelease` — create repo with conventional commits, run binary, assert version bump
- `TestDateRollingRelease` — create repo, run binary, assert YYYY.MM.DD version
- `TestNumericRollingRelease` — create repo, run binary, assert incremented number
- `TestDryRun` — run with dry-run, assert no tag created
- `TestBootstrap` — fresh repo with no tags, assert bootstrap version
- `TestSkipNoConventionalCommits` — only non-conventional commits, assert skipped
- `TestMonorepo` — two packages, assert each gets own version
- `TestPartialFailure` — monorepo where one package config is invalid, assert other still releases

**Effort:** S-M (human: ~3h / CC: ~20 min)
**Priority:** P1 — needed before shipping
**Depends on:** Local binary test passing, git-cliff available in CI

---

## GitHub Actions workflow test

**What:** Create a test repo (or use action-releaser itself) that references the action from a PR Docker image (`pr-{N}` tag) and runs a real release workflow.

**Why:** The only way to verify the action works in its actual runtime environment — GitHub Actions runner, `$GITHUB_OUTPUT`, Docker container, real GitHub API.

**How:**
1. Push a PR → test.yaml builds `pr-{N}` Docker image
2. Create a test workflow in a sandbox repo that uses `docker://ghcr.io/dnd-it/action-releaser:pr-{N}`
3. Run the action with `dry-run: true` first to verify outputs
4. Run the action for real to create an actual GitHub release
5. Verify the release exists via `gh release view`

**Effort:** M (human: ~2h / CC: ~15 min)
**Priority:** P2 — after integration tests pass
**Depends on:** Docker image pushed to GHCR, test repo with write access

---

## Webhook/Slack notification on release

**What:** Optional `webhook-url` input that POSTs release details (version, changelog URL, tag) after success.

**Why:** Team visibility without checking GitHub.
**Effort:** S (human: ~2h / CC: ~10 min)
**Priority:** P3
**Depends on:** v1 adopted by at least one team

---

## git-cliff binary checksum verification

**What:** Pin git-cliff version in Dockerfile and verify SHA256 of the downloaded binary.

**Why:** Org-wide tool with `contents:write` — high-impact if supply chain compromised.
**Effort:** S (human: ~1h / CC: ~5 min)
**Priority:** P2
**Depends on:** v1 shipped
