# Decision records

Long-lived decisions for `dforge`. Each record states the decision, the context
that forced it, what it costs, and the evidence behind it. Records are not
rewritten when a later decision replaces them — a superseding record is added and
the old one is marked `superseded`.

| # | Title | Status |
| --- | --- | --- |
| [0001](0001-scope-and-ownership.md) | Scope, autonomy, and the `dflow` boundary | accepted; Decision 1 superseded in part by 0006 |
| [0002](0002-governance-profiles.md) | Governance profiles and derived commit conventions | accepted |
| [0003](0003-persistent-memory-and-branch-notes.md) | Persistent memory contract and branch notes | accepted |
| [0004](0004-foundation-discipline.md) | Foundation discipline (PRD-first) | accepted |
| [0005](0005-configuration-schema.md) | `.dforge.yaml` schema | accepted |
| [0006](0006-adopting-existing-repositories.md) | Adoption of existing repositories | accepted |
| [0007](0007-domain-separated-harness.md) | Domain-separated harness artifacts | accepted |
| [0008](0008-committed-vs-local-artifacts.md) | Committed versus machine-local generated artifacts | accepted |
| [0009](0009-build-scope-and-sequencing.md) | Build scope, sequencing, and why `dforge` is not a subcommand | accepted |
| [0010](0010-enforcement-surface.md) | Enforcement surface: who gets stopped, and by what | accepted |
| [0011](0011-deterministic-verification.md) | Deterministic verification versus AI judgement | accepted |

## Conventions

- **This repository is open source.** Never name an internal project,
  repository, organisation or module path. Cite evidence as a repository-relative
  path **without the project prefix** (`.agents/rules/code-conventions.md:43`, not
  `some-project/.agents/rules/code-conventions.md:43`) and describe the project by
  its role — "a Laravel + Inertia application", "the Go TUI CLI" — never by its
  name. The mapping from each role to the real repository is recorded in Engram,
  not here.
- **The one exception is another open-source tool.** A tool that is itself
  released as open source may be named, because naming it discloses nothing and a
  boundary is unintelligible without the name of the thing on the other side. The
  test is whether the named thing is **already open source**, not whether it
  belongs to the same organisation: `dflow` qualifies, the application projects do
  not.
- **Anonymising the actor is not permission to soften the finding.** Numbers,
  magnitudes and mechanisms stay exact. The evidence is what makes these records
  credible; the identity of the witness is not part of the argument.
- `NNNN-kebab-case-title.md`, numbered in creation order, never renumbered.
- **Status** is one of `proposed`, `accepted`, `superseded`, `rejected`.
- A record **superseded in part** keeps its original text. The status names the
  superseding record and the amended decision, so the history stays readable and
  nobody re-argues a settled point from a stale copy.
- **Evidence** cites a repository-relative `path:line`. A decision without
  evidence is an opinion; the inventory exists so that these decisions do not
  have to be re-argued from memory.
- Open questions stay in the record as `Open` until resolved. A record is not
  accepted while it hides an unresolved choice.
