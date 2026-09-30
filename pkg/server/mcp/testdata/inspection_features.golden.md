# Inspection `2026-09-24-0130-a1b2`

- Name: prod-cluster-1 restart investigation
- Type: Google Kubernetes Engine (`gcp-gke`)

To change the enabled features, call `update_inspection_features` with the IDs of all features to enable. Then call `dry_run_inspection`.

## Features

| ID | Name | Description | Enabled |
| --- | --- | --- | --- |
| `k8s-audit-log` | Kubernetes Audit Logs | Kubernetes audit logs from Cloud Logging. | yes |
| `k8s-event-log` | Kubernetes Event Logs | Kubernetes events such as scheduling failures and OOM kills. | yes |
| `k8s-node-log` | Kubernetes Node Logs | kubelet and container runtime logs from Cloud Logging. | no |
