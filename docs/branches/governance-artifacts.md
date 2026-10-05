# governance-artifacts

Branch `hotfix/governance-artifacts` · Issue #1 · Type: chore

## What changed

Adds the GitHub governance layer before any implementation work starts:

- `.dflow.yaml` and the generated workflow document — branch model: `main` is
  parked and manual, `develop` is the living base and automatic, no `uat` branch.
- Strict issue forms: `bug_report`, `feature_request`, `decision_proposal`,
  `chore`, with blank issues disabled.
- `PULL_REQUEST_TEMPLATE.md` — the issue link is required and the
  non-negotiables are a checklist.
- `issue-gate` workflow — fails a pull request that references no issue, or one
  that does not carry `status:approved`.
- Decision [0012](../decisions/0012-issue-linked-pull-requests.md) records the
  hybrid governance: conventional commits, GitHub Issues as mandatory intake.
- Label taxonomy: 11 labels across type, status and priority; 9 GitHub defaults
  outside the taxonomy removed.

## Why

A pull request may only open from an issue the owner approved. Without the forms,
the labels and the gate, that rule is a note rather than a mechanism
([0010](../decisions/0010-enforcement-surface.md)).

## Verification

- All seven YAML files parse; every form applies its type label and enforces its
  required fields.
- Labels were created before the forms that reference them — GitHub silently
  ignores a missing label.
- The gate fails on: no referenced issue, a referenced pull request, a closed
  issue, or a missing `status:approved` label.
- `main` is protected: pull request only, `enforce_admins`, linear history, no
  force pushes, no deletions. Zero required approvals, because GitHub forbids
  self-approval and a solo repository would deadlock.

## Correction

The first two review rounds produced five CRITICAL findings on the gate job. The
owner closed the split the same day: **the gate was rebuilt inside
`issue-gate.yml` with those findings as design requirements**, and then **left
unreviewed by RDD** on his explicit decision after a third round returned three
more CRITICAL findings on the same loop. The agent self-review is recorded as a
proposal, not a verdict; the named residual risk is an unknown silent path
leaving a stale pass. Detail in `odd/tasks/governance-artifacts.md`.
