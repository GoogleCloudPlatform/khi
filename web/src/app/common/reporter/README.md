# Generic Reporter in KHI

This directory contains the infrastructure for hierarchical event reporting in Kubernetes History Inspector (KHI). This system can be used for analytics, logging, or any other event tracking needs.

## Privacy and Tracking in OSS KHI

- **No Default Tracking**: The open-source version of KHI (including pre-built images) **does not** perform any user tracking or data collection. We value user privacy and do not collect any data from your usage of the official OSS build.
- **Purpose of the Infrastructure**: The reporting interfaces and classes are included in the OSS codebase to allow developers who fork the project, as well as our internal environments, to gather usage statistics (or other telemetry) for their own users from their own deployments.
- **Customization for Forks**: If you fork this repository, you can leverage this infrastructure to record user interactions or system events within your custom KHI instances. You can do this by providing a custom implementation of `Reporter` (e.g., sending data to your own analytics backend or logging service).

## Technical Overview

The system uses Angular's hierarchical dependency injection to allow different parts of the application to add context (labels) to reported events as they bubble up or are delegated to parent reporters.

- `Reporter`: The abstract base class and DI token.
- `ConsoleReporter`: A simple implementation that logs events to the console.
- `HierarchicalReporter`: An implementation that merges local labels and delegates to a parent reporter.
- `provideReporterContext(labels)`: A helper function to add static labels at a component or module level in the DI tree.

## Usage

### 1. Basic Usage (Injecting and Sending Events)

Inject the `Reporter` and use the `send` method to report events.

```typescript
import { Component, inject } from "@angular/core";
import { Reporter } from "./common/reporter/reporter";

@Component({
  // ...
})
export class MyComponent {
  private reporter = inject(Reporter);

  logAction() {
    this.reporter.send({ action: "click", target: "submit-button" });
  }
}
```

### 2. Adding Context Labels in Components

Use `provideReporterContext` in the `providers` array of a component to add static labels for that component and its children. These labels will be merged with the event data.

```typescript
import { Component } from "@angular/core";
import { provideReporterContext } from "./common/reporter/reporter";

@Component({
  // ...
  providers: [provideReporterContext({ feature: "my-feature" })],
})
export class MyFeatureComponent {
  // ...
}
```

When events are sent from within `MyFeatureComponent` or its children, they will automatically include `{ feature: 'my-feature' }`.
