# Dry run of `2026-09-24-0130-a1b2`

Errors: 1, warnings: 0. Fix the fields with errors and call `dry_run_inspection` again.

## Fields

### Target

#### `clusterName`
- Label: Cluster name
- Description: The name of the GKE cluster.
- Type: text
- Value: `prod-cluster-1`
- Suggestions: `prod-cluster-1`, `staging-cluster`

#### `location`
- Label: Location
- Description: The region or zone of the cluster.
- Type: text
- Suggestions: `us-central1`
- Error: Location is required.

### Time range

#### `endTime`
- Label: End time
- Description: The end of the query range in RFC3339 format.
- Type: text
- Value: `2026-09-24T02:00:00Z`
- Default: `2026-09-24T06:40:00Z`

## Queries

### K8s audit logs (`k8s-audit`)
Estimated logs: 152000

```text
resource.type="k8s_cluster"
resource.labels.cluster_name="prod-cluster-1"
```
