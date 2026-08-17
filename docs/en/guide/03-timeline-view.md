# Timeline View

[< Previous: Inspection](./02-inspection.md) | [Next: Filtering >](./04-filtering.md)

---

The Timeline View is the core visualization interface of KHI. It allows you to intuitively explore and observe state transitions and log events across all Kubernetes cluster resources along a chronological timeline.

## UI Structure

![Timeline View UI Structure](/docs/en/images/timeline-ui-structure.png)

The Timeline View is composed of five primary components:

| Number | Component | Description |
| :--- | :--- | :--- |
| **1** | **Timeline Histogram / Time Ruler** | Displays the time ruler and a histogram showing the frequency of logs categorized by severity. |
| **2** | **Timeline Index** | Displays the hierarchical tree of resources by Cluster, apiVersion, Kind, Namespace, Resource Name, and subresources or containers. |
| **3** | **Timeline Body** | Graphically renders resource state transitions and related log events on the timeline tracks. |
| **4** | **Legend** | Explains the colors and icons used in the selected timeline. |
| **5** | **Hover Popup** | Displays a detail popover showing the logs and events occurring around the mouse cursor when hovering over the timeline. |

## Timeline Zooming and Navigation

In the Timeline View, you can zoom in and out smoothly to transition between high-level cluster trends and millisecond-level incident details. Zooming is always centered at the mouse cursor position.

![Timeline Zoom Controls](/docs/en/images/timeline-zoom-controls.png)

### Control Methods

- **With a Trackpad**:
  - Perform a two-finger pinch gesture over the timeline body to quickly zoom in and out centered at the cursor.
- **With a Standard Mouse and Keyboard**:
  - Hold down the `Shift` key and rotate the mouse scroll wheel to zoom in and out centered at the cursor.
- **Scrolling and Panning**:
  - Drag the timeline canvas to move across different time ranges.

## Timeline Structure

In KHI, timelines are organized as a multi-root tree structure.
Parent-child relationships are built based on rules defined for each resource type, and displayed as a hierarchical list. Each timeline has an associated name and a timeline type.

### Timeline Index Components

Each row in the timeline index on the left contains the following interactive elements and metadata:

![Timeline Index Structure](/docs/en/images/timeline-index-structure.png)

| Number | Component | Description |
| :--- | :--- | :--- |
| **1** | **Expand / Collapse Button** | Toggles the visibility of child timelines in the hierarchy tree. |
| **2** | **Timeline Name** | Shows the name of the cluster, namespace, resource, or attribute at this hierarchy level. |
| **3** | **Timeline Type** | A label chip indicating the type of timeline, such as `k8sCluster`, `apiVersion`, `kind`, `namespace`, or `resource`. |
| **4** | **Timeline Menu (︙)** | Context menu for quickly filtering by this timeline or isolating its child elements. |

### Resource Hierarchy Example (Pod)

For example, a Pod resource timeline has the following hierarchy from top to bottom:

| Level | Timeline Type | Example | Description |
| :--- | :--- | :--- | :--- |
| **1** | `k8sCluster` | `p0-gke-basic-1` | Root node representing the Kubernetes cluster |
| **2** | `apiVersion` | `core/v1` | API version of the resource |
| **3** | `kind` | `pod` | Resource Kind |
| **4** | `namespace` | `1-2-deployment-update` | Namespace where the resource belongs |
| **5** | `resource` | `nginx-deployment-surge-9964c...` | Specific resource instance name |
| **6** | `subresource` / `condition` / `container` | `binding`, `Ready`, `nginx` | Subresources, conditions, or containers associated with the resource |

### Timeline Body Display

![Timeline Body Display](/docs/en/images/timeline-body-elements.png)

Depending on how logs are associated with resources, information is displayed on the timeline in two different forms:

| Number | Component | Description |
| :--- | :--- | :--- |
| **1** | **Revision** | A continuous horizontal bar representing the persistent state of a resource over time. |
| **2** | **Event** | A diamond marker indicating the occurrence of an individual log event associated with the resource. |

#### Revision

A revision is displayed when a log indicates that a resource's state has changed.
For example, because Kubernetes audit logs include the resource manifest at that point in time, the duration starting from that audit log until the next modifying audit log forms a single continuous state (revision).

The meaning of each revision is represented by its color and icon. The number displayed at the right end of a revision bar indicates the revision sequence number, starting from 0 for the first detected change and incrementing by 1. This number corresponds to the revision numbers shown in the History View, making it easy to identify when rapid consecutive changes have occurred even if individual bars appear compressed on the timeline.

#### Event

Events represent logs that do not directly translate into resource state transitions. For example, kubelet or containerd logs associated with a Pod represent point-in-time occurrences and are visualized as discrete diamond markers.

The top half of an event diamond indicates its severity, while the bottom half indicates the log category.
Higher-severity event markers are rendered on top of lower-severity markers, ensuring that critical logs remain visible even in dense log clusters.

#### Checking the Legend

The colors used for revisions and log events vary depending on the timeline type.

| Revisions Legend | Events Legend |
| :---: | :---: |
| ![Revisions Legend](/docs/en/images/timeline-legend-revisions.png) | ![Events Legend](/docs/en/images/timeline-legend-events.png) |

Because there are many distinct color variations across different resource types, you do not need to memorize them all. Whenever you encounter an unfamiliar color or icon, simply click on the timeline to inspect its legend and review the meaning of the revisions and events used in that track.

#### Dashed and Striped Revisions

While revision colors vary across different resource types, dashed and striped patterns follow consistent conventions throughout KHI.

The following image shows an example of container revision legends:

![Container Revision Legend](/docs/en/images/timeline-legend-container.png)

1. **Dashed Revisions**: Represent states where the resource is not actively running, such as "Deleted", "Terminated", or "Waiting".
2. **Striped (Hatched) Revisions**: Represent states where information is insufficient to determine the exact status. For example, when a deletion log confirms that a resource existed, but the creation log falls outside the query timeframe and initial details are unknown.

While the exact status is always described in the legend, recognizing these two common patterns helps you quickly understand resource health at a glance.

### Hover Popup

![Hover Popup](/docs/en/images/timeline-hover-popup.png)

When you hover your mouse cursor over elements in the timeline, a popup appears showing details of logs occurring around the cursor position.

| Number | Component | Description |
| :--- | :--- | :--- |
| **1** | **Timeline Track** | A vertical timeline showing logs and revisions around the cursor. Unlike the main horizontal canvas, item heights do not represent time duration. |
| **2** | **Verb** | Indicates the operation type that modified resource state and created a new revision. For example, creation is blue and deletion is red, matching the colors used in the History View. |
| **3** | **Log Type** | Color indicator representing the log category, matching the bottom half of the event diamond marker. |
| **4** | **Severity** | Color indicator representing log severity, matching the top half of the event diamond marker. |
| **5** | **Timestamp** | The exact timestamp when the log occurred. |
| **6** | **Log Summary** | A one-line summary generated from the structured log. It automatically resolves raw identifiers like container IDs into human-readable names. |
