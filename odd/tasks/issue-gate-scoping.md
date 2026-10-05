# issue-gate-scoping

Scope the `artifacts` job of `issue-gate.yml` to pull-request events.
Issue #3 · Branch `hotfix/issue-gate-scoping` (from `main`) · Type: chore

## Stages

- [x] **Fix** — one line: `if: github.event_name == 'pull_request'` on the
  `artifacts` job. Root cause: when the `issues` trigger was added for R1-001,
  the job was not scoped — on any issue event `github.head_ref` is empty, the
  slug derivation refuses, and the run fails. First observed on the
  `issues:closed` fired by merging PR #2.
- [x] **Bookkeeping** — branch note and Task Notes updated.
- [x] **Native review** — the tree is frozen and submitted for review before
  commit. The verdict is expected as a native receipt recorded in Engram, never
  in this file; no approval is ever recorded here.

## Evidence

- The failed run: `issue-gate.yml` on `issues:closed` over `84f3b45` —
  "Branch '' carries no type prefix; the artifact slug cannot be derived."
- The gate job passed on that run and is untouched: its issue-event
  re-evaluation is the R1-001 requirement.

## Commit identity

Recorded in Engram after the commit, not in this file: writing the SHA here
would be a tracked write after review approval.
