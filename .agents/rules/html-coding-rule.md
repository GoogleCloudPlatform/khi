---
trigger: glob
globs: **/*.html
---

# HTML(Angular template) Coding rules

1. Use built-in control flow (`@if`, `@for`, `@switch`) in templates instead of structural directives (`*ngIf`, `*ngFor`).
2. Do not define any ARIA attributes. We are not ready to address these level of accessibility yet.
3. Complex calculation on template is disallowed. Use computed field in Typescript side if the calculation become 2 or more sentences. This typically for allowing emitting events directly from template like `(click)="click.emit($event)`.
