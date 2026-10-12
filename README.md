<div align="center">
  <pre>
██████╗ ███████╗ ██████╗ ██████╗  ██████╗ ███████╗
██╔══██╗██╔════╝██╔═══██╗██╔══██╗██╔════╝ ██╔════╝
██║  ██║█████╗  ██║   ██║██████╔╝██║  ███╗█████╗  
██║  ██║██╔══╝  ██║   ██║██╔══██╗██║   ██║██╔══╝  
██████╔╝██║     ╚██████╔╝██║  ██║╚██████╔╝███████╗
╚═════╝ ╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝
  </pre>
</div>

<!-- The two trailing spaces on the 3rd and 4th art lines are padding, not
     noise: they keep all six art lines the same width, which is what makes the
     centering align the letters. A whitespace-stripping hook would silently
     break the banner. -->

# dforge

> Project bootstrap for AI-agent harnesses and engineering policy.

One command prepares a repository so that any agent — and any developer — starts
from the same rules, the same documentation skeleton, and the same definition of
"done". The engineering core is stack-agnostic; everything that depends on the
stack lives in a preset.

> **Status: v0.1.0 — first public release.** The releasing layer (version
> provenance, local publishing via GoReleaser, checksum-verifying installers,
> and self-update) is implemented, reviewed, and published. Phase 1 —
> `dforge check` — is next.

For the problem it solves, its scope and boundaries, and the shape of what it
generates, read [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md). The
[decision records](docs/decisions/README.md) are canonical.

## 📦 Installation

### ⚡ Option 1: One-Line Installer (Recommended)

#### 🐧 Linux and macOS

```bash
curl -fsSL https://github.com/yepizrene-devoost/dforge/releases/latest/download/install.sh | sh
```

Resolves the latest release, downloads `dforge_<OS>_<arch>.tar.gz` together
with its published SHA-256 checksums, verifies the archive before touching
your system, and installs the binary to `$HOME/.local/bin` without `sudo`. If
that directory is not on your `PATH` yet, the installer prints the exact line
to add.

#### 🪟 Windows (PowerShell)

```powershell
irm https://github.com/yepizrene-devoost/dforge/releases/latest/download/install.ps1 | iex
```

Runs the same flow for `dforge_Windows_<arch>.zip`, verifies the SHA-256
checksum, and installs to `%LOCALAPPDATA%\Programs\dforge`. It adds that
directory to your user `PATH`, so open a new terminal (or sign out and back
in) for the change to take effect.

#### ⚙️ Installer options

- `DFORGE_VERSION` pins a version instead of installing the latest release.
  Both installers accept `0.1.0` and `v0.1.0`.
- `DFORGE_INSTALL_DIR` overrides the install directory. Shell installer only;
  defaults to `$HOME/.local/bin`.

### 🛠 Option 2: Go Install

```bash
go install github.com/yepizrene-devoost/dforge@latest
```

### 📥 Option 3: Manual Download

Grab the archive for your platform from the
[Releases page](https://github.com/yepizrene-devoost/dforge/releases), verify
it against the published `checksums.txt`, extract it, and move the binary onto
your `PATH`.

### 🔄 Keeping up to date

Installed binaries update themselves with `dforge update`: it resolves the
latest release, verifies the checksum, and replaces the binary atomically.
Development and snapshot builds refuse to self-update.

## Documentation map

| Document | Contents |
| --- | --- |
| [ARCHITECTURE](docs/ARCHITECTURE.md) | The problem, scope, boundaries and generated layout — the why and the how |
| [RELEASING](RELEASING.md) | How a release is published: local `make release`; CI never publishes |
| [decisions/0001](docs/decisions/0001-scope-and-ownership.md) | Scope, autonomy, and the `dflow` boundary |
| [decisions/0002](docs/decisions/0002-governance-profiles.md) | Governance profiles and derived commit conventions |
| [decisions/0003](docs/decisions/0003-persistent-memory-and-branch-notes.md) | Memory contract and branch notes |
| [decisions/0004](docs/decisions/0004-foundation-discipline.md) | Foundation discipline (PRD-first) |
| [decisions/0005](docs/decisions/0005-configuration-schema.md) | `.dforge.yaml` schema |
| [decisions/0006](docs/decisions/0006-adopting-existing-repositories.md) | Adopting existing repositories |
| [decisions/0007](docs/decisions/0007-domain-separated-harness.md) | Domain-separated harness artifacts |
| [decisions/0008](docs/decisions/0008-committed-vs-local-artifacts.md) | Committed versus machine-local generated artifacts |
| [decisions/0009](docs/decisions/0009-build-scope-and-sequencing.md) | Build scope, sequencing, and the kill criterion |
| [decisions/0010](docs/decisions/0010-enforcement-surface.md) | Enforcement surface: who gets stopped, and by what |
| [decisions/0011](docs/decisions/0011-deterministic-verification.md) | Deterministic verification versus AI judgement |
| [decisions/0012](docs/decisions/0012-issue-linked-pull-requests.md) | Issue-linked pull requests and the issue lifecycle |

## Roadmap

Sequencing and the kill criterion that governs it:
[0009](docs/decisions/0009-build-scope-and-sequencing.md).

**Phase 0 — no code.** Two or three template repositories, one per stack, holding
the core harness and a short `AGENTS.md`. Solves "a new project starts from zero"
with zero code and zero maintenance, and authors the reference content once, in a
place a human can read.

**Delivered before Phase 1 — the release layer.** Version provenance (`dev` /
`snapshot-*` / tag markers), local publishing through GoReleaser, read-only CI
release checks, checksum-verifying installers, and `dforge update` — the
mechanism this repository used to ship itself. See [RELEASING](RELEASING.md).

**Phase 1 — `dforge check`.** A single binary that only **verifies**: commit format
against the derived profile, functional lines against the review budget, memory
digest and branch note presence, the ignore region, gate and domain consistency,
and exceptions or advisory rollouts whose `review_by` has passed — it **fails** on
these rather than warning about them. It generates nothing. Small, testable, and
the only thing that stops the drift.

**Phase 2 — generation and adoption.** Embedded asset trees with idempotent,
marker-aware writes, plus the reconciliation planner. Only if Phase 1 survives.
This is the one genuinely new component: the organization has no `embed.FS` usage
today.

> **Kill criterion.** If, three months after Phase 1, `check` is not running in the
> CI or pre-commit path of at least three projects, the generator is not built. A
> verification tool that nobody runs has already proved the organization will not
> maintain the contract.

Commit, push, pull request and release decisions are always human.

## License

MIT — see [LICENSE](LICENSE).
