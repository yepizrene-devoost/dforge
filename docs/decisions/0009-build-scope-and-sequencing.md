# 0009 — Build scope, sequencing, and why `dforge` is not a subcommand

- **Status:** accepted
- **Date:** 2026-10 (design phase)

## Context

Two questions were open after the design phase: whether `dforge` is worth building
at all, and whether it should be a separate binary or a subcommand of the
organization's existing branch-flow tool, `dflow`. Both are answered here, because they
have the same answer.

### The uncomfortable evidence

The inventory did not find missing rules. It found the **same rules written five
times, drifting**. One project declared a memory file must stay concise and
reached 685 lines anyway. Another had two files contradicting each other on branch
bases and nobody noticed.

**Writing rules does not change behaviour.** A generator that produces more
documents would produce a **seventh location** for rules unless it also verifies
them.

### The genuine adoption signal

The organization has already demonstrated that its developer-facing CLIs get
adopted: a second developer adopted the branch tool and no longer starts a project
without it. That refutes the concern that the team will not take up a tool.

But it does **not** transfer automatically, and the reason matters. Branch
management is an *immediate, felt pain* — the developer notices the benefit on the
first branch. Rules consistency is a *diffuse* pain: nobody feels it in a single
sitting, and the cost only appears months later as drift. A tool that relieves
acute pain sells itself; a tool that prevents slow decay has to prove its value,
which is why this record ends with a kill criterion rather than a plan.

## Decision 1 — `dforge` is a separate binary. `dflow harness` is rejected.

Each tool has **one responsibility**, and neither absorbs the other's:

| Tool | Responsibility |
| --- | --- |
| `dflow` | control the branch flow |
| `dforge` | the engineering harness and its verification |

A single-responsibility tool is easier to reason about, easier to test, and easier
to adopt. Bolting project bootstrap onto a branch manager would blur the purpose of
both and give every user a dependency on a feature they do not need.

### The corollary: a tool documents its own interface

The branch tool's agent-facing document exists so **agents know how to use that
tool** — it publishes its own interface. It is not a harness generator and it does
not express engineering policy.

This narrows the overlap considerably. The two tools are not competing for the
same role; each answers a different question:

| Question | Answered by |
| --- | --- |
| *How do I use this tool?* | the tool itself |
| *What are our engineering rules, and is this repository following them?* | `dforge` |

The only genuinely shared resource is `AGENTS.md`, resolved with delimited sections
([0001](0001-scope-and-ownership.md)). One tool owning another's documentation
would be the mistake this decision avoids.

### Rejected alternatives

- **`dflow harness`.** Rejected: it makes a branch manager responsible for
  engineering policy, and the branch tool refuses repositories with no commits —
  which is exactly the state of every new project.
- **Folding `dforge` into an existing framework integration.** Rejected: those are
  stack-specific, and the core is not.

## Decision 2 — The product is adoption and verification, not generation

Generation is the cheap part of the value and the expensive part of the work:

| Half | Value | Cost |
| --- | --- | --- |
| Generation (scaffolding engine, presets, template content) | low — largely replaceable by template repositories | **high**: thousands of lines plus perpetual template maintenance |
| Adoption ([0006](0006-adopting-existing-repositories.md)) | **high** — nothing else does this | moderate |
| Verification (`check`) | **high** — this is what stops the drift | low, and no templates to maintain |

Three capabilities have no substitute:

1. **Adopting existing repositories.** A template repository cannot retrofit a
   repository that already exists, and there are twenty with six competing rule
   locations.
2. **Verifying the contract after day one.** A template never runs again.
3. **Deriving** a commit convention from the governance profile
   ([0002](0002-governance-profiles.md)) instead of maintaining four divergent
   copies.

Generating files is not on that list.

## Decision 3 — Build order

### Phase 0 — no code

Two or three template repositories, one per stack, holding the core harness and a
short `AGENTS.md`. This solves "a new project starts from zero" with **zero code
and zero maintenance**. It also produces the reference content that a later
generator would emit, so the work is not wasted either way.

### Phase 1 — `dforge check`

A single binary that only **verifies**: it reads `.dforge.yaml`, validates commit
messages against the derived format, counts functional lines against the review
budget, requires the memory digest and the branch note, checks the ignore region
([0008](0008-committed-vs-local-artifacts.md)), and **fails** on exceptions and
advisory rollouts whose `review_by` has passed. It generates nothing.

Small, testable, and it is the only thing that converts the three non-negotiables
into something that cannot silently erode.

### Phase 2 — generation and adoption

Only if Phase 1 survives its kill criterion. The scaffolding engine — embedded
asset trees with idempotent, marker-aware writes — is the one genuinely new
component, since the organization has no `embed.FS` usage today.

## Decision 4 — Kill criterion

> **If, three months after Phase 1, `check` is not running in the CI or pre-commit
> path of at least three projects, the generator is not built.**

The criterion is deliberately blunt. A verification tool that nobody runs has
already proved that the organization will not maintain the contract, and building
its generator would multiply the cost of a conclusion already reached. It also
keeps this project honest about the difference between the value that was argued
and the value that was used.

## Consequences

- `README`'s roadmap is reordered to match this record.
- `dforge`'s public documentation names `dflow`, the tool it integrates with.
  `dflow` is itself released as **open source**, so naming it discloses nothing,
  and the ownership boundary is unintelligible without it — the two form one
  open-source tool family. The convention that internal application projects and
  organisation names are never named still applies to everything else
  ([`docs/decisions/README.md`](README.md)).
- Phase 0 producing template repositories first means the harness content is
  authored and reviewed once, in a place a human can read, before any generator
  claims to emit it.
- A tool that verifies must be able to run in a repository it does not own and
  without network access.

## Evidence

- Measured drift: three commit formats, four branch models, three TDD strictness
  levels, six rule locations, 290 branch notes with no tooling, one retrogressive
  memory file at 685 lines, and two repository READMEs repeating the same sections
  at the same line numbers.
- The branch tool's own definition: *"Manage Git branches with a configurable
  workflow"*.
- The branch tool refuses repositories with no commits.
- No `embed.FS` usage anywhere in the organization; generation is single-document,
  from Go string constants.
