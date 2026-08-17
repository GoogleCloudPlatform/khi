# Introduction

[Next: UI Layout >](./01-ui-overview.md)

---

In distributed systems like Kubernetes, where large volumes of logs are generated simultaneously across numerous components, it can be extremely difficult during an incident to immediately determine "which log to look at first."

Especially in managed Kubernetes services on the cloud, there are many critical logs recorded beyond user-deployed application logs—such as control plane, node, and controller logs—that are vital for troubleshooting. Without familiarity with these logs and how to use them, finding necessary information quickly is difficult, often leading to extended resolution times or an inability to identify the root cause.

Furthermore, Kubernetes resources are highly dynamic. With automatic scaling via Horizontal Pod Autoscaler (HPA) and automated control plane or node upgrades performed by managed services, internal system states change continuously. Incidents where a transient failure occurs and resolves itself through self-healing before anyone notices are common. In such scenarios, even if the timeframe of an incident is known, identifying the exact resources (Pods, Nodes, etc.) involved is a significant challenge.

For example, when investigating a momentary outage occurring late at night, simply searching through massive log volumes to locate the exact names of Pods that restarted or stopped requires substantial expertise and effort.

Traditional log storage and search systems excel at long-term aggregation and extracting logs matching specific keyword queries. This works well when unique identifiers—such as a specific Pod name—are already known. However, log queries fundamentally return a "single log stream" (a chronological list of text entries). When there is no single point of failure and multiple resources interact in complex ways, operators must open multiple browser tabs and manually cross-reference timestamps across different query results.

![Traditional Log Search Approach](/docs/en/images/overview-traditional-logs.png)

KHI (Kubernetes History Inspector) is a log visualizer designed to address these troubleshooting challenges. Rather than focusing on long-term storage or wide-text searches, KHI's core concept is to **"enumerate all relevant resources in a given timeframe as comprehensively as possible, tracking and visualizing multiple log streams interactively along a unified timeline."**

![KHI Timeline View](/docs/en/images/overview-timeline.png)

As shown above, KHI organizes log data across **two intuitive dimensions: resources and time**, providing a unified high-level overview that lets you spot resources involved in an issue at a glance.

![KHI Topology View](/docs/en/images/overview-topology.png)

Furthermore, by graphically representing Pod scheduling states, parent-child hierarchies, and resource dependencies at specific points in time, KHI transforms fragmented log text into an intuitive format that humans can easily analyze and understand.

KHI represents a paradigm shift in log analysis. For users accustomed to single-stream search and manual cross-tab verification, this approach may feel new at first.

This guide covers KHI's features in detail. Beyond reading the instructions, we encourage you to launch KHI, explore logs from your own environment, and master interactive, deep log analysis skills.

## Guide Structure

| Chapter | Title | Summary |
| :--- | :--- | :--- |
| **01** | [UI Layout](./01-ui-overview.md) | Panes overview, dockable window management, and layout switching |
| **02** | [Inspection](./02-inspection.md) | Using the start screen, creating new inspections via log queries, and loading files |
| **03** | [Timeline View](./03-timeline-view.md) | Understanding resource trees, revision bars, event markers, and timeline controls |
| **04** | [Filtering](./04-filtering.md) | Filtering timelines and filtering logs |
| **05** | [Log View](./05-log-view.md) | Inspecting detailed log entries and structured payloads for selected timelines |
| **06** | [History View](./06-history-view.md) | Reviewing revision history and tracking manifest diffs across changes |
| **07** | [Topology View](./07-topology-view.md) | Visualizing parent-child dependencies and resource relationships |
