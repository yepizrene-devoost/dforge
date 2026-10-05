# Project Memory

Committed memory for this repository. This file is the contract for any agent or
person without access to another developer's local tooling: it is self-contained
and never references local-only stores. It holds durable decisions and standing
constraints; what a branch changed lives in its branch note under `docs/branches/`.

## Repository Guidance

- **This repository is public.** Internal project, organisation and module names
  never appear in its artifacts. The one exception is a tool that is itself open
  source (`dflow`). See the conventions in `docs/decisions/README.md`.
- **The review gate is a label.** A pull request may only reference issues
  carrying `status:approved`; the owner approves issues and agents never do.
- **Commit subjects** follow the derived conventional format, with no ticket
  reference and no file paths.
- **Review budgets**: 600 functional lines per commit (excluding documentation,
  comments and formatting), 400 per pull-request slice, 2000 per file.

## Workflow Decisions

- Governance is `autonomous` for commits with GitHub Issues as the mandatory
  intake — [0012](../docs/decisions/0012-issue-linked-pull-requests.md).
- Branch model (`.dflow.yaml`): `main` is parked and manual — pull request only,
  branch-protected; `develop` is the living base and automatic; there is no `uat`
  branch.
- Intake taxonomy: `bug` → `fix:`, `feature` → `feat:`, `decision` → a record in
  `docs/decisions/`, `chore` → `docs:` / `refactor:` / `chore:`. Blank issues are
  disabled.
- Priority labels order the approval queue. Exactly one per approved issue,
  applied by the owner at approval time, never inferred from the issue text.

## Task Notes

<!-- Per-branch durable decisions only. The branch narrative lives in
     docs/branches/<slug>.md; what proves global is promoted to Workflow
     Decisions or to a record in docs/decisions/, then this heading is pruned. -->

### governance-artifacts

- `main` branch protection intentionally requires **zero approving reviews**
  (pull request only, `enforce_admins: true`). GitHub forbids self-approval, so
  raising it to 1 in a solo repository deadlocks every pull request. Revisit only
  when a second reviewer exists.
