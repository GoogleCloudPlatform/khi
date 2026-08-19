# Filtering

[< Previous: Timeline View](./03-timeline-view.md) | [Next: Log View >](./05-log-view.md)

---

The KHI toolbar provides powerful filtering features to efficiently find specific timelines and logs within massive datasets. You can switch between Standard Mode for intuitive UI controls and Advanced Mode for writing Common Expression Language (CEL) expressions.

## UI Structure

### Standard Mode

By default, the toolbar opens in Standard Mode:

![Standard Mode Toolbar](/docs/en/images/filter-normal-toolbar.png)

| Number | Component                   | Description                                                                 |
| :----- | :-------------------------- | :-------------------------------------------------------------------------- |
| **1**  | **Switch to Advanced Mode** | Toggles between Standard Mode and Advanced Mode with CEL expressions.       |
| **2**  | **Add Timeline Filter**     | Opens the dialog to add a timeline filtering rule.                          |
| **3**  | **Min Severity Filter**     | Filters logs so that only logs at or above the selected severity are shown. |
| **4**  | **Log Search Field**        | Searches or filters log text by keywords.                                   |
| **5**  | **Timezone Setting**        | Changes the display timezone for logs and timelines.                        |

### Advanced Mode

Clicking the search-gear icon on the left switches the toolbar to Advanced Mode:

![Advanced Mode Toolbar](/docs/en/images/filter-advanced-toolbar.png)

| Number | Component                    | Description                                                                                        |
| :----- | :--------------------------- | :------------------------------------------------------------------------------------------------- |
| **1**  | **Switch to Standard Mode**  | Returns to Standard Mode.                                                                          |
| **2**  | **Positive Timeline Filter** | Enter a CEL expression to include matching timelines.                                              |
| **3**  | **Negative Timeline Filter** | Enter a CEL expression to exclude matching timelines.                                              |
| **4**  | **Log Filter**               | Enter a CEL expression to evaluate and filter individual logs by message, severity, or attributes. |
| **5**  | **CEL Help Button**          | Displays help for available fields and syntax.                                                     |
| **6**  | **Filter Settings Button**   | Opens advanced settings for the filtering pipeline.                                                |
| **7**  | **Timezone Setting**         | Changes the display timezone for logs and timelines.                                               |

## Basic Filtering Concepts

When using Standard Mode filtering, the most important principle to remember is:

> **"KHI first filters the timelines, and then filters logs within those matching timelines."**

Filtering timelines is much faster than evaluating millions of individual log lines. Filtering timelines first to narrow down target resources and then refining logs will ensure fast performance.

## Filtering in Standard Mode

### Timeline Filtering

#### Filtering Timelines by Regular Expression

Clicking the "Add filter" button opens the timeline filter dialog. In most cases, filtering timeline names with a regular expression is the quickest approach.

![Timeline Regex Filter Dialog](/docs/en/images/filter-regex-pattern.png)

| Number | Component                    | Description                                                         |
| :----- | :--------------------------- | :------------------------------------------------------------------ |
| **1**  | **Filter Pattern (Regex)**   | Enter a regular expression pattern to match against timeline names. |
| **2**  | **Include / Exclude Toggle** | Choose whether to include or exclude matching timelines.            |

If no timeline type is specified, KHI matches the regular expression across all timeline types and hierarchy levels.

For example, searching for `pod` will match:

- All Pods in all namespaces (because every Pod is a descendant of the `pod` kind timeline)
- Any other resources that contain `pod` in their name
- All PodDisruptionBudgets in all namespaces (because `PodDisruptionBudget` contains `pod`)

Because matching works regardless of position in the hierarchy tree, you can quickly isolate related resource families.
For example, given a Deployment named `app-deployment`, a ReplicaSet named `app-deployment-abcde`, and a Pod named `app-deployment-abcde-fghij`:

- Entering `app-deployment` matches the Deployment, ReplicaSet, and Pod.
- Entering `app-deployment-abcde` matches only the ReplicaSet and Pod.

#### Filtering by Timeline Type

If you want to restrict filtering to a specific category (such as Kind or Namespace), you can specify the Timeline Type.

![Timeline Type Dropdown](/docs/en/images/filter-type-dropdown.png)

Clicking the Timeline Type input field displays a list of available types. The number in parentheses next to each type indicates the number of unique values present in the current dataset.

When a timeline type is selected, you can use the "Selection" mode to pick items from a checklist in addition to regex filtering.

![Selection Mode Filter](/docs/en/images/filter-type-selection.png)

### Log Filtering

In addition to timeline filtering, you can filter logs by minimum severity or text keywords. Timelines with no matching logs are automatically hidden from the view.

You can also use regular expressions in the log search field. The search evaluates all structured log fields as well as the generated log summary text.

## Advanced Filtering Pipeline

> [!NOTE]
> Advanced filter mode is an experimental feature. Syntax and behavior may change in future releases.

In Advanced Mode, KHI evaluates timelines and logs through a 6-step filtering pipeline:

1. **Extract Timelines with Positive Filters**: Identifies timelines that match the positive CEL expression.
2. **Add Descendant Timelines**: Automatically includes child timelines (such as subresources and containers) belonging to the matched timelines.
3. **Exclude Timelines with Negative Filters**: Removes timelines that match the negative CEL expression.
4. **Apply Log Filters**: Evaluates the log CEL expression against each log in the remaining timelines and removes non-matching logs.
5. **Remove Empty Timelines**: Excludes timelines that contain no active logs from the final view.
6. **Add Ancestor Timelines**: Automatically adds parent and ancestor timelines to maintain the visible tree structure.

Standard Mode filtering is internally translated into Advanced Mode CEL filters before execution.

Advanced Mode enables highly flexible filtering, such as matching specific revision fields (e.g. manifest contents) or inspecting custom structured log attributes. For complete syntax and available fields, click the CEL Help button on the toolbar.
