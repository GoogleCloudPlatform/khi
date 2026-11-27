---
name: khi-swarm-workflow
description: >-
  Primary multi-agent swarm workflow for multi-file/multi-package development,
  refactoring, bug fixing, and code-quality verification across KHI (`pkg/`,
  `internal/`, `web/`, `scripts/`). Use when acting as Coordinator or top-level
  task lead in an active swarm to analyze, implement, audit, verify, and prepare
  user-review walkthroughs for changes.
---

# KHI Multi-Agent Swarm Workflow

Use this workflow to coordinate parallel development, refactoring, bug fixing, and quality verification across KHI (`pkg/`, `internal/`, `web/`, `scripts/`).

## Roles and Model Assignments

Pin the exact model below on every `swarm{action: "add"}` call and enforce strict role boundaries:

- **Coordinator / Task Lead** (`Gemini 4 Neon (High)`, main workspace or assigned share workspace):
  - Orchestrates phases, tracks dependencies, dispatches parallel workers, merges `share` branches, and escalates blocking decisions to the user.
  - **No hands-on coding or deep file reading**—delegate investigation to Scouts/Planner and edits to Implementers.
- **Planner** (`Gemini 4 Neon (XHigh)`, `workspace: "inherit"`):
  - Read-only design lead. Fans out parallel **Scouts** (and optional experimental **Implementers** in `workspace: "share"`) to verify invariants and feasibility, kills all helpers before reporting `[done]`, and outputs:
    - Cohesive disjoint-file workstreams (zero overlapping files across parallel Implementers; avoid over-splitting tightly coupled abstractions/tests across too many workers)
    - **`Non-Goals`** (explicitly untouched working algorithms/logic & passing suites) and **1-to-1 change justification** (every edit tied to a concrete bug, explicit rule/AST violation, or genuine coverage gap)
    - Explicit behavioral invariants, **State-Transition & Interleaving Contracts** (for concurrent/stateful edits), and regression traps
    - `QUESTIONS_FOR_USER` (only genuine architectural or behavioral forks; otherwise `NONE`)
- **Scout** (`Gemini 4 Hydrogen (High)`, `workspace: "inherit"`):
  - Read-only codebase, caller, test, and spec investigator reporting concrete `file:line` citations.
- **Implementer** (`Gemini 4 Hydrogen (High)`, `workspace: "share"` when parallel / `"inherit"` when solo):
  - Executes one disjoint workstream and reads applicable `.agents/rules/` files. Never chases 100% statement/mutant coverage via white-box internal pokes or timer/spin-poll waits.
  - **Surgical Diff**: follow `.agents/rules/go-coding-rule.md`—never rewrite already-passing `*_test.go` files wholesale, rename/split passing `Test*` funcs, or mass-add boilerplate comments to untouched test cases; keep test edits proportional and directly tied to concrete bugs, coverage gaps, surviving mutants, or `MUSTFIX` limits, with no unrequested locks or working-algorithm rewrites.
  - Commits on a local branch (`git checkout -b <worker>` + `git add -A && git commit`) when in `share`—or stages with `git add -A` when in `inherit`—**before** running `make qa-fast BASE=<BASE_REV>` (`must` + `advisorycheck`) and targeted `go build <pkgs>` / `go test -race -count=1 <pkgs>` / `make test-web`, so new files aren't missed and advisory slips don't trigger polish rounds. Never runs `git push` or `jj commit`/`jj new` in a shared main workspace.
- **Auditor** (`Gemini 4 Hydrogen (XHigh)`, `workspace: "inherit"`):
  - Read-only triage of `QAReport.advisory[]` (including any Gemini Decisions findings when `CODEQUALITY_DECISIONS=true` is configured) and assigned scopes; consults `scripts/codequality/rules/*.yaml` directly when Decisions API is disabled or offline.
  - 100% statement/mutant coverage is NOT a goal—mark GC zeroing, sealed-interface guards, defensive branches, and interface boilerplate `NO_ACTION`; never recommend white-box internal pokes, timer/spin waits, deleting safety guards, or adding `CODEQUALITY_ALLOW`/`//nolint` solely to clear advisories.
- **Verifier** (`Gemini 4 Neon (XHigh)`, `workspace: "inherit"`):
  - Read-only final gate comparing Planner instructions, `Non-Goals`, and invariants against full `git diff <BASE_REV>` and `qa-fast-advisory` signals (`advisorycheck`), outputting `VERDICT: PASS` or `VERDICT: FAIL` (`file:line`).
  - **Disposable Breaker** (`Gemini 4 Hydrogen (High)`, `workspace: "share"`): whenever `git diff <BASE_REV>` touches concurrency/locks/goroutines, stateful/re-entrant APIs (`Start`/`Stop`, repeated `Parse`), `defer`/panic ordering, or non-trivial arithmetic/bounds, Verifier spawns a disposable **Breaker** (NO branch creation; respects the `share` codegen boundary; preserves any board-isolation `since` floor; write `Report ONLY to @<verifier>` and never write negative `@<lead>` or `@coordinator` mentions in Breaker's brief).
    - Breaker writes adversarial `go test -race -count=1 <target-pkgs>` falsification probes (blocking fakes during concurrent `Start`/`Stop`, repeated calls & mid-call error rollback, `defer`/panic-vs-`Wait()` order, `MaxInt`/boundary inputs).
    - Verifier `kill`s Breaker **without merging** to discard scratch probes, records `Breaker: PASS | FAIL | N/A`, embeds any confirmed contract-violating minimal reproducer in `VERDICT: FAIL` so the Fix Implementer fixes the bug AND preserves the regression test, and never uses a killed unmerged Breaker to justify `NO_ACTION` on an uncovered committed branch or surviving mutant.
  - **Adversarial Verification Lens**: before issuing `VERDICT: PASS`, emit a concise **5-bullet adversarial trace** with concrete `file:line` citations (or `N/A` with justification) plus a 1-line `Breaker: PASS | FAIL | N/A` summary:
    - `[lock/lifecycle]`: lock-release ordering and lifecycle interleaving (`Start`/`Stop`, repeated `Parse`/error state).
    - `[bounds/overflow]`: arithmetic/contract bounds (signed-int overflow, negative inputs, `defer`/panic ordering).
    - `[barrier contention]`: deterministic concurrency synchronization confirming real goroutine contention before release (no optimistic checks, timer/spin waits, or excessive `testing.Short()` stress cuts that skip contention/refill paths).
    - `[non-short stress & Goodhart]`: non-`testing.Short()` stress reductions, excessive `testing.Short()` stress cuts skipping contention/refill paths, newly introduced `0%`-covered production helpers, single-task/coincidental-pass assertions, comment-vs-assertion mismatches, deleted guards, weakened/hollow assertions, `time.Sleep`, suppressions, `testing.Short()` skips, and caller drift.
    - `[scope & diff minimality]`: blocking `FAIL` on `Non-Goal` edits, unrequested features/locks, wholesale rewrites of already-passing `*_test.go` files, mass-added boilerplate comments on untouched test cases, gratuitous `Test*` renames surfaced by `qa-fast-advisory` `test_removed`, overlapping sibling test harnesses, or test edits not tied to concrete bugs, coverage gaps, surviving mutants, or `MUSTFIX` limits.
- **Walkthrough Reviewer** (`Gemini 4 Neon (XHigh)`, `workspace: "inherit"`):
  - Strictly read-only on repository files; runs once after Verifier `VERDICT: PASS` to build a user-review checklist comparing the **original verbatim user request** (not just Planner's plan), Planner's decisions/`Non-Goals`/`QUESTIONS_FOR_USER`, external callers, retained advisories, and full `git diff <BASE_REV>`.
  - Drafts `<appDataDir>/brain/<conversation-id>/walkthrough.md` (published by Coordinator as a user-facing artifact; **never** written into the repository working tree) covering:
    - `[implicit decisions]`: specs, defaults, error handling, edge-case rules, or API/config behaviors decided unilaterally without explicit user input (`file:line`, rationale, and alternatives to confirm).
    - `[regression surface]`: caller- or user-visible behavioral shifts (nil/empty/ordering/concurrency/timeout/error semantics), why they appear safe, and exact `file:line` spots warranting human inspection.
    - `[retained advisories & trade-offs]`: `NO_ACTION` uncovered branches/mutants, intentionally retained exported-signature `MUSTFIX` items, and out-of-scope follow-ups.
    - `[manual & unverified boundaries]`: live GCP/external-API, UI/visual, or environment-dependent behaviors not covered by offline automated tests.
    - `[review guide & verification]`: recommended file reading order and concise summary of executed checks.
  - Emits `STATUS: READY` (non-blocking—surfaces all implicit decisions and trade-offs for user judgment) or `STATUS: BLOCKING_REGRESSION` (`file:line`, **only** for a concrete accidental regression bug missed by Verifier; triggers at most **1** targeted fix + Verifier + fresh Walkthrough Reviewer pass, after which any residual risk is escalated in `walkthrough.md`).

## Rules and Automated Signals

Do not duplicate Go/TS coding rules in worker briefs. Point workers directly to:

- **Repository Rules**: `.agents/rules/` (`go-coding-rule.md`, `ts-coding-rule.md`, `comment-guideline.md`, `storybook-guideline.md`, `khi-task-rule.md`, `proto-coding-rule.md`, `test-data-rule.md`) and `.agents/skills/backward_compatibility_policy/SKILL.md`.
- **Declarative Decision Rules**: `scripts/codequality/rules/*.yaml` (opt-in via `CODEQUALITY_DECISIONS=true` when API credentials are configured—triaged by Auditors through `QAReport.advisory[]`; otherwise consulted directly by Auditors offline).
- **Blocking Gates**: `make qa-fast BASE=<BASE_REV>`, `make qa-fast-must BASE=<BASE_REV>`, `make qa-slow-must BASE=<BASE_REV>`, `make pre-commit`, `go build ./...`, targeted `go build <pkgs>` / `go test -race -count=1 <pkgs>`, and `make test-web`.
- **Empty-Tree Full-Package `MUSTFIX` Gate** (package/directory conformance or cleanup tasks): in addition to `make qa-fast-must BASE=<BASE_REV>`, run `go run ./scripts/codequality qa --speed=fast --base=4b825dc642cb6eb9a060e54bf8d69288fbee4904 --paths=<target-paths> --filter=must` so pre-existing violations above hard limits aren't masked by the differential ratchet (note in the final report any exported-signature violations intentionally retained to avoid breaking out-of-scope callers).
- **Advisory JSONs** (input for Auditors and Verifier; always pass `--base=<BASE_REV>` so merged commits are not skipped by `qa`'s `CurrentOnly` default):

```bash
go run ./scripts/codequality qa --base=<BASE_REV> --speed=fast --filter=advisory --format=json
go run ./scripts/codequality qa --base=<BASE_REV> --speed=slow --filter=advisory --format=json
```

## Operational Hygiene

- **Fresh Agents Per Task and Round**: Spawn a fresh agent for every workstream, audit shard, fix round, verification pass, and walkthrough review. Never reuse stale workers across rounds.
- **Merge, Kill, Then Delete Branch**: `swarm{action: "kill"}` deletes `share` workspaces permanently, and git refuses to delete a branch checked out in a live worktree until `kill` removes it. Verify the worker's commit SHA, run `git merge --no-edit <worker-branch>` in the main workspace, call `swarm{action: "kill"}`, then run `git branch -D <worker-branch>` (Planners/workers spawning helpers must also `kill` all helpers before `[done]`; **exception**: Verifier's disposable `Breaker` runs without branch creation and is `kill`ed **without merging** to discard scratch probes).
- **Share Worktree Codegen Boundary**: `workspace: "share"` omits `.gitignore`d generated/embedded assets (`pkg/generated/zzz_register_inspection.go`, `pkg/server/dist/browser/`, `pkg/model/khifile/v6/style/assets/`). Verify targeted packages (`go build <pkgs>`, `go test -race <pkgs>`) in `share`, and run repo-wide `go build ./...` / `make qa-slow-must` only in a generated workspace.
- **Board Discipline**: Treat `@name` strictly as a wakeup doorbell—never CC `@coordinator` on peer-to-peer handoffs, **ban negative `@<other-agent>` mentions** in worker briefs (even in prohibitions; write `Report ONLY to @<lead>`) so non-target coordinators are never woken spuriously, and forward any brief-level board-isolation (`since` floor) constraints when spawning helpers. Cite `#seq` instead of re-quoting posts; sleep with `swarm_wait(since: <highest_seq_read>)` and never poll.

## Pipeline

1. **Plan**: Spawn a fresh **Planner** (`Gemini 4 Neon (XHigh)`, `inherit`). Planner fans out parallel **Scouts** (`Gemini 4 Hydrogen (High)`, `inherit`), kills helpers, and outputs cohesive disjoint workstreams, `Non-Goals`, 1-to-1 change justifications, invariants/state-transition contracts, and `QUESTIONS_FOR_USER`. If `QUESTIONS_FOR_USER` is not `NONE`, Coordinator escalates to the user before editing code.
2. **Parallel Disjoint Implement + Merge/Kill**: Record `BASE_REV=$(git rev-parse HEAD)`. Spawn one fresh **Implementer** (`Gemini 4 Hydrogen (High)`, `share` if >1, `inherit` if solo) per workstream. Each Implementer stages/commits edits on a local branch (`git add -A` + commit) before running `make qa-fast BASE=<BASE_REV>` and targeted package checks. Coordinator verifies the commit SHA, runs `git merge --no-edit <worker-branch>`, calls `swarm{action: "kill"}`, and runs `git branch -D <worker-branch>`.
3. **Blocking Gates + Capture Advisory JSONs**: In the generated main workspace, run blocking gates (`make qa-fast-must BASE=<BASE_REV>`, plus empty-tree `go run ./scripts/codequality qa --speed=fast --base=4b825dc642cb6eb9a060e54bf8d69288fbee4904 --paths=<target-paths> --filter=must` for package/directory conformance or cleanup tasks, `make qa-slow-must BASE=<BASE_REV>` as applicable, `go build ./...`, targeted `go test -race -count=1`, `make test-web`) and capture `qa --base=<BASE_REV> --speed=fast --filter=advisory --format=json` and `--speed=slow --filter=advisory --format=json`. Note any intentionally retained exported-signature violations in the final report.
4. **Parallel Advisory Audit**: Shard `QAReport.advisory[]` items (and any assigned review scopes) by file or package across fresh **Auditors** (`Gemini 4 Hydrogen (XHigh)`, `inherit`). Collect `ACTION_REQUIRED` / `NO_ACTION` verdicts and `kill` all Auditors.
5. **Targeted Fresh Fix Round (if `ACTION_REQUIRED`)**: Spawn fresh **Implementer(s)** on disjoint scopes; each stages/commits before running `make qa-fast BASE=<BASE_REV>`. Verify SHA, `git merge --no-edit`, `kill`, and `git branch -D`, then re-run blocking gates (`BASE=<BASE_REV>`) and advisory capture (`--base=<BASE_REV>`).
6. **Fresh Verifier Gate**: Spawn a fresh **Verifier** (`Gemini 4 Neon (XHigh)`, `inherit`) with Planner instructions, `Non-Goals`, invariants, `git diff <BASE_REV>` (not bare `git diff`), and `qa-fast-advisory` signals; require the 5-bullet adversarial trace (including `[scope & diff minimality]`) with `file:line` citations and 1-line `Breaker:` summary (`PASS` / `FAIL` / `N/A`; disposable `Breaker` spawned & `kill`ed unmerged when diff triggers) before `VERDICT: PASS`. On `VERDICT: FAIL`, run a fresh fix (preserving any Breaker reproducer as a regression test) + gate + fresh Verifier pass. On `VERDICT: PASS`, `kill` the Verifier.
7. **Fresh Walkthrough Reviewer**: Spawn a fresh **Walkthrough Reviewer** (`Gemini 4 Neon (XHigh)`, `inherit`) with the **verbatim user request**, Planner's plan/`Non-Goals`/`QUESTIONS_FOR_USER`, `git diff <BASE_REV>`, and Auditor/Verifier outcomes. On `STATUS: BLOCKING_REGRESSION` (concrete accidental regression bug only; max **1** bounce), run a fresh fix + gate + fresh Verifier + fresh Walkthrough Reviewer pass. On `STATUS: READY`, `kill` the Walkthrough Reviewer.
8. **Publish `walkthrough.md` & Report to User**: Coordinator writes the Walkthrough Reviewer's checklist to `<appDataDir>/brain/<conversation-id>/walkthrough.md` as a user-facing artifact (never inside the repository), then replies concisely linking `walkthrough.md` and highlighting implicit decisions or confirmation items needing user input.
