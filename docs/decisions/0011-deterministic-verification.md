# 0011 — Deterministic verification versus AI judgement

- **Status:** accepted
- **Date:** 2026-10 (design phase)

## Context

[0009](0009-build-scope-and-sequencing.md) made `check` the product, and
[0010](0010-enforcement-surface.md) specified who receives its failures. The
remaining question is what kind of judge `check` is: whether the rules are
evaluated mechanically, or whether a model interprets them.

The temptation is real. Several core rules read like matters of judgement — "the
test proves something", "the note tells the truth" — and a model could be asked to
score them.

## Decision 1 — `check` never uses a model. The verdict is a predicate.

Every rule `check` evaluates is reduced to a mechanical operation on data that
already exists:

| Rule | Mechanical form | Data source |
| --- | --- | --- |
| Commit format | a regex derived from the governance profile | `git log` over the range |
| Review budget | count changed lines, minus docs, comments and formatting | `git diff --numstat` plus an exclusion list |
| Branch note | the file exists for the branch slug and is non-empty | filesystem |
| Memory digest | exists, within its line cap, no local-tool identifiers | filesystem plus regex |
| Expiries | compare the date against today | `time.Now()` against the YAML field |
| Committed versus local | the marked region contains what is declared | parse `.gitignore` |
| Gate consistency | every `always: true` domain appears in the generated gate | parse `AGENTS.md` |

None of these requires judgement. All of them return the same result on any
machine, on any day, for the same diff. **Determinism is not that the rules are
simple; it is that each one has been reduced to a predicate.**

### Why not a model

| A model would cost | What it takes away |
| --- | --- |
| The same diff can pass today and fail tomorrow | reproducibility |
| There is no line to point at, only an opinion | auditability |
| A model call per rule, per pull request, per repository | cost and latency |
| "The model said it was fine" | the modern form of "near-strict" |

The last row is the decisive one. The inventory's failure mode was rules that are
interpreted differently by whoever reads them. A model in the verification path
reintroduces exactly that, with more confidence and less traceability.

## Decision 2 — The boundary: what the check owns, and what review owns

| The check verifies | Human review owns |
| --- | --- |
| that a test failed before the change was written | whether the test **proves** anything |
| that a branch note exists | whether the note **tells the truth** |
| that `MEMORY.md` is within its cap | whether what was written is a **real decision** |
| that the subject matches the derived regex | whether the message **describes** the change |
| that the pull request is within budget | whether the change **should be split** |

The left column is mechanical and mandatory. The right column is judgement, and it
is where the review conversation belongs. `check` makes the left column
non-negotiable so that review time is spent on the right column. It does not
replace review; it protects it.

## Decision 3 — AI proposes; the check decides

A model still has a place, and it is on the proposing side:

| Where | What it does |
| --- | --- |
| writing the code | the work itself; the rules say how |
| assisting a human reviewer | a second pass over the right-hand column above |
| `dforge audit` (Phase 2) | **report** which proposed changes would violate the rules before they are made — still deterministic, because it compares against the schema rather than interpreting |
| proposing a fix | when `check` fails, an agent can draft the correction, and `CODEOWNERS` guarantees a human approves it |

The asymmetry is the rule: **a model proposes, a predicate decides.** The moment a
model is allowed to decide compliance, the drift the inventory measured returns —
with more confidence and no trace.

## Decision 4 — How the check runs in CI

[0010](0010-enforcement-surface.md) specified *where* `check` runs and who receives
the failure. The mechanics:

```yaml
# .github/workflows/dforge.yml — generated; a file dforge owns
name: dforge
on: [pull_request]
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }     # the diff needs history
      - uses: actions/setup-go@v5
        with: { go-version: 'stable' }
      - run: go install github.com/<org>/dforge@latest
      - run: dforge check --base "${{ github.base_ref }}"
```

- `--base` gives the range to compare against: functional line counts and
  commit-message validation are computed over that diff.
- `dforge` is a static Go binary with no runtime and no network use at check time;
  the only network access is installing the binary itself, which satisfies
  [0009](0009-build-scope-and-sequencing.md).
- The process runs on the runner, which the agent neither controls nor sees the
  token of.

What `dforge` generates is the workflow file. What it cannot set is the
branch-protection rule that makes this check *required* — that remains the setup
prerequisite in [0010](0010-enforcement-surface.md).

## Consequences

- Every rule added to the core must be expressible as a predicate, or it does not
  go into `check` — it goes into review. That gate is deliberately uncomfortable:
  it forces each new rule to be stated precisely.
- `check` needs a `--base` argument and a defined diff-exclusion list. The
  exclusion list is part of the contract, because "functional lines" is only
  deterministic if the exclusions are too.
- A rule that cannot be made mechanical is not silently dropped: it is documented
  as a review obligation in the same file as the checked rules, so nothing is lost
  by being uncheckable.

## Open

**The exact diff-exclusion list.** Which file extensions and paths count as
non-functional is not yet defined. It must be written down before `check` exists,
because two implementations with different exclusions produce different counts for
the same commit and the budget stops meaning anything.

## Evidence

- The inventory's failure mode: rules interpreted differently by whoever reads them.
- [0009](0009-build-scope-and-sequencing.md) — `check` must run without network
  access, which rules out model calls by construction.
- [0010](0010-enforcement-surface.md) — the enforcement surface and the two valid
  actions on failure.
