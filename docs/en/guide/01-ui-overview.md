# UI Layout

[< Previous: Introduction](./00-introduction.md) | [Next: Inspection >](./02-inspection.md)

---

![KHI UI Layout](/docs/en/images/overview-ui-structure.png)

KHI's main interface is composed of multiple panes and views designed for efficient log investigation. When you open KHI, the default interface displays the numbered components shown above:

| Number | Component | Description |
| :--- | :--- | :--- |
| **1** | **Header** | Displays application menus, server connection status, and server memory usage. |
| **2** | **Toolbar** | Provides controls for filtering visible timelines and logs, and changing the display timezone. |
| **3** | **Timeline View** | Shows an intuitive overview of logs as resource-specific timelines. |
| **4** | **Log View** | Displays a chronological list of logs matching the selected timeline or active filters, and shows details for the selected log. |
| **5** | **History View** | Displays the revision history and detailed changes for the selected resource. |

The fundamental workflow in KHI is to get an overview of resource changes across the Timeline View to identify anomalies, and then investigate details using the Log View and History View.

## View Menu and Dockable Windows

Except for the header and toolbar, each view is implemented as a dockable window that you can freely resize and reposition by dragging and dropping.

If you close a view, you can reopen it at any time from the **View** menu in the header.

![KHI View Menu](/docs/en/images/overview-view-menu.png)

KHI also provides preset layouts tailored for different investigation scenarios. You can switch between them using keyboard shortcuts (`Ctrl+1` through `Ctrl+3` on Windows/Linux, `Cmd+1` through `Cmd+3` on macOS):

| Preset Layout | Shortcut | Layout Composition | Purpose |
| :--- | :--- | :--- | :--- |
| **Default layout** | `Ctrl+1` / `Cmd+1` | Timeline View + Log View + History View | Standard layout for exploring the timeline while simultaneously viewing logs and revision history for selected resources. |
| **State view layout** | `Ctrl+2` / `Cmd+2` | Timeline View + History View | Focused layout for tracking resource state transitions and comparing manifest diffs. |
| **Topology view layout** | `Ctrl+3` / `Cmd+3` | Timeline View + Topology View | Specialized layout for inspecting resource dependencies and parent-child relationships at specific points in time. |
