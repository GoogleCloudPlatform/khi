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
