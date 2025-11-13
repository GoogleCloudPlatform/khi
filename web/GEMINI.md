> **Note:** In this document, "I" refers to the Gemini assistant, and "you" refers to the user.

# GEMINI.md for `web` Directory

This document outlines the development conventions and guidelines for the frontend application located in the `web/` directory. All code within this directory must adhere to these rules, in addition to the global standards defined in the root [`GEMINI.md`](../GEMINI.md).

## 1. Architecture Overview

The frontend is a modern Angular application built with TypeScript. It is responsible for providing an interactive user interface for visualizing Kubernetes log data.

- **Core Framework**: Angular
- **Component Library**: Angular Material is used for UI components to ensure a consistent look and feel.
- **State Management**: We primarily use Angular Signals for component-level and service-level state management due to their fine-grained reactivity and performance benefits. RxJS is used for handling complex asynchronous operations like API calls and user input events.
- **Styling**: SCSS is used for styling, following the BEM naming convention to keep styles scoped and maintainable.

## 2. Coding Conventions

We adhere to the official [Angular Style Guide](https://angular.io/guide/styleguide) and the conventions in the root `GEMINI.md`. The following rules are specific to this `web` directory.

### Component Design

- **Standalone by Default**: All new components, directives, and pipes **must** be `standalone: true`. This is the default for modern Angular development.
- **Signals for Inputs**: Use the `input()` signal function for component inputs instead of the `@Input()` decorator. This improves type safety and integration with the signal-based reactivity model.

    ```typescript
    // Preferred
    import { input } from '@angular/core';
    export class MyComponent {
      user = input.required<User>();
    }
    ```

- **New Control Flow**: In component templates, use the new built-in control flow syntax (`@if`, `@for`, `@switch`) instead of the older structural directives (`*ngIf`, `*ngFor`, `*ngSwitch`). This offers better type checking and performance.

    ```html
    <!-- Preferred -->
    @for (item of items; track item.id) {
      <li>{{ item.name }}</li>
    } @empty {
      <p>No items found.</p>
    }
    ```

### State Management

- **Signals for Local State**: Use signals (`signal()`, `computed()`) for managing component-level state. They are efficient and easy to reason about.
- **RxJS for Async**: Use RxJS for handling complex asynchronous event streams, such as those from `HttpClient` or `@angular/forms`. Convert RxJS observables to signals using `toSignal` from `@angular/core/rxjs-interop` when binding to the template.

### Styling (SCSS)

- **BEM Naming**: Use the BEM (Block, Element, Modifier) naming convention for component-scoped styles to avoid style conflicts and improve readability.

    ```scss
    // .khi-card { ... }
    // .khi-card__header { ... }
    // .khi-card--active { ... }
    ```

- **`@use` for Imports**: Use the modern `@use` rule to import SCSS partials (like `_variables.scss`). This prevents global style pollution.
- **Nesting Depth**: Limit SCSS nesting to a maximum of **3 levels** to maintain low specificity and readability.

## 3. Testing Strategy

- **Component Tests**: All components should have corresponding `.spec.ts` files with adequate test coverage for their public API and user interactions.
- **Harnesses**: When testing components that use Angular Material, leverage the [Component Test Harnesses](https://material.angular.io/guide/testing) to interact with them in a robust way.
- **Mocks and Spies**: Use `jasmine.createSpyObj` or similar techniques to mock services and dependencies.

## 4. Build and Development

- **Development Server**: Run `make watch-web` to start the local development server with live reloading at `http://localhost:4200`.
- **Production Build**: Run `make build-web` to create a production-optimized build in the `pkg/server/dist/` directory.
- **Proxy**: The `proxy.conf.mjs` file is configured to proxy API requests from `http://localhost:4200/api` to the backend server. This is used during development to avoid CORS issues.
