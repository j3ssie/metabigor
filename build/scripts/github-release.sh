#!/usr/bin/env bash
set -euo pipefail

# Tag the current commit and publish a GitHub release with goreleaser.
#
# The tag/title is the version in internal/core/constants.go — the same constant
# `make npm-publish` ships to npm — so the npm package and the GitHub release
# for a given version always come from one number. goreleaser builds the
# cross-platform archives, generates the changelog from the commit range, and
# uploads everything (`mode: replace` in .goreleaser.yaml means re-running for
# an existing release replaces its artifacts instead of failing).
#
# Usage (normally via `make github-release`):
#   github-release.sh
#
# Env / make vars:
#   VERSION      = release tag/title (default: parsed from internal/core/constants.go)
#   TAG_TARGET   = commit-ish the new tag points at (default: HEAD)
#   GITHUB_TOKEN = required by goreleaser to create the release
#   ALLOW_DIRTY  = 1 to skip the clean-worktree check (goreleaser will still refuse)
#   DRY_RUN      = 1 to build locally and skip tagging + publishing

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

VERSION_FILE="$ROOT/internal/core/constants.go"
TAG_TARGET="${TAG_TARGET:-HEAD}"
DRY_RUN="${DRY_RUN:-}"

die()  { printf '\033[31m[!] %s\033[0m\n' "$*" >&2; exit 1; }
info() { printf '\033[36m[*]\033[0m %s\n' "$*"; }

command -v git        >/dev/null 2>&1 || die "git not found."
command -v goreleaser >/dev/null 2>&1 || die "goreleaser not found — https://goreleaser.com/install/"

VERSION="${VERSION:-}"
if [ -z "$VERSION" ]; then
  [ -f "$VERSION_FILE" ] || die "version file not found: $VERSION_FILE"
  VERSION="$(grep -E '^[[:space:]]*VERSION[[:space:]]*=' "$VERSION_FILE" | head -1 | cut -d '"' -f 2)"
fi
[ -n "$VERSION" ] || die "could not determine VERSION"

if [ "$DRY_RUN" = "1" ]; then
  info "DRY_RUN=1 — building $VERSION locally, no tag and no upload"
  # Snapshot builds take their version from the newest git tag unless told
  # otherwise; pin it to $VERSION so the dry run produces the same artifacts a
  # real release would.
  METABIGOR_VERSION="${VERSION#v}" exec goreleaser release --snapshot --clean --skip=publish
fi

[ -n "${GITHUB_TOKEN:-}" ] || die "GITHUB_TOKEN not set — goreleaser needs it to create the release."

# goreleaser refuses to release from a dirty worktree; fail here with a message
# that says what to do instead of a stack of goreleaser diagnostics.
if [ "${ALLOW_DIRTY:-}" != "1" ] && [ -n "$(git status --porcelain)" ]; then
  git status --short >&2
  die "worktree is dirty — commit or stash before releasing $VERSION."
fi

# --- Git tag (create + push if missing) --------------------------------------
if git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null; then
  info "git tag $VERSION already exists locally"
else
  info "creating annotated git tag $VERSION at $TAG_TARGET..."
  git tag -a "$VERSION" -m "Release $VERSION" "$TAG_TARGET"
fi

if git ls-remote --exit-code --tags origin "refs/tags/$VERSION" >/dev/null 2>&1; then
  info "git tag $VERSION already present on origin"
else
  info "pushing tag $VERSION to origin..."
  git push origin "refs/tags/$VERSION"
fi

# --- Build + publish ----------------------------------------------------------
info "building and publishing GitHub release $VERSION..."
goreleaser release --clean

info "GitHub release $VERSION published."
