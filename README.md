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

> **Status: design phase.** This repository holds decisions and specifications,
> not implementation. No `dforge` binary exists yet. See
> [`docs/decisions/`](docs/decisions/README.md) and the
> [build order](docs/decisions/0009-build-scope-and-sequencing.md).

## The problem this solves

An inventory of the 20 projects in the workspace found the same policies written
by hand, five times, and drifting apart:

| Symptom | Measured |
| --- | --- |
| Competing rule directories | `.agents/workflows/`, `.agents/quality/`, `.agents/rules/`, `.ai/rules/`, `.antigravity/`, plus `rules.md` and `style.md` at repo root |
| Commit message formats in use | 3 incompatible (`type(scope): desc`, `TICKET-ID: type(scope) - desc`, `type(scope) - desc`) |
| Branch models in use | 4 different base/merge conventions |
| TDD strictness | "no exceptions" vs "near-strict" vs `strict_tdd: false` |
| Branch notes | **290 files** across 6 projects, with zero tooling creating or indexing them |
| Duplicated documentation | The Laravel Sail README is copy-pasted across projects: a Laravel API under Jira management and the team-managed Laravel application repeat the same sections at the same line numbers |
| Memory coverage | 7 projects have a committed `MEMORY.md`; the rest have none |

The rules are not missing. They are **unowned and unverified**, so each project
re-derives them, and the copies drift.

The drift is also **temporal, not project-specific**. The projects were built by
hand over a long period, and harness quality tracks *when* a project was created
rather than what it is: knowledge accumulated continuously since the organization
adopted AI tooling, so newer projects are materially better structured than older
ones. Left alone, that gradient is permanent — the projects that predate the
knowledge keep the weakest rules forever, and the rules that do exist are the ones
written when the least was known.

**Project identities are withheld.** The evidence comes from private repositories,
so each finding names a project by its role rather than its name. The identities
are recorded privately, not here.

## Scope

`dforge` generates and maintains:

- Agent instruction files (`AGENTS.md`, `CLAUDE.md`) with delimited, idempotent
  sections that other tools can share.
- The engineering core: code conventions, strict SOLID, strict ACID, strict TDD,
  review budgets, domain separation.
- The persistent memory contract (`.agents/MEMORY.md`) and branch notes
  (`docs/branches/`).
- The documentation skeleton: `docs/README.md` index, `decisions/`, `domains/`,
  `features/`.
- Governance profile: commit convention derived from how the project is managed.
- Verification of all of the above, so the policy is checkable rather than
  aspirational.
- **Adoption of existing repositories.** A detection-driven reconciliation plan
  that indexes rules it does not own, never clobbers a project's own rules, and
  records dated, attributed exceptions where a core rule cannot be met yet.

## Non-goals

`dforge` explicitly does **not** touch:

- **Branch lifecycle.** When a project declares a branch provider, that tool owns
  branch types, bases, merge rules and the finish flow; `dforge` never declares
  branch names. When a project declares none, `dforge` states the invariant and
  nothing else.
- **Stack-specific architecture.** Directory layout, framework boundaries and
  domain code structure are preset concerns, not core concerns.
- **Business content.** PRDs, business rules, design systems and brand identity
  belong to each project.
- **Infrastructure and deployment.** CI pipelines, Docker and hosting are
  project-specific.
- **Per-project compatibility matrices.** `dforge` stays autonomous: it holds no
  knowledge of any specific repository, so adoption is detection-driven rather
  than a list of known exceptions.
- **Cleaning up a repository.** Stray scripts and loose artifacts are not
  `dforge`'s concern. A bootstrap tool that starts deleting files nobody asked it
  to delete is a hazard, not a convenience.
- **Overriding a team's flow decision.** Adoption is a rollout, not an edit.
  `dforge` may report that a project does not follow the organization's model;
  switching that project is a conversation with its team, not an `--apply`.

## Two independent axes

The governing insight from the inventory: **how a project is built** and **how it
is governed** are orthogonal. The Laravel application that keeps per-concern
quality rule files, a Laravel application and the Laravel POS application are
Laravel/React projects governed autonomously; a Laravel API under Jira management
is a Laravel project governed through Jira. Stack does not predict governance.

```text
                    governance profile
                    jira | github | autonomous
                            │
   preset                   │
   laravel-modern-web ──────┼────── profile: autonomous
   go-cli ──────────────────┼────── profile: github
   flutter-mobile ──────────┼────── profile: jira
```

- **Preset** (stack) — test runner, linter, package manager, entrypoints,
  directory layout, stack skills.
- **Profile** (governance) — where work items live and therefore what a commit
  must reference.

## Ownership boundary with `dflow`

| Asset | Owner | Rule |
| --- | --- | --- |
| `.dflow.yaml` | `dflow` | `dforge` reads it if present; never writes it |
| `.agents/workflows/dflow.md` | `dflow` | `dforge` neither touches nor duplicates it |
| `<!-- dflow:workflow-reference -->` block in `AGENTS.md` | `dflow` | `dforge` writes only its own delimited block |
| Branch lifecycle (`start`/`finish`/`status`/`delete`) | `dflow` | never reimplemented |
| Branch names and bases | `dflow` | `dforge` declares the invariant, not the names |
| `.dforge.yaml` | `dforge` | single source of truth |
| `.agents/harness/*` | `dforge` | derived from the schema |
| `.agents/MEMORY.md` | `dforge` | contract, template and verification |
| `docs/**` | `dforge` | `docs/branches/` is unclaimed by `dflow` |

`AGENTS.md` is the only shared file. `dforge` reuses the marker semantics already
implemented and tested in `dflow/pkg/agent/reference.go`: replace between HTML
comment markers, append when there are none, and **change nothing** when a
marker is orphaned or a legacy heading is found.

## Configuration

A single `.dforge.yaml` at the repository root, mirroring the `.dflow.yaml`
precedent already present in 15 of 20 projects. It is the complete record of the
onboarding: `dforge sync` can regenerate everything from it without asking again.

Schema and rationale: [0005-configuration-schema](docs/decisions/0005-configuration-schema.md).

## Generated layout

```text
AGENTS.md                    Tier 0 gate (generated, budgeted) + local rules
.gitignore                   dforge region for machine-local paths (delimited)
.dforge.yaml                 single source of truth
.agents/
  MEMORY.md                  committed memory digest (mandatory, self-contained)
  harness/
    index.md                 generated: moment/trigger → file
    core/                    generated: conventions, solid, acid, tdd,
                             review-budget, commits, branches, memory, docs, domains
    presets/<stack>/         generated: stack depth, same frontmatter contract
    project/                 never written by dforge; exists only if the
                             project has rules of its own
  workflows/dflow.md         GENERATED BY DFLOW — never touched
  skills/
docs/
  README.md                  index + Branch Notes
  branches/<slug>.md         one note per branch
  domains/<slug>/README.md   one folder per business domain
  decisions/                 long-lived decision records
```

Layout rationale: [0007](docs/decisions/0007-domain-separated-harness.md).

## Documentation map

| Document | Contents |
| --- | --- |
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

## Value: verification, not generation

Generation is the cheap half of the value and the expensive half of the work. A
template repository can seed a new project; it cannot retrofit the twenty that
already exist, it never runs again after day one, and it cannot derive a commit
convention from how a project is governed. Those three capabilities are why
`dforge` exists, and none of them is generation. See
[0009](docs/decisions/0009-build-scope-and-sequencing.md).
