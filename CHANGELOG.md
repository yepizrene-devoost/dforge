# Changelog

This file is generated and curated: `make changelog` renders the unreleased
Conventional Commits into `CHANGELOG.draft.md` with git-cliff, and a human
curates that draft into the `## 📦 vX.Y.Z – Title` sections below. The draft is
gitignored and never committed; only the curated sections live here.

`make release` extracts the newest `## 📦` section verbatim as the GitHub
release body, so every section is re-curated on the release branch before the
tag is created — its heading must name the exact version being tagged.

## 📦 v0.1.0 – First public release

- **Version reporting** — `dforge version` (plain, `--json`) reports the
  build-time version marker plus the commit the binary was built from, stamped
  by Go's own VCS info; development builds keep the honest `dev` marker instead
  of claiming a released version.
- **Local publishing** — `make release` publishes a GitHub release from an
  exact local tag with GoReleaser: validated release body extracted from this
  file, preflighted `.env` token and tag, cross-platform archives
  (Linux/macOS/Windows, amd64/arm64) with a `checksums.txt` manifest, and a
  post-publish module-proxy warm-up.
- **Installers** — POSIX and Windows installers fetch the release archive and
  verify its SHA-256 against `checksums.txt` before installing, without sudo.
- **Self-update** — `dforge update` upgrades an installed binary in place with
  release discovery, checksum verification and atomic replacement; development
  and snapshot builds are refused by the release-provenance gate.
