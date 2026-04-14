---
trigger: glob
globs: **/*.md
---

# KHI Markdown Standards

- Run `make lint-markdown` to check style and formatting.

## 2. Language and Style

- **Keep your English plain and simple**. Avoid complex vocabulary or overly long sentences. Ensure that the documentation is easy to understand for non-native speakers.
- **We also provide Japanese documents**:
  - When you update markdown files intended to be read by AI agents (e.g., `.agents/*/_/*.md`), you just need to provide them in English.
  - When you update markdown files under `docs/en` folder, please update the corresponding file under `docs/ja`.
  - When you update `README.md` files under a code folder (e.g., `pkg/task/`), please update the corresponding file under `README_ja.md`.
