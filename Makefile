BINARY    := metabigor
MODULE    := github.com/j3ssie/metabigor
VERSION   := $(shell cat internal/core/constants.go | grep 'VERSION =' | cut -d '"' -f 2)
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILDDATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS   := -s -w -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.buildDate=$(BUILDDATE)'
GOFLAGS   := -trimpath
GOBIN_PATH := $(shell go env GOPATH)/bin

# npm versions are unprefixed semver; constants.go keeps the leading 'v'.
NPM_VERSION := $(patsubst v%,%,$(VERSION))
NPM_OUT_DIR := build/dist-npm

# Recursive (=) so the extra shell call only runs when `make help` references it.
DESC = $(shell grep 'DESC =' internal/core/constants.go | cut -d '"' -f 2)

.PHONY: help build test clean install build-all fmt vet lint e2e update-ip-data release snapshot \
        bump-version npm-build npm-pack npm-publish github-release

# help is declared first for readability, so pin the default goal: bare `make`
# still builds, as it always has.
.DEFAULT_GOAL := build

# printf, not echo: dash and bash-as-sh disagree on whether echo expands \033.
help:
	@printf '\n \033[32mMetabigor $(VERSION)\033[0m - $(DESC)\n'
	@printf ' \033[36mCommit: $(COMMIT)\033[0m\n'
	@printf ' \033[34m────────────────────────────────────────────────────────\033[0m\n\n'
	@printf '\033[33m  BUILD\033[0m\n'
	@printf '    make build            Build ./bin/metabigor and copy it to $$GOPATH/bin\n'
	@printf '    make install          go install into $$GOPATH/bin\n'
	@printf '    make build-all        Cross-compile every platform into dist/\n'
	@printf '    make clean            Remove bin/ and dist/\n\n'
	@printf '\033[33m  TEST & QUALITY\033[0m\n'
	@printf '    make test             Unit tests with the race detector\n'
	@printf '    make e2e              End-to-end CLI contract suite (test/run-e2e.sh)\n'
	@printf '    make fmt              gofmt -w -s\n'
	@printf '    make vet              go vet\n'
	@printf '    make lint             golangci-lint (must be installed)\n\n'
	@printf '\033[33m  DATA\033[0m\n'
	@printf '    make update-ip-data   Refresh the embedded ASN/country databases in public/\n\n'
	@printf '\033[33m  RELEASE\033[0m\n'
	@printf '    make bump-version     Bump VERSION in internal/core/constants.go\n'
	@printf '    make snapshot         Local goreleaser build (cross-platform binaries -> dist/)\n'
	@printf '    make npm-build        Stage @j3ssie/metabigor packages -> $(NPM_OUT_DIR)/\n'
	@printf '    make npm-pack         npm-build + inspectable .tgz tarballs\n'
	@printf '    make npm-publish      Publish @j3ssie/metabigor to npm (rebuilds if stale)\n'
	@printf '    make github-release   Tag, push, and publish the GitHub release\n'
	@printf '    make release          Raw goreleaser release (expects the tag to exist already)\n\n'
	@printf '\033[33m  RELEASE FLOW\033[0m\n'
	@printf '    make bump-version\n'
	@printf '    git commit -am "Release <new version>"   \033[90m# goreleaser refuses a dirty worktree\033[0m\n'
	@printf '    make npm-publish\n'
	@printf '    make github-release\n\n'
	@printf '\033[33m  VARIABLES\033[0m\n'
	@printf '    PART=patch|minor|major|pre|release  bump-version: which part to bump (default: patch)\n'
	@printf '    LABEL=beta                          bump-version: set the prerelease label\n'
	@printf '    SET=v2.3.0                          bump-version: use an exact version\n'
	@printf '    DRY_RUN=1                           bump-version, npm-publish, github-release: preview only\n'
	@printf '    ALLOW_DIRTY=1                       github-release: skip the clean-worktree check\n\n'

build:
	@mkdir -p bin
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o bin/$(BINARY) ./cmd/metabigor
	@cp bin/$(BINARY) $(GOBIN_PATH)/$(BINARY)

install:
	go install $(GOFLAGS) -ldflags '$(LDFLAGS)' ./cmd/metabigor

test:
	go test -race -count=1 ./...

fmt:
	gofmt -w -s .

vet:
	go vet ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Install: https://golangci-lint.run/usage/install/"; exit 1; }
	golangci-lint run ./...

e2e: build
	@echo "Running end-to-end tests..."
	@cd test && ./run-e2e.sh

update-ip-data:
	@echo "Updating embedded databases..."
	@wget -O public/ip-to-asn.csv.zip https://github.com/iplocate/ip-address-databases/raw/refs/heads/main/ip-to-asn/ip-to-asn.csv.zip
	@echo "ASN database updated at public/ip-to-asn.csv.zip"
	@wget -O public/ip-to-country.csv.zip https://github.com/iplocate/ip-address-databases/raw/refs/heads/main/ip-to-country/ip-to-country.csv.zip
	@echo "Country database updated at public/ip-to-country.csv.zip"
	@echo "All databases updated successfully"

clean:
	rm -rf bin/ dist/

build-all: clean
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/$(BINARY)-linux-amd64       ./cmd/metabigor
	GOOS=linux   GOARCH=arm64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/$(BINARY)-linux-arm64       ./cmd/metabigor
	GOOS=darwin  GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/$(BINARY)-darwin-amd64      ./cmd/metabigor
	GOOS=darwin  GOARCH=arm64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/$(BINARY)-darwin-arm64      ./cmd/metabigor
	GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/$(BINARY)-windows-amd64.exe ./cmd/metabigor

snapshot:
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed. Install: https://goreleaser.com/install/"; exit 1; }
	@echo "Building snapshot release with goreleaser (version $(NPM_VERSION))..."
	METABIGOR_VERSION=$(NPM_VERSION) goreleaser release --snapshot --clean --skip=publish

release:
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed. Install: https://goreleaser.com/install/"; exit 1; }
	@echo "Creating release with goreleaser..."
	@echo "Note: Ensure you have a git tag and GITHUB_TOKEN set"
	goreleaser release --clean

# Bump the single source-of-truth VERSION in internal/core/constants.go. npm
# versions are immutable, so every release needs a new number here before
# npm-publish. Override with PART=minor|major|pre|release, LABEL=<label>, or
# SET=<explicit version>. DRY_RUN=1 previews only.
bump-version:
	@PART="$(PART)" LABEL="$(LABEL)" SET="$(SET)" DRY_RUN="$(DRY_RUN)" bash build/scripts/bump-version.sh

# Tag the current commit and publish the GitHub release via goreleaser.
# Needs GITHUB_TOKEN, a clean worktree, and gh-free (goreleaser does the upload).
github-release:
	@VERSION="$(VERSION)" TAG_TARGET="$(TAG_TARGET)" ALLOW_DIRTY="$(ALLOW_DIRTY)" DRY_RUN="$(DRY_RUN)" bash build/scripts/github-release.sh

# --- npm distribution -----------------------------------------------------
# Publish the metabigor binary to npm as @j3ssie/metabigor. The binary ships
# gzipped inside per-platform optional-dependency packages (codex-style: one npm
# name, version-suffixed platform builds). See build/npm/build.mjs.

# "yes" when the goreleaser binaries are missing OR were built for a different
# version than internal/core/constants.go — i.e. stale after a version bump.
#
# Detected via the goreleaser archive name, NOT by grepping the binary:
# goreleaser names each archive `metabigor_v<version>_<os>_<arch>.tar.gz` from
# the version it actually built and `--clean` wipes dist/ at the start of every
# run, so the archive name is an authoritative record of what the binaries are.
# A substring grep of the binary would false-match: a build embeds docs and
# skill files that mention other version strings. Recursive (=) so it only runs
# when referenced by the npm-build/npm-pack guards, not on every `make`.
NPM_NEEDS_BUILD = $(shell bins=$$(ls dist/metabigor_*_*/metabigor dist/metabigor_*_*/metabigor.exe 2>/dev/null); arcs=$$(ls dist/metabigor_v$(NPM_VERSION)_* 2>/dev/null); if [ -z "$$bins" ] || [ -z "$$arcs" ]; then echo yes; else echo no; fi)

# Stage the npm packages from goreleaser output. Runs `make snapshot` first if
# the binaries are missing or stale, so a version bump never ships a mismatched
# binary under a fresh (immutable) npm version.
npm-build:
	@if [ "$(NPM_NEEDS_BUILD)" = "yes" ]; then \
		echo "Binaries missing or stale for $(VERSION) — running 'make snapshot'..."; \
		$(MAKE) snapshot; \
	fi
	@echo "Staging npm packages (version $(NPM_VERSION))..."
	METABIGOR_VERSION=$(NPM_VERSION) node build/npm/build.mjs

# Stage + produce inspectable .tgz tarballs (npm pack) for each package.
npm-pack:
	@if [ "$(NPM_NEEDS_BUILD)" = "yes" ]; then \
		echo "Binaries missing or stale for $(VERSION) — running 'make snapshot'..."; \
		$(MAKE) snapshot; \
	fi
	@echo "Staging + packing npm tarballs (version $(NPM_VERSION))..."
	METABIGOR_VERSION=$(NPM_VERSION) node build/npm/build.mjs --pack
	@echo "Tarballs written to $(NPM_OUT_DIR)/"

# Publish to npm: platform packages FIRST (so the main package's
# optionalDependencies resolve), then the main package. Every run pins the
# `latest` dist-tag to the version being published, so
# `npm install -g @j3ssie/metabigor` always installs this version.
#
# Auth comes from ~/.npmrc, either as a literal token or as
#   //registry.npmjs.org/:_authToken=${NPM_TOKEN}
# which npm interpolates from the environment. The guard below checks the real
# precondition — that npm is authenticated — instead of any one way of getting
# there. Set DRY_RUN=1 to preview (no auth needed; --dry-run only packs).
npm-publish: npm-build
	@echo "Publishing @j3ssie/metabigor ($(NPM_VERSION)) to npm [latest -> $(NPM_VERSION)]..."
	@set -e; \
		if [ "$(DRY_RUN)" = "1" ]; then \
			DRY="--dry-run"; \
			echo "DRY RUN — nothing will be published"; \
		else \
			DRY=""; \
			if ! npm whoami >/dev/null 2>&1; then \
				echo "\033[31m[!] not authenticated to npm — run 'npm login', or export NPM_TOKEN if ~/.npmrc reads it.\033[0m"; \
				echo "\033[31m    Use DRY_RUN=1 to preview without publishing.\033[0m"; \
				exit 1; \
			fi; \
		fi; \
		for d in $(NPM_OUT_DIR)/metabigor-*/; do \
			ptag=$$(basename "$$d" | sed 's/^metabigor-//'); \
			echo "  publishing platform package [$$ptag]"; \
			( cd "$$d" && npm publish --access public --tag "$$ptag" $$DRY ) \
				|| { echo "\033[31m[!] publish failed: $$ptag\033[0m"; exit 11; }; \
		done; \
		echo "  publishing $(NPM_OUT_DIR)/metabigor/ (main) [tag=latest]"; \
		( cd $(NPM_OUT_DIR)/metabigor && npm publish --access public --tag latest $$DRY ) \
			|| { echo "\033[31m[!] publish failed: main\033[0m"; exit 12; }; \
		if [ "$(DRY_RUN)" != "1" ]; then \
			echo "  pointing 'latest' dist-tag at $(NPM_VERSION)"; \
			npm dist-tag add @j3ssie/metabigor@$(NPM_VERSION) latest \
				|| { echo "\033[31m[!] dist-tag add failed\033[0m"; exit 13; }; \
			resolved=""; \
			for i in 1 2 3 4 5 6; do \
				resolved=$$(npm dist-tag ls @j3ssie/metabigor --prefer-online 2>/dev/null \
					| sed -n 's/^latest: //p'); \
				[ "$$resolved" = "$(NPM_VERSION)" ] && break; \
				echo "  latest still '$$resolved' (npm registry cache lag) — retry $$i/6 in 10s"; \
				sleep 10; \
			done; \
			if [ "$$resolved" != "$(NPM_VERSION)" ]; then \
				echo "\033[31m[!] latest resolved to '$$resolved', expected '$(NPM_VERSION)' after retries\033[0m"; \
				echo "\033[31m    Publish likely succeeded — verify: npm dist-tag ls @j3ssie/metabigor\033[0m"; \
				exit 14; \
			fi; \
			echo "  verified: latest -> $$resolved"; \
		fi
	@echo "npm publish complete"
