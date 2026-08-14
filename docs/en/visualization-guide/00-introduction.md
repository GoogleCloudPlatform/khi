# Introduction

[Next: UI Layout >](./01-ui-overview.md)

---

In distributed systems like Kubernetes, many components generate large volumes of logs simultaneously. When an issue occurs, quickly determining which logs to check first can be very difficult.

Managed Kubernetes services in the cloud record many logs essential for troubleshooting beyond application logs, such as logs from the control plane, nodes, and controllers. Without familiarity with these logs and how to use them, engineers may struggle to find the needed information quickly, resulting in prolonged issues or inconclusive investigations.

Furthermore, Kubernetes resources are highly dynamic. With automatic scaling via Horizontal Pod Autoscaler (HPA) and automated control plane or node upgrades, the internal cluster state changes constantly. Because of this, issues often appear suddenly and may self-heal before anyone notices. In such situations, even if you know the exact time range of an incident, identifying the specific resources (such as Pods or nodes) involved is a major challenge.

For example, when investigating a brief outage in the middle of the night, simply finding the exact names of Pods that restarted or stopped requires specialized skills and significant effort.

Standard log storage and search systems excel at storing massive volumes of logs over long periods and filtering them by keywords. This approach works well when you already know unique identifiers such as the affected Pod name. However, these tools primarily provide a single log stream as a chronological list of text. When multiple resources interact and there is no single failed component, you often have to open multiple browser tabs and manually compare timelines across separate search results.

![Traditional Log Search Approach](/docs/en/images/overview-traditional-logs.png)

Kubernetes History Inspector (KHI) is a log visualizer designed to address these troubleshooting challenges. Rather than focusing on long-term log storage or broad text searches, KHI's core concept is to **comprehensively list all resources related to a specific time window, and interactively track and visualize multiple log streams along a timeline**.

![KHI Timeline View](/docs/en/images/overview-timeline.png)

As shown above, KHI presents log data along two axes: **Resources** and **Time**. This provides an immediate overview to spot the resources involved in an issue across large volumes of logs.

![KHI Topology View](/docs/en/images/overview-topology.png)

In addition, KHI allows you to graphically inspect Pod scheduling states, parent-child relationships, and resource dependencies at any specific point in time. It converts fragmented text logs into intuitive visual context for faster analysis.

The log analysis approach offered by KHI represents a paradigm shift in troubleshooting. For users accustomed to traditional single-stream log queries who manually correlate timelines across multiple browser tabs, this workflow might initially feel unfamiliar.

This documentation suite provides a comprehensive guide to KHI's features. Beyond simply understanding each feature, we encourage you to launch KHI, "explore" logs from your own environment, and master interactive, deep log analysis skills.

## Guide Structure

| Chapter | Title | Summary |
| :--- | :--- | :--- |
| **01** | [UI Layout](./01-ui-overview.md) | Panes overview, dockable window management, and layout switching |
| **02** | [Startup and Inspections](./02-startup-and-inspections.md) | Using the start screen, creating new inspections via log queries, and loading files |
| **03** | [Timeline View](./03-timeline-view.md) | Understanding resource trees, revision bars, event markers, and timeline controls |
| **04** | [Filtering and CEL Expressions](./04-filtering-and-cel.md) | Timeline filtering and advanced querying using Common Expression Language (CEL) |
| **05** | [Log View](./05-log-view.md) | Inspecting detailed log entries and structured payloads for selected timelines |
| **06** | [History View and Diff](./06-history-and-diff.md) | Reviewing revision history and tracking manifest diffs across changes |
| **07** | [Graph (Topology) View](./07-graph-view.md) | Visualizing parent-child dependencies and resource relationships |
| **08** | [Style Override](./08-style-override.md) | Customizing colors and display styles for revisions and events |
| **09** | [Shortcuts Reference](./09-shortcuts-reference.md) | Complete reference of keyboard shortcuts available in KHI |
