# Inspection Summary: prod-cluster-1 restart investigation

- **ID**: `2026-09-24-0130-a1b2`
- **Type**: Google Kubernetes Engine
- **Time Range**: 2026-09-24T01:00:00Z - 2026-09-24T02:00:00Z

## Core Labels

| Key | Value |
| --- | --- |
| clusterName | prod-cluster-1 |
| projectId | my-gcp-project |

## Common Properties

| Property | Value |
| --- | --- |
| duration | 1h |

---

## Kubernetes Audit Logs

### Executed Queries

#### K8s audit logs
```text
resource.type="k8s_cluster"
resource.labels.cluster_name="prod-cluster-1"
```

### Key Properties

| Property | Value |
| --- | --- |
| timelineCount | 342 |
| totalLogs | 1520 |

### Task Reports

#### Audit Log Mapper

| Property | Value |
| --- | --- |
| mappedLogs | 1520 |
| verbs | create, delete, patch, update |

#### List Audit Logs

| Property | Value |
| --- | --- |
| fetchedLogs | 1520 |

Split the query into 4 time ranges because of the log volume.

### Insights

- 3 audit logs failed to parse and were skipped.

---

## Shared & Filter Context

### Key Properties

| Property | Value |
| --- | --- |
| nodeCount | 3 |
