# 0002 — Governance profiles and derived commit conventions

- **Status:** accepted, one open question
- **Date:** 2026-10 (design phase)

## Context

The inventory found three incompatible commit formats in use:

| Format | Where |
| --- | --- |
| `type(scope): description` | `AGENTS.md:39`, in the Go TUI CLI; `.agents/workflows/commits.md`, in the Laravel application that keeps per-concern quality rule files |
| `TICKET-ID: type(scope) - description` | `.agents/workflows/commit-rules.md`, in a Laravel + Inertia application under company project management; `rules.md:24`, in another Laravel application |
| `type(scope) - description` | `.cursorrules`, in a Laravel application |

These are not stylistic disagreements. They track a real difference: some projects
answer to a Jira dashboard, and work items live there; others are managed
autonomously and have no external ticket. Documenting the convention as a free
choice lets a project declare a format its own management model contradicts.

The inventory also ruled out a binary model. The Laravel POS application is managed neither by
Jira nor autonomously: it is organised **inside GitHub with issues, milestones and
epics**, and its own documentation delegates to them — *"Las fases corresponden a
los milestones de GitHub. Cada epic de dominio en GitHub enlaza a su carpeta
correspondiente aquí"* (`docs/README.md`, in the Laravel POS application).

## Decision 1 — Governance is three-valued

```yaml
project:
  management: jira | github | autonomous
```

| Profile | Work item lives in | Traceability source |
| --- | --- | --- |
| `jira` | Jira | ticket key, mandatory in the commit subject |
| `github` | GitHub Issues, grouped by milestones and epics | issue number, mandatory in the commit subject |
| `autonomous` | The repository itself | a **declared task location**, plus `docs/branches/<slug>.md` |

`autonomous` does not mean "no process". It means the work item has no external
system of record, so the repository artifacts become the record. This is why
branch notes are mandatory rather than optional for these projects.

The task location is **declared, not assumed**:

```yaml
harness:
  task_location: odd/tasks/     # or docs/tasks.md, or any project file
```

One project keeps a single `docs/tasks.md`; others keep one file per work item
under `odd/tasks/`. Both are legitimate autonomous records, and `check` must not
assume a directory the project does not use. This mirrors `memory.committed_path`
in [0003](0003-persistent-memory-and-branch-notes.md): where the record lives is a
project fact, not a tool default.

### Governance is a lifecycle state, not a permanent identity

A profile changes as a project does. A pre-release project whose backlog fits in
one repository file is a different animal from the same project after its first
deployment, when the volume of work justifies an external tracker. Moving from
`autonomous` to `jira` or `github` is therefore a **schema change followed by
`dforge sync`** — never a text edit to a generated file, because the derived
commit regex, the branch-note template and the harness statements must all change
together or not at all.

## Decision 2 — The commit format is derived, never declared

`.dforge.yaml` does **not** accept a free-form `commit_format`. The format follows
from `management`:

| `management` | Derived subject | Example |
| --- | --- | --- |
| `jira` | `<PREFIX>-<n>: type(scope) - description` | `PROJ-161: fix(billing) - correct proration` |
| `github` | `type(scope): description (#<n>)` | `feat(orders): add split payment (#128)` |
| `autonomous` | `type(scope): description` | `feat(tui): support multi-word names` |

### Why derived

A declared format can contradict the management model, and nothing detects it.
A derived format can be **verified**:

```text
dforge check  →  reads project.management
              →  derives the expected regex
              →  validates the commit range
              →  fails on mismatch
```

This is the difference between a policy document and a policy. The inventory
showed the rules are already written and still diverge, so writing them again in
a better layout changes nothing.

### Declared versus observed

`management` is declared, so it can be wrong. `check` therefore compares the
declaration against the history it can observe: commits matching none of the
derived patterns, a `jira` project whose subjects carry no ticket key, a `github`
project whose subjects carry no issue number, or a project declaring a branch
provider it visibly does not follow. A mismatch is a finding, never a silent
pass. A harness that trusts its own configuration file is worth nothing.

An explicit `commits.format` override remains available for genuine exceptions,
but declaring one that contradicts `management` is a validation error, not a
silent pass.

## Decision 3 — Governance is orthogonal to stack

This is the load-bearing insight, and it comes from the project list itself:

| Project | Stack | Governance |
| --- | --- | --- |
| the Go TUI CLI | Go + Bubble Tea | autonomous |
| `dflow` | Go + cobra | autonomous |
| the Laravel application that keeps per-concern quality rule files | Laravel + Inertia | autonomous |
| a Laravel application | Laravel | autonomous |
| the Laravel POS application | Laravel + Inertia | github |
| the React SPA | React + Vite | autonomous |
| a Laravel API under Jira management | Laravel API | jira |
| a Laravel API under Jira management | Laravel API | jira |

Three of these are Laravel and they are not governed the same way. A preset
therefore cannot imply a profile, and a profile cannot imply a preset. They are
configured independently and validated independently.

## Consequences

- The onboarding must ask the governance profile **before** generating
  `commits.md`, `AGENTS.md` or the branch-note template, because all three embed
  the derived format.
- `jira` additionally requires `ticket_prefix`. Without it the derived format is
  unverifiable, so onboarding fails rather than guessing.
- Changing `management` after onboarding is a **re-generation**, not a text edit:
  `dforge sync` rewrites the derived artifacts.
- The harness owes the `autonomous` profile a stronger task artifact, since
  nothing external provides traceability.

## Open

**`github` subject form.** `type(scope): description (#<n>)` follows GitHub's own
squash-merge output and keeps Conventional Commits intact. The alternative —
`#<n> type(scope): description` — leads with the issue number and mirrors the
Jira shape. This record stays `accepted` with the first form as the default;
confirm before implementing `dforge check`, since the regex is the contract.

## Evidence

- `AGENTS.md:39`, in the Go TUI CLI; `.agents/workflows/commits.md:1`, in the Laravel application that keeps per-concern quality rule files — Conventional Commits.
- `.agents/workflows/commit-rules.md`, in a Laravel + Inertia application under company project management — `TICKET-ID: type(scope) - description`, explicitly overriding the `dflow` default.
- `rules.md:24`, in another Laravel application — same override, stated as a hard rule.
- `docs/README.md`, in the Laravel POS application — domains mapped to GitHub milestones; epics link to domain folders; issue templates present under `.github/`.
- `odd/tasks/`, in the Laravel POS application — `{ticket#}-{slug}.md` naming, consistent with an issue-backed work item.
