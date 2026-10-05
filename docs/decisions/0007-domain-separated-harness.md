# 0007 — Domain-separated harness artifacts

- **Status:** accepted
- **Date:** 2026-10 (design phase)
- **Resolutions:** the three items left open at review are closed below

> This record changes the generated layout in
> [0005](0005-configuration-schema.md). It was accepted after review; the
> resolutions taken at acceptance are recorded at the end of this document.

## Context

Today the rules accumulate in one file. In the Laravel application that keeps
per-concern quality rule files, `AGENTS.md` is 231 lines before its
injected Boost block, which is itself another ~230 (`AGENTS.md:26-231`); in a
Laravel + Inertia application under company project management, `AGENTS.md` is
~310. Meanwhile the same content also lives in six
competing locations, so nobody knows which file is authoritative.

Two existing patterns in the workspace show the way out, and neither was applied
consistently:

1. **The Laravel application that keeps per-concern quality rule files already separates by concern and indexes it.** `.agents/quality/`
   holds `solid.md`, `acid.md` and `tdd.md` as separate files, and
   its `AGENTS.md:100` describes `.ai/rules/index.md` mapping file globs to
   rule files. This is the right shape; it just coexists with three other
   locations.
2. **`dflow` already publishes discoverable documents with frontmatter.**
   `.agents/workflows/dflow.md` opens with `name`, `description: "Trigger: …"`,
   `metadata.source`, `metadata.generated_by`. `.atl/skill-registry.md` is a
   generated registry with a `Trigger` column. The team already reads this shape.

There is also a real hazard to design against. One Laravel application's `.cursorrules` and its
own `rules.md` give contradictory branch instructions. **Splitting rules across
files increases the chance of exactly that**, unless the split has a rule that
prevents duplication. That rule is the load-bearing part of this record.

## Decision 1 — Two tiers

| Tier | Location | Loaded | Contents |
| --- | --- | --- | --- |
| **0 — the gate** | inline in `AGENTS.md`, inside `dforge` markers | always | every rule that must bind on every single change, in its shortest form |
| **1 — the domains** | one file per domain under `.agents/harness/` | by trigger | the depth: rationale, examples, edge cases, what is *not* allowed |

**Why the gate stays inline.** You cannot rely on an agent to go read the TDD rule
before writing code. Rules that are always applicable and mechanically checkable
must live in the file that is always read. Moving them into a subdirectory means
they will be obeyed inconsistently, and an inconsistently obeyed non-negotiable is
not a non-negotiable.

**Why depth moves out.** Detail is conditional. The full treatment of ACID,
including the argument about what it does *not* cover, is worth reading when
touching persistence and is pure context cost when editing a Blade template.
Moving it out is what keeps the always-read file small enough to actually be read.

## Decision 2 — The rule that makes the split safe

> **Every `always: true` domain states its rule in its own frontmatter, and that
> statement is generated into the gate verbatim.**

A file carries **depth, never the sole existence of a rule**. Nothing that must
bind universally lives only in a domain file. `dforge check` verifies both
directions: every `always: true` domain appears in the gate, and no normative text
is duplicated between the gate and a domain file.

This is the answer to the problem of one Laravel application whose `.cursorrules` contradicted its own `rules.md`. There is exactly one source for
each rule, and one generated view of it.

## Decision 3 — Directory equals write ownership

```text
.agents/harness/
  index.md        generated     discovery: trigger → file, subject → file
  core/           generated     dforge overwrites on every sync
  presets/        generated     dforge overwrites on every sync
  project/        never written by dforge; exists only if the project
                                has rules of its own
```

The tree lists **possible** locations, not a manifest. `index.md` and `core/`
always exist, `presets/<stack>/` exists for the declared preset, and `project/`
exists only where a project actually has something of its own to say.

`dforge sync` regenerates `index.md`, `core/` and `presets/`, and **never touches
`project/`**. Idempotence becomes structural rather than a promise: the
directory a file sits in declares who owns it. This is also how adoption
([0006](0006-adopting-existing-repositories.md)) has somewhere safe to put a
project's own rule without a conflict.

### Nothing is created empty

`dforge` never creates an empty generated artifact. A directory with nothing in it
carries no information, produces noise, and teaches the team that generated files
are clutter — the same reasoning that makes an empty `PRD-MASTER.md` worse than no
PRD at all ([0004](0004-foundation-discipline.md)).

So a new project gets `index.md`, `core/` and its preset, and usually **no
`project/` at all**. It appears the first time the project has a rule of its own,
which may be months later, or once when an adopted project keeps rules the human
chose to retain. Most repositories will never have one, and that is the intended
outcome: a directory that exists only where it is earned.

### Why this boundary has to be a directory

The rule that makes regeneration safe is: **`sync` may overwrite everything it
owns, so it must have exactly one place it is forbidden to touch.** Without that
boundary, `sync` is destructive and nobody can run it. Regenerating the harness
only becomes a safe, repeatable operation once there is a region that is
officially not its business.

Markers alone are not enough for that, for two reasons.

1. **A marker is a contract between writers; a directory is a fact.** `AGENTS.md`
   has three writers — the branch tool, the framework's own injected block, and
   the human. Markers only hold if every one of them implements them correctly. A
   directory cannot be violated by a writer that does not know it exists.
2. **`AGENTS.md` carries a size budget** (Decision 7). Project-specific rules are
   conditional depth, exactly like the core domains: the money convention matters
   when touching billing, not when editing a template. Piling them into the
   always-read file recreates the monolith this record exists to prevent.

### The same question, worked through

Take a project rule such as "monetary amounts are stored in minor units".

- It is not universal, so it must not live in `core/` — the next `sync` would
delete it.
- It is not a framework fact, so it must not live in `presets/` — it is this
product's decision, not the stack's.
- So it lives in `project/`, which `sync` never writes, and it is indexed in
`index.md` so an agent finds it by trigger and by moment like any other rule.

`project/` is also the landing zone for adopted rules. After adoption resolves a
project's existing rule files, the rules the human decides to keep as dforge rules
need a home that survives regeneration — and this is the only one that does. The
rules the human decides *not* to adopt stay where they are and are indexed in
place; `project/` is for the ones that become dforge's.

## Decision 4 — Frontmatter contract

Adopted from `dflow`'s generated documents so the shape is already familiar:

```yaml
---
name: strict-tdd
domain: testing          # subject matter
when: [edit, verify]     # the moment in the work where it binds
always: true             # its summary is emitted into the AGENTS.md gate
trigger: "writing or changing behavior, adding or fixing tests"
summary: "RED → GREEN → REFACTOR for every change. No exceptions."
metadata:
  source: dforge
  generated_by: dforge sync
  version: 1
---
```

`when` is the field that answers *"dominios de flujo"*: rules do not bind at all
times, they bind at a moment. The five moments follow the work lifecycle:

| `when` | Moment | Domains that bind |
| --- | --- | --- |
| `plan` | before the first write | `domains`, `solid` |
| `edit` | while writing | `conventions`, `solid`, `acid`, `tdd` |
| `verify` | running the checks | `tdd`, `review-budget` |
| `commit` | staging and committing | `commits`, `review-budget` |
| `close` | closing the task | `memory`, `docs` |

Indexing by subject alone tells an agent *what exists*. Indexing by moment as well
tells it **what applies right now**, which is the part that was missing.

## Decision 5 — The core domain set is fixed

| File | Domain | `when` | `always` |
| --- | --- | --- | --- |
| `conventions.md` | code | `edit` | yes |
| `solid.md` | design | `plan`, `edit` | yes |
| `acid.md` | data | `edit` | yes |
| `tdd.md` | testing | `edit`, `verify` | yes |
| `review-budget.md` | review | `verify`, `commit` | yes |
| `commits.md` | vcs | `commit` | yes |
| `branches.md` | vcs | `plan`, `commit` | yes |
| `memory.md` | memory | `close` | yes |
| `docs.md` | documentation | `close` | no |
| `domains.md` | architecture | `plan` | yes when `prd-first` |

Fixed, not configurable. A configurable core is how the six competing locations
happened.

## Decision 6 — Presets add domains; they never overwrite core

`presets/laravel.md`, `presets/go.md`, `presets/flutter.md` and so on add
stack-specific depth under the same frontmatter contract. A preset may add a
domain and may specialize a core rule for the stack, but it cannot contradict one
— a contradiction is a `conflicting` class during adoption and a validation error
in `check`.

Extensible domains are expected, not exceptional. Evidence: `code-conventions.md`,
in a Laravel + Inertia application under company project management, already carries money-in-minor-units, a timezone helper
convention and a localization rule — real, useful, and not part of any core.

## Decision 7 — The gate has a budget

Tier 0 targets **≤ 60 generated lines**. Exceeding it means either a rule is not
actually `always`, or it belongs in a domain file with only its `summary` inline.
Without a budget, the gate becomes the monolithic `AGENTS.md` this record exists
to prevent.

## Resolved at acceptance

1. **Name of the project-owned directory: `project/`.** `overrides/` would have
described it by what it does to core, but a project-specific rule is not an
override of anything — it is the project's own rule, and the directory should say
so. Chosen explicitly at review.
2. **Humans may add normative text inline in `AGENTS.md`, outside the markers:**
yes. `dflow`'s marker semantics already permit it, and forbidding it would fight
the humans who maintain the file. `check` **warns** when inline text duplicates a
core domain, because that duplication is the drift this record exists to prevent.
3. **`docs.md` is `always: true`.** The rule that every task leaves a branch note
is universally binding, and the load-bearing rule of this record is that nothing
binding on every change may live *only* in a domain file. Omitting its summary
from the gate would mean inconsistent obedience, and a non-negotiable obeyed
inconsistently is not a non-negotiable. Its `summary` is kept to one line to
respect the gate budget of [Decision 7](#decision-7--the-gate-has-a-budget).

## Consequences

- Agents need the index. Because the gate carries every `always: true` summary,
  an agent that reads only `AGENTS.md` still receives every binding rule.
- The generated surface grows from one file to about a dozen. That is a real
  cost, and it buys three things: a small always-read file, mechanical discovery
  by moment, and a writable place for project rules that cannot conflict.
- `dforge check` gains two structural validations: gate/domain consistency, and
  absence of duplicated normative text.
- Testing is no longer enforced by prose alone: `tdd.md` is `always: true`, so its
  summary appears in the gate of every project.

## Evidence

- `.agents/quality/{solid,acid,tdd}.md`, in the Laravel application that keeps per-concern quality rule files — per-concern rule files already in use.
- `AGENTS.md:100`, in the Laravel application that keeps per-concern quality rule files — `.ai/rules/index.md` mapping file globs to rule files.
- `dflow`'s generated `.agents/workflows/dflow.md` — `name` / `description: "Trigger: …"` / `metadata.source` / `metadata.generated_by`.
- `.atl/skill-registry.md` — a generated registry with a `Trigger` column, present in several projects.
- `.agents/rules/code-conventions.md`, in a Laravel + Inertia application under company project management — money, dates and localization conventions: legitimate non-core domains.
- one Laravel application's `.cursorrules` against its own `rules.md` — the duplication hazard this record designs against.
