# releasing-layer

Branch `feature/releasing-layer` · Issue #5 · Type: feature

## What changed

The complete release layer of `dforge`, built and reviewed in five slices plus
a documentation slice:

- **Go foundation** (`08f9f3c`) — `go.mod` (stdlib-only), `main.go` with the
  `-X main.version` injection target, `dforge version [--json]` reporting the
  `buildvcs` stamp, Makefile (`build`, `build-all`, `test`, `vet`, `check`,
  `tidy`, `install`, `clean`), gojira-style `.golangci.yml`.
- **Version provenance** (same commit) — `internal/release`: `dev` and
  `snapshot-` markers, `HasReleaseProvenance` gating release comparisons.
- **Release plumbing** (`27550f6`) — `.goreleaser.yaml` (deprecation-free,
  `release.mode: replace`, checksums, installers as assets), `cliff.toml`,
  curated `CHANGELOG.md` seed, the release-notes extractor whose awk program
  is byte-identical between Makefile and CI, and the `release`,
  `warm-module-proxy`, `lint`, `release-contract` targets.
- **CI release checks** (`049a873`) — read-only workflow: pinned GoReleaser
  `goreleaser check` gate, contract tests, changelog contract.
- **Installers** (`e34f5fb`) — `scripts/install.sh` and `scripts/install.ps1`
  verifying SHA-256 before extraction, contract tests with local fixtures (no
  network), the `windows-installer` CI job under PS 5.1 + pwsh, and the POSIX
  contract wired into the ubuntu job.
- **Self-update** (`dfd26b0`) — `dforge update`: provenance gate first, latest
  release resolution, checksum verification, atomic replacement with a
  rename-aside dance for Windows.
- **Runbook and docs** (this slice) — `RELEASING.md`, this note, Task Notes,
  README refresh.

## Why

Issue #5 (owner-approved) with four decisions: local publishing only (CI never
publishes), GitHub releases + installers + `go install` with module-proxy
warm-up, checksums-only integrity, and full self-update in v1. Reference
architectures: `dflow` and `gojira`.

## Verification

- Every slice was frozen and reviewed natively before its commit; all five
  review verdicts were approved. Slice 5 required one bounded correction
  (79/80 diff lines): `os.Rename` over a running Windows executable always
  fails, fixed with the rename-aside dance and validated by the provider's
  targeted validator. Detail in `odd/tasks/releasing-layer.md`.
- `make check` green throughout; installer contract tests green (12 cases);
  the awk byte-identity diff between Makefile and CI re-verified empty.
- The `.ps1` files were never executed on this host (no pwsh installed); the
  `windows-installer` CI job is their first real run under both engines.

## Notes

- The owner approved two budget exceptions during the feature (installers
  slice: 798 functional lines tests-inclusive; self-update slice: 1121) —
  production-only counts stayed well under the 600-line budget.
- Known follow-up from review (non-blocking): download buffering is uncapped
  and integrity is checksums-only by decision — both documented in the
  review receipts.
