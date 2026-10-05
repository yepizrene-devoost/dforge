# Documentation

- **`decisions/`** — long-lived decision records. [`decisions/README.md`](decisions/README.md)
  is the index: records are numbered in creation order, never renumbered, and a
  superseded record keeps its original text.
- **`branches/`** — one note per branch: `branches/<slug>.md`, where the slug is
  the branch name without its type prefix. A branch that changes application
  behaviour has a note here before it merges.

## Branch Notes

| Branch | Note | Issue |
| --- | --- | --- |
| `hotfix/governance-artifacts` | [governance-artifacts](branches/governance-artifacts.md) | [#1](https://github.com/yepizrene-devoost/dforge/issues/1) |
