# 0012 — Issue-linked pull requests and the issue lifecycle

- **Status:** accepted
- **Date:** 2026-10 (design phase)

## Context

[0002](0002-governance-profiles.md) modelled two places a work item can live: in an
external tracker whose key lands in every commit subject (`jira`), or nowhere
outside the repository (`autonomous`). It did not model the shape this repository
actually needs: **commits stay conventional with no ticket, while every pull
request must open from an issue that the owner has approved.**

The owner set that rule while configuring the repository's own governance
artifacts, and it creates a hybrid neither profile covered. It is a better shape
than either pure profile — the traceability lives in one place, the pull request,
instead of polluting every commit subject — but it has to be written down, because
an unrecorded governance rule is exactly the drift this project exists to prevent.

## Decision 1 — Governance is `autonomous` for commits; issues are the mandatory intake

Commit subjects follow the derived conventional format and carry **no ticket
reference**. The issue link lives in the **pull request body** (`Closes #N`), never
in the subjects of its commits.

This is deliberately not the `github` profile of [0002](0002-governance-profiles.md):
that profile puts the issue number in every subject. Requiring it here would buy
nothing — the pull request already names the issue, and the squashed merge onto the
base branch carries that link into history.

## Decision 2 — Four issue types, mirroring the pull-request taxonomy

Strict issue forms, with blank issues disabled:

| Form | Label | Commit type it produces | Covers |
| --- | --- | --- | --- |
| `bug_report` | `bug` | `fix:` | something broken or against the records |
| `feature_request` | `feature` | `feat:` | a capability that does not exist |
| `decision_proposal` | `decision` | a record in `docs/decisions/` | a change to a rule, boundary or policy |
| `chore` | `chore` | `docs:` / `refactor:` / `chore:` | everything that changes no rule and no behaviour |

Three facts make this set complete rather than convenient:

1. Every pull request is one of: fixes a defect, adds a capability, or changes a
   rule — or it changes none of those, which is `chore`.
2. The `chore` form carries a **guard**: a required confirmation that the change
   touches no rule, boundary or policy. Without it, "documentation" becomes the
   escape hatch that bypasses decisions.
3. The boundary is stated, not implied: **a change that adds, removes or modifies a
   rule is a decision; one that only clarifies, corrects or extends prose is a
   chore.**

The `chore` form's `Kind` dropdown maps directly onto the conventional commit type,
which is what will let `check` compare an issue's label against its commits' types
once it exists.

## Decision 3 — The approval gate

| Label | Meaning | Applied by |
| --- | --- | --- |
| `status:needs-approval` | opened; awaiting a decision | the form's author or the agent |
| `status:approved` | the owner approved it; pull requests may reference it | **the owner** |
| `status:rejected` | declined; the reason is recorded in a comment | **the owner** |

- A pull request may reference only issues that carry `status:approved`, and every
  issue it references must carry it.
- **The agent never approves its own work.** Approval is the owner's act, exactly
  as [0010](0010-enforcement-surface.md) separates preparing a decision from making
  one.
- Rejection is a decision too and carries a reason; a silent close is not a
  rejection.

## Decision 4 — Priority orders the approval queue

`priority:critical` · `priority:high` · `priority:medium` · `priority:low`.

Exactly one on every approved issue, applied by the owner at approval time — not by
the reporter at creation time, because self-reported priority is noise and approval
is where the decision happens. Priority never gets inferred from an issue's text;
it is read only from the label.

Its concrete purpose: since the owner approves issues, priority answers *"which
approval comes first?"*. It is a queue order, not decoration.

## Decision 5 — Enforcement

A rule with no surface is a note ([0010](0010-enforcement-surface.md)). Three
mechanisms carry this one:

1. **Blank issues are disabled** (`config.yml`), so nothing enters the tracker
   outside the four forms.
2. **The forms apply their own type label**, so classification does not depend on
   anyone remembering. Prerequisite: the labels must exist before the forms are
   used — GitHub silently ignores a label a form does not have.
3. **The issue-approval gate — its own work unit, pending.** A workflow that
   fails a pull request whose referenced issues are missing or unapproved. Its
   first two review rounds produced five CRITICAL findings on that job — stale
   passes after approval revocation, rate-limit exhaustion from an untrusted
   body, silent truncation — so it ships alone in a focused slice instead of
   inside the governance batch. Until it lands, the rule is carried by the
   pull-request template and review discipline, and `main`'s protection does
   not require its check.
4. **The artifact gate — active in this slice.** A job derives the slug from the
   branch name (`hotfix/governance-artifacts` → `governance-artifacts`) and fails
   when `odd/tasks/<slug>.md` or `docs/branches/<slug>.md` is absent. That makes
   the traceability contract of [0002](0002-governance-profiles.md) and
   [0003](0003-persistent-memory-and-branch-notes.md) mechanical instead of
   remembered. The work order itself lives in `AGENTS.md`, where an agent reads it
   at session start — what the gate can verify is the artifacts, not the moment
   they were created.

The gate's failure message names the missing thing and the rule it comes from, per
[0010](0010-enforcement-surface.md) Decision 5.

## Consequences

- The owner is the bottleneck by design: nothing reaches `main` without an
  approved issue behind it. That is the point, and it is cheap for a
  single-maintainer repository.
- The first issue predates strict mode, because the forms that disable blank
  issues arrive in the same pull request as the rule. It was created free-form and
  labelled by hand — a bootstrap exception that stops existing the moment this
  merges.
- `check` gains a future validation: the issue's type label against the types of
  the commits it produced.

## Evidence

- The rule as set by the owner while configuring this repository's governance.
- GitHub does not natively block a pull request for referencing an unapproved
  issue; the workflow is the only mechanism.
- GitHub silently ignores a form label that does not exist in the repository.
- `devoost/dflow/AGENTS.md` — the existing organisation convention that priority is
  read from `priority:*` labels and never inferred from issue text.
