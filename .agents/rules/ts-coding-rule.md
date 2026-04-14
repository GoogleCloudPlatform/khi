---
trigger: glob
globs: **/*.ts
---

## Typescript Coding rules

* Use TSDoc-style comments for all public types, functions, and methods.
* Test files must be named with `A.spec.ts` if `A.ts` exists. Do not define spec files by scenarios.
* Do NOT use `any`.
* Make fields `readonly` as much as possible.
* Define enum type rather than define a union of string literal types.
* Write import path in absolute form from the project base path like `src/app/...`.

## Angular coding rules

* Use Angular signal rather than RxJS.
* Define templates and styles in separate files. Do not define them in @Component.
* Import `KHIIconRegistrationModule` when the component uses `MatIconModule`.
* Use `input()` or `input.required()` for component inputs instead of `@Input`.
* Use `output()` instead of `@Output`.
* Use `model()` for two-way bindings.
* Use `signal()` and `computed()` for component-level state management.
* To export enum type to the template, use protected readonly and it must be placed before any other fields, methods or constructor.

## Smart-Dumb Component Architecture

To maintain a clean separation of concerns and improve testability, we adopt the Smart-Dumb component strategy.

### Directory Structure

Each feature or complex component should use the following directory structure:

```text
foo/
  components/           # Place non-smart (Dumb) components here
    foo-layout.component(.ts|.scss|.html|.spec.ts|.stories.ts) # A layout dumb component
    bar.component(.ts|.scss|.html|.spec.ts|.stories.ts) # Other dumb component used from the layout component
    ...
  types/                # Place component-specific types/ViewModels here (not shared outside foo/)
  foo-smart.component.ts
  foo-smart.component.scss
  foo-smart.component.html
```

### Dependency Rules

1. **Smart Components**:
   * Responsible for state management and data fetching.
   * Allowed to depend on Angular Services.
   * **MUST NOT** contain any elements other than layout components in their templates. They must only act as a bridge to propagate information from Services to the underlying layout components.
   * **MUST NOT** have any styles except `:host { display: contents }`. This ensures that smart components do not contribute to the visual layout and remain purely logical containers.
2. **Dumb Components**:
   * Responsible only for rendering UI and propagating events.
   * **MUST NOT** depend on Angular Services. They should only communicate via Inputs (`input()`, `model()`) and Outputs (`output()`).
   * **MUST** have a corresponding Storybook story (`*.stories.ts`) to verify its visual states independently.
   * **DO NOT** use suffixes like `-dumb` or `-ui` in the component name. Name them based on their semantic meaning (e.g., `user-profile`, `data-table`).
   * **Layout Components** (which are a specialized type of dumb components) are responsible for the structural layout, including placing other dumb components and standard HTML elements (like `<div>`).
