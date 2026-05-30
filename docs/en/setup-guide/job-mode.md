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
  --job-inspection-features ALL \
  --job-inspection-values '{"<form-field-id>":"<value>"}' \
  --job-export-destination ./inspection.khi
```

When the command succeeds, `inspection.khi` can be opened later with KHI.

If you use a released binary or run KHI from another directory, replace `./khi` with the path to that binary.

## **Inspection values**

`--job-inspection-values` must be a JSON object. Each key is a KHI form field ID and each value is the value that would be submitted from the web UI for the same field.

For example, a Google Cloud inspection usually includes common fields such as project, location, query end time, and duration, plus inspection-specific fields:

```json
{
  "cloud.google.com/common/input-project-id": "my-project",
  "cloud.google.com/common/input-location": "us-central1",
  "cloud.google.com/common/input-end-time": "2026-01-15T10:00:00Z",
  "cloud.google.com/common/input-duration": "2h",
  "<inspection-specific-field-id>": "<value>"
}
```

The exact set of required fields depends on the inspection type and enabled features. A practical way to build the JSON is to run the same inspection once in the web UI, note the form field IDs and values for that inspection, then reuse the same values in job mode.

## **Selecting features**

Use `ALL` when the job should include every feature supported by the selected inspection type:

```bash
--job-inspection-features ALL
```

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

The following example runs a GKE inspection and writes the result to `./gke-inspection.khi`:

```bash
./khi \
  --job-mode \
  --job-inspection-type gcp-gke \
  --job-inspection-features ALL \
  --job-inspection-values '{
    "cloud.google.com/common/input-project-id": "my-project",
    "cloud.google.com/common/input-location": "us-central1",
    "cloud.google.com/common/input-end-time": "2026-01-15T10:00:00Z",
    "cloud.google.com/common/input-duration": "2h",
    "<inspection-specific-field-id>": "<value>"
  }' \
  --job-export-destination ./gke-inspection.khi
```

The command runs once, waits for the inspection to finish, and exits after writing the KHI file.
