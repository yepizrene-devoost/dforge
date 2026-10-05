# 0006 — Adoption of existing repositories

- **Status:** accepted
- **Date:** 2026-10 (design phase)
- **Supersedes:** [0001](0001-scope-and-ownership.md) Decision 1, in part, and its consequence 2

## Context

It is a requirement that `dforge` both **create new repositories** and **adopt
existing ones**. `0001` had excluded migration; this record reverses that part
while keeping the reasoning that produced it.

That reasoning still holds: a migration engine with a per-repository compatibility
matrix is a different product with a different failure mode. So adoption must be
**detection-driven** — no project names, no exception list, no hardcoded
knowledge of any repository. Autonomy survives; only the capability is added.

### What adoption has to reconcile

| Condition | Evidence |
| --- | --- |
| Six locations already hold rule content | `.agents/workflows/`, `.agents/quality/`, `.agents/rules/`, `.ai/rules/`, `.antigravity/`, root `rules.md` / `style.md` |
| A repository's own rules contradict each other | one Laravel application's `.cursorrules` says "features base on `uat`, merge to `develop`"; its own `rules.md` says otherwise |
| A non-negotiable is declared off | `openspec/config.yaml:16`, in the legacy Laravel application — `strict_tdd: false` |
| A non-negotiable is declared loosely | `.agents/rules/code-conventions.md:43`, in a Laravel + Inertia application under company project management — SOLID "pragmatically" |
| The directory `dflow` owns is already shared | `.agents/workflows/`, in the team-managed Laravel application, holds three hand-written workflows next to where `dflow` writes `dflow.md` |
| A project with no harness at all | the team-managed Laravel application: no `AGENTS.md`, no `CLAUDE.md`, no `.dflow.yaml`; rules in `.antigravity/rules.md`, and ~15 ad-hoc `test_*.php` scripts loose at the repository root |

### Why retrofitting matters

The projects were built by hand, incrementally, and their harness quality is a
function of **when** they were created rather than what they are. Harness
knowledge accumulated continuously since the organization adopted AI tooling, so
newer projects are materially better structured than older ones. The drift is
**temporal, not project-specific** — and it compounds, because every new project
re-derives what the previous one had already learned.

Adoption is the mechanism that pushes today's accumulated standard backwards into
the projects that predate it. Without it, the organization keeps a permanent
gradient in which the oldest code carries the weakest rules, and the rules that do
exist are the ones written when the least was known.

### The transition problem

Several projects are maintained by teams that decide their own branching flow,
and at least one is not following the organization's branch tooling at all.
Authority to add rules is not in question; what is in question is **when** a rule
becomes binding for a team that is still working by hand.

A harness installed at full enforcement into a team that has not adopted it gets
switched off the first time it blocks legitimate work — and then the whole
harness is discredited, not just the one rule. Adoption therefore needs a rollout
state, and the branch model needs the option of being absent.

## Decision 1 — Two verbs, one schema

| Verb | Target | Behaviour |
| --- | --- | --- |
| `dforge init` | new or empty repository | writes the full harness |
| `dforge adopt` | existing repository | detects, reconciles, writes only what is safe |

Both read and write the same `.dforge.yaml`. Adoption is not a special mode with
its own configuration.

## Decision 2 — Adoption is read-only until applied

`dforge adopt` prints a reconciliation plan and **changes nothing**. Writing
requires an explicit `--apply`.

Precedent: `dflow init` refuses to overwrite a hand-edited config without
`--force` (`cmd/commands/init.go:95-105`). Adoption inherits that
discipline rather than trusting the operator to be careful.

## Decision 3 — Five reconciliation classes, fixed actions

Every detected path falls into exactly one class, and the action is not
negotiable:

| Class | Detection | Action |
| --- | --- | --- |
| `absent` | target does not exist | create |
| `identical` | content equals what would be generated | no-op |
| `mergeable` | `dforge` markers already present | replace between the markers only |
| `foreign` | rule content in a location `dforge` does not own | **index it; never move or rewrite it** |
| `conflicting` | existing normative text contradicts a core rule | **stop; one human decision per conflict** |

`foreign` and `conflicting` are the classes that make adoption safe.

`foreign` matters because the six existing rule locations are not junk — they are
somebody's decisions, and several are better than a generated default. Adoption
records them in the domain index so an agent knows where to look, which is the
opposite of fighting them.

`conflicting` never auto-resolves. Silently overwriting a human's rule is the
worst outcome; silently keeping both contradictory statements is the second worst
— it is exactly the failure above in one Laravel application whose `.cursorrules` contradicted its own `rules.md`, where an agent receives two
incompatible instructions and picks one at random.

## Decision 4 — Existing content wins; core rules cannot be quietly ignored

**Additive** content always wins: extra conventions, stack notes, project
specifics, and anything in a location `dforge` does not own are preserved intact.

**Core non-negotiables** are different. SOLID, ACID, TDD, the base-branch
invariant, mandatory memory and branch notes are either complied with or
recorded as a **declared exception**:

```yaml
exceptions:
  - rule: tdd
    reason: "legacy module predates the test harness; tracked in <ref>"
    owner: <name>
    recorded: 2026-10
    review_by: 2027-04
```

Three requirements make this honest rather than a loophole:

1. The exception is **stated in the generated harness**. An agent must never
   receive two contradictory instructions; it must receive one instruction plus a
   visible, attributed deviation.
2. `owner` and `review_by` are mandatory. An exception is a dated decision, not a
   permanent waiver.
3. `dforge check` **fails on an expired `review_by`**. An exception that nobody
   revisits becomes a silently compliant rule, which is worse than a conflict.

## Decision 5 — Adoption has a rollout mode, and flow autonomy is respected

Adoption is not a single event. A harness installed at full enforcement into a
team that is still working by hand will be switched off the first time it blocks
legitimate work. Adoption therefore declares how it is being introduced:

```yaml
rollout:
  mode: enforced | advisory
  owner: <name>
  review_by: 2027-04             # required whenever mode is advisory
```

| Mode | Behaviour | When |
| --- | --- | --- |
| `advisory` | `check` reports and does not block; the harness is present and read | while a team transitions |
| `enforced` | `check` blocks on violation | once the team has adopted it |

`dforge adopt --plan` may run against any repository: describing is always safe.
`--apply` records the mode and the accountable owner in the adoption record.

**`review_by` is required whenever the mode is `advisory`.** Advisory is a
transition state, not a permanent exemption. Without an expiry nothing forces the
transition to end, and the state becomes **invisible**: `check` reports and passes,
so the exemption survives by attrition rather than by decision. An exception nobody
revisits is worse than a conflict, because a conflict is at least noticed.

Past `review_by`, `check` fails until someone either promotes the project to
`enforced` or renews the window with a new date and a reason. A **date** is used
rather than a duration so the answer is deterministic: `check` does not need to
know when the rollout began, and the same configuration produces the same result on
any machine on any day.

**The expiry is not subject to the mode.** `advisory` governs whether *rule*
violations block; it does not govern whether the rollout's own expiry blocks. If it
did, an advisory project would never be reviewed, and the mode would become a
permanent exemption disguised as a transition. Who receives that failure, where
`check` runs, and why an agent must not be able to extend its own deadline:
[0010](0010-enforcement-surface.md).

### The branch model is the sharpest case

Branching is where a team's own decision most often differs from the
organization's, and where overriding it unilaterally does the most damage. A
project may therefore legitimately declare **no branch provider at all**:

```yaml
branches:
  provider: none       # the project runs its own flow, not yet migrated
```

`dforge` then states only the invariant — never work on a base branch — and says
nothing about branch names, because there is no model to defer to. It never
manufactures one.

### Declared versus observed

A project may declare a branch provider it does not actually follow. That is a
detectable condition, not an assumption: `check` compares the declaration against
the history it can observe and reports a mismatch instead of trusting the
declaration. A harness that believes its own configuration file is worth nothing.

## Decision 6 — What adoption never does

- Never deletes or rewrites a project-specific rule.
- Never moves rule content out of a foreign location.
- Never touches git: no commits, no branch changes, no remote changes, no history
  rewriting.
- Never modifies a file it does not own to resolve a conflict.
- Never resolves a `conflicting` class by itself.
- Never overrides a project's branch flow decision. It may report that a project
does not follow the organization's model; switching that project is a rollout and
a conversation with its team, not an edit.
- Never cleans up repository debris. The loose `test_*.php` scripts and stray
  artifacts at the root of the team-managed Laravel application are not
  `dforge`'s concern; a bootstrap tool that starts deleting files nobody asked it
  to delete is a hazard, not a convenience.

## Decision 7 — Adoption is recorded

Every adoption writes a record under `docs/decisions/` listing each detected
conflict, its class, its resolution and its owner. That directory and its record
convention already exist ([0004](0004-foundation-discipline.md)), so adoption
leaves the audit trail in a place the team already reads.

## Consequences

- `dforge check` must distinguish **compliant** from **excepted**, and report the
  exception list. A check that cannot express a declared deviation will be
  disabled the first time it blocks legitimate work.
- Adoption is not a single command on a divergent repository: it is a plan plus
  one approval per conflict. That is deliberate — it is bounded and reviewable,
  and the number of conflicts is itself the finding that matters.
- `0001` consequence 2 ("correct by construction for new projects only") no longer
  holds.
- Adoption increases the number of writable locations `dforge` must reason about,
  which is why the five-class taxonomy is fixed rather than configurable.

## Evidence

- `cmd/commands/init.go:95-105` — refusal to overwrite without `--force`.
- one Laravel application's `.cursorrules` against its own `rules.md` — a repository contradicting itself on branch bases.
- `openspec/config.yaml:16`, in the legacy Laravel application — `strict_tdd: false`.
- `.agents/rules/code-conventions.md:43`, in a Laravel + Inertia application under company project management — pragmatic SOLID, against `.agents/quality/solid.md:8`, in the Laravel application that keeps per-concern quality rule files.
- `.agents/workflows/`, in the team-managed Laravel application — three hand-written workflows; `.antigravity/rules.md` in the same application; no `AGENTS.md`; no `.dflow.yaml`.
- `.agents/quality/`, in the Laravel application that keeps per-concern quality rule files, and `.agents/rules/`, in a Laravel + Inertia application under company project management — existing per-concern rule files, the shape [0007](0007-domain-separated-harness.md) formalizes.
