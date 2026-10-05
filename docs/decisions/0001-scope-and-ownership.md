# 0001 — Scope, autonomy, and the `dflow` boundary

- **Status:** accepted; Decision 1 and consequence 2 superseded in part by [0006](0006-adopting-existing-repositories.md); the boundary with `dflow` clarified by [0009](0009-build-scope-and-sequencing.md)
- **Date:** 2026-10 (design phase)

## Context

`dforge` was designed after an inventory of the 20 projects in a single
development workspace, covering their agent-harness files, documentation
and process artifacts, and stack/tooling manifests. The inventory had one
purpose: separate what is transversal from what belongs to each project, and
establish what is `dflow`'s property versus `dforge`'s.

The inventory also surfaced an existing second writer. `dflow` — a Go CLI whose
stated purpose is *"Manage Git branches with a configurable workflow"* — also
publishes its own interface for agents: `dflow init` writes
`.agents/workflows/dflow.md` and injects a reference block into `AGENTS.md`.
15 of the 20 projects already carry a `.dflow.yaml`.

That output is **tool-usage documentation, not engineering policy** — it exists so
agents know how to use that tool. [0009](0009-build-scope-and-sequencing.md) fixes
the boundary as one responsibility per tool.

## Decision 1 — `dforge` is autonomous

`dforge` has **no knowledge of any existing project** and operates on any
directory, empty or occupied. It contains no project-specific defaults, no
exception list, and **no per-project compatibility matrix**.

> **Amended by [0006](0006-adopting-existing-repositories.md).** The original
text of this decision also excluded adoption outright. `dforge` now supports
adopting existing repositories through detection-driven reconciliation. The
reasoning below still governs *how*: without it, adoption would become the
compatibility matrix described in the next paragraph.

### Why

The inventory informed the design; it is not a deployment target. Treating it as
one would turn a bootstrap tool into a migration engine with a permanent
compatibility matrix — a different product with a different failure mode. It
would also make `dforge` unusable for the next project that does not exist yet,
which is the entire point.

### Consequences

- Presets describe **capabilities** (test runner, linter, package manager), never
  project names.
- The governance profile is answered during onboarding, not inferred from a
  repository.
- `dforge` degrades gracefully: if `.dflow.yaml` is absent it omits the workflow
  section and continues. It never fails for missing optional context, and it
  never authors a `.dflow.yaml` on the project's behalf — that would mean
  inventing a branch model.

## Decision 2 — Ownership boundary with `dflow`

| Asset | Owner | Rule |
| --- | --- | --- |
| `.dflow.yaml` | `dflow` | `dforge` reads it read-only; never writes it |
| `.agents/workflows/dflow.md` | `dflow` (`init`, `agent`) | `dforge` neither touches nor duplicates it |
| `dflow:workflow-reference` block in `AGENTS.md` | `dflow` | `dforge` writes only its own delimited block |
| Branch lifecycle (`start`/`finish`/`status`/`delete`) | `dflow` | never reimplemented |
| Branch names, bases and merge targets | `dflow` (via `.dflow.yaml`) | `dforge` declares the invariant only |
| `.dforge.yaml` | `dforge` | single source of truth |
| `.agents/harness/*`, `.agents/MEMORY.md` | `dforge` | derived from the schema |
| `docs/**` including `docs/branches/` | `dforge` | unclaimed by `dflow` |

### The invariant, not the model

`dforge` states **"no change is ever made on a base branch; every change opens a
work branch"**. It does not state which branches exist.

This split matters because the organisational drift is measurable: four different
branch models are in use, and one project's `.cursorrules` contradicts its own
`rules.md`. If `dforge` also wrote branch names, it would create a *second*
source of truth for the same fact and guarantee the same drift.

### `AGENTS.md` is the only shared file

`dforge` reuses the marker semantics already implemented and tested in
`dflow/pkg/agent/reference.go`:

- Block delimited by HTML comment markers, so a later run replaces only its own
  range.
- A file with no markers gets the block appended after existing bytes.
- **A lone marker, or a legacy heading without markers, is left untouched.**

Verification: `pkg/agent/reference.go:16-17`, `:35-39`, `:57-87`.

## Decision 3 — One configuration artifact

A single `.dforge.yaml` at the repository root is the sole source of truth;
everything else is generated. Rationale and rejection of the
"section inside `AGENTS.md`" alternative: [0005](0005-configuration-schema.md).

## Consequences

1. **A multi-file scaffolding engine is required.** The organization has no
   `embed.FS` usage anywhere; `dflow` generates single documents from Go string
   constants. Sowing asset trees per preset is the one genuinely new component.
2. **Adoption is detection-driven, not project-specific.** `dforge` still holds
   no knowledge of any particular repository, so it cannot be validated by
   consulting a list of known projects. *(Original consequence: "`dforge` cannot
   be validated against the existing projects; it is correct by construction for
   new projects only." Superseded in part by
   [0006](0006-adopting-existing-repositories.md).)*
3. **`docs/branches/` is the highest-value unclaimed territory.** 290 files
   across 6 projects are maintained by hand, with no tooling creating, indexing
   or validating them.

## Evidence

- `cmd/root/root.go:24` — `"Manage Git branches with a configurable workflow"`.
- `cmd/commands/init.go:418-422` — the only file `dflow init` writes under `.agents`.
- `pkg/agent/reference.go:16-17` — `referenceStartMarker` / `referenceEndMarker`.
- `cmd/commands/report.go` — not a command: shared JSON shapes. No command reads or writes branch notes.
- 17 `.dflow.yaml` files present in the tree; 15 belong to active projects. Absent in the team-managed Laravel application and in the four Flutter applications. The two remaining files sit in the archive and test directories, which are not projects.
- `.atl/skill-registry.md` is git-ignored everywhere (`.gitignore:2` in the Go TUI CLI; `.git/info/exclude:8` in the Laravel POS application), so it never travels with a repository.
