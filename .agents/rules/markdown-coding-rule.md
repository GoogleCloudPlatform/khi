---
trigger: glob
globs: **/*.md
---

# KHI Markdown Standards

When developing or modifying Markdown files in the KHI project, you **must** adhere to the following rules and best practices.

## 1. Verifications

- Run `make lint-markdown` to check style and formatting.
- **MUST** invoke a subagent for code review before asking the user for verification. See Section 4 for the procedure. **If you make any modifications based on the review, you MUST run the review again to verify the changes.**

## 2. Language and Style

- **Keep your English plain and simple**. Avoid complex vocabulary or overly long sentences. Ensure that the documentation is easy to understand for non-native speakers.
- **We also provide Japanese documents**:
  - When you update markdown files intended to be read by AI agents (e.g., `.agents/*/_/*.md`), you just need to provide them in English.
  - When you update markdown files under `docs/en` folder, please update the corresponding file under `docs/ja`.
  - When you update `README.md` files under a code folder (e.g., `pkg/task/`), please update the corresponding file under `README_ja.md`.

## 3. Formatting

- Avoid MD029 errors by ensuring proper list item prefixes or structure (e.g., indenting bullet points under a numbered list).

- Follow standard Markdown best practices as enforced by `markdownlint`.

## 4. Subagent Review Guidelines

Follow these rules to perform code reviews using a single temporary subagent.

- **Define**: Use `define_subagent` to create a temporary subagent specialized for Markdown/Documentation.
- **Invoke**: Use `invoke_subagent` to start the review.
- **Capabilities**: The subagent can read files, perform web searches, and list directories.

### Reviewer Role

**Markdown Reviewer** (`md_reviewer`): Focuses on clarity (plain English), formatting (markdownlint compliance), and I18n requirements.

### Example Format

**`define_subagent`**

```json
{
  "name": "md_reviewer",
  "description": "General Markdown reviewer for clarity, formatting, and I18n.",
  "prompt_sections": [
    {
      "title": "Checklist",
      "content": "- Plain and simple English\n- No markdownlint errors\n- Japanese translation if required by rules"
    },
    {
      "title": "Constraints",
      "content": "- Do NOT execute CLI commands. `run_command` tool is strictly prohibited."
    }
  ],
  "tool_names": [
    "view_file",
    "search_web",
    "read_url_content",
    "list_dir",
    "grep_search"
  ]
}
```

**`invoke_subagent`**

```json
{
  "TypeName": "md_reviewer",
  "Role": "Markdown Reviewer",
  "Prompt": "Review the changes in: [file paths]"
}
```
