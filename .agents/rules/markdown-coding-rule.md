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
- **We also provides Japanese documents**:
  - When you update markdown files intented to be read from AI agents(e.g .agents/\*_/_.md), you just need to provide it in English.
  - When you update markdown files under `docs/en` folder, please update corresponded file under `docs/ja`.
  - When you update README.md file under some code folder(e.g pkg/task/), please update corresponded file under `README_ja.md.

## 3. Formatting

- Avoid MD029 errors by ensuring proper list item prefixes or structure (e.g., indenting bullet points under a numbered list).
- Follow standard Markdown best practices as enforced by `markdownlint`.
