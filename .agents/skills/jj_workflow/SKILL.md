---
name: jj-workflow
description: Guidelines and procedures for using Jujutsu (jj) in KHI, including change management, workspace isolation, pushing commits with auto-generated bookmarks, addressing PR review feedback without squashing, and resolving conflicts incrementally from oldest to newest commit.
---

# KHI Jujutsu (`jj`) Workflow Guidelines

This guide outlines the standard version control workflow for working on the Kubernetes History Inspector (KHI) project using Jujutsu (`jj`).

---

## 1. Core Principles

- **Prefer Jujutsu (`jj`) Commands**: Always use `jj` commands instead of raw Git commands whenever possible.
- **Working Copy is Always a Change**: In Jujutsu, the working copy (`@`) is treated as an in-progress change/commit at all times.
- **Describe vs. New**:
  - `jj describe -m "..."`: Updates the commit message of the _current_ change (`@`). It does not create a new commit.
  - `jj new`: Commits and freezes the current change, then creates a new, empty working-copy change (`@`) on top of it.
- **Task Boundaries**: Always run `jj new` upon completing a logical unit of work or before switching tasks.
- **Workspace Isolation**: Always make changes strictly within your assigned workspace. You may inspect or reference other workspaces, but never modify files or commit changes in another workspace.

---

## 2. Workspace Isolation

> [!IMPORTANT]
> **Never modify files or state in other workspaces.**
>
> - Inspecting or referencing other workspaces (e.g., viewing files or reading logs) is allowed.
> - However, you must NEVER modify, edit, or create files in another workspace.
> - All changes, commits, and development work must take place strictly within your assigned workspace.

---

## 3. Standard Development Flow

### Starting a New Change

Before starting work on a new feature or bugfix, verify that your working copy is clean with `jj status`, then create a new change:

```bash
# Create a new working-copy change
jj new -m "feat(scope): short description"
```

### Developing and Reviewing Changes

Edit files as needed. Check your progress and inspect changes:

```bash
jj status
jj diff
```

Update the commit message as work evolves:

```bash
jj describe -m "feat(scope): detailed message"
```

### Testing and Formatting

Before finalizing any commit or boundary, execute KHI test and verification commands:

```bash
make pre-commit
make lint-go && make lint-web
make test-go && make test-web
```

### Freezing the Change

When the task is complete and verified:

```bash
jj new
```

This freezes your completed change into the commit graph history (`@-`) and moves your working copy (`@`) to a new empty change.

---

## 4. Pushing Changes & Remote Bookmarks

> [!IMPORTANT]
> **Do not invent custom remote branch names manually.**
> Always let `jj` generate and manage bookmark names automatically.
> The remote `origin` points directly to `GoogleCloudPlatform/khi`.

When you need to push a change to the remote repository for review or PR creation:

```bash
# Push the change at @; jj automatically creates and attaches a bookmark named push-<change_id>
jj git push -c @
```

- Jujutsu automatically assigns a unique bookmark name (e.g., `push-qpzqrvsmmnvr`) and pushes it to the remote Git repository (`origin`).
- Use this auto-generated bookmark name as the head branch when opening a Pull Request on GitHub.
- For stacked PRs involving multiple linear commits, push each commit to create its corresponding remote bookmark:

```bash
jj git push -c <rev1> -c <rev2>
```

- If the change already has a tracked remote bookmark, update it with:

```bash
jj git push --tracked
```

---

## 5. Addressing PR Review Feedback (No Squashing)

> [!IMPORTANT]
> **Do NOT squash review fixes into the original commit (`jj squash` is forbidden here).**
> Reviewers must be able to inspect incremental diffs across review rounds. Always preserve review iteration history.

When review comments require code changes:

1. **Create a New Child Revision on the PR Commit**:
   Start a new change directly on top of the PR commit:

   ```bash
   # If @ is already the PR commit:
   jj new

   # Or target the PR commit/bookmark explicitly:
   jj new <pr-bookmark-or-commit>
   ```

2. **Apply the Fixes**:
   Modify the source files to address the reviewer feedback.

3. **Verify and Format**:
   Run formatters and tests to ensure no regressions:

   ```bash
   make format-go && make format-web
   make test-go && make test-web
   ```

4. **Set a Descriptive Commit Message**:

   ```bash
   jj describe -m "fix(scope): address review comments on XYZ"
   ```

5. **Advance or Move the PR Bookmark to the New Commit**:
   Update the PR bookmark to track the newly created fix commit (`@`):

   ```bash
   # Option A: Advance the closest bookmark forward to @
   jj bookmark advance

   # Option B: Move the specific PR bookmark to @
   jj bookmark move <pr-bookmark-name> --to @
   ```

6. **Cascade Rebase for Stacked PRs (When Applicable)**:
   If this PR has child PRs stacked on top of it, linearly rebase the child PR's root commit onto the updated bookmark:

   ```bash
   jj rebase -s <first-commit-of-next-pr> -d <pr-bookmark-name>
   ```

   Do NOT use merge commits in a stack.

7. **Push the Updated PR**:

   ```bash
   jj git push
   ```

   The GitHub PR will automatically show the new incremental commit, allowing reviewers to verify changes easily.

---

## 6. Stacked PR Management & Post-Merge Rebasing

KHI uses GitHub Stacked PRs (`gh stack`) and Merge Queue for development.

### Linear Stack Graph in Jujutsu

Keep all PR commits strictly linear without branching or merge commits:
`main@origin -> rev1 (PR 1) -> rev2 (PR 2) -> rev3 (PR 3) ...`

Each PR commit has its own auto-generated bookmark (`push-xxxx`) pushed to `origin`.

### Post-Squash Merge Rebasing

GitHub's Merge Queue merges PRs using the `SQUASH` method, resulting in a single squashed commit on `main@origin`. The original individual commit history of the merged PR is replaced on `main`.

When a bottom PR (e.g., PR 1) is squash-merged into `main`:

1. **Fetch the Updated Remote**:

   ```bash
   jj git fetch --remote origin
   ```

2. **Rebase Remaining Stack Commits onto `main@origin`**:
   Rebase the root commit of the remaining unmerged PRs (e.g., PR 2) directly onto `main@origin`:

   ```bash
   jj rebase -s <first-commit-of-remaining-bottom-pr> -d main@origin
   ```

3. **Abandon Old Pre-Squash Revisions**:
   Abandon the old local commits of the merged PR to keep the revision graph clean:

   ```bash
   jj abandon <old-squashed-commit-range>
   ```

4. **Push the Rebased Stack**:

   ```bash
   jj git push
   ```

---

## 7. Conflict Resolution Workflow (Oldest to Newest)

In Jujutsu, merge and rebase conflicts do not abort operations. Instead, conflicts are recorded as first-class states directly on the affected commits.

> [!IMPORTANT]
> **Always resolve conflicts starting from the oldest conflicted commit first.**
> Resolving the oldest commit frequently resolves downstream conflicts automatically and prevents cascading conflict cycles.

### Step-by-Step Resolution Procedure

1. **Identify Conflicted Commits**:
   List all revisions currently in a conflicted state:

   ```bash
   jj log -r 'conflicts()'
   ```

2. **Select the Oldest Conflicted Commit**:
   Examine the commit graph and pick the oldest (root-most) ancestor among the conflicted revisions. Let this be `<conflicted-rev>`.

3. **Spawn a Working Copy on the Conflicted Commit**:
   Create a new child revision directly on the conflicted commit:

   ```bash
   jj new <conflicted-rev>
   ```

   The working copy (`@`) now contains the conflicting files with conflict markers (`<<<<<<<`, `>>>>>>>`, `%%%%%%%`).

4. **Resolve Conflict Markers**:
   Edit all conflicting files and resolve the markers manually to achieve the desired clean state.

5. **Verify the Resolution**:
   Run linters and tests to verify that the resolution builds cleanly and passes all test suites:

   ```bash
   make lint-go && make lint-web
   make test-go && make test-web
   ```

6. **Squash the Resolution into the Conflicted Commit**:
   Once all tests pass, squash the working copy back into the parent commit (`<conflicted-rev>`):

   ```bash
   jj squash
   ```

   This moves the clean, resolved content into `<conflicted-rev>`, clearing its conflicted state.

7. **Repeat for Remaining Downstream Conflicts**:
   Jujutsu automatically rebases any child commits on top of the resolved commit. Re-check for remaining conflicts:

   ```bash
   jj log -r 'conflicts()'
   ```

   If any commits still have conflicts, repeat steps 2 through 6 for the next oldest conflicted commit until `conflicts()` is completely empty.

---

## 8. Pre-flight Checklist

Before finalizing any task or pushing to a remote repository:

- [ ] Worked strictly within the assigned workspace and made no cross-workspace modifications.
- [ ] Executed `make pre-commit` to format all code and documentation files.
- [ ] Passed all linters and tests (`make lint-go`, `make lint-web`, `make test-go`, `make test-web`).
- [ ] For PR reviews: created a new incremental commit with `jj new` without squashing.
- [ ] For PR reviews: advanced or moved the PR bookmark with `jj bookmark advance` or `jj bookmark move`.
- [ ] For stacked PR reviews: cascade-rebased downstream stack commits with `jj rebase -s <next-pr-root> -d <pr-bookmark>`.
- [ ] For conflict resolution: resolved from oldest to newest commit using `jj new` -> test -> `jj squash`.
