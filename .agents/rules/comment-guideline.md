---
trigger: always_on
---

## Code Commenting Guidelines

You MUST adhere to the following rules for all code comments.

### 1. Content and Philosophy (The "Why" not "What")

- **Focus on Intent:** Explain _why_ a piece of code exists or why a specific approach was taken, especially for complex or non-obvious logic. Do not simply restate what the code is doing in English.
- **Avoid Redundancy:** Do not repeat information that is obvious from the code itself (e.g., do not say "Increment i" for `i++`).
- **Self-Documenting Code First:** Strive for clear naming and structure that reduces the need for implementation comments. Use comments only when the code's logic isn't obvious.

### 2. English Style and Grammar

- **Active Voice:** Use active voice for clarity and brevity (e.g., "Returns the user ID" instead of "The user ID is returned").
- **Proper Grammar:** Use correct spelling, punctuation, and grammar. This reflects the quality and care put into the codebase.
- **Sentence Structure:** End all comments with a period (.), even for short single-line comments.
- **Third-Person Verbs:** For function and method documentation, start with a third-person singular verb (e.g., "Calculates...", "Checks...", "Sends...").

### 3. Documentation Comments (API level)

- **First Sentence Summary:** Start all doc comments (e.g., TSDoc, GoDoc) with a concise single-sentence summary on its own line.
- **Public APIs:** Documentation comments are mandatory for all public classes, methods, and fields.

### 4. Critical Constraints

- **Preserve Context:** NEVER remove existing comments during refactoring unless the code they document is also deleted.
- **No Chatting:** Do not use the code or comments to communicate with reviewers or explain your personal thoughts; keep it technical and professional. If you need to tell something to user, just tell it in the message, not in comments.

To ensure all comments comply with the rule above, use the following subagent to verify comment quality:

Follow these rules to perform code reviews using a temporary subagent before asking the user to verify your changes.

- **Define**: Use `define_subagent` to create a temporary subagent specialized for reviewing comments.
- **Invoke**: Pass the modified file paths (TS, HTML, SCSS, GLSL, Go, Makefile) to the subagent during `invoke_subagent`.
- **Capabilities**: The subagent can read files.

Use following formats to define and invoke subagents.

**`define_subagent`**

```json
{
  "name": "temp_comment_reviewer",
  "description": "Reviews code comments against project standards.",
  "prompt_sections": [
    {
      "title": "Checklist",
      "content": "<Please include the Code Commenting Guidelines here>"
    }
  ],
  "tool_names": ["view_file"]
}
```

**`invoke_subagent`**

```json
{
  "TypeName": "temp_comment_reviewer",
  "Role": "Professional comment reviewer",
  "Prompt": "Review the changes in: [file paths]"
}
```
