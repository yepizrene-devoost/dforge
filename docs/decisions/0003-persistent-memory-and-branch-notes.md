# 0003 — Persistent memory contract and branch notes

- **Status:** accepted
- **Date:** 2026-10 (design phase)

## Context

Two facts from the inventory shape this decision.

**`.agents/MEMORY.md` exists in 7 projects and is tracked by git in all 7** —
a legacy Laravel application, several other Laravel applications, a Laravel API
and web application, an Astro static site, and `dflow`. It is
already a committed artifact, not a local
one.

**The durable memory story is uneven.** The Go TUI CLI has no committed memory at all —
its `.agents/` contains only `workflows/` — and relies on Engram as its durable
store. The Laravel application that keeps per-concern quality rule files keeps its
backlog in `docs/tasks.md`. The Laravel POS application and
the React SPA have none, though the Laravel POS application has something better in
`docs/decisions/OD-001…015`. Someone cloning the Go TUI CLI without the author's Engram
finds no recorded decision anywhere.

Engram is available to exactly one developer, on one machine, and only when it is
running. That asymmetry is the reason this record exists.

## Decision 1 — Committed memory is mandatory and always repository-scoped

`.agents/MEMORY.md` is always required. It is the only memory artifact a teammate
or an agent can rely on without any local tooling.

The contract belongs to the repository because the reader is, by definition, the
person who **does not** have the author's local tools. A contract that only one
developer can satisfy is not a contract.

## Decision 2 — Engram is detected, never declared

```yaml
memory:
  committed: required      # repository contract — always
  local:
    hint: auto             # detected at runtime, never configured in the repo
```

Engram's availability is a **developer-local fact, not a repository fact**.
Declaring it in a committed `.dforge.yaml` would mean a colleague without Engram
clones the repository and begins by failing a non-negotiable rule, for a tool they
do not have. That erodes the authority of every other rule in the file.

The repository declares only the portable contract. Engram appears in the
generated harness as a detected capability. For an author who has it, the
practical behaviour is `both`; for everyone else it degrades to the file with no
warning and nothing missing.

## Decision 3 — The flow is one-way

```text
Engram                →    .agents/MEMORY.md      →    AGENTS.md
working log                curated digest              session-start block
local, dense,              committed, portable,        "if Engram is
ephemeral, searchable      self-contained              available, read it"
```

**Engram → `MEMORY.md`, never the reverse.** `MEMORY.md` is not a cache of
Engram; it is its published distillation.

This is already an idiom in the organization: in the Go TUI CLI and the legacy Laravel application,
`HISTORY.md` is a curated narrative changelog, distinct from the generated
`CHANGELOG.md`. Same relationship, different artifact pair.

## Decision 4 — Three artifacts at task close, with distinct scopes

This is the rule that prevents the failure mode observed in a Laravel + Inertia application under company project management.

| Artifact | Scope | Contents | Lifetime |
| --- | --- | --- | --- |
| `docs/branches/<slug>.md` | **one branch** | what changed, why, verification evidence, files touched | closes with the branch |
| `.agents/MEMORY.md` | **outlives the branch** | durable decisions, non-obvious traps, standing constraints, conventions that were settled | pruned; hard cap |
| Engram | everything, unfiltered | whatever is not worth committing | local |

**The test that makes it checkable:** *if it only makes sense to someone who read
that pull request, it belongs in the branch note, not in `MEMORY.md`.*

### The observed failure mode

In one Laravel + Inertia application under company project management,
`.agents/MEMORY.md` reached **685 lines** under a section headed
`## Ticket Notes`, containing 15+ sections of the form `### PROJ-162-design-choice`
— one per ticket, never pruned. The heading convention is exactly what
its own `AGENTS.md` mandates. What failed is the **content granularity**: the
file became a per-ticket changelog, duplicating the branch notes.

A heading convention does not prevent this. A line cap plus an explicit
"decisions, not changelog" rule does.

## Decision 5 — The file must be self-contained

Because the reader is the person without Engram:

- No `engram://` references.
- No memory observation identifiers (`mem_get_observation(1234)`).
- No "see my local memory".

`dforge check` rejects these mechanically. This rule exists **only** because
Engram is single-developer today, and it is what protects the rest of the team.

## Decision 6 — Format contract

```yaml
memory:
  committed: required
  committed_path: .agents/MEMORY.md   # override allowed: the Laravel POS application → docs/decisions/
  content: decisions                  # NOT a changelog
  self_contained: true
  max_lines: 400                      # a Laravel + Inertia application under company project management has 685 → forces pruning
  format:
    required_sections: [Repository Guidance, Workflow Decisions, Branch Notes]
    branch_heading: "### <branch-slug-without-type>"
```

`committed_path` accepts an alternative digest destination.
`docs/decisions/OD-001…015`, in the Laravel POS application, **already is** a well-structured durable
decision record; forcing a parallel `MEMORY.md` there would duplicate it rather
than consolidate.

`max_lines: 400` is a curation budget with teeth. The cap is what turns
"keep it concise" — a rule one Laravel + Inertia application under company project management already had and still reached 685 lines —
into something enforcement can see.

## Decision 7 — Branch notes and their index

Convention, already established across 6 projects with **290 files**:

```text
docs/branches/<branch-slug>.md      created or updated when the branch
                                    changes application behaviour
docs/README.md                      gains the note under a `Branch Notes` section
```

`dflow` does not read or write `docs/branches/` — verified: its `report.go` holds
shared JSON shapes, not a command. This is unclaimed territory with the largest
existing manual workload, which makes it the highest-value thing `dforge`
automates.

Two naming variants exist today (`<branch-slug>.md` and `ARA-XXXX-slug.md`). For
`github` and `jira` profiles the ticket key is part of the slug; for `autonomous`
it is not. The branch slug is the join key between the branch, its note, and its
`MEMORY.md` heading, so it must be derived, not hand-typed.

## The generated session-start block

One `AGENTS.md` serves both kinds of developer:

```markdown
## Session Start

1. Read `.agents/MEMORY.md` — the repository contract. **Always.**
   It is the context base for any agent or person without access to
   another developer's local tooling.
2. If Engram is available in this environment, consult the project's recent
   context for decisions not yet distilled into the file. If it is not,
   continue with step 1: nothing mandatory is missing.
3. Check `docs/README.md` → Branch Notes for work in progress.
```

Step 1 never depends on step 2. The same file is correct for an author with
Engram and for a teammate without it.

## Consequences

- Onboarding asks about memory only to confirm `committed_path`; there is nothing
  to choose about `committed`.
- `dforge check` needs three memory validations: file exists and is non-empty,
  line count within cap, and no local-tool identifiers.
- A `dforge memory digest` command is **not** part of this decision. Whether an
  author distills Engram into the file by hand or with help is a personal
  workflow; `dforge` owns the shape and the verification.

## Evidence

- 7 tracked `.agents/MEMORY.md` files, verified with `git ls-files --error-unmatch`.
- `.agents/MEMORY.md`, in a Laravel + Inertia application under company project management — 685 lines, `## Ticket Notes` with 15+ `### LOY-…` sections.
- `.agents/rules/code-conventions.md:34-35`, in a Laravel + Inertia application under company project management — the branch-note rule and the `docs/README.md` index requirement.
- `AGENTS.md`, in the Go TUI CLI — Engram as durable memory; its `.agents/` contains only `workflows/`, no `MEMORY.md`.
- Branch notes: 174 in the legacy Laravel application, 56 in a Laravel API and web application, 41 in a Laravel + Inertia application under company project management, 15 in the Astro static site, 3 in a Laravel application, 1 in another Laravel application.
- `cmd/commands/report.go` — shared JSON shapes, not a branch-note command.
- `docs/decisions/`, in the Laravel POS application — `OD-001`–`OD-015` decision records.
