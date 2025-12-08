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

We follow the **Smart (Container) vs. Dumb (Presentational) Component** pattern to maintain separation of concerns.

- **Smart Components (Containers)**:
  - Responsible for state management, data fetching, and business logic.
  - Pass data to dumb components via inputs.
  - Handle events emitted by dumb components.
  - Often correspond to top-level route components or major feature containers.
- **Dumb Components (Presentational)**:
  - Focus purely on rendering the UI and handling user interactions.
  - Receive data via `input()` signals.
  - Emit user actions via  `output()`.
  - Should not contain complex business logic or direct service dependencies (except for purely presentational services).

- **Standalone by Default**: All new components, directives, and pipes **must** be `standalone: true`.
- **Signals for Inputs**: Use the `input()` signal function for component inputs.

    ```typescript
    // Preferred
    import { input } from '@angular/core';
    export class MyComponent {
      user = input.required<User>();
    }
    ```

- **New Control Flow**: Use `@if`, `@for`, `@switch`.

### State Management & Services

- **Signals for Local State**: Use signals (`signal()`, `computed()`) for component-level state.
- **RxJS for Async**: Use RxJS for complex async streams, converting to signals with `toSignal` for template binding.

#### Key Services

The following services play critical roles in the application architecture:

- **`InspectionDataLoaderService`**: Responsible for loading inspection data from various sources (backend, local file) and parsing it into the application's data model.
- **`InspectionDataStoreService`**: Acts as the central store for the loaded inspection data, holding the global state accessible throughout the application.
- **`SelectionManagerService`**: Manages the user's current selection state, including selected logs, timelines, and revisions, and handles the synchronization between different views.
- **`BackendAPI`**: An interface (and implementation) for communicating with the backend server to fetch data or perform actions.

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
