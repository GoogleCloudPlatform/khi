# Topology View

[< Previous: History View](./06-history-view.md)

---

The Topology View visualizes cluster resources and their relationships as an intuitive topology graph. It reconstructs and renders the cluster state at the exact timestamp of the currently selected log.

![Topology View](/docs/en/images/topology-view.png)

## Opening Topology View

To open the Topology View, select **Topology view layout** from the **View** menu in the top header. The topology pane will open on the right side of the screen.

## Features and Best Practices

- **Point-in-Time State Reconstruction**: Reconstructs the state and relationships of Kubernetes resources (e.g. Deployments, DaemonSets, Nodes, Pods, Services) corresponding to the timestamp of the active log event.
- **Performance Notice**: In large clusters with hundreds or thousands of resources, opening the Topology View without filtering can lead to high resource consumption in the browser. When analyzing large datasets, it is recommended to narrow down resources using the toolbar timeline filter before switching to the Topology View.
