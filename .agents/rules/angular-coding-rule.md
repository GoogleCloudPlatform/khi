---
trigger: glob
globs: **/*.ts, **/*.html, **/*.scss
---

# KHI Angular Standards

When developing or modifying Angular code in the KHI project, you **must** adhere to the following rules and best practices.

## General Coding Rules for Typescript

1. **Verifications**

   - Run `make build-web` to check compilation.
   - Run `make lint-web` to check style.
   - Run `make test-web-headless` to check unit tests.
   - Run `make build-storybook` to check compilation for storybook.
   - **MUST** invoke a subagent for code review before asking the user for verification. See Section 4 for the procedure. **If you make any modifications based on the review, you MUST run the review again to verify the changes.**

2. **Comments**:
   - Use TSDoc-style comments for all public types, functions, and methods.
3. **Naming Conventions**:
   - Component selectors should have a khi- prefix.
4. Test files must be named with A.spec.ts if A.ts exists. Do not define spec files by scenarios.
5. Do not use any.

## Modern Angular Conventions

These rules apply when creating new components or refactoring existing ones:

1. **Standalone Components**:
   - Explicitly list dependencies in the `imports` array.
2. **Signals Paradigm**:
   - Use `input()` or `input.required()` for component inputs instead of `@Input`.
   - Use `output()` instead of `@Output`.
   - Use `model()` for two-way bindings.
   - Use `signal()` and `computed()` for component-level state management.
3. **Control Flow**:
   - Use built-in control flow (`@if`, `@for`, `@switch`) in templates instead of structural directives (`*ngIf`, `*ngFor`).
4. **Access Modifiers**:
   - Members accessed only from the template must be `protected`.
   - Members used only within the TypeScript class must be `private`.
   - Use `readonly` for properties that are not reassigned.
5. **RxJS to Signal Conversion**:
   - If a service returns an Observable, convert it to a Signal in the component using `toSignal`.
6. **Icons**:
   - When importing `MatIconModule`, you must also import `KHIIconRegistrationModule`.
7. Styles and template must be defined in an independent file. Do not directly supply them in @Component

## General Coding Rules for SCSS

1. Do not use color literal in SCSS files like `background-color: #FF00FF;`.
   Define semantically meaningful color variables at the top of the SCSS file to use the color like background-color: $dialog-background-color;.
2. Use color palette from Material with mat.m2-get-color-from-palette rather than specifying color codes.
3. Prefer display: grid rather than display: flex. Use grid-template field rather than specifying grid-template-areas, grid-template-columns or grid-template-rows separately.
4. KHI's color scheme is light theme.

## 4. Subagent Review Guidelines

> [!IMPORTANT]
> **DO NOT FORGET** to invoke the subagent for code review after making changes. You must complete the subagent review before asking the user to verify your implementation.

Follow these rules to perform code reviews using a temporary subagent before asking the user to verify your changes.

- **Define**: Use `define_subagent` to create a temporary subagent specialized for Angular and Typescript.
- **Invoke**: Pass the modified file paths (TS, HTML, SCSS) to the subagent during `invoke_subagent`.
- **Capabilities**: The subagent can read files and perform web searches (e.g., to reference external style guides or documentation).

### Review Checklist Focus

The subagent must verify:

- **Simplicity & Duplication**: Can any parts be written more simply and concisely? Are there any duplicated implementations?
- **Signals & Modern APIs**: Are Signals (`input`, `output`, `computed`, etc.) and built-in control flow (`@if`, `@for`) used correctly instead of legacy decorators and directives?
- **Access Modifiers**: Are members accessed only from templates marked as `protected`? Are others properly scoped (`private` or `public` readonly)?
- **Style & SCSS**: Are color literals avoided in SCSS? Are semantic variables or Material palettes used?
- **Standalone Components**: Are all component dependencies listed in the `imports` array? Is `KHIIconRegistrationModule` included if `MatIconModule` is present?
- **Testing**: Do test cases sufficiently cover realistic component interactions and state changes?

### Example Format

**`define_subagent`**

```json
{
  "name": "temp_angular_reviewer",
  "description": "Reviews Angular code against project standards.",
  "prompt_sections": [
    {
      "title": "Checklist",
      "content": "- Code simplicity and no duplication\n- Use of Signals and modern control flow\n- Proper access modifiers (protected for template accessed)\n- No color literals in SCSS\n- Independent style and template files\n- Proper icon module registration"
    }
  ],
  "tool_names": ["view_file", "search_web"]
}
```

**`invoke_subagent`**

```json
{
  "TypeName": "temp_angular_reviewer",
  "Role": "Angular Code Reviewer",
  "Prompt": "Review the changes in: [file paths]"
}
```
