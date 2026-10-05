<!-- dforge:session-protocol -->
## Session Protocol

1. Read `.agents/MEMORY.md` before the first substantive reply. It is the
   committed contract of this repository and is self-contained.
2. Work order for every task
   ([0012](docs/decisions/0012-issue-linked-pull-requests.md) and
   [0002](docs/decisions/0002-governance-profiles.md)):

   approved issue → `dflow start <type> <slug>` → **create
   `odd/tasks/<slug>.md` first** → work → branch note and Task Notes as each
   stage closes → freeze for native review → commit → `dflow finish`.

3. The pull-request gate is mechanical: it fails on a missing or unapproved
   issue, and on a missing `odd/tasks/<slug>.md` or `docs/branches/<slug>.md`.
   Nothing here is optional.
4. This repository is public: internal project, organisation and module names
   never appear in its artifacts. Tools that are themselves open source may be
   named (`dflow`).
<!-- /dforge:session-protocol -->

<!-- dflow:workflow-reference -->
## dflow Workflow
Read `.agents/workflows/dflow.md` for branch types, merge rules, and finish flow.
<!-- /dflow:workflow-reference -->
