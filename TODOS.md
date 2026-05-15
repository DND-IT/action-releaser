# TODOS

Punch list from the 2026-05-15 code-quality review. Grouped by category, roughly
ordered by impact within each group.

---

## Active bugs

### 0. Direct-mode release notes are one release behind
**Symptom:** GitHub Release notes contain commits from the *previous* release range, not the commits introduced in the current release. Observed in `DND-IT/cms-cassandra-keepalive`:
- v0.1.1 (tagged on commit `ae9cafd`, #5) — body shows the "[0.1.0]" section with commits before v0.1.0.
- v0.1.2 (tagged on commit `14ae2cd`, #6) — body shows commits #4 and #5 (the v0.1.0..v0.1.1 range); commit #6 is missing.

**Root cause:** In `cmd/action-releaser/main.go`, `changelog.Generate(cfg)` is called at line 161 — *before* `gitutil.CreateTag(...)` inside `directRelease` (line 317). But `internal/changelog/changelog.go:18-25` assumes the opposite:

```go
// In PR mode no tag has been created yet, so --latest would resolve to the
// previous release's range. Use --unreleased to capture all commits since
// the last tag. In direct mode the tag is created before this runs, so
// --latest correctly resolves to the new release's range.
rangeFlag := "--latest"
if cfg.ReleaseMode == "pr" {
    rangeFlag = "--unreleased"
}
```

The comment is wrong about direct mode. Since the tag doesn't yet exist when `git-cliff` runs, `--latest` resolves to the *previous* tag and emits the prior release's range. Commit `b0f0afb` ("fix: use --unreleased flag for git-cliff in PR release mode") fixed this for PR mode but missed direct mode.

**Fix options:**
1. *(Recommended)* Drop the `pr`-only branch in `changelog.go` and always use `--unreleased`. The tag is never created before `changelog.Generate` in either mode, so `--unreleased` is correct universally.
2. Reorder `main.go` so `directRelease` creates and pushes the tag before calling `changelog.Generate`. Keeps `--latest` semantically correct but risks orphan tags if changelog generation fails.

**Where:**
- `cmd/action-releaser/main.go` (lines 160-165, 306-323)
- `internal/changelog/changelog.go` (lines 17-26)
- `internal/changelog/changelog_test.go` (assertions will need updating)
- `integration/integration_test.go` — add a regression test that runs a real release flow and asserts the changelog contains the *current* commits, not the previous range.

**Priority:** P0 — every direct-mode user is shipping wrong release notes today.

---

## Stale code & docs

### 1. Delete `internal/strategy/daterolling.go`
**What:** Remove the file entirely.
**Why:** Commit `4a316f8` ("feat!: replace date-rolling/numeric-rolling with calver, drop numeric-rolling") shipped the removal in the CHANGELOG, but the file was never deleted. `strategy.New` only recognises `semver`/`calver`, so `DateRolling` is unreachable. It's a near-duplicate of `calver.go` and confuses readers.
**Priority:** P1 (one-line cleanup, no risk)

### 2. Mark or delete `docs/designs/action-releaser.md`
**What:** Either annotate `STATUS: SUPERSEDED — see CHANGELOG and current code` at the top, or delete.
**Why:** Document still describes a three-strategy design (semver, date-rolling, numeric-rolling) and includes `daterolling.go` in the file layout. Anyone reading it gets the wrong mental model.
**Priority:** P1

### 3. Delete `gitutil.HasConventionalCommits`
**What:** Remove the function.
**Why:** No callers. `strategy.hasConventionalCommitsSinceRef` (in `semver.go`) is a near-identical duplicate that *is* used.
**Where:** `internal/gitutil/gitutil.go:90`
**Priority:** P1

### 4. Remove or wire up `Manifest.Strategy`
**What:** Either populate the field from `cfg.VersionStrategy` in `CreateOrUpdate`, or delete it from the struct.
**Why:** Currently written as `Strategy: ""` with the comment "filled by caller if needed" — no caller ever fills it. Dead JSON field in a persisted manifest.
**Where:** `internal/releasepr/releasepr.go:33,97`
**Priority:** P2

### 5. Rewrite or delete this file
**What:** After the rest of the punch list is cleared, replace `TODOS.md` with a short living "known limitations / future work" doc, or delete entirely.
**Why:** Prior contents were a pre-v1 planning doc claiming items were "P1 before first release" — but the project is on 0.3.1 and those items are done.
**Priority:** P3 (do last)

---

## Correctness & robustness

### 6. Replace error-message substring sniffing with typed checks
**What:** Use `errors.As(err, &gerr *github.ErrorResponse)` plus structured field checks instead of `strings.Contains(err.Error(), "already_exists")`.
**Why:** Brittle — breaks the moment GitHub rewords an error message. go-github already returns typed errors with status codes.
**Where:**
- `internal/release/release.go:71` (`already_exists` check on 422)
- `internal/releasepr/releasepr.go:432` (`ensureLabel`)
**Priority:** P2

### 7. Paginate `DetectMerge` or document the 100-PR ceiling
**What:** Either loop with `ListOptions.Page` until the merged release PR is found (with a sensible upper bound), or add a comment stating the limitation.
**Why:** Repos with frequent PRs can push the merged release PR off the first page, and the merge silently goes undetected. The action then fails to create a release.
**Where:** `internal/releasepr/releasepr.go:160`
**Priority:** P2

### 8. Fix `baseBranch` derivation
**What:** Stop defaulting to `GITHUB_REF_NAME`. Either accept an explicit `base-branch` input on the action, or derive from `GITHUB_BASE_REF` when present (PR events) and only fall back to `GITHUB_REF_NAME` on push events. At minimum, document the constraint in `action.yaml`.
**Why:** `GITHUB_REF_NAME` is the *triggering ref*, not necessarily the PR base. On `workflow_dispatch` against a non-main branch, this picks the wrong base; `DetectMerge` then searches PRs against the wrong target.
**Where:** `cmd/action-releaser/main.go:53,193`
**Priority:** P2

### 9. Move `gitutil` package init side-effects to an explicit call
**What:** Delete the `init()` in `gitutil.go` and move the `safe.directory` config into `ConfigureAuth` (or a new `Init()` called from `main`).
**Why:** A package init that mutates the user's global git config on import is surprising. Fine in the Docker container, but the package is now untestable in any environment where you don't want that side-effect.
**Where:** `internal/gitutil/gitutil.go:13`
**Priority:** P2

### 10. `FindBuiltinConfig` should error on missing template
**What:** Add a second return value (or a `MustFindBuiltinConfig`) and have callers fail loudly if the bundled `cliff-templates/<strategy>.toml` cannot be found in any candidate path.
**Why:** Currently returns `""` silently, causing git-cliff to fall back to its keepachangelog default — exactly the regression `d163602` fixed. A missing bundled template should be a build-time/runtime hard error, not a silent fallback.
**Where:** `internal/strategy/semver.go:113`
**Priority:** P2

### 11. Read manifest from repo root, not CWD
**What:** Resolve `ManifestFile` against the workspace root (`GITHUB_WORKSPACE` or a git-rev-parse result) instead of relying on `os.Getwd`.
**Why:** If a workflow sets `working-directory:` to a subdir, `os.ReadFile(".release-pending.json")` silently misses and forces the (slower) recalculation path. The user gets a release, but not the intended one.
**Where:** `internal/releasepr/releasepr.go:438`
**Priority:** P3

### 12. Audit `//nolint:errcheck` calls in `gitutil`
**What:** For each `exec.Command(...).Run() //nolint:errcheck`, either log the error or document why it's safe to swallow.
**Why:** `git config user.name` or `git remote set-url` failing silently produces mystifying push failures three steps later. The lint suppressions hide real diagnostic signal.
**Where:** `internal/gitutil/gitutil.go:19,37,38,44`
**Priority:** P3

---

## Style & structure

### 13. Fix duplicate step numbering in `main.go`
**What:** Renumber the comment-as-outline in `run()` — there are two consecutive "3." after the PR-mode early return.
**Where:** `cmd/action-releaser/main.go:42,69`
**Priority:** P3

### 14. Replace `os.Exit(2)` in `processPackage` with sentinel error
**What:** Return a typed `partialFailure` error from `run()` and have `main()` translate it into the exit code.
**Why:** Mid-function `os.Exit` skips deferred cleanups and mixes control-flow styles.
**Where:** `cmd/action-releaser/main.go:112`
**Priority:** P3

### 15. Document `pr-url` vs `release-pr-url` duplication
**What:** Either add a comment in `setOutputs` explaining the back-compat reason both outputs carry the same value, or pick one and deprecate the other in `action.yaml`.
**Where:** `cmd/action-releaser/main.go:388,394`
**Priority:** P3

### 16. Simplify `actionOutputs` / `setOutputs`
**What:** Either use a `map[string]string` driven directly, or generate the pairs from struct tags. The current parallel-struct-plus-slice pattern is hand-rolled and easy to get out of sync.
**Where:** `cmd/action-releaser/main.go:362-404`
**Priority:** P3

### 17. Dedupe `runReleaser` and `readOutput` in integration tests
**What:** Extract the common 20 lines of setup (output file, env, command) into one helper that both wrappers call.
**Where:** `integration/integration_test.go:96,120`
**Priority:** P3

---

## Test coverage

### 18. Unit-test `release.Create` with `httptest`
**What:** Spin up an `httptest.Server`, point a `github.Client` at it (`BaseURL`), and assert on the retry/classification logic for 401, 403, 422 (`already_exists`), 5xx-then-200, and 5xx-then-5xx-then-5xx.
**Why:** The most branchy code in the project (status-code switch + exponential backoff) has zero direct tests. Currently only covered indirectly through integration.
**Where:** `internal/release/release.go:34`
**Priority:** P2

### 19. Unit-test `gitutil` parsing
**What:** At minimum `ListTags` (newline split, empty input, trailing whitespace) and `TagExists`. Use a temp repo in `t.TempDir()`.
**Why:** Pure shell-out parsing with no tests. Easy to regress when adding a flag.
**Where:** `internal/gitutil/gitutil.go`
**Priority:** P2

### 20. Expand `releasepr` unit tests
**What:** Add tests for `PrependChangelog` (empty existing, existing with header, existing without header), `ReleaseBranchName` edge cases (no prefix, monorepo prefix), and the `findPendingPR` label-filter path using a mocked transport.
**Why:** These are pure functions with subtle behaviour and high blast-radius if they regress.
**Where:** `internal/releasepr/releasepr_test.go`
**Priority:** P3

### 21. Smoke test `cmd/action-releaser/main.go`
**What:** Add a small `main_test.go` that exercises `setOutputs` end-to-end against a temp `GITHUB_OUTPUT` file, asserting every documented action output is written.
**Why:** Currently `[no test files]`. Documented outputs and the `actionOutputs` struct can drift apart silently.
**Priority:** P3
