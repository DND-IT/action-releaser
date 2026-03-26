#!/usr/bin/env bash
set -euo pipefail

# Local end-to-end test for action-releaser.
# Creates temp git repos and runs the binary against each strategy.

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BINARY="${REPO_ROOT}/action-releaser"
export CLIFF_TEMPLATES_DIR="${REPO_ROOT}/cliff-templates"
PASS=0
FAIL=0
TESTS=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BOLD='\033[1m'
NC='\033[0m'

cleanup() {
  rm -rf "$WORKDIR" 2>/dev/null || true
}

assert_output() {
  local name="$1" key="$2" expected="$3" file="$4"
  TESTS=$((TESTS + 1))
  local actual
  actual=$(grep "^${key}=" "$file" | head -1 | cut -d= -f2-)
  # Handle heredoc multi-line outputs — grab the value between delimiters.
  if [ -z "$actual" ]; then
    actual=$(awk "/^${key}<</{found=1; next} found && /^RELEASER_EOF_/{found=0; next} found{print}" "$file")
  fi
  if [ "$expected" = "*nonempty*" ]; then
    if [ -n "$actual" ]; then
      echo -e "  ${GREEN}✓${NC} ${name}: ${key} is non-empty"
      PASS=$((PASS + 1))
    else
      echo -e "  ${RED}✗${NC} ${name}: ${key} expected non-empty, got empty"
      FAIL=$((FAIL + 1))
    fi
  elif [ "$actual" = "$expected" ]; then
    echo -e "  ${GREEN}✓${NC} ${name}: ${key}=${actual}"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}✗${NC} ${name}: ${key} expected '${expected}', got '${actual}'"
    FAIL=$((FAIL + 1))
  fi
}

# Build binary.
echo -e "${BOLD}Building binary...${NC}"
(cd "$(dirname "$0")/.." && go build -o action-releaser ./cmd/action-releaser)
echo ""

WORKDIR=$(mktemp -d)
trap cleanup EXIT

# Helper: create a git repo with conventional commits.
setup_repo() {
  local dir="$1"
  mkdir -p "$dir" && cd "$dir"
  git init -q
  git config user.email "test@test.com"
  git config user.name "Test"
}

# ─────────────────────────────────────────────────
# TEST 1: Semver — normal bump
# ─────────────────────────────────────────────────
echo -e "${BOLD}TEST 1: Semver — normal bump from v0.1.0${NC}"
REPO1="$WORKDIR/test-semver"
setup_repo "$REPO1"
git commit --allow-empty -m "feat: initial feature"
git tag -a v0.1.0 -m "v0.1.0"
git commit --allow-empty -m "feat: add new feature"
git commit --allow-empty -m "fix: correct a bug"

OUTPUT1="$WORKDIR/output1"
touch "$OUTPUT1"
GITHUB_OUTPUT="$OUTPUT1" INPUT_VERSION_STRATEGY=semver INPUT_TAG_PREFIX=v INPUT_DRY_RUN=true \
  "$BINARY" 2>&1 || true

assert_output "semver-bump" "version" "*nonempty*" "$OUTPUT1"
assert_output "semver-bump" "skipped" "false" "$OUTPUT1"
assert_output "semver-bump" "dry-run" "true" "$OUTPUT1"
assert_output "semver-bump" "previous-version" "v0.1.0" "$OUTPUT1"
echo ""

# ─────────────────────────────────────────────────
# TEST 2: Semver — skip (no conventional commits)
# ─────────────────────────────────────────────────
echo -e "${BOLD}TEST 2: Semver — skip when no new conventional commits${NC}"
REPO2="$WORKDIR/test-semver-skip"
setup_repo "$REPO2"
git commit --allow-empty -m "feat: initial"
git tag -a v1.0.0 -m "v1.0.0"
git commit --allow-empty -m "wip: not a conventional commit"

OUTPUT2="$WORKDIR/output2"
touch "$OUTPUT2"
GITHUB_OUTPUT="$OUTPUT2" INPUT_VERSION_STRATEGY=semver INPUT_TAG_PREFIX=v INPUT_DRY_RUN=true \
  "$BINARY" 2>&1 || true

assert_output "semver-skip" "skipped" "true" "$OUTPUT2"
echo ""

# ─────────────────────────────────────────────────
# TEST 3: Semver — bootstrap (no tags)
# ─────────────────────────────────────────────────
echo -e "${BOLD}TEST 3: Semver — bootstrap on fresh repo${NC}"
REPO3="$WORKDIR/test-semver-bootstrap"
setup_repo "$REPO3"
git commit --allow-empty -m "feat: first feature ever"

OUTPUT3="$WORKDIR/output3"
touch "$OUTPUT3"
GITHUB_OUTPUT="$OUTPUT3" INPUT_VERSION_STRATEGY=semver INPUT_TAG_PREFIX=v INPUT_DRY_RUN=true \
  "$BINARY" 2>&1 || true

assert_output "semver-bootstrap" "version" "0.1.0" "$OUTPUT3"
assert_output "semver-bootstrap" "skipped" "false" "$OUTPUT3"
echo ""

# ─────────────────────────────────────────────────
# TEST 4: Date-rolling
# ─────────────────────────────────────────────────
echo -e "${BOLD}TEST 4: Date-rolling — first release of the day${NC}"
REPO4="$WORKDIR/test-date"
setup_repo "$REPO4"
git commit --allow-empty -m "some work"

TODAY=$(date -u +%Y.%m.%d)

OUTPUT4="$WORKDIR/output4"
touch "$OUTPUT4"
GITHUB_OUTPUT="$OUTPUT4" INPUT_VERSION_STRATEGY=date-rolling INPUT_TAG_PREFIX=v INPUT_DRY_RUN=true \
  "$BINARY" 2>&1 || true

assert_output "date-rolling" "version" "$TODAY" "$OUTPUT4"
assert_output "date-rolling" "skipped" "false" "$OUTPUT4"
echo ""

# ─────────────────────────────────────────────────
# TEST 5: Date-rolling — collision (second release)
# ─────────────────────────────────────────────────
echo -e "${BOLD}TEST 5: Date-rolling — second release same day${NC}"
REPO5="$WORKDIR/test-date-collision"
setup_repo "$REPO5"
git commit --allow-empty -m "first release"
git tag -a "v${TODAY}" -m "v${TODAY}"
git commit --allow-empty -m "more work"

OUTPUT5="$WORKDIR/output5"
touch "$OUTPUT5"
GITHUB_OUTPUT="$OUTPUT5" INPUT_VERSION_STRATEGY=date-rolling INPUT_TAG_PREFIX=v INPUT_DRY_RUN=true \
  "$BINARY" 2>&1 || true

assert_output "date-collision" "version" "${TODAY}.2" "$OUTPUT5"
echo ""

# ─────────────────────────────────────────────────
# TEST 6: Numeric-rolling
# ─────────────────────────────────────────────────
echo -e "${BOLD}TEST 6: Numeric-rolling — increment from v3${NC}"
REPO6="$WORKDIR/test-numeric"
setup_repo "$REPO6"
git commit --allow-empty -m "init"
git tag -a v1 -m "v1"
git commit --allow-empty -m "second"
git tag -a v2 -m "v2"
git commit --allow-empty -m "third"
git tag -a v3 -m "v3"
git commit --allow-empty -m "fourth"

OUTPUT6="$WORKDIR/output6"
touch "$OUTPUT6"
GITHUB_OUTPUT="$OUTPUT6" INPUT_VERSION_STRATEGY=numeric-rolling INPUT_TAG_PREFIX=v INPUT_DRY_RUN=true \
  "$BINARY" 2>&1 || true

assert_output "numeric-rolling" "version" "4" "$OUTPUT6"
assert_output "numeric-rolling" "previous-version" "v3" "$OUTPUT6"
echo ""

# ─────────────────────────────────────────────────
# TEST 7: Numeric-rolling — bootstrap
# ─────────────────────────────────────────────────
echo -e "${BOLD}TEST 7: Numeric-rolling — bootstrap (no tags)${NC}"
REPO7="$WORKDIR/test-numeric-bootstrap"
setup_repo "$REPO7"
git commit --allow-empty -m "first"

OUTPUT7="$WORKDIR/output7"
touch "$OUTPUT7"
GITHUB_OUTPUT="$OUTPUT7" INPUT_VERSION_STRATEGY=numeric-rolling INPUT_TAG_PREFIX=v INPUT_DRY_RUN=true \
  "$BINARY" 2>&1 || true

assert_output "numeric-bootstrap" "version" "1" "$OUTPUT7"
echo ""

# ─────────────────────────────────────────────────
# TEST 8: Config file (.release.yml)
# ─────────────────────────────────────────────────
echo -e "${BOLD}TEST 8: Config from .release.yml${NC}"
REPO8="$WORKDIR/test-config"
setup_repo "$REPO8"
cat > .release.yml <<'YAML'
version-strategy: numeric-rolling
tag-prefix: "build-"
YAML
git commit --allow-empty -m "init"

OUTPUT8="$WORKDIR/output8"
touch "$OUTPUT8"
GITHUB_OUTPUT="$OUTPUT8" INPUT_DRY_RUN=true \
  "$BINARY" 2>&1 || true

assert_output "config-file" "version" "1" "$OUTPUT8"
echo ""

# ─────────────────────────────────────────────────
# TEST 9: Invalid strategy
# ─────────────────────────────────────────────────
echo -e "${BOLD}TEST 9: Invalid strategy should fail${NC}"
REPO9="$WORKDIR/test-invalid"
setup_repo "$REPO9"
git commit --allow-empty -m "init"

TESTS=$((TESTS + 1))
if GITHUB_OUTPUT=/dev/null INPUT_VERSION_STRATEGY=invalid INPUT_DRY_RUN=true "$BINARY" 2>&1; then
  echo -e "  ${RED}✗${NC} invalid-strategy: expected failure but got success"
  FAIL=$((FAIL + 1))
else
  echo -e "  ${GREEN}✓${NC} invalid-strategy: correctly failed"
  PASS=$((PASS + 1))
fi
echo ""

# ─────────────────────────────────────────────────
# TEST 10: Shallow clone detection
# ─────────────────────────────────────────────────
echo -e "${BOLD}TEST 10: Shallow clone should fail${NC}"
REPO10_ORIGIN="$WORKDIR/test-shallow-origin"
setup_repo "$REPO10_ORIGIN"
git commit --allow-empty -m "feat: init"

REPO10="$WORKDIR/test-shallow"
git clone --depth 1 "file://$REPO10_ORIGIN" "$REPO10" 2>/dev/null
cd "$REPO10"

TESTS=$((TESTS + 1))
if GITHUB_OUTPUT=/dev/null INPUT_VERSION_STRATEGY=semver INPUT_TAG_PREFIX=v INPUT_DRY_RUN=true "$BINARY" 2>&1; then
  echo -e "  ${RED}✗${NC} shallow-clone: expected failure but got success"
  FAIL=$((FAIL + 1))
else
  echo -e "  ${GREEN}✓${NC} shallow-clone: correctly rejected"
  PASS=$((PASS + 1))
fi
echo ""

# ─────────────────────────────────────────────────
# SUMMARY
# ─────────────────────────────────────────────────
echo -e "${BOLD}════════════════════════════════${NC}"
echo -e "${BOLD}RESULTS: ${PASS}/${TESTS} passed, ${FAIL} failed${NC}"
echo -e "${BOLD}════════════════════════════════${NC}"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
