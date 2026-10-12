# releasing-layer

Release layer: local publishing, installers, self-update and CI release checks.
Issue #5 · Branch `feature/releasing-layer` (from `develop` via `dflow start`) · Type: feature

Owner-approved decisions (issue #5): local `make release` publishing (CI never
publishes), read-only CI release checks, checksums-only integrity, GitHub
releases + installers + `go install` with module-proxy warm-up, full
`dforge update` in v1, runbook adapted to this repository's flow.

Reference maps: `dflow` and `gojira` release layers (Engram topic key
`dforge/release-layer/reference-maps`).

## Stages

- [x] **Go foundation** — `go.mod` (module `github.com/yepizrene-devoost/dforge`, go
  1.26.1, stdlib-only), `main.go` with `var version = "dev"` ldflags target wired
  through `cmd.SetVersion`/`cmd.Execute`, `cmd/version.go` (`dforge version [--json]`
  with buildvcs stamp), 6 deterministic unit tests, Makefile (`build`, `build-all`,
  `test`, `vet`, `check`, `tidy`, `install`, `clean`), gojira-identical
  `.golangci.yml`, rooted `.gitignore` entries. Verified: `go build`, `go vet`,
  `go test`, `make build`, `make build-all` (6 binaries), `make check`, JSON shape.
  Deviations: stdlib `flag` dispatcher instead of cobra (zero-external-deps
  constraint); `go.sum` absent (no external deps — Go writes no empty go.sum).
- [x] **Version provenance** — `internal/release/provenance.go` (package
  `release`, stdlib-only): `DevMarker`/`SnapshotPrefix` constants,
  `ClassifyMarker`/`MarkerKind` (`MarkerRelease`, `MarkerNone`, `MarkerDev`,
  `MarkerSnapshot`), `HasReleaseProvenance` mirroring dflow's semantics (trim,
  exact `dev`, `snapshot-` prefix). 19 table-driven subtests, test-first (RED
  observed: undefined symbols; GREEN: all pass). Documented decisions: `0.1.0`
  has provenance (predicate classifies markers, not tag shape — tag-shape
  validation belongs to release checks); `DEV` is `MarkerRelease` (exact,
  case-sensitive match; the build only emits lowercase `dev`). Verified: `go
  build`, `go vet`, `go test ./...`, stage-1 tests uncached, `make check`.
  Accepted deviations: `snapshot-` requires the dash (dflow matches broader
  `snapshot`); bare `snapshot` would classify as release — our GoReleaser
  snapshot template is `snapshot-<short>`, so this is safe.
- [x] **GoReleaser config** — `.goreleaser.yaml` (format v2, deprecation-free):
  single CGO-off build linux/darwin/windows × amd64/arm64 with
  `IsSnapshot`/`Tag` ldflags, tar.gz archives (zip on windows), dflow-style
  `title .Os` name template (`dforge_Linux_x86_64.tar.gz`), LICENSE/README/
  CHANGELOG bundled, `checksums.txt`, `release.mode: replace`, no changelog
  block (commented rationale), snapshot `snapshot-{{ .ShortCommit }}`,
  installers as `release.extra_files` (later slice; non-matching globs ignored).
  LICENSE exists in the root, so it is bundled. `goreleaser check` validated,
  no deprecation warnings.
- [x] **Changelog** — `cliff.toml` (conventional commits,
  `filter_unconventional`, `protect_breaking_commits`, `tag_pattern
  "v[0-9].*"`, feat/fix/perf grouped, `docs(odd)` and `docs` skipped,
  refactor/test/chore → Internal, catch-all last, heading-free output);
  `make changelog` draft target; `CHANGELOG.md` seed with
  generated-and-curated header and `## 📦 v0.1.0 – First public release`
  section so the CI changelog contract is green from day one. git-cliff smoke
  test produced grouped draft without title/date/version.
- [x] **Release-notes extractor** — `make release-notes`: one self-contained
  awk block extracting the newest `## 📦` section, exact `$(VERSION)` semver
  match (prerelease accepted), non-blank body, temp file with cleanup on
  failure. `tests/release-contract.sh` (5 extraction cases + release preflight
  cases with fake goreleaser/curl and a git fixture: byte-exact extraction,
  prerelease acceptance, mismatch/malformed/empty/missing-section rejections,
  missing `.env`/token, dev refusal, untagged and mismatched tag guards, stale
  notes, happy path asserting exact goreleaser args, proxy URL and no token
  leak). `make release-contract` green.
- [ ] **CI release checks** — read-only workflow: `goreleaser check` with pinned
  version, changelog-contract assertion mirroring the Makefile awk, run on
  push/PR to `main`+`develop`.
- [ ] **Installers** — `scripts/install.sh` (POSIX) and `scripts/install.ps1`
  (PowerShell): fetch latest release, verify SHA-256 against `checksums.txt`,
  install without sudo; contract tests for the install scripts.
- [ ] **Self-update** — `dforge update`: provenance gate (refuses `dev` and
  `snapshot-*`), download latest release for the current platform, verify
  checksum, replace binary atomically; tests for provenance and checksum logic.
- [x] **Proxy warm-up** — `make warm-module-proxy`: curls
  `proxy.golang.org/github.com/yepizrene-devoost/dforge/@v/$(VERSION).info`,
  refuses `dev`, exits non-zero on curl failure for honest standalone retry;
  `make release` swallows warm-up failure with a warn (release never fails on
  warm-up). `make release` chains `release-notes` → preflights (`.env`, token,
  dev refusal, tag-on-HEAD, notes-version match) → `goreleaser release --clean
  --release-notes` → warm-up. Pinned `lint` target (golangci-lint 2.11.4,
  hard-fail on mismatch). Plain `make` (not `$(MAKE)`) in the release recipe so
  `make -n` stays a dry run — dflow's documented workaround.
- [ ] **Runbook and bookkeeping** — `RELEASING.md` adapted to this repository
  (`release/*` from `develop` → PR into `main` → tag on `main` → `make release`;
  tags never moved or deleted; `References #N` on release PRs),
  `docs/branches/releasing-layer.md`, `.agents/MEMORY.md` Task Notes.
- [ ] **README refresh** — update `README.md`: the `Status: design phase` note
  (line ~26) must reflect that the binary skeleton and the release layer exist;
  the `## Roadmap` section (line ~197) gains the release layer as delivered
  work between Phase 0 and Phase 1; the documentation map lists `RELEASING.md`.
  Prose-only changes: no rule or boundary moves, so this stays chore-level
  within this feature.
- [ ] **Native review** — freeze the tree for native review before commit. The
  verdict is expected as a native receipt recorded in Engram, never in this
  file; no approval is ever recorded here.

## Review slices

Declared before the first freeze, per the owner's instruction to keep candidates
small (review budgets: 600 functional lines per commit, 400 per pull-request
slice):

- **Slice 1** — Go foundation + version provenance (stages 1–2) plus this task
  document and the `.gitignore` bookkeeping. Everything else stays out of the
  tree until this slice is reviewed and committed.
- **Slice 2** — release plumbing: `.goreleaser.yaml`, `cliff.toml`,
  `CHANGELOG.md` seed, release-notes extractor, `tests/release-contract.sh`,
  `release`/`warm-module-proxy`/`lint` Makefile targets (stages 3–5, 9).
- **Slice 3** — CI release checks workflow (stage 6).
- **Slice 4** — installers with checksum verification + contract tests (stage 7).
- **Slice 5** — `dforge update` self-update (stage 8).
- **Slice 6** — runbook, branch note, Task Notes, README refresh (stages 10, 12).
  Passive documentation: may skip native review if trivial.

Each slice: implement → verify → tick its stages here → freeze → native review →
commit → next slice.

## Notes

- Commit convention: `type(scope): description`, English, lowercase, no period,
  no ticket IDs, no file paths. Commit subjects stay conventional with no issue
  reference (decision 0012); the PR body carries the issue link.
- The module path must avoid internal-name disclosure (public repository).
- Public tags are never moved, force-pushed, or deleted: a wrong public tag
  means a new patch release.
- `make release` preflights: token in `.env`, exact tag on HEAD, release notes
  matching the tag version.

## Commit identity

Recorded in Engram after each work-unit commit, not in this file: writing SHAs
here would be a tracked write after review approval, which mints a new candidate
and forces a re-review for a bookkeeping line.
