---
trigger: glob
globs:
  - pkg/task/**/*.go
  - pkg/model/khifile/v6/style/**/*.go
---

# KHI Description Standards

When developing or modifying task, inspection, or style definitions in the KHI project, you **must** adhere to the following standards for `description` fields and labels.

## 1. RevisionState

When registering a `RevisionState`, both `label` and `description` must be provided according to their display context:

### Label

- **Display Context**: Rendered as a concise legend item in the UI.
- **Style**: Short phrase expressing the status clearly (e.g., `Processing operation`, `Component is running`).

### Description

- **Display Context**: Rendered as a tooltip providing deeper context.
- **Style**: Complete sentence or detailed markdown text (starting with a capital letter and ending with a period). If the description spans multiple lines, use Go raw string literals (``` ` ```) to format multiline text clearly.
- **Format**: `The [resource/component] [is/has] [state description].`
- **Example**:

  ```go
  `The condition is set to **True**.

  **Note**: **True** does not always indicate a healthy state (e.g., Ready=True is healthy, but DiskPressure=True is unhealthy).`
  ```

## 2. TimelineType Descriptions

- **Grammar Style**: Noun phrase (starting with a capital letter and ending **without** a period).
- **Role and Format**:
  - **Grouping Timelines**: If the element purely groups child timelines hierarchically.
    - **Format**: `Grouping timeline for [category/resource types]`
    - **Example**: `"Grouping timeline for GKE nodepools"`
  - **Entity Representation Timelines**: If the timeline represents an abstract or standalone entity as a whole.
    - **Format**: `Timeline representing [a/an entity]`
    - **Example**: `"Timeline representing a Multi-Cloud cluster"`
  - **Lifecycle, State Transitions & Field History Timelines**: If the timeline tracks specific state changes, life cycles, or log events of an entity.
    - **Format**: Directly describe the tracked states or events (e.g., `Lifecycle states and logs of...`, `Phase transitions of...`).
    - **Examples**:

      ```go
      "Lifecycle states and logs of the container"
      "Phase transitions of the pod (from .status.phase)"
      ```

- **Intent**: Clearly state the role and entity the timeline represents. Avoid using the word `Container` to describe grouping elements, as it can be confused with Kubernetes containers.

## 3. FeatureTask Descriptions

- **Grammar Style**: Imperative verb phrase (starting with a capitalized base verb such as `Gather`, `Parse`, etc., and ending with a period).
- **UX Requirements**: To help users decide whether to enable the feature, clearly include:
  1. **Log Source**: What specific log types are required.
  2. **Troubleshooting Value**: What operations or issues it helps investigate (e.g., autoscaling decisions, network status).
  3. **Caveats (If any)**: Any specific prerequisites, configuration needs, or performance impact.
- **Format**: `Gather [log type] logs to visualize [troubleshooting value]. [Caveats if any].`
- **Example**:

  ```go
  "Gather Cluster Autoscaler logs to visualize autoscaling decisions and actions on the timelines of the affected resources."
  ```

- **Intent**: Help users determine whether the feature is relevant to their current debugging task and what log prerequisites apply.

## 4. InspectionType Descriptions

- **Grammar Style**: Imperative verb phrase (starting with `Gather and parse`, etc., and ending with a period).
- **Format**: `Gather and parse [product] logs ([supported log types]) to visualize [overall operations] on timelines.`
- **Example**:

  ```go
  "Gather and parse Google Kubernetes Engine (GKE) cluster logs (Kubernetes audit, event, node, container, GCE audit, Network, and Cluster Autoscaler logs) to visualize cluster operations on timelines."
  ```

- **Intent**: Describe the full scope of gathered log sources and the primary visualization goal of the inspection.
