# dforge Makefile

BINARY_NAME = dforge
BIN_DIR     = bin

# Channel marker only. A tag is injected when HEAD is exactly on it; every
# other revision keeps the `dev` marker, so the development tree never claims a
# released version. The commit identity is deliberately NOT injected here: the
# binary self-reports it from Go's own buildvcs stamp (vcs.revision,
# vcs.modified), keeping one source of truth per fact.
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || echo dev)

# The .golangci.yml config is versioned, so the linter itself is pinned: a
# drifting golangci-lint would either false-pass or false-fail the same tree.
# `make lint` hard-fails on any other installed version.
GOLANGCI_LINT_VERSION = 2.11.4

LDFLAGS = -s -w -X main.version=$(VERSION)

.PHONY: all
all: build

# Build the local binary with the version marker injected via ldflags.
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) (version: $(VERSION))..."
	go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) .

# Cross-platform build (linux/darwin/windows x amd64/arm64, CGO disabled),
# for testing outside GoReleaser.
.PHONY: build-all
build-all:
	@mkdir -p "$(BIN_DIR)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o "$(BIN_DIR)/$(BINARY_NAME)-linux-amd64" .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o "$(BIN_DIR)/$(BINARY_NAME)-linux-arm64" .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o "$(BIN_DIR)/$(BINARY_NAME)-darwin-amd64" .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o "$(BIN_DIR)/$(BINARY_NAME)-darwin-arm64" .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o "$(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe" .
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o "$(BIN_DIR)/$(BINARY_NAME)-windows-arm64.exe" .

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: check
check: vet test

.PHONY: tidy
tidy:
	go mod tidy

# 🔎 Linter with a pinned version (see GOLANGCI_LINT_VERSION above).
.PHONY: lint
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint v$(GOLANGCI_LINT_VERSION) is required but is not installed" >&2; exit 1; }
	@installed_version=$$(golangci-lint version 2>/dev/null | awk 'NR == 1 { print $$4 }'); \
	if [ "$$installed_version" != "$(GOLANGCI_LINT_VERSION)" ]; then \
		echo "golangci-lint version mismatch: required v$(GOLANGCI_LINT_VERSION), found v$${installed_version:-unknown}" >&2; \
		exit 1; \
	fi
	golangci-lint run ./...

# 📝 Generate the changelog draft from Conventional Commits with git-cliff.
# The draft is curated by hand into CHANGELOG.md and never committed.
.PHONY: changelog
changelog:
	@command -v git-cliff >/dev/null 2>&1 || { \
		echo "❌  git-cliff is not installed. Install it first: https://git-cliff.org/docs/installation"; \
		exit 1; \
	}
	@echo "📝 Generating CHANGELOG.draft.md from Conventional Commits..."
	@git cliff --unreleased --config cliff.toml -o CHANGELOG.draft.md
	@echo "✅ Draft ready. Curate it into CHANGELOG.md, then discard it: it is never committed."

# 📝 Extract the newest version section from CHANGELOG.md into RELEASE_NOTES.md
# (generated, gitignored, never committed). The awk program below is one
# self-contained block: the CI release check runs the byte-identical program.
# Output goes to a temporary file and only lands as RELEASE_NOTES.md once it
# holds the exact $(VERSION) semver section (prereleases like -rc.N accepted)
# with content below its heading — an empty or mismatched body would publish a
# wrong GitHub release. A rejected run removes both the temporary file and any
# pre-existing RELEASE_NOTES.md, so a failure never leaves a stale release body
# behind for a bare GoReleaser invocation to publish unvalidated.
.PHONY: release-notes
release-notes:
	@set -eu; \
	tmp=RELEASE_NOTES.md.tmp; \
	fail() { rm -f "$$tmp" RELEASE_NOTES.md; echo "❌  $$1" >&2; exit 1; }; \
	awk -v tag="$(VERSION)" ' \
		BEGIN { found = 0; content = 0 } \
		$$1 == "##" && $$2 == "📦" { \
			if (found) exit; \
			if ($$3 == tag && tag ~ /^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$$/) { found = 1; print; next } \
			exit 1 \
		} \
		found { print; if ($$0 ~ /[^[:space:]]/) content = 1 } \
		END { if (!found || !content) exit 1 } \
	' CHANGELOG.md > "$$tmp" \
		|| fail "release notes rejected: the newest '## 📦' section must be '$(VERSION)' (exact semver tag) and contain content below its heading."; \
	mv "$$tmp" RELEASE_NOTES.md; \
	echo "📝 RELEASE_NOTES.md written from the newest CHANGELOG.md section (generated, never committed)."

# 🧪 Contract test for the release-notes extraction above: runs the same awk
# program from this Makefile against fixture changelogs (plain bash + awk + git).
.PHONY: release-contract
release-contract:
	bash tests/release-contract.sh

# 🌐 Request the tagged version directly from the public Go module proxy. This
# deliberately bypasses the local Go module cache so a successful request proves
# that proxy.golang.org can resolve the published tag. A warm-up failure is
# reported with a standalone retry command: it must never fail the release that
# already succeeded.
.PHONY: warm-module-proxy
warm-module-proxy:
	@set -eu; \
	fail() { echo "❌  $$1" >&2; exit 1; }; \
	[ "$(VERSION)" != dev ] \
		|| fail "make warm-module-proxy needs an explicit release tag: retry with VERSION=vX.Y.Z."; \
	url="https://proxy.golang.org/github.com/yepizrene-devoost/dforge/@v/$(VERSION).info"; \
	echo "🌐 Warming public Go module proxy: $$url"; \
	if curl --fail --silent --show-error --retry 4 --retry-delay 2 --retry-all-errors --connect-timeout 10 --max-time 30 "$$url"; then \
		echo; \
		echo "✅ Public Go module proxy resolved $(VERSION)."; \
	else \
		echo; \
		echo "⚠️  Public Go module proxy warm-up failed for $(VERSION). Retry with: make warm-module-proxy VERSION=$(VERSION)" >&2; \
		exit 1; \
	fi

# 🚀 Publish a GitHub release with GoReleaser. The release body is only the
# newest version section of CHANGELOG.md, extracted (and validated) by the
# release-notes prerequisite, so a stale changelog can never overwrite an
# already correct release body. This preflights .env/GITHUB_TOKEN, refuses to
# run off-tag (gojira's exact-tag-on-HEAD pattern), and re-checks that the
# extracted body names the tag being published before GoReleaser runs. After a
# successful publish the module proxy is warmed; a failed warm-up prints its
# retry command but never fails the already-published release. The warm-up is
# invoked as plain `make`, not `$(MAKE)`: GNU make executes recipe lines that
# contain the literal `$(MAKE)` even under `make -n`, which would turn that
# parse check into a real preflight run.
.PHONY: release
release: release-notes
	@set -eu; \
	fail() { echo "❌  $$1" >&2; exit 1; }; \
	command -v goreleaser >/dev/null 2>&1 || fail "goreleaser is not installed: install it before releasing."; \
	[ -f .env ] \
		|| fail ".env is missing: GoReleaser loads GITHUB_TOKEN from it, so create it before releasing."; \
	[ -r .env ] \
		|| fail ".env exists but cannot be read, so it cannot be sourced: check its permissions before releasing."; \
	set -a; . ./.env || fail "Could not source .env: fix its shell syntax before releasing."; set +a; \
	[ -n "$${GITHUB_TOKEN:-}" ] \
		|| fail ".env defines no GITHUB_TOKEN: GoReleaser needs it to publish the GitHub release."; \
	[ "$(VERSION)" != dev ] \
		|| fail "make release needs a tag, but HEAD is not on an exact tag (VERSION=dev): check out the release commit and create its tag first."; \
	tag_commit=$$(git rev-parse --verify "refs/tags/$(VERSION)^{commit}" 2>/dev/null) \
		|| fail "release version $(VERSION) must be an exact local tag on HEAD"; \
	head_commit=$$(git rev-parse --verify HEAD 2>/dev/null) \
		|| fail "release version $(VERSION) must be an exact local tag on HEAD"; \
	[ "$$tag_commit" = "$$head_commit" ] \
		|| fail "release version $(VERSION) must be an exact local tag on HEAD"; \
	notes_version=$$(head -n 1 RELEASE_NOTES.md | awk '{print $$3}'); \
	[ "$$notes_version" = "$(VERSION)" ] \
		|| fail "RELEASE_NOTES.md carries the $$notes_version section but the tag being released is $(VERSION): curate CHANGELOG.md so its newest '## 📦' section matches the tag."; \
	echo "🚀 Running GoReleaser with .env"; \
	goreleaser release --clean --release-notes=RELEASE_NOTES.md; \
	make --no-print-directory warm-module-proxy VERSION="$(VERSION)" || :

# Install into $GOPATH/bin with the version marker injected.
.PHONY: install
install:
	go install -ldflags="$(LDFLAGS)" .

# Remove the local build output.
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
