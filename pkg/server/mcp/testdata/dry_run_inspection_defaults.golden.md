# Dry run of `2026-09-24-0130-a1b2`

Errors: 0, warnings: 0. Ready to run with the values below. Call `run_inspection` with the parameters to start the inspection.
These fields use default values because you did not specify them. Pass them explicitly if they are not what you intend:
- `clusterName`: `prod-cluster-1`
- `includeEvents`: `false`

## Fields

Pass text fields as strings, set fields as arrays of strings, and checkbox fields as booleans. Upload files for file fields with `request_file_upload`.

### General

#### `clusterName`
- Label: Cluster name
- Type: text
- Value: `prod-cluster-1` (default)
- Suggestions: `prod-cluster-1`, `staging-cluster`

#### `includeEvents`
- Label: Include events
- Type: checkbox
- Value: `false` (default)

### Time range

#### `endTime`
- Label: End time
- Type: text
- Value: `2026-09-24T02:00:00Z`
- Default: `2026-09-24T06:40:00Z`

## Queries

### K8s audit logs (`k8s-audit`)
```text
resource.type="k8s_cluster"
```
