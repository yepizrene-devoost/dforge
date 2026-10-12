# readme-install-docs

Installation documentation in `README.md` reflecting the current project state.
Issue #9 · Branch `bugfix/readme-install-docs` (from `develop` via `dflow start`) · Type: chore

## Stages

- [x] **README documentation** — `## 📦 Installation` added: one-line installer
  (`curl … | sh` / `irm … | iex`, the exact commands exercised against the
  published `v0.1.0` release), installer options (`DFORGE_VERSION`,
  `DFORGE_INSTALL_DIR`), `go install`, manual download with checksum
  verification, and `dforge update` for keeping current. Status note updated:
  `v0.1.0 — first public release` (was `pre-release`). Changes live on the
  tree, uncommitted, for owner review.
- [x] **Task record and branch note** — this file and
  `docs/branches/readme-install-docs.md` at the release slug.
- [x] **README restructure** — design content moved to the new
  `docs/ARCHITECTURE.md` (problem, scope, non-goals, two independent axes —
  ASCII diagram replaced by a table, ownership boundary with `dflow`,
  configuration, generated layout, value). README keeps: banner (restored
  byte-exact after a transcription slip, verified by diff against HEAD),
  tagline, status, installation, documentation map (ARCHITECTURE first row),
  roadmap, license. Prose relocated between documents; decisions 0001–0012
  remain canonical. No external anchor links existed to README sections.
- [ ] **Commit and finish (owner confirmation pending)** — conventional commit
  on this branch and `dflow finish` to `develop` (auto target; no promotion to
  `main` this cycle, per the owner).

## Evidence

- Documented commands match the published artifacts: asset names
  (`dforge_<OS>_<arch>.tar.gz|zip`), env vars (`DFORGE_VERSION`,
  `DFORGE_INSTALL_DIR`) and install dirs (`$HOME/.local/bin`,
  `%LOCALAPPDATA%\Programs\dforge`) cross-checked against `scripts/install.sh`,
  `scripts/install.ps1` and `.goreleaser.yaml`.
- The `curl …/releases/latest/download/install.sh | sh` command was exercised
  by the owner against the published release (Darwin/arm64, checksum verified).

## Commit identity

Recorded in Engram after the commit, not in this file.
