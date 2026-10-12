# Releasing dforge

How a reviewed commit on `main` becomes a published release. CI validates the
release path on every push; **CI never publishes** — a human runs
`make release` locally, on purpose (issue #5).

## Model

| Concern | Decision |
| --- | --- |
| Publisher | Human, local `make release` with a GitHub token in `.env` |
| CI role | Read-only validation: `goreleaser check`, contract tests, changelog contract (`.github/workflows/release-checks.yml`) |
| Version source | Annotated tag `vX.Y.Z` on `main`; the Makefile derives it with `git describe --tags --exact-match`, GoReleaser injects `{{ .Tag }}` |
| Commit identity | Go's `buildvcs` stamp — never injected by hand |
| Integrity | SHA-256 `checksums.txt`, verified by the installers and `dforge update`; no artifact signing |
| Changelog | git-cliff draft → human-curated `## 📦 vX.Y.Z – Title` section → awk-extracted release notes |
| Reruns | `release.mode: replace` — republishing the same tag replaces the release body and assets |

## Prerequisites

- `goreleaser` and `git-cliff` installed locally (CI pins GoReleaser `v2.18.2`).
- A fine-grained GitHub token with **Contents: write** only, in `.env` as
  `GITHUB_TOKEN`. `.env` is gitignored — never commit it.
- The `release-checks` workflow green on `main`: it validates exactly what
  `make release` will consume (the awk program is byte-identical by design).

## Runbook

1. **Branch.** From an up-to-date `develop`:
   `dflow start release v0.1.0`. Release branches finish to `develop`
   (automatic) and `main` (pull request).
2. **Curate the changelog.** `make changelog` writes `CHANGELOG.draft.md`
   (gitignored, never committed). Curate the new section into `CHANGELOG.md`
   as `## 📦 vX.Y.Z – Title` with concise bullets. Commit it.
3. **Finish.** `dflow finish` merges the release branch into `develop` and
   opens the pull request toward `main`. The pull request body references the
   tracking issue with `References #N` — never `Closes`/`Fixes`/`Resolves`
   while the issue still tracks tagging and publication.
4. **Merge the pull request** into `main` (the issue must carry
   `status:approved` — the issue gate enforces it).
5. **Tag.** On a clean local `main` at the merged commit:
   `git tag -a vX.Y.Z -m "vX.Y.Z"` and push the tag. This is its own explicit
   decision, separate from the merge.
6. **Publish.** `make release`. The target refuses to run unless: `.env` with
   `GITHUB_TOKEN` exists, `HEAD` is exactly `refs/tags/vX.Y.Z`, and the newest
   `## 📦` changelog section matches the tag version with non-blank content.
   It then runs GoReleaser (cross-compiled archives + `checksums.txt` +
   installer scripts) and warms the module proxy (`make warm-module-proxy`;
   the release never fails on warm-up — it warns and prints the retry
   command).

## Tag policy

Public tags are never moved, force-pushed, or deleted. A wrong public tag is
not corrected in place: publish a new patch release. `release.mode: replace`
exists so a *failed or interrupted* publish of the same tag can be rerun
safely — it does not license retagging.

## Self-update

Released binaries update themselves with `dforge update`: provenance gate
(refuses `dev` and `snapshot-*` builds), latest-release resolution, checksum
verification, atomic replacement (on Windows via a rename-aside dance; the
superseded binary sits at `<target>.old` until the next update removes it).
