# Timeline CEL Reference

Timeline CEL expressions filter which resource timelines are displayed or excluded in KHI. Expressions must evaluate to a boolean value (`bool`).

## Variables

The following variables are available in the timeline evaluation scope:

- `name` (`string`): The name of the resource represented by the timeline (e.g. pod name, node name).
- `timelineType` (`string`): The type of the timeline (e.g. `Pod`, `Node`, `Namespace`).
- `path` (`map<string, string>`): Hierarchical resource path properties (e.g. `path["namespace"]`, `path["kind"]`, `path["resource"]`).
- `t` (`map<string, dyn>`): The root timeline object, exposing `t.name`, `t.timelineType`, and `t.path`.

## Timeline Path Keys

The following keys can be accessed via `path["<key>"]`:

| Key | Timeline Type | Description |
| --- | --- | --- |
| `path["airflow"]` | Airflow | Timeline representing a Managed Airflow environment |
| `path["apiversion"]` | apiVersion | Grouping timeline for Kubernetes API versions |
| `path["autoscaler"]` | autoscaler | Logs of the GKE Cluster Autoscaler |
| `path["component"]` | Component | Logs of the generic Airflow component |
| `path["components"]` | Components | Grouping timeline for Airflow backend components |
| `path["condition"]` | condition | Status conditions of the resource (from .status.conditions) |
| `path["container"]` | container | Lifecycle states and logs of the container |
| `path["controlplane"]` | controlplane | Timeline representing control plane components of the cluster |
| `path["controlplanes"]` | controlplanes | Control Plane |
| `path["csm"]` | csm | CSM Traffic logs related to this resource |
| `path["csmcp-connection"]` | csmcp-connection | CSM CP Connection |
| `path["csmcp-log"]` | csmcp-log | CSM CP Log associated with the Pod |
| `path["dag"]` | DAG | Timeline representing an Airflow DAG |
| `path["dag file"]` | DAG File | Timeline representing an Airflow DAG definition file |
| `path["dag files"]` | DAG files | Grouping timeline for parsed DAG files |
| `path["dag run"]` | DAG Run | Timeline representing an Airflow DAG run |
| `path["dags"]` | DAGs | Grouping timeline for Airflow DAGs |
| `path["endpoint"]` | endpoint | Pod serving status (from EndpointSlice) |
| `path["event-exporter"]` | event-exporter | GKE Event Exporter logs |
| `path["gcp_resource"]` | gcp_resource | Timeline representing a Google Cloud resource |
| `path["gcp_resource_type"]` | gcp_resource_type | Grouping timeline for Google Cloud resource types |
| `path["gke"]` | gke | Control plane operations and lifecycle logs of the GKE cluster |
| `path["k8scluster"]` | k8sCluster | Grouping timeline for Kubernetes clusters |
| `path["kind"]` | kind | Grouping timeline for Kubernetes API resource kinds |
| `path["managed airflow"]` | Managed Airflow | Timeline for Managed Airflow |
| `path["mig"]` | mig | Managed Instance Group (MIG) logs for the nodepool |
| `path["multicloudcluster"]` | multicloudCluster | Timeline representing a Multi-Cloud cluster |
| `path["multicloudnodepool"]` | multicloudNodepool | Timeline representing a Multi-Cloud nodepool |
| `path["namespace"]` | namespace | Grouping timeline for Kubernetes namespaces |
| `path["neg"]` | neg | Associated NEG serving status |
| `path["node-component"]` | node-component | Logs of non-containerized system components on the node |
| `path["nodepool"]` | nodepool | Grouping timeline for GKE nodepools |
| `path["nodepools"]` | nodepools | Node Pools |
| `path["onpremcluster"]` | onpremCluster | Timeline representing an On-Prem Kubernetes cluster |
| `path["onpremnodepool"]` | onpremNodePool | Timeline representing an On-Prem nodepool |
| `path["operation"]` | operation | Google Cloud operations associated with the resource |
| `path["other_gke_resources"]` | other_gke_resources | Other GKE Resources |
| `path["owns"]` | owns | Child resources owned by the resource (from .metadata.ownerReferences) |
| `path["parser"]` | Parser | Logs of the DAG Processor Manager instance. Same DAG file can be parsed from multiple DAG Processor Manager instances at the same time thus this is shown as separated timelines. |
| `path["pod"]` | pod | Phase transitions of the pod (from .status.phase) |
| `path["project"]` | project | Timeline representing a Google Cloud project |
| `path["resource"]` | resource | Lifecycle states and logs of the Kubernetes resource |
| `path["serialport"]` | serialport | Serial port logs of the node |
| `path["subresource"]` | subresource | Lifecycle states and logs of the Kubernetes subresource |
| `path["task instance"]` | Task Instance | Execution states of the Airflow task instance |

## Severity Constants

Severities are integer constants used with `hasSeverity` and `minSeverity`:

| Constant | Value | Description |
| --- | --- | --- |
| `UNKNOWN` | 0 | UNKNOWN severity |
| `INFO` | 1 | INFO severity |
| `WARNING` | 2 | WARNING severity |
| `ERROR` | 3 | ERROR severity |
| `FATAL` | 4 | FATAL severity |

## Functions

### `match` / `M`

Matches values in the timeline path against substrings or regular expressions.

- `match(value string) bool` / `M(value string) bool`:
  Checks if any value across all timeline path keys contains `value` (case-insensitive substring or regex).
- `match(values list<string>) bool` / `M(values list<string>) bool`:
  Checks if any value across all timeline path keys matches any pattern in `values`.
- `match(key string, value string) bool` / `M(key string, value string) bool`:
  Checks if the path property `key` contains `value` (case-insensitive substring or regex).
- `match(key string, values list<string>) bool` / `M(key string, values list<string>) bool`:
  Checks if the path property `key` matches any pattern in `values`.

### `revision_body` / `RB`

Matches YAML/JSON field paths in recorded resource revision bodies.

- `revision_body(value string) bool` / `RB(value string) bool`:
  Checks if any revision body contains `value` anywhere in its contents.
- `revision_body(values list<string>) bool` / `RB(values list<string>) bool`:
  Checks if any revision body contains any pattern in `values`.
- `revision_body(fieldPath string, value string) bool` / `RB(fieldPath string, value string) bool`:
  Checks if the dot-separated field `fieldPath` in any revision body matches `value`.
- `revision_body(fieldPath string, values list<string>) bool` / `RB(fieldPath string, values list<string>) bool`:
  Checks if the dot-separated field `fieldPath` in any revision body matches any pattern in `values`.

### `minSeverity`

- `minSeverity(severity int) bool`:
  Returns `true` if the timeline contains at least one event or log with a severity level equal to or greater than the given constant.

### `hasSeverity`

- `hasSeverity(severity int) bool`:
  Returns `true` if the timeline contains at least one event or log with the exact given severity level.
- `hasSeverity(severities list<int>) bool`:
  Returns `true` if the timeline contains at least one event or log matching any severity level in `severities`.

## Common Patterns

Filter by namespace:

```cel
path["namespace"] == "kube-system"
```

Filter by resource kind:

```cel
path["kind"] == "Pod"
```

Filter by resource name regex:

```cel
match("resource", "^coredns-.*")
```

Filter timelines with errors:

```cel
minSeverity(ERROR)
```

Filter by revision body field:

```cel
revision_body("status.phase", "Failed")
```

Combine conditions:

```cel
path["kind"] == "Pod" && path["namespace"] == "default" && minSeverity(WARNING)
```
