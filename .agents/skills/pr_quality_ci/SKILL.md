---
name: pr-quality-ci
description: >-
  Runs KHI pre-push code quality gates with Cloud Build CI parity, creates or
  updates GitHub Pull Requests, monitors GitHub Actions and Cloud Build checks
  asynchronously, reruns transient CI failures, triages sticky codequality
  reports and automated/human review comments, pushes incremental jj fix
  commits, and posts review replies and PR summary comments. Use when verifying
  code quality before pushing, creating a PR, watching PR CI, rerunning or
  fixing CI failures, or addressing PR review and codequality advisories.
---

# KHI Code Quality & PR CI Workflow

This guide defines the end-to-end workflow for preventing pre-push rework and autonomously monitoring, rerunning, repairing, and commenting on GitHub Pull Requests in Kubernetes History Inspector (KHI).

Follow the `jj-workflow` skill for Jujutsu branch, stack, and conflict mechanics. Follow `.agents/rules/go-coding-rule.md`, `.agents/rules/ts-coding-rule.md`, and `.agents/rules/proto-coding-rule.md` for language-specific style rules and thresholds.

---

## 1. Core Principles

- **Two Commands Cover Local Verification**: Use `make pre-commit` (`make qa-fast-must` + formatters) for fast inner-loop auto-formatting and static checks on `@`, and `make pre-push` (`make qa-all-must` + `make qa-slow-advisory`) before pushing to run full GitHub Actions and Cloud Build CI-parity checks automatically scoped to changed files.
- **Fix Root Causes, Never Game Metrics**: Never delete GoDoc comments to reduce line counts, never split functions into 6+ parameter pass-through helpers, never delete defensive guards or capacity pre-allocations to silence `func_uncovered_lines`/`func_surviving_mutants` advisories, and never suppress invariant or unreachable branches with `CODEQUALITY_ALLOW`.
- **Green Checkmarks Are Necessary, Not Sufficient**: Go/TS coverage, mutation, redundancy (`qa-slow-advisory`), offline diff anti-Goodhart signals (`qa-fast-advisory` via `advisorycheck`), and `Similarity` matches emit non-blocking `ADVISORY` items (Exit `0`) in JSON and the sticky PR comment (`<!-- khi-codequality-report -->`), and `gemini-code-assist[bot]` posts inline review comments even when all checks pass. Always inspect both after checks complete.
- **No-Squash Incremental Commits**: Never run `jj squash` once a PR is pushed. Always create incremental child commits (`jj new <pr-bookmark>`) so reviewers can inspect round-by-round diffs.

---

## 2. Stage 1: Local Verification (`make pre-commit` & `make pre-push`)

Local verification is consolidated into two diff-aware `make` targets so you do not need to memorize per-language command matrices.

### 1. Inner-Loop Auto-Format & Static Check (`make pre-commit`)

Run `make pre-commit` while your working copy (`@`) contains modified files (before `jj new`):

```bash
make pre-commit
```

- Automatically detects modified files in `@` (exiting with an explicit error if `@` is empty) and runs **only** the formatters and fast static checks for touched domains: Apache-2.0 headers (`add-licenses`), Proto (`format-proto`, `build-proto`), package-scoped Go (`format-go`), Web (`format-web`), config YAML/JSON (`format-misc`), Markdown (`format-md`), and `make qa-fast-must` (which runs `lint-go` including `codequality conventions`, `lint-web`, `lint-md`, `lint-proto`, and fast Go AST rules).
- Combine with targeted unit tests (for example, `go test ./pkg/path/to/pkg/...`) while iterating on specific logic.

### 2. Pre-Push Full CI-Parity Gate (`make pre-push`)

Run `make pre-push` before `jj git push`:

```bash
# Standard PR targeting main:
make pre-push

# Stacked child PR targeting a parent PR bookmark:
make pre-push BASE=<parent-push-bookmark>
```

- Inspects the full branch diff (`fork_point(BASE | @)..@`, defaulting to `BASE=trunk()`), so it works identically **before or after** `jj new`.
- Runs `make qa-all-must` across only the CI suites affected by your branch diff in fail-fast order, matching `.github/workflows/pullrequest.yaml` and `.cloudbuild/github-codequality.yaml`:
  - Static & license checks (`check-licenses`, `lint-md`, `check-format-misc`, `lint-proto` + `build-proto`)
  - Go parity (`lint-go` including `codequality conventions` and `unused-exports`, `qa-fast-must`, `test-go`, cross-OS `build-go-binaries`)
  - Frontend parity (`check-format-web`, `lint-web`, `build-web`, `build-web-dev`, `build-storybook`, `test-web`)
  - Differential `qa-slow` with **unlimited mutants** (`CODEQUALITY_MAX=0`, `CODEQUALITY_TIMEOUT=10m`)
- Optional environment overrides: add `USE_CONTAINER=true` when headless Chromium is available only inside the builder container, or `CODEQUALITY_SIMILARITY=true` (when not already enabled in `~/.config/khi/codequality.mk`) to preview Vertex AI AST-normalized function similarity advisories locally.

---

## 3. Stage 2: Remediating `codequality` Findings

Resolve `codequality` findings in this strict priority order:

1. **`RATCHET_REGRESSION` (`after > before` on a legacy unit)**:
   - Legacy units already exceeding a default budget enforce a strict no-worsening ratchet (`after <= before`).
   - `CODEQUALITY_ALLOW` **never** suppresses `RATCHET_REGRESSION` (adding one causes an `unused CODEQUALITY_ALLOW` `DirectiveError`). Refactor or add tests until `after <= before`.
2. **Advisory Coverage, Mutation & Diff Signals (`qa-slow-advisory`, `qa-fast-advisory`)**:
   - **Never Game Advisory Metrics**: `func_uncovered_lines`, `func_surviving_mutants`, `test_redundancy` (`qa-slow-advisory`), and `advisorycheck` signals (`test_removed`, `test_cases_reduced`, `test_assertion_weakened`, `test_skip_added`, `test_stress_reduced`, `test_sleep_added`, `suppression_added`, `orphan_test_file` in `qa-fast-advisory`) report Exit `0` for human/auditor review. Never delete legitimate defensive guards, pre-sizing hints, or pre-existing tests—and never add `CODEQUALITY_ALLOW`—just to silence advisory items.
   - **Step A — Delete genuinely dead code**: Remove only provably unreachable branches or duplicate subtests introduced in your current change.
   - **Step B — Strengthen assertions**: Compare whole structs with `cmp.Diff(tc.want, got)` and add boundary inputs (`ROR`/`LCR` operators). In TypeScript, test real DOM/layout behavior in Headless Chromium or extract pure `*calculator.ts` / `*.viewmodel.ts` helpers.
   - **Step C — Document intentionally retained advisories**: When an advisory item stems from a genuine external I/O boundary, equivalent mutant, or deliberate guard, leave the code clean and note the rationale in the PR summary comment if helpful rather than adding `CODEQUALITY_ALLOW`.
3. **Redundant Tests (`test_redundancy`: `SAME PATH` / `NO OUTPUT CHECK`)**:
   - If a subtest or Jasmine `it(...)` spec added in your current change is dominated on the same branch-polarity path or checks only `err == nil` / `toBeTruthy()`, assert distinct output values so it kills a unique mutant, or consolidate the duplicate case you introduced.
   - Never delete pre-existing test cases or weaken pre-existing assertions from `main` without asking the user first.
4. **Function Similarity Advisories (`Similarity`)**:
   - Inspect each matched candidate (`[shared]` helper or domain function). Reuse or extract a shared helper when both units represent the same domain logic; if symmetry is intentional (e.g., decoupled standalone CLIs or separate protocol schemas), document that rationale in your PR comment in Stage 5.

---

## 4. Stage 3: Creating the PR & Asynchronous CI Monitoring

### Pushing & Opening the Pull Request

Verify, freeze, and push an auto-generated `push-*` bookmark per `jj-workflow`:

```bash
make pre-commit
make pre-push
jj describe -m "feat(scope): concise summary"
jj new
jj git push -c @-
```

Create the PR on `GoogleCloudPlatform/khi` (`--base main` for root PRs, or `--base <parent-push-bookmark>` for stacked PRs) with `## Summary` and `## Verification` sections:

```bash
gh pr create --repo GoogleCloudPlatform/khi \
  --base main \
  --head <push-bookmark> \
  --title "feat(scope): concise summary" \
  --body-file <scratch-body-file>
```

### KHI CI Check Suite Overview

| Check / Workflow | Runner | Typical Duration | What It Verifies |
| :--- | :--- | :--- | :--- |
| `PR Tests / markdownlint` | GitHub Actions | ~10s | `markdownlint-cli2` across `.md` files |
| `PR Tests / license-header-check` | GitHub Actions | ~35s | Google LLC Apache-2.0 headers (`make check-licenses`) |
| `PR Tests / proto-checks` | GitHub Actions | ~1–2m | `make lint-proto` and `make build-proto` |
| `PR Tests / build-web` | GitHub Actions | ~3m | Production frontend bundle |
| `PR Tests / backend-tests` | GitHub Actions | ~4–5m | `go test ./...`, `check-format-misc`, `codequality conventions`, `golangci-lint` |
| `PR Tests / frontend-tests` | GitHub Actions | ~5m | `make test-web`, `check-format-web`, `lint-web` |
| `PR Tests / build-frontend (prod, dev)` | GitHub Actions | ~4–5m | Angular `prod`/`dev` builds + `npm run build-storybook` |
| `PR Tests / build-binaries` | GitHub Actions | ~5–6m | Cross-OS/arch Go binaries (`windows`, `linux`, `darwin`) |
| `CodeQL / Analyze (*)` | GitHub Actions | ~1–3m | Static security analysis for Go, JS/TS, and Actions |
| `github-codequality (kubernetes-history-inspector)` | Cloud Build | ~6–30m | Differential Go/TS AST, coverage, unlimited mutation, redundancy, similarity + sticky PR report |
| `github-deploy-ondemand (kubernetes-history-inspector)` | Cloud Build | ~15–65m | Container image build and preview deployment |

### Asynchronous Monitoring (No Busy-Polling)

Never run `sleep` loops in a shell to poll CI. Watch checks asynchronously with `--fail-fast` so execution wakes immediately on the first failure without waiting up to an hour for `github-deploy-ondemand`:

```bash
gh pr checks <pr_number> --repo GoogleCloudPlatform/khi --watch --fail-fast --interval 30
```

- Run the command above as a background command (or use native non-blocking timer/scheduler tools when inspecting intermediate check states).
- Exit codes for `gh pr checks`: `0` (all passed), `1` (at least one failed), `8` (checks pending when invoked without `--watch`).

---

## 5. Stage 4: Failure Triage, Transient CI Re-Runs & Review Inspection

Whenever `--watch` wakes on a failure OR when `PR Tests/*` and `github-codequality` complete:

### 1. Resolve Current PR `HEAD` Commit

```bash
gh pr view <pr_number> --repo GoogleCloudPlatform/khi --json headRefOid,statusCheckRollup,reviews,comments
```

Record `HEAD_SHA` (`headRefOid`) and its 12-character prefix `SHORT_SHA` (`${HEAD_SHA:0:12}`).

### 2. Inspect Failing Logs

- **GitHub Actions (`PR Tests/*`, `CodeQL/*`, `Code Quality PR Comment`)**:
  Extract `<run_id>` from `detailsUrl` (`.../actions/runs/<run_id>/job/...`) and read failed step logs:

  ```bash
  gh run view <run_id> --repo GoogleCloudPlatform/khi --log-failed
  ```

- **Cloud Build (`github-codequality`, `github-deploy-ondemand`)**:
  1. Read the Check Run log tail from GitHub:

     ```bash
     gh api "repos/GoogleCloudPlatform/khi/commits/${HEAD_SHA}/check-runs" \
       --jq '.check_runs[] | select(.name | startswith("github-codequality")) | .output.text'
     ```

  2. When full logs are needed (for example, early compilation/setup exit or truncated output), extract `<build_id>` from `detailsUrl` (`.../builds;region=asia-northeast1/<build_id>?project=...`) and run:

     ```bash
     gcloud builds log <build_id> --region=asia-northeast1 --project=kubernetes-history-inspector
     ```

### 3. Re-Run CI When the Failure Is Transient (No Code Push Needed)

Classify the failure before editing any files:

- **Re-run CI (transient / infrastructure flake)** when logs show external network or registry errors (`npm ci` 5xx/timeout, `apt-get` mirror failure, `go mod download` or cache timeout, image pull error), Cloud Build runner preemption, transient Vertex AI / GitHub API `429`/`502`/`503`/`504` errors, or an isolated headless browser disconnect on unmodified specs that pass locally.
- **Do NOT re-run CI (deterministic failure)** for compile, format, lint, license header, proto, test assertion, AST budget, uncovered line, or surviving mutant failures — proceed to Stage 5 to fix them locally.

Use the runner-specific re-run command (cap transient retries at **2 attempts** per check on the same `HEAD_SHA`):

- **GitHub Actions Re-Run (Failed Jobs Only)**:

  ```bash
  gh run rerun <run_id> --repo GoogleCloudPlatform/khi --failed
  ```

  > [!NOTE]
  > Do not pass the browser URL's `/job/<number>` directly to `gh run rerun --job <id>` (GitHub returns `404 NOT FOUND`). Prefer `--failed`, or look up `databaseId` via `gh run view <run_id> --repo GoogleCloudPlatform/khi --json jobs --jq '.jobs[] | {name, databaseId}'`.

- **Cloud Build Check Run Re-Run (Targeted)**:
  Look up the failed `<check_run_id>` on `HEAD_SHA` and re-request only that check run:

  ```bash
  gh api "repos/GoogleCloudPlatform/khi/commits/${HEAD_SHA}/check-runs" \
    --jq '.check_runs[] | select(.name | startswith("github-codequality")) | {id, name, conclusion}'

  gh api --method POST "repos/GoogleCloudPlatform/khi/check-runs/<check_run_id>/rerequest"
  ```

  - **Fallback (`/gcbrun`)**: Only if a Cloud Build Check Run was never created (missed webhook) or `rerequest` does not start a new build, post `/gcbrun` on the PR (`gh pr comment <pr_number> --repo GoogleCloudPlatform/khi --body "/gcbrun"`). Prefer `rerequest` first because `/gcbrun` re-triggers every Cloud Build PR check including `github-deploy-ondemand`.

After triggering a re-run, return immediately to Stage 3 asynchronous monitoring. If the check fails identically after 2 transient retries, treat it as a deterministic bug or escalate to the user.

### 4. Verify Sticky `codequality` Report Freshness

> [!WARNING]
> **Stale Sticky Comment Race**: `.github/workflows/codequality-comment.yaml` runs asynchronously (`on: check_run: [completed]`) ~15–45 seconds *after* `github-codequality` finishes. On multi-commit PRs, `<!-- khi-codequality-report -->` retains the **previous** commit's report until updated. Always confirm the footer contains ``Commit: `<SHORT_SHA>` `` before acting on it.

Once the sticky comment matches `SHORT_SHA`, inspect all sections — including non-blocking TypeScript findings (`Advisory`) and `Similar Existing Go/TypeScript Functions (Advisory)`.

### 5. Inspect Inline Review Comments (`gemini-code-assist[bot]` & Reviewers)

```bash
gh api "repos/GoogleCloudPlatform/khi/pulls/<pr_number>/comments"
```

Evaluate every bot or human comment critically against KHI's architecture and rules:

- **Accept & fix** genuine bugs, edge cases, UTF-8/rune slicing bugs, missing error handling, or clear simplifications.
- **Decline & explain** suggestions that conflict with KHI conventions (for example, `backward-compatibility-policy` forbids adding defensive `nil` fallbacks, legacy wrapper shims, or unused optional parameters to internal code).

---

## 6. Stage 5: Automated Code Fix, Incremental Push & PR Commenting

### 1. Push Fixes as a New Incremental Commit (No Squashing)

Batch all deterministic CI fixes, valid review fixes, and actionable advisory fixes from the current round into a single incremental commit:

```bash
# 1. Create a new child commit on top of the PR bookmark
jj new <pr-bookmark>

# 2. Edit files to resolve all triaged issues

# 3. Auto-format active changes and run full CI-parity verification
make pre-commit
make pre-push          # or: make pre-push BASE=<parent-push-bookmark>

# 4. Describe the commit and advance the PR bookmark
jj describe -m "fix(scope): address CI and review feedback"
jj bookmark move <pr-bookmark> --to @

# 5. If child PRs are stacked on top of <pr-bookmark>, cascade rebase linearly:
# jj rebase -s <first-commit-of-child-pr> -d <pr-bookmark>

# 6. Push the updated bookmark and leave @ on a clean working copy
jj git push --tracked
jj new
```

### 2. Reply to Review Threads & Post Concise PR Comments

Leave a clear, low-noise record on GitHub:

- **Inline Review Thread Replies**:
  Reply directly to each addressed or declined inline comment (`<comment_id>` from `pulls/<pr_number>/comments`):

  ```bash
  gh api "repos/GoogleCloudPlatform/khi/pulls/<pr_number>/comments/<comment_id>/replies" \
    -f body="Fixed in \`<new_short_sha>\`: <concise explanation of the fix>."
  ```

  When declining a comment, reply on the same thread stating why (citing the invariant or KHI policy such as `backward-compatibility-policy`).

- **Top-Level PR Summary Comment**:
  Post a concise comment (`gh pr comment <pr_number> --repo GoogleCloudPlatform/khi --body-file <scratch-comment-file>`) only when:
  1. **Automated CI Code Repair**: Summarize which check failed, the root cause, and the fix pushed in `` `<new_short_sha>` ``.
  2. **Retained Sticky Report Advisories**: When `<!-- khi-codequality-report -->` lists `Similarity` matches or TypeScript advisory items that are intentionally retained (e.g., intentional symmetry across decoupled CLI tools), explain why each retained item was kept as-is so human reviewers have immediate context.
  - **No Spam**: Never post status-polling comments ("Watching CI...", "Checks passed!"). Post at most one top-level summary comment per push round.

### 3. Re-Verify & Enforce Escalation Stop Conditions

After pushing `<new_short_sha>`, return to **Stage 3** to monitor CI on the updated commit.

**Mandatory Stop & Escalation Conditions** — stop automated retries and ask the user immediately if:

- Automated code-fix push rounds reach **3 rounds** (or transient re-runs reach **2 retries** on the same commit) without passing.
- A check fails due to persistent external issues (`cla/google`, Cloud Build IAM/quota errors, prolonged GitHub/GCP outages).
- Resolving a finding requires **deleting pre-existing test cases or weakening pre-existing assertions** (forbidden by root `GEMINI.md` without user approval).
- Resolving a comment or failure requires an architectural change or `.khi` file backward-compatibility decision.
