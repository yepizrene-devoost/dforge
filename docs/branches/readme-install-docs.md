# readme-install-docs

Branch `bugfix/readme-install-docs` · Issue #9 · Type: chore

## What changed

`README.md` becomes an orient-and-install document: a new `## 📦 Installation`
section (three install paths, installer options, `dforge update`), the status
note moving from `pre-release` to `v0.1.0 — first public release`, and every
design section relocated to the new `docs/ARCHITECTURE.md` (problem, scope,
non-goals, two independent axes, ownership boundary with `dflow`,
configuration, generated layout, value). The ASCII axes diagram becomes a
table. `README.md` keeps banner, tagline, status, installation, documentation
map (ARCHITECTURE.md first row), roadmap and license. The banner block was
restored byte-exact after a transcription slip, verified by diff against HEAD.

Documentation only: no rule, boundary, policy or behaviour changes — the
decision records remain canonical. Not promoted to `main` this cycle — the
instructions reach the published page with the next release promotion.

## Why

A newcomer had no install path; the owner installed the tool by mirroring the
`dflow` command. The `README` must reflect the project's current state.

## Verification

- Documented commands cross-checked against `scripts/install.sh`,
  `scripts/install.ps1` and `.goreleaser.yaml` (asset names, env vars, install
  dirs).
- The one-line installer command was exercised by the owner against the
  published `v0.1.0` (Darwin/arm64, checksum verified, installed to
  `~/.local/bin`).
