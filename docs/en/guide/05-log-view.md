# Log View

[< Previous: Filtering](./04-filtering.md) | [Next: History View >](./06-history-view.md)

---

The Log View allows you to inspect and investigate individual structured logs in chronological order. It seamlessly synchronizes with timeline selections, allowing you to quickly focus on logs relevant to specific resources and inspect their full attributes.

## UI Structure

![Log View UI Structure](/docs/en/images/log-view-structure.png)

| Number | Component | Description |
| :--- | :--- | :--- |
| **1** | **Timeline Selection Filter Toggle** | Toggles whether to automatically filter logs based on the currently selected timeline resource. |
| **2** | **Include / Exclude Descendant Logs** | Toggles whether logs from child timelines (such as containers and subresources) are included in the view. |
| **3** | **Log Count Header** | Shows the number of filtered logs relative to the total number of logs in the dataset. |
| **4** | **Log List** | Chronological list of logs displaying severity indicator, timestamp, and summary. |
| **5** | **Log Hover Popup** | Displays the full wrapped log summary and message when hovering over an item in the log list. |
| **6** | **Log Properties Bar** | Displays metadata of the selected log including log type, severity, timestamp, and related resources. |
| **7** | **Log Detail Pane** | Shows the full structured payload of the selected log in YAML format. |

## Log Filtering and Timeline Selection

By default, KHI displays only the logs associated with the resource currently selected on the timeline. Selected timelines are highlighted in light green.

### 1. Showing Logs for the Selected Resource and Its Children (Default)

In the default mode, KHI displays logs for both the selected timeline and all of its descendant elements (such as containers and subresources).

![Selected Resource with Descendants](/docs/en/images/log-filter-pod-with-children.png)

In the example above, out of 180,965 total logs in the dataset, 467 logs relevant to the selected Pod and its child containers are filtered and displayed in the Log View.

### 2. Excluding Descendant Logs (Focus on Target Resource Only)

To inspect only logs directly associated with the selected timeline without child events, click the "Include / Exclude Descendant Logs" toggle button (Number 2).

![Excluding Descendant Logs](/docs/en/images/log-filter-pod-only.png)

Child container logs are removed, narrowing the list to 77 logs belonging strictly to the Pod itself. This is useful when focusing solely on Pod-level state changes.

### 3. Disabling Timeline Selection Filter (All Logs)

Clicking the "Timeline Selection Filter Toggle" button (Number 1) disables the selection-based filtering and displays logs across all timelines in chronological order.

![Disabling Selection Filter to Show All Logs](/docs/en/images/log-filter-all-timelines.png)

This mode is helpful when investigating cluster-wide events occurring around a specific point in time across multiple resources. Because the volume of logs can be large, it is best used in combination with toolbar text search.

## Log List

![Log List Elements](/docs/en/images/log-list-elements.png)

Each row in the log list displays the following elements to help quickly identify and scan log events:

| Number | Component | Description |
| :--- | :--- | :--- |
| **1** | **Log Type** | A color strip indicating the log category, matching the bottom half of event diamond markers on the timeline. |
| **2** | **Severity** | An indicator showing log severity (e.g. INFO, WARNING, ERROR), matching the top half of event diamond markers on the timeline. |
| **3** | **Timestamp** | The timestamp when the log occurred. |
| **4** | **Log Summary** | A one-line summary generated automatically from the structured log payload. |

## Related Resources

In KHI, a single log can be associated with multiple resources. For example, a kubelet log indicating that a Pod was started relates to both the Pod itself and the Node where the kubelet daemon runs.

KHI automatically links such logs to all relevant resources. In the log detail pane, the "Related resources" list allows you to navigate between associated resource timelines.

![Related Resources Hierarchy](/docs/en/images/log-related-resources.png)

Hovering over a resource in the list displays a popover showing its full hierarchy within the cluster, making it easy to understand where the resource resides.
