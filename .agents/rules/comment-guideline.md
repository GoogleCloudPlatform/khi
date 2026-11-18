---
trigger: always_on
---

## Code Commenting Guidelines

You MUST adhere to the following rules for all code comments.

### 1. Content and Philosophy (The "Why" not "What")

- **Focus on Intent:** Explain _why_ a piece of code exists or why a specific approach was taken, especially for complex or non-obvious logic. Do not simply restate what the code is doing in English.
- **Avoid Redundancy:** Do not repeat information that is obvious from the code itself (e.g., do not say "Increment i" for `i++`).
- **Self-Documenting Code First:** Strive for clear naming and structure that reduces the need for implementation comments. Use comments only when the code's logic isn't obvious.
- **Strict Alignment with Logic:** Every comment must accurately match the actual implementation behavior, data structures, synchronization primitives, boundary conditions, and error/fallback handling. When refactoring code (for example, changing a map implementation, lock strategy, or error contract), update or remove affected comments in the same change so stale claims never remain.
- **Test Case Explanations:** Place a concise comment immediately before each test case explaining the scenario/preconditions and expected behavior, and ensure the test setup, invocation, and assertions genuinely verify that described scenario.

### 2. English Style and Grammar

- **Active Voice:** Use active voice for clarity and brevity (e.g., "Returns the user ID" instead of "The user ID is returned").
- **Proper Grammar:** Use correct spelling, punctuation, and grammar. This reflects the quality and care put into the codebase.
- **Sentence Structure:** End all comments with a period (.), even for short single-line comments.
- **Third-Person Verbs:** For function and method documentation, start with a third-person singular verb (e.g., "Calculates...", "Checks...", "Sends...").

### 3. Documentation Comments (API level)

- **First Sentence Summary:** Start all doc comments (e.g., TSDoc, GoDoc) with a concise single-sentence summary on its own line.
- **GoDoc Convention:** On exported Go declarations (`type`, `func`, `method`, `const`, `var`), start the GoDoc comment with the exact declared symbol name followed by a third-person singular verb (e.g., `// ParseFilter parses ...`) and end it with a period (`.`).
- **Public APIs:** Documentation comments are mandatory for all public classes, methods, and fields.
