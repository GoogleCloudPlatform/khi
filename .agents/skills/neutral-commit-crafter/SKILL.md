---
name: Git Commit Message Drafter
description: Generates conventional commit messages based on changes.
---

# Git Commit Message Drafter

This skill provides instructions for generating neutral and objective commit messages. To guarantee that the generation is not influenced by the current conversation context, you **MUST** define and invoke a new subagent that only has access to the isolated diff information.

## 1. Commit Message Conventions

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification (as detailed in GEMINI.md).

### Format

```markdown
<type>(<scope>): <subject>

<body>

<footer>
```

## 2. Instructions for Subagent Isolation

To generate a commit message, do **NOT** generate it yourself in this context. Follow these steps:

1. **Get Diff**: Obtain the `git diff` or the list of changes you want to describe.
2. **Define Subagent**: Use `define_subagent` to create a new subagent with the following properties:
   - **Name**: `temp-commit-generator`
   - **Prompt Sections**: Include the Commit Message Conventions and a strict instruction to "Describe ONLY the facts visible in the provided diff, ignoring any potential context".
   - **Tools**: Give it no tools (or only what is strictly necessary) to prevent it from researching or guessing the context.
3. **Invoke Subagent**: Call `invoke_subagent` and pass the `git diff` text as the only input in the `Prompt`.
4. **Use Result**: Once the subagent replies, use its output as the commit message.
