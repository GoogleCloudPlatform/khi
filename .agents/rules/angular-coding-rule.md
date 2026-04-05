---
trigger: glob
globs: **/*.ts, **/*.html, **/*.scss
---

# KHI Angular Standards

When developing or modifying Angular code in the KHI project, you **must** adhere to the following rules and best practices.

## 1. Verifications

1. **Build Verification**: Run `make build-web` to check compilation.
2. **Lint Verification**: Run `make lint-web` to check style.
3. **Test Verification**: Run `make test-web-headless` to check unit tests.
4. **Storybook Verification**: Run `make build-storybook` to check compilation for storybook.
5. **Review Verification**: Before asking the user to verify your changes, you MUST invoke a subagent for code review. See Section 4 for the procedure. **If you make any modifications based on the review, you MUST run the review again to verify the changes.**
6. **Restart on Correction**: If you make any corrections during a verification phase, you MUST restart the verification from the beginning for that phase.

## 2. General Coding Rules

1. **Comments**:
   - Use TSDoc-style comments for all public types, functions, and methods.
2. **Naming Conventions**:
   - Component selectors should have a `khi-` prefix.
3. **Test File Naming**: Test files must be named with `A.spec.ts` if `A.ts` exists. Do not define spec files by scenarios.
4. **Type Safety**: Do not use `any`.

## 3. Language-Specific Conventions (TypeScript & SCSS)

### 3.1 Modern Angular Conventions

These rules apply when creating new components or refactoring existing ones:

1. **Standalone Components**:
   - Components are standalone by default. **DO NOT** add `standalone: true` to the `@Component` decorator.
   - Explicitly list dependencies in the `imports` array.
2. **Signals Paradigm**:
   - Use `input()` or `input.required()` for component inputs instead of `@Input`.
   - Use `output()` instead of `@Output`.
   - Use `model()` for two-way bindings.
   - Use `signal()` and `computed()` for component-level state management.
3. **Control Flow**: Use built-in control flow (`@if`, `@for`, `@switch`) in templates instead of structural directives (`*ngIf`, `*ngFor`).
4. **Access Modifiers**:
   - Members accessed only from the template must be `protected`.
   - Members used only within the TypeScript class must be `private`.
   - Use `readonly` for properties that are not reassigned.
5. **RxJS to Signal Conversion**: If a service returns an Observable, convert it to a Signal in the component using `toSignal`.
6. **Icons**: When importing `MatIconModule`, you must also import `KHIIconRegistrationModule`.
7. **File Separation**: Styles and templates must be defined in separate files. Do not directly supply them in `@Component`.

## Smart-Dumb Component Architecture

To maintain a clean separation of concerns and improve testability, we adopt the Smart-Dumb component strategy.

### Directory Structure

Each feature or complex component should use the following directory structure:

```text
foo/
  components/           # Place non-smart (Dumb) components here
  types/                # Place component-specific types/ViewModels here (not shared outside foo/)
  foo-smart.component.ts
  foo-smart.component.scss
  foo-smart.component.html
```

### Dependency Rules

1. **Smart Components**:
   - Responsible for state management and data fetching.
   - Allowed to depend on Angular Services.
2. **Dumb Components**:
   - Responsible only for rendering UI and propagating events.
   - **MUST NOT** depend on Angular Services. They should only communicate via Inputs (`input()`, `model()`) and Outputs (`output()`).
   - **MUST** have a corresponding Storybook story (`*.stories.ts`) to verify its visual states independently.
   - **DO NOT** use suffixes like `-dumb` or `-ui` in the component name. Name them based on their semantic meaning (e.g., `user-profile`, `data-table`).

## General Coding Rules for SCSS

1. Do not use color literal in SCSS files like `background-color: #FF00FF;`.

   Define semantically meaningful color variables at the top of the SCSS file to use the color like background-color: $dialog-background-color;.

2. Use color palette from Material with mat.m2-get-color-from-palette rather than specifying color codes.
3. Prefer display: grid rather than display: flex. Use grid-template field rather than specifying grid-template-areas, grid-template-columns or grid-template-rows separately. **When using `grid-template`, you MUST define area names for all grid tracks.**
4. **DO NOT** use `repeat()` in grid layouts unless the number of elements is dynamic or unknown.
5. KHI's color scheme is light theme.
6. **Order SCSS properties by functional groups** (outside-in approach) to improve readability:
   - **Positioning**: Properties that determine the location of the element.
     - `position`, `z-index`, `top`, `right`, `bottom`, `left`
   - **Display & Layout**: Properties that determine how the element and its children are laid out.
     - `display`, `grid`, `grid-template`, `flex`, `gap`, `justify-content`, `align-items`
   - **Box Model (Sizing & Spacing)**: Properties related to dimensions and margins.
     - `margin`, `border`, `padding`, `width`, `height`, `min-width`, `max-width`
   - **Typography**: Properties related to text.
     - `font-family`, `font-size`, `font-weight`, `line-height`, `text-align`, `color`
   - **Visuals (Backgrounds & Decoration)**: Properties related to the visual appearance.
     - `background`, `background-color`, `box-shadow`, `opacity`
   - **Transitions & UI**: Properties related to interaction and animation.
     - `transition`, `cursor`, `user-select`, `pointer-events`

## 4. Subagent Review Guidelines

> [!IMPORTANT]
> **DO NOT FORGET** to invoke the subagent for code review after making changes. You must complete the subagent review before asking the user to verify your implementation.

Follow these rules to perform code reviews using a temporary subagent before asking the user to verify your changes.

Follow these rules to perform code reviews using three parallel temporary subagents with distinct perspectives to catch more issues.

- **Define**: Use `define_subagent` to create three temporary subagents with distinct roles specialized for Angular/Typescript.
- **Invoke**: Invoke all three subagents in parallel using `invoke_subagent`. Do not wait for one to finish before invoking the next.
- **Capabilities**: The subagents can read files, perform web searches, and read URL contents.

### Reviewer Roles

1. **QA Engineer** (`angular_standards_reviewer`): A QA engineer obsessed with coding standards, style guides, and consistency. Focuses on Smart-Dumb architecture, Signals usage, SCSS ordering, Responsive Grid semantics, and Dumb component naming.
2. **Senior Architect** (`angular_logic_reviewer`): A senior architect with deep knowledge of system design. Focuses on ViewModel definitions, state mutations, interaction side-effects, and visual excellence (Wow factor).
3. **Senior Test Engineer** (`angular_test_reviewer`): A senior test engineer obsessed with thorough testing. Focuses on Storybook coverage, element accessibility, and automated component test edge-cases.

### Example Format

**`define_subagent` (QA Engineer)**

```json
{
  "name": "angular_standards_reviewer",
  "description": "QA Engineer obsessed with style guidelines and architecture.",
  "prompt_sections": [
    {
      "title": "Persona",
      "content": "You are a strict QA engineer who cannot tolerate any violation of coding standards, SCSS ordering, or naming conventions."
    },
    {
      "title": "Checklist",
      "content": "- Smart-Dumb architecture compliance\n- Signals usage over legacy decorators\n- SCSS property ordering and grid tracking\n- Semantic naming"
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

**`define_subagent` (Senior Architect)**

```json
{
  "name": "angular_logic_reviewer",
  "description": "Senior Architect focusing on logic correctness and interaction states.",
  "prompt_sections": [
    {
      "title": "Persona",
      "content": "You are a senior architect who values clean component design, efficient state management, and visual excellence."
    },
    {
      "title": "Checklist",
      "content": "- ViewModel definitions and state scoping\n- Proper use of protected for template access\n- Visual excellence and interaction smoothness"
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

**`define_subagent` (Senior Test Engineer)**

```json
{
  "name": "angular_test_reviewer",
  "description": "Senior Test Engineer obsessed with test coverage and Storybook.",
  "prompt_sections": [
    {
      "title": "Persona",
      "content": "You are a senior test engineer who believes that every UI component needs a Storybook story and thorough interaction tests."
    },
    {
      "title": "Checklist",
      "content": "- Storybook stories for all dumb components\n- Element accessibility and semantic semantics\n- Tests covering realistic interactions"
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

**`invoke_subagent` (Parallel Invocations)**

Call these tools in a single turn without waiting between them.

```json
[
  {
    "TypeName": "angular_standards_reviewer",
    "Role": "Angular Standards Reviewer",
    "Prompt": "Review the changes in: [file paths]"
  },
  {
    "TypeName": "angular_logic_reviewer",
    "Role": "Angular Logic Reviewer",
    "Prompt": "Review the changes in: [file paths]"
  },
  {
    "TypeName": "angular_test_reviewer",
    "Role": "Angular Test Reviewer",
    "Prompt": "Review the changes in: [file paths]"
  }
]
```
