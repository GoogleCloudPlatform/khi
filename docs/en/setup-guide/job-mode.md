# **Running KHI in Job Mode**

KHI usually starts a local web server and runs inspections from the browser. Job mode runs one inspection directly from the command line and writes the result to a `.khi` file. This is useful for scheduled jobs, alerting pipelines, or other automation that needs to generate a KHI file without starting the web UI.

## **Required flags**

Job mode is enabled with `--job-mode`. When it is enabled, all of the following flags are required:

| Flag | Description |
| --- | --- |
| `--job-inspection-type` | Inspection type ID to run. |
| `--job-inspection-features` | Comma-separated feature IDs to enable. Use `ALL` to enable every feature available for the inspection type. |
| `--job-inspection-values` | JSON object containing the inspection parameters. These are the same form values used by the web UI. |
| `--job-export-destination` | Path where the generated `.khi` file is written. |

Available inspection type IDs include:

| Inspection type ID | Target |
| --- | --- |
| `gcp-gke` | Google Kubernetes Engine |
| `gcp-gke-on-aws` | GKE on AWS |
| `gcp-gke-on-azure` | GKE on Azure |
| `gcp-gdcv-for-vmware` | Google Distributed Cloud for VMware |
| `gcp-gdcv-for-baremetal` | Google Distributed Cloud for bare metal |
| `gcp-composer` | Cloud Composer |
| `oss-kubernetes-from-files` | OSS Kubernetes audit log files |

## **Command format**

```bash
./khi \
  --job-mode \
  --job-inspection-type gcp-gke \
  --job-inspection-features <feature-id>,<feature-id> \
  --job-inspection-values '{"<form-field-id>":"<value>"}' \
  --job-export-destination ./inspection.khi
```

When the command succeeds, `inspection.khi` can be opened later with KHI.

If you use a released binary or run KHI from another directory, replace `./khi` with the path to that binary.

## **Inspection values**

`--job-inspection-values` must be a JSON object. Each key is a KHI form field ID and each value is the value that would be submitted from the web UI for the same field.
Text fields use JSON strings. Set fields use JSON arrays of strings.

For example, a Google Cloud inspection usually includes common fields such as project, location, query end time, and duration, plus inspection-specific fields:

```json
{
  "cloud.google.com/common/input-project-id": "my-project",
  "cloud.google.com/common/input-location": "us-central1",
  "cloud.google.com/common/input-end-time": "2026-01-15T10:00:00Z",
  "cloud.google.com/common/input-duration": "2h",
  "<inspection-specific-text-field-id>": "<value>",
  "<inspection-specific-set-field-id>": ["<value>"]
}
```

The exact set of required fields depends on the inspection type and enabled features. The most reliable way to build the JSON is to run the same inspection once in the web UI, open the browser developer tools, and check the request payload sent to `POST /api/v3/inspection/<inspection-id>/run` or `POST /api/v3/inspection/<inspection-id>/dryrun`. The `parameters` object in that request uses the same field IDs and values that job mode expects.

For a GKE inspection, the commonly required field IDs are:

| Field ID | Value |
| --- | --- |
| `cloud.google.com/common/input-project-id` | Google Cloud project ID. |
| `cloud.google.com/common/input-location` | Cluster location, such as `us-central1` or `us-central1-a`. |
| `cloud.google.com/common/input-end-time` | Query end time in RFC3339 format. |
| `cloud.google.com/common/input-duration` | Query duration, such as `2h`. |
| `cloud.google.com/k8s/input-cluster-name` | GKE cluster name. |
| `cloud.google.com/k8s/input-namespaces` | Namespace filter array. Use `["@all_cluster_scoped", "@all_namespaced"]` to include all scopes. |
| `cloud.google.com/k8s/input-kinds` | Kubernetes kind filter array. Use `["@default"]` for common kinds or `["@any"]` for every kind. |

Some optional GKE features add their own fields. Include these fields when the selected feature list needs them:

| Feature area | Field ID | Example value |
| --- | --- | --- |
| Node and serial port logs | `cloud.google.com/k8s/input/node-name-filter` | `[]` |
| Container logs | `cloud.google.com/log/k8s-container/input/query-namespaces` | `["@managed"]` |
| Container logs | `cloud.google.com/log/k8s-container/input/query-podnames` | `["@any"]` |
| Control plane component logs | `cloud.google.com/log/k8s-control-plane/input/component-names` | `["@any", "-apiserver"]` |
| CSM access logs | `cloud.google.com/log/csm-accesslog/input/response-flags` | `["@any", "-OK"]` |
| CSM resource audit logs | `cloud.google.com/log/csm-accesslog/input/fleet-project-id` | `"my-project"` |

## **Selecting features**

Use `ALL` when the job should include every feature supported by the selected inspection type:

```bash
--job-inspection-features ALL
```

`ALL` enables optional features that are disabled by default in the web UI. For GKE, this includes node logs, container logs, control plane component logs, CSM logs, and serial port logs, so the extra fields listed above may also be required.

To enable only specific features, pass their IDs as a comma-separated list:

```bash
--job-inspection-features feature-a,feature-b
```

You can inspect feature IDs from the web server API before switching the same configuration to job mode:

```bash
curl http://localhost:8080/api/v3/inspection/types
curl -X POST http://localhost:8080/api/v3/inspection/types/gcp-gke
curl http://localhost:8080/api/v3/inspection/<inspection-id>/features
```

Use the `inspectionID` returned by the `POST` request in the final URL.

## **Example**

The following example runs the default GKE feature set and writes the result to `./gke-inspection.khi`:

```bash
./khi \
  --job-mode \
  --job-inspection-type gcp-gke \
  --job-inspection-features khi.google.com/k8s-common-auditlog/k8s-auditlog-parser-tail#gcp,cloud.google.com/log/k8s-event/timeline-mapper#default,cloud.google.com/log/gke-api/timeline-mapper#default,cloud.google.com/log/compute-api/timeline-mapper#default,cloud.google.com/log/network-api/timeline-mapper#default,cloud.google.com/gke/log/autoscaler/timeline-mapper#default \
  --job-inspection-values '{
    "cloud.google.com/common/input-project-id": "my-project",
    "cloud.google.com/common/input-location": "us-central1",
    "cloud.google.com/common/input-end-time": "2026-01-15T10:00:00Z",
    "cloud.google.com/common/input-duration": "2h",
    "cloud.google.com/k8s/input-cluster-name": "my-cluster",
    "cloud.google.com/k8s/input-namespaces": ["@all_cluster_scoped", "@all_namespaced"],
    "cloud.google.com/k8s/input-kinds": ["@default"]
  }' \
  --job-export-destination ./gke-inspection.khi
```

The command runs once, waits for the inspection to finish, and exits after writing the KHI file.
