# governance-artifacts

GitHub governance layer: issue intake, approval gate, pull-request template.
Issue #1 · Branch `hotfix/governance-artifacts` (from `main`) · Type: chore

## Stages

- [x] **Branch model** — `dflow init`: `main` manual, `develop` automatic, no
  `uat` branch. Owner decision, applied and verified.
- [x] **Branch protection** — `main` pull-request-only, `enforce_admins`,
  linear history, no force pushes, no deletions. Zero required approvals on
  purpose: GitHub forbids self-approval, and 1 would deadlock a solo repository.
- [x] **Label taxonomy** — 11 labels (type / status / priority); 9 GitHub
  defaults outside the taxonomy removed.
- [x] **Issue forms** — `bug_report`, `feature_request`, `decision_proposal`,
  `chore`, plus `config.yml` with blank issues disabled.
- [x] **Pull-request template** — issue link required, non-negotiables as a
  checklist.
- [x] **Enforcement** — `issue-gate` workflow: fails a PR that references no
  issue, a pull request instead of an issue, or an unapproved issue.
- [x] **Decision 0012** — hybrid governance recorded and indexed.
- [x] **Memory and branch note** — `.agents/MEMORY.md` (Task Notes),
  `docs/branches/governance-artifacts.md`, `docs/README.md` index. Added after
  the owner caught the omission against the PR checklist.
- [x] **Native review** — the tree is frozen and submitted for review before
  commit. The verdict is expected as a native receipt recorded in Engram, never
  in this file; no approval is ever recorded here.
- [x] **Slice 2 — issue-approval gate, rebuilt in-repo.** The owner closed the
  split the same day: the gate returns inside `.github/workflows/issue-gate.yml`
  with the five findings as design requirements — issue events re-evaluate open
  PRs and post a fresh check run per head; only closing-keyword lines are
  parsed; the count is checked before any cap so the rejection is reachable; and
  per-PR API failures never abort the loop, so no stale pass survives.
- [x] **Disposition — slice 2 left unreviewed by RDD.** The owner explicitly
  cancelled the review of the rebuilt gate after three rounds of findings had
  landed on this job. The agent self-review covered syntax, the five design
  requirements and the three corrections, and is recorded as a proposal, not a
  verdict. The named residual risk is an unknown silent path leaving a stale
  pass after an approval revocation. The lineage
  `review-5c9198e34bfc23d1` remains in `correction_required` as the record.

## Correction (R1-001, R1-002)

The first lineage (`review-bea45411021ea045`, risk tier high, 4 lenses) returned
`correction_required` with two CRITICAL candidate-caused findings on
`.github/workflows/issue-gate.yml`:

- **R1-001** — the gate ran only on pull-request events, so a revoked approval
  left a stale pass behind: the rule was not enforced at merge time.
- **R1-002** — the gate parsed every `#<digits>` token in the untrusted PR body
  with one API call per number, so a padded body could exhaust the rate limit and
  break the required check for legitimate PRs.

A bounded correction plan of **131 lines** (63 added, 34 replaced × 2) was
declared and accepted against the frozen candidate, and the correction was
applied: the gate now also runs on issue events (`edited`, `labeled`,
`unlabeled`, `closed`, `reopened`), re-evaluates every open PR that references
the issue, and posts a fresh check run on each PR head; the body parser reads
only `Closes` / `Fixes` / `Resolves` lines and caps at 10 references.

The provider's targeted validator refused its binding twice with nothing mutated
(once as a transport failure for a missing model assignment on the
`review-validator` role — fixed in the routing config — and once after the
assignment). A fresh lineage over the corrected workspace then returned three
more CRITICAL findings on the same job (dead cap guard with silent truncation,
stale check runs under `set -e`).

**Resolution: the gate job ships as its own work unit.** Two review rounds
produced five CRITICAL findings, all concentrated on that one job, while the
other seventeen files passed clean every round. The gate job was pulled out of
this slice; the findings above are its design requirements, and its required
check was removed from `main`'s protection until it lands. The remaining slice
(templates, labels, memory, notes, decision 0012) passed three review rounds
with no findings and is frozen alone.

## Evidence

- All seven YAML files parse; every form applies its type label.
- 75 relative links resolved, 0 broken. Zero identifier leaks (public repo).
- `main` tree hash identical to the reviewed candidate: `0642c2e4…` for the
  foundation commit; this task's candidate is hashed at freeze.

## Commit identity

Recorded in Engram after the commit, not in this file: writing the SHA here
would be a tracked write after review approval, which mints a new candidate and
forces a re-review for a bookkeeping line.
