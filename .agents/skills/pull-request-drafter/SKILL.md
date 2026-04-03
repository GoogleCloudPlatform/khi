---
name: Pull Request Drafter
description: Use this skill when you want to draft a GitHub Pull Request with a standardized structure.
---

# Pull Request Drafter

This skill provides instructions for generating a structured GitHub Pull Request title and description. It ensures that alternative solutions are brainstormed neutrally by an isolated subagent.

## 1. Output Format

You must output the **Title** and the **Description** separately.

1. **Title**: Output the Title for the Pull Request in English (outside the description code block).
2. **Description Code Block**: The entire description (English and Japanese) **MUST** be wrapped in a single markdown code block (triple backticks) to make it easy to copy.

Inside the description code block, it must contain the following sections in order:

1. `## Motivation`: Explain why this change is needed. If the motivation is unclear from the context or the diff, you **MUST** ask the user for clarification.
2. `## Changes`: Summarize what was changed. Do not just list changed files; describe the impact. If changes are complex, include a Mermaid diagram to visualize relationships or transitions.
3. `## Alternative considered`: Summarize the brainstormed alternatives and explain why they were not chosen for the current plan.
4. **Japanese Translation**: The full Japanese translation of the Title and Description sections, appended at the bottom.

## 2. Instructions for Alternative Considered

To generate the "Alternative considered" section, follow these steps:

1. **Obtain Data**: Get the `git diff` and determine the `Motivation`.
2. **Define Subagent**: Use `define_subagent` to create a temporary subagent.
   - **Name**: `alternative-brainstormer`
   - **Properties**: Strictly limit its context to prevent bias. Provide only the `git diff` and the `Motivation`. Do not provide previous conversation history.
3. **Invoke Subagent**: Ask the subagent to "Brainstorm alternative solutions based ONLY on the provided diff and motivation. Do not use any other context."
4. **Evaluate Output**: Once you receive the subagent's response, **evaluate why those alternatives might NOT be suitable for the current plan or context**. If you have doubts or unresolved trade-offs, you **MUST** ask the user for clarification.
5. **Integrate Output**: Combine the subagent's response and your evaluation for the `## Alternative considered` section.

> [!IMPORTANT]
> If the `Motivation` changes during the drafting process, you **MUST** re-invoke the subagent to regenerate the `Alternative considered` section, as the alternatives strongly depend on the motivation.

## 3. Translation

After generating the English title and description, translate the entire text into Japanese and append it at the bottom of the output.
