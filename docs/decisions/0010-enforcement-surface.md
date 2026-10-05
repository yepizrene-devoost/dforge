# 0010 — Enforcement surface: who gets stopped, and by what

- **Status:** accepted
- **Date:** 2026-10 (design phase)

## Context

[0006](0006-adopting-existing-repositories.md) introduced an advisory rollout with a
`review_by` date, and [0009](0009-build-scope-and-sequencing.md) made `check` the
product. Neither answered two questions that decide whether the date means
anything:

1. **Who receives the failure?** A date that only a human remembers is not a
   deadline — it is a note.
2. **Does `advisory` silence its own expiry?** If it does, an advisory project is
   never reviewed, which is the opposite of the intent.

A rule that depends on someone remembering it is not enforced. It is documented.

## Decision 1 — The reviewer is a human owner. Never the agent.

`rollout.owner` names the person accountable. `check` does not decide whether to
promote a project to `enforced`; the owner does. The tool's only job is to make the
decision **impossible to skip silently**.

The agent's role is the inverse, and it is a real role: **make the decision cheap
and informed.** When a rollout expires, the agent can report exactly what
enforcement would start blocking, so the owner decides with evidence instead of
guessing. It prepares the decision; it never makes it.

## Decision 2 — The expiry is not subject to the mode

This is the contradiction that had to be resolved:

| What | Controlled by `rollout.mode`? |
| --- | --- |
| Violations of harness rules (TDD, commit format, review budget, memory, branch notes) | **Yes** — `advisory` reports, `enforced` blocks |
| The rollout's own `review_by` expiring | **No — always blocks, in both modes** |

`advisory` is a statement about *the rules inside the harness*. It is not a
statement about *how long the harness itself may stay provisional*. If the expiry
were advisory too, then `advisory` would be self-preserving: it could never expire,
because expiring is the one thing it would refuse to report. The mode would become
a permanent exemption disguised as a transition.

## Decision 3 — Where `check` runs

The expiry only works where `check` actually executes. Three surfaces, each for a
different reason:

| Surface | Purpose |
| --- | --- |
| Pre-commit hook | immediate feedback, before a change is even described. **Advisory only** — an agent bypasses it with `--no-verify`, or by simply not installing it |
| CI on every pull request | **the gate** — the failure blocks the merge |
| Scheduled run (nightly) | catches repositories nobody is committing to, where no pull request would ever trigger the check |

The scheduled run is not redundant. A dormant repository is exactly where an
expired exemption can sit unnoticed for months, precisely because nothing happens
in it to trigger anything else.

## Decision 4 — An agent must not be able to extend its own exemption

A failing check creates pressure to make it green, and the cheapest edit is to bump
`review_by`. An agent under that pressure will do it if nothing stops it, and then
the deadline is decorative.

Two requirements close that hole, and neither is code:

1. **`.dforge.yaml` is protected by `CODEOWNERS`.** Any change to the file —
   including `review_by`, `rollout.mode` and `exceptions` — requires human
   approval. Extending an exemption becomes a **reviewed act**, visible in a diff.
   This is a repository-setup requirement, not something `dforge` can enforce by
   itself, and it must be stated in the generated harness so nobody assumes the
   tool is doing it.
2. **A renewal requires a `reason`.** The schema will not accept a new `review_by`
   without one, so an extension carries an explanation a reviewer can disagree
   with. A bare date bump is deliberately not expressible.

Without these, the honest description of the mechanism would be "a date the agent
can move", which is worse than no date, because it looks like a control.

## Decision 5 — The failure message is part of the contract

A check that fails without saying what to do trains people to bypass it. The
expiry failure must name the rule, the date, and the only two valid actions:

```text
advisory rollout expired on 2027-04  (owner: <name>)
  .dforge.yaml → rollout.review_by

Two valid actions:
  1. promote to enforced:  mode: enforced
  2. renew with a reason:  review_by: <new date> + reason: "<why>"
```

Both actions are edits to a file under `CODEOWNERS`, so both reach a human. There
is no third action, and the message does not imply one.

## Decision 6 — The verdict is not the stop. The binding is.

`check` is a judge, not a jail. Running it produces a verdict; something else must
make that verdict binding, and that something is **server-side**, not local:

| Layer | Role | Can an agent bypass it? |
| --- | --- | --- |
| `dforge check` | produces the verdict | — it *is* the verdict |
| CI runner on the pull request | the bailiff: executes the check server-side | only if the workflow does not exist, or exists but is not *required* |
| Branch protection on the base branches | the jail: refuses the merge until the check passes | **only with an admin token** |
| Local pre-commit hook | feedback only | **yes, trivially** |

Three consequences follow, stated bluntly:

1. **A repository without required status checks is not protected.** `check`
   running locally is advice. Enforcement begins where a merge is refused.
2. **A repository with no CI is not protected at all**, because nothing ever runs
   the verdict. This is why the kill criterion in
   [0009](0009-build-scope-and-sequencing.md) measures whether `check` runs in CI —
   not whether it exists.
3. **The permission boundary is the real control.** If the agent's credentials can
   edit branch protection, every layer above is void. The agent's token must have
   neither admin rights on the repository nor the ability to change required
   checks. That is an operations requirement outside `dforge`'s reach, and it
   belongs in the generated harness's setup checklist next to `CODEOWNERS`.

What `dforge` can and cannot do about this follows directly:

- **Can:** generate the CI workflow file, because it is a file `dforge` owns; fail
  loudly; name the two valid actions.
- **Cannot:** make itself binding. That is a property of the repository's
  configuration, which is exactly why this record states prerequisites instead of
  claiming enforcement.

## Consequences

- **A setup requirement is created** for every adopted repository: a protected
  branch plus `CODEOWNERS` covering `.dforge.yaml`. This belongs in the generated
  harness as a checklist item, because `dforge` cannot verify it from the
  filesystem and a check that silently cannot be trusted is worse than a documented
  prerequisite.
- `check` needs a `reason` field on `rollout`, and validation must reject a
  renewal that changes `review_by` without one.
- The nightly run needs somewhere to run — a scheduled CI job per repository, or a
  single organization-level sweep. Left open deliberately; the requirement is that
  *some* scheduled surface exists.
- The advisory mode's value is preserved: a team still gets room to adopt. What it
  loses is the ability to stay in transition without anyone deciding.

## Open

**Where the scheduled sweep lives.** Options are a per-repository scheduled
workflow, which duplicates configuration across twenty repositories, or one
organization-level job that reads each repository's `.dforge.yaml`. The second is
better suited to `dforge`'s autonomy — one binary that can be pointed at many
repositories — but it does not exist yet.

**Verifying the prerequisites.** Whether the required check exists, whether the
base branches are protected and whether `CODEOWNERS` covers `.dforge.yaml` can only
be confirmed through the API, which conflicts with
[0009](0009-build-scope-and-sequencing.md)'s requirement that `check` run without
network access. Candidate for a separate `dforge audit` command in Phase 2, so the
setup requirement becomes verifiable instead of merely documented.

## Evidence

- The measured failure mode this prevents: a project declared `strict_tdd: false`
  and remained that way, because nothing ever asked again. An exception nobody
  revisits becomes a silently compliant rule.
- `devoost/dflow/cmd/commands/init.go:95-105` — the same discipline one level down:
  the tool refuses to overwrite a hand-edited config without an explicit flag.
- [0006](0006-adopting-existing-repositories.md) Decision 4 already requires
  `owner` and `review_by` on exceptions; this record applies the identical
  mechanism to rollouts and specifies who receives the result.
