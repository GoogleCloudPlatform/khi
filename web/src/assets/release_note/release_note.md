# 🚀 Kubernetes History Inspector v0.56.0

## Breaking changes on `.khi` File Schema

**[IMPORTANT] Compatibility Warning: This version includes breaking changes with no backwards compatibility for older `.khi` files.**

If you need to open `.khi` files created with previous versions, please continue to use v0.55.x.  
This format update significantly reduces file sizes, allowing much larger log datasets to be rendered smoothly in the frontend.

| Scenario |   Logs    | Old Format | New Format | Reduction |
| :------: | :-------: | :--------: | :--------: | :-------: |
|  Basic   |  223,464  |  128.2 MB  |  19.4 MB   |   x6.6    |
|  Large   | 1,386,843 |  885.3 MB  |  144.4 MB  |   x6.1    |

The new file schema is organized based on Protocol Buffers, enabling future updates while maintaining compatibility.  
Previously, the timeline data structure strictly assumed Kubernetes resource hierarchies (`APIVersion` -> `Kind` -> `Namespace` -> `Resource` -> `Subresource`). This made it difficult to properly visualize logs from non-Kubernetes environments like Cloud Service Mesh or Managed Airflow without forcing them into virtual namespaces. This release introduces intuitive timeline structures to resolve this issue and support a wider variety of service logs in the future.

## Enhanced Timeline and Log Search

<img src="assets/release_note/filter_ui.png" alt="New filter UI overview" width="512px">

> **New Filter UI**

With the introduction of various timeline structures, we overhauled the timeline filtering experience. Beginners can simply click `+ Add filter` to add a filter builder and narrow down displayed timeline events.

In addition to traditional group selection filters, we added support for complex search queries essential for real-world troubleshooting, such as "timelines containing Pods with a specific IP" or "Pods with specific labels."

<img src="assets/release_note/advanced_filter_ui.png" alt="Advanced Filter UI supporting CEL expressions" width="512px">

> **Advanced Filter UI**

The Advanced Filter UI allows you to write flexible query conditions using Common Expression Language (CEL) expressions.

- _(Example)_ `revision_body("metadata.labels.role", "critical-component")` : Filters timelines that have a moment with a specific label.

_Note: For a detailed guide on writing CEL expressions, click the help button in the top-right toolbar._

<img src="assets/release_note/context_menu.png" alt="Context menu added to the right of timeline names" width="512px">

> **Context Menu for Timelines**

You can now exclude specific timelines directly from the view by clicking the three-dot menu marker to the right of a timeline name. This allows you to intuitively hide unnecessary noise without writing filters.

## In-View Search Support (Ctrl + F)

Previously, searching within manifests or log messages required relying on browser-native search. You can now search directly inside each view.

<img src="assets/release_note/body-search.png" alt="Search UI inside log view and history view" width="512px">

> **In-View Search UI**

Pressing `Ctrl + F` (or `Cmd + F` on Mac) inside the log view or history view opens a dedicated search UI. Matches are highlighted and you can jump between search results.

## Integrated Topology View

The topology view, which visualizes resource relationships at a specific point in time, has been integrated into the main window as a docking pane.

<img src="assets/release_note/topology.png" alt="Topology view showing cluster resource relationships interactively" width="512px">

> **Topology View**: Clicking any position on the timeline interactively draws the cluster resource relationship diagram at that exact moment.
