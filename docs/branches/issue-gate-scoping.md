# issue-gate-scoping

Branch `hotfix/issue-gate-scoping` · Issue #3 · Type: chore

## What changed

One line in `issue-gate.yml`: the `artifacts` job is now scoped to pull-request
events (`if: github.event_name == 'pull_request'`).

## Why

The `issues` trigger added for the approval re-evaluation (R1-001) left the
`artifacts` job unscoped. On any issue event `github.head_ref` is empty, the
slug derivation refuses, and the run fails — first observed on the
`issues:closed` fired by merging PR #2. The gate job is untouched: running on
issue events is exactly its R1-001 requirement.

## Verification

- YAML parses; the artifacts job carries the event condition.
- The gate job is unchanged.
- The first pull request opened after this fix exercises the scoped job
  directly: it runs on `pull_request` with a non-empty `head_ref`.
