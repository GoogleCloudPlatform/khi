# Storybook Title Guidelines

When creating or updating Storybook files (`*.stories.ts`), you MUST adhere to the following rules for defining the `title` property in the default export. This ensures a consistent folder structure in the Storybook UI.

## Structure

The `title` should follow a maximum 3-level hierarchy:
`[Feature Area] / [Sub Area (if applicable)] / [Component Name]`

### 1. Feature Area (Root Level)

- Represents the main feature module (based on the `web/src/app` directory structure).
- Use **PascalCase**.
- Examples: `Log`, `Timeline`, `Diff`, `Graph`, `Shared`, `Dialogs`.
- **Exception**: Global singleton components (like `Header`) can be placed at the root level without nesting (e.g., `title: 'Header'`).

### 2. Sub Area (Optional)

- Use only when the feature is large and has clear subdivisions.
- Use **PascalCase**.
- Examples:
  - Dialogs: Use the specific dialog name (e.g., `Dialogs/Startup`, `Dialogs/Progress`).
  - Timeline: Place main components under `Timeline/Main` and toolbar components under `Timeline/Toolbar`.
- **Rule**: NEVER use generic intermediary folders like `Components` (e.g., `Shared/Components/YamlViewer` is WRONG; `Shared/YamlViewer` is CORRECT).

### 3. Component Name

- The name of the component in **PascalCase**.
- **Rule**: Omit the `Component` suffix from the name. (e.g., `TimelineChartComponent` -> `TimelineChart`).

### 4. Individual Stories (Leaf Nodes)

- The individually exported variables in the `*.stories.ts` file will automatically become the leaf nodes (the actual stories) under the folder defined by the `title`. You don't need to specify the story name in the `title` itself.

## Examples

- `src/app/shared/components/search-bar/search-bar.stories.ts` -> `title: 'Shared/SearchBar'`
- `src/app/timeline-toolbar/components/cel-input.stories.ts` -> `title: 'Timeline/Toolbar/CelInput'`
- `src/app/dialogs/new-inspection/components/set-parameter.stories.ts` -> `title: 'Dialogs/NewInspection/SetParameter'`
- `src/app/header/components/header.component.stories.ts` -> `title: 'Header'`
- `src/app/log/components/type-severity.stories.ts` -> `title: 'Log/TypeSeverity'`
