These release notes only include featured changes. For other minor changes and bug fixes, please see the [GitHub Releases](https://github.com/GoogleCloudPlatform/khi/releases) page.

# v0.57.0 (July 9 2026)

## Enhanced Diff View Tooltips

We have added new tooltips to the Diff view to provide more context about changes.

><img src="assets/release_note/diff-managed-field.png" alt="Managed fields tooltip showing editor information" width="512px">
>
> **Field Editor Information**
>
> You can now see who edited each field by hovering your mouse over it in the Diff view. This information is extracted from `metadata.managedFields`, which would be almost impossible to read manually.

> <img src="assets/release_note/diff-mutating-webhook.png" alt="Mutating webhook tooltip showing mutation source" width="512px">
>
> **Mutating Webhook Identification**
>
> If a change was made by a `MutatingWebhookConfiguration`, a tooltip will now show this information. This is parsed from the JSONPatch payload log label in the audit log, which would be practically impossible to read manually. This feature helps you easily identify which change was caused by which webhook, even when original requests and multiple webhook mutations are mixed.

# v0.56.6 (July 1 2026)

## Job Mode Command Generator

In the dialog to start a new inspection, KHI now generates a copy-pasteable CLI command based on your selected inspection target, enabled features, and parameter inputs. You can easily copy the command to execute KHI in job mode in automated environments.

<img src="assets/release_note/job_mode.png" alt="Job mode command generation UI" width="512px">

Copy the comand after filling out the parameters on the forms.

# v0.56.0 (June 20 2026)

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
