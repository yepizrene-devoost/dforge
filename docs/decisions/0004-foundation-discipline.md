# 0004 — Foundation discipline (PRD-first)

- **Status:** accepted
- **Date:** 2026-10 (design phase)

## Context

The Laravel POS application has the strongest documentation structure in the workspace, and the
reason is a process, not a format. It is a private POS for a restaurant that is
about to open. Before the first line of code, a long planning session defined the
product and closed the decisions: a master PRD, then `OD-001` through `OD-015` —
one record per open decision. The project is still in its **foundation stage**:
almost no business logic exists yet, because persistence and the database schema
were deliberately settled first, so that no gap would propagate into the domain
layer.

The structure that resulted:

| Artifact | Role |
| --- | --- |
| `docs/PRD-MASTER.md` | canonical product truth; edited only to record a decision |
| `docs/decisions/OD-NNN.md` | one card per open decision, resolved by the owner |
| `docs/domains/<slug>/README.md` | one folder per business domain: responsibility, PRD IDs covered, related open decisions, phase |
| `docs/features/` | feature specs that cross domain boundaries |
| `docs/README.md` | the index, plus the domain-to-phase table |

And the rules that keep it honest, quoted from `docs/README.md`, in the Laravel POS application:

- The PRD *"No se edita salvo para registrar una decisión que resuelve una Open Decision"*.
- Decisions are *"se actualiza cuando el propietario del producto la resuelve; nunca se asume una respuesta implícita en una spec o en código"*.
- *"ninguna spec puede reducir o contradecir silenciosamente el PRD (§21.4)"*.
- Phases map to GitHub milestones, and each domain epic links to its folder.

## Decision

`dforge` treats this as a first-class, opt-in module, not as the Laravel POS application's private
convention:

```yaml
harness:
  discipline: prd-first | iterative
```

`prd-first` generates the skeleton above and the rules that govern it. `iterative`
generates the harness without a PRD scaffold.

### Why this is in scope

`dforge`'s stated boundary excludes *stack-specific architecture*. This module is
neither architecture nor stack-specific: it is a **documentation and process
protocol**, and the inventory shows it works — it produced the best-organized
project in the workspace. The distinction the module preserves is that `dforge`
generates the **structure and the rules**, never the content. Writing the PRD is
the human's long session; `dforge` only guarantees that when the session ends,
the place for each conclusion already exists and the rules for it are written
down.

### Why opt-in

A `prd-first` scaffold is overhead for a tool with one job, like the Go TUI CLI or
`dflow`. Forcing it everywhere would produce empty skeletons that get deleted —
worse than not generating them, because it teaches the team that generated
artifacts are noise.

## The protocol, separated from the content

| `dforge` generates | The human provides |
| --- | --- |
| `PRD-MASTER.md` template with its edit rule stated at the top | the product |
| `decisions/README.md` index and the `OD-NNN` record template | the open decisions and their resolutions |
| `domains/<slug>/README.md` template and the domain table in `docs/README.md` | the domain list |
| the rule that a spec may not silently reduce the PRD | the specs |
| the rule that persistence and schema precede logic | the schema |

The `domains/` list itself is onboarding input, not something `dforge` invents. For
a `github`-governed project the phase column maps to milestones; for other
profiles it stays a plain phase label.

## Decision — persistence precedes logic

When `discipline: prd-first`, the generated rules state that the persistence layer
and the database schema are settled before domain logic is written, and that a
gap found later is a decision record — not a silent migration.

This is the one rule in this module with a technical edge, and it is worth
stating explicitly because it is the reason the Laravel POS application has no rework: a schema
decided upfront against a complete product definition is cheaper than one grown
incrementally against an incomplete one. It is stack-agnostic: it constrains
*order of work*, not the shape of the tables.

## Interaction with other decisions

- **`domains/` vs domain code layout.** The domain **folder documentation** is
  core and stack-agnostic. The domain **directory layout in code** is a preset
  concern — `app/Domain/<X>/` in Laravel is not `internal/<x>/` in Go. Keeping
  them in different layers is what lets the documentation model be reusable
  without pretending the code structure is.
- **[0002](0002-governance-profiles.md).** `prd-first` pairs naturally with the
  `github` profile, because milestones give phases an external home. The two
  remain independent settings — neither implies the other.
- **[0003](0003-persistent-memory-and-branch-notes.md).** `OD-NNN` records and the
  `MEMORY.md` digest answer different questions: an `OD` is a decision taken
  before or across work; `MEMORY.md` accumulates durable findings during work. A
  project may legitimately use `committed_path: docs/decisions/` for its digest.

## Consequences

- Onboarding for `prd-first` must collect the domain list, which makes it a longer
  onboarding than `iterative`.
- An empty `PRD-MASTER.md` is worse than none. If `prd-first` is selected and the
  PRD is not written within the foundation stage, the generated rules require the
  project to record that explicitly rather than leave a misleading template.
- `dforge check` can validate structure — PRD exists and is non-empty, every
  domain folder has a `README.md`, every referenced `OD` exists — but can never
  validate product quality. That limit is stated in the generated rules so no one
  trusts the check further than it goes.

## Evidence

- `docs/README.md`, in the Laravel POS application — the full index: PRD-MASTER, DESIGN_SYSTEM, `domains/`, `decisions/` (OD-001–OD-015), `features/`, plus the domain-to-phase table with its GitHub-milestone mapping.
- `docs/domains/`, in the Laravel POS application — 24 domain folders: `organization`, `identity-access`, `events`, `audit`, `catalog`, `floor-tables`, `payments`, `sessions-accounts`, `orders`, `production`, `printing`, `cash-day`, `tax-billing`, `inventory`, `purchasing`, `costing`, `commercial`, `reporting`, `reliability`, `local-hybrid`, `security`, `hardware`, `ux`, `integrations`.
- `docs/PRD-MASTER.md`, in the Laravel POS application — canonical product truth, edit-restricted by rule.
