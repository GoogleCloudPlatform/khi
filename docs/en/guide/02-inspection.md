# Inspection

[< Previous: UI Layout](./01-ui-overview.md) | [Next: Timeline View >](./03-timeline-view.md)

---

When you start KHI, the welcome screen appears. From this screen, you can create new inspections or load previously exported `.khi` inspection files.

## Start Screen Structure

![Start Screen Structure](/docs/en/images/startup-elements.png)

| Number | Component | Description |
| :--- | :--- | :--- |
| **1** | **New Inspection** | Collects and processes logs from sources like Cloud Logging or JSON Lines audit log files to create a new inspection dataset. |
| **2** | **Open .khi file** | Loads and reopens a previously exported `.khi` file. |
| **3** | **Inspection List** | Displays active and completed inspections. Click "Open" to launch the visualization view, or "Download" to export the dataset as a `.khi` file. |

## Managing Inspections

### Renaming an Inspection

Clicking the title of an inspection card in the list allows you to edit its name inline.

![Renaming an Inspection](/docs/en/images/startup-rename-inspection.png)

This is useful for tagging datasets with cluster names, incident IDs, or investigation notes for easy identification.

## Creating a New Inspection

In KHI, the process of collecting logs and preprocessing them into a structured dataset (`.khi` file) is called an "inspection". Unless you already have an exported `.khi` file, creating a new inspection is the first step toward visualizing your cluster logs.

Clicking the **New Inspection** button on the start screen launches a 3-step setup wizard.

### 1. Select Cluster Type

Choose the type of Kubernetes cluster you want to inspect.

![Select Cluster Type](/docs/en/images/startup-new-inspection-cluster.png)

### 2. Select Log Types to Query

A list of available log sources will appear based on the selected cluster type. Check the log types you wish to include in the investigation.

![Select Log Types](/docs/en/images/startup-new-inspection-log-types.png)

### 3. Input Parameters and Filter Preview

Enter the required query parameters based on the selected log types, such as the time window, Project ID, Cluster name, and resource Kinds.

![Input Parameters and Filter Preview](/docs/en/images/startup-new-inspection-parameters.png)

As you fill in the parameters, the generated Cloud Logging filter expressions are previewed in real time on the right pane.

> [!NOTE]
> If you select OSS Kubernetes, instead of Cloud Logging query parameters, you will be prompted to upload local audit log files.

### Command for Job Mode

At the bottom of the right pane, KHI generates the equivalent CLI command for running this inspection in Job Mode.

To automate `.khi` file generation within CI/CD pipelines or automated incident response scripts, see the [Job Mode Guide](../setup-guide/job-mode.md).
