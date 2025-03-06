# Inspection types

> [!WARNING]
> 🚧 This reference document is under construction. 🚧

Log querying and parsing procedures in KHI is done on a DAG based task execution system.
Each tasks can have dependency and KHI automatically resolves them and run them parallelly as much as possible. 

Inspection type is the first menu item users will select on the `New inspection` menu. Inspection type is usually a cluster type.
KHI filters out unsupported parser for the selected inspection type at first.

<!-- BEGIN GENERATED PART: inspection-type-element-header-gcp-gke -->
## [Google Kubernetes Engine](#gcp-gke)

<!-- END GENERATED PART: inspection-type-element-header-gcp-gke -->
<!-- BEGIN GENERATED PART: inspection-type-element-header-features-gcp-gke -->
### Features

| Feature task name | Description |
| --- | --- |
|[Kubernetes Audit Log](./features.md#kubernetes-audit-log)|Gather kubernetes audit logs and visualize resource modifications.|
|[Kubernetes Event Logs](./features.md#kubernetes-event-logs)|Gather kubernetes event logs and visualize these on the associated resource timeline.|
|[Kubernetes Node Logs](./features.md#kubernetes-node-logs)|Gather node components(e.g docker/container) logs. Log volume can be huge when the cluster has many nodes.|
|[Kubernetes container logs](./features.md#kubernetes-container-logs)|Gather stdout/stderr logs of containers on the cluster to visualize them on the timeline under an associated Pod. Log volume can be huge when the cluster has many Pods.|
|[GKE Audit logs](./features.md#gke-audit-logs)|Gather GKE audit log to show creation/upgrade/deletion of logs cluster/nodepool|
|[Compute API Logs](./features.md#compute-api-logs)|Gather Compute API audit logs to show the timings of the provisioning of resources(e.g creating/deleting GCE VM,mounting Persistent Disk...etc) on associated timelines.|
|[GCE Network Logs](./features.md#gce-network-logs)|Gather GCE Network API logs to visualize statuses of Network Endpoint Groups(NEG)|
|[Autoscaler Logs](./features.md#autoscaler-logs)|Gather logs related to cluster autoscaler behavior to show them on the timelines of resources related to the autoscaler decision.|
|[Kubernetes Control plane component logs](./features.md#kubernetes-control-plane-component-logs)|Gather Kubernetes control plane component(e.g kube-scheduler, kube-controller-manager,api-server) logs|
|[Node serial port logs](./features.md#node-serial-port-logs)|Gather serialport logs of GKE nodes. This helps detailed investigation on VM bootstrapping issue on GKE node.|
<!-- END GENERATED PART: inspection-type-element-header-features-gcp-gke -->
<!-- BEGIN GENERATED PART: inspection-type-element-header-gcp-composer -->
## [Cloud Composer](#gcp-composer)

<!-- END GENERATED PART: inspection-type-element-header-gcp-composer -->
<!-- BEGIN GENERATED PART: inspection-type-element-header-features-gcp-composer -->
### Features

| Feature task name | Description |
| --- | --- |
|[Kubernetes Audit Log](./features.md#kubernetes-audit-log)|Gather kubernetes audit logs and visualize resource modifications.|
|[Kubernetes Event Logs](./features.md#kubernetes-event-logs)|Gather kubernetes event logs and visualize these on the associated resource timeline.|
|[Kubernetes Node Logs](./features.md#kubernetes-node-logs)|Gather node components(e.g docker/container) logs. Log volume can be huge when the cluster has many nodes.|
|[Kubernetes container logs](./features.md#kubernetes-container-logs)|Gather stdout/stderr logs of containers on the cluster to visualize them on the timeline under an associated Pod. Log volume can be huge when the cluster has many Pods.|
|[GKE Audit logs](./features.md#gke-audit-logs)|Gather GKE audit log to show creation/upgrade/deletion of logs cluster/nodepool|
|[Compute API Logs](./features.md#compute-api-logs)|Gather Compute API audit logs to show the timings of the provisioning of resources(e.g creating/deleting GCE VM,mounting Persistent Disk...etc) on associated timelines.|
|[GCE Network Logs](./features.md#gce-network-logs)|Gather GCE Network API logs to visualize statuses of Network Endpoint Groups(NEG)|
|[Autoscaler Logs](./features.md#autoscaler-logs)|Gather logs related to cluster autoscaler behavior to show them on the timelines of resources related to the autoscaler decision.|
|[Kubernetes Control plane component logs](./features.md#kubernetes-control-plane-component-logs)|Gather Kubernetes control plane component(e.g kube-scheduler, kube-controller-manager,api-server) logs|
|[Node serial port logs](./features.md#node-serial-port-logs)|Gather serialport logs of GKE nodes. This helps detailed investigation on VM bootstrapping issue on GKE node.|
|[(Alpha) Composer / Airflow Scheduler](./features.md#alpha-composer--airflow-scheduler)|Airflow Scheduler logs contain information related to the scheduling of TaskInstances, making it an ideal source for understanding the lifecycle of TaskInstances.|
|[(Alpha) Cloud Composer / Airflow Worker](./features.md#alpha-cloud-composer--airflow-worker)|Airflow Worker logs contain information related to the execution of TaskInstances. By including these logs, you can gain insights into where and how each TaskInstance was executed.|
|[(Alpha) Composer / Airflow DagProcessorManager](./features.md#alpha-composer--airflow-dagprocessormanager)|The DagProcessorManager logs contain information for investigating the number of DAGs included in each Python file and the time it took to parse them. You can get information about missing DAGs and load.|
<!-- END GENERATED PART: inspection-type-element-header-features-gcp-composer -->
<!-- BEGIN GENERATED PART: inspection-type-element-header-gcp-gke-on-aws -->
## [GKE on AWS(Anthos on AWS)](#gcp-gke-on-aws)

<!-- END GENERATED PART: inspection-type-element-header-gcp-gke-on-aws -->
<!-- BEGIN GENERATED PART: inspection-type-element-header-features-gcp-gke-on-aws -->
### Features

| Feature task name | Description |
| --- | --- |
|[Kubernetes Audit Log](./features.md#kubernetes-audit-log)|Gather kubernetes audit logs and visualize resource modifications.|
|[Kubernetes Event Logs](./features.md#kubernetes-event-logs)|Gather kubernetes event logs and visualize these on the associated resource timeline.|
|[Kubernetes Node Logs](./features.md#kubernetes-node-logs)|Gather node components(e.g docker/container) logs. Log volume can be huge when the cluster has many nodes.|
|[Kubernetes container logs](./features.md#kubernetes-container-logs)|Gather stdout/stderr logs of containers on the cluster to visualize them on the timeline under an associated Pod. Log volume can be huge when the cluster has many Pods.|
|[MultiCloud API logs](./features.md#multicloud-api-logs)|Gather Anthos Multicloud audit log including cluster creation,deletion and upgrades.|
|[Kubernetes Control plane component logs](./features.md#kubernetes-control-plane-component-logs)|Gather Kubernetes control plane component(e.g kube-scheduler, kube-controller-manager,api-server) logs|
<!-- END GENERATED PART: inspection-type-element-header-features-gcp-gke-on-aws -->
<!-- BEGIN GENERATED PART: inspection-type-element-header-gcp-gke-on-azure -->
## [GKE on Azure(Anthos on Azure)](#gcp-gke-on-azure)

<!-- END GENERATED PART: inspection-type-element-header-gcp-gke-on-azure -->
<!-- BEGIN GENERATED PART: inspection-type-element-header-features-gcp-gke-on-azure -->
### Features

| Feature task name | Description |
| --- | --- |
|[Kubernetes Audit Log](./features.md#kubernetes-audit-log)|Gather kubernetes audit logs and visualize resource modifications.|
|[Kubernetes Event Logs](./features.md#kubernetes-event-logs)|Gather kubernetes event logs and visualize these on the associated resource timeline.|
|[Kubernetes Node Logs](./features.md#kubernetes-node-logs)|Gather node components(e.g docker/container) logs. Log volume can be huge when the cluster has many nodes.|
|[Kubernetes container logs](./features.md#kubernetes-container-logs)|Gather stdout/stderr logs of containers on the cluster to visualize them on the timeline under an associated Pod. Log volume can be huge when the cluster has many Pods.|
|[MultiCloud API logs](./features.md#multicloud-api-logs)|Gather Anthos Multicloud audit log including cluster creation,deletion and upgrades.|
|[Kubernetes Control plane component logs](./features.md#kubernetes-control-plane-component-logs)|Gather Kubernetes control plane component(e.g kube-scheduler, kube-controller-manager,api-server) logs|
<!-- END GENERATED PART: inspection-type-element-header-features-gcp-gke-on-azure -->
<!-- BEGIN GENERATED PART: inspection-type-element-header-gcp-gdcv-for-baremetal -->
## [GDCV for Baremetal(GKE on Baremetal, Anthos on Baremetal)](#gcp-gdcv-for-baremetal)

<!-- END GENERATED PART: inspection-type-element-header-gcp-gdcv-for-baremetal -->
<!-- BEGIN GENERATED PART: inspection-type-element-header-features-gcp-gdcv-for-baremetal -->
### Features

| Feature task name | Description |
| --- | --- |
|[Kubernetes Audit Log](./features.md#kubernetes-audit-log)|Gather kubernetes audit logs and visualize resource modifications.|
|[Kubernetes Event Logs](./features.md#kubernetes-event-logs)|Gather kubernetes event logs and visualize these on the associated resource timeline.|
|[Kubernetes Node Logs](./features.md#kubernetes-node-logs)|Gather node components(e.g docker/container) logs. Log volume can be huge when the cluster has many nodes.|
|[Kubernetes container logs](./features.md#kubernetes-container-logs)|Gather stdout/stderr logs of containers on the cluster to visualize them on the timeline under an associated Pod. Log volume can be huge when the cluster has many Pods.|
|[OnPrem API logs](./features.md#onprem-api-logs)|Gather Anthos OnPrem audit log including cluster creation,deletion,enroll,unenroll and upgrades.|
|[Kubernetes Control plane component logs](./features.md#kubernetes-control-plane-component-logs)|Gather Kubernetes control plane component(e.g kube-scheduler, kube-controller-manager,api-server) logs|
<!-- END GENERATED PART: inspection-type-element-header-features-gcp-gdcv-for-baremetal -->
<!-- BEGIN GENERATED PART: inspection-type-element-header-gcp-gdcv-for-vmware -->
## [GDCV for VMWare(GKE on VMWare, Anthos on VMWare)](#gcp-gdcv-for-vmware)

<!-- END GENERATED PART: inspection-type-element-header-gcp-gdcv-for-vmware -->
<!-- BEGIN GENERATED PART: inspection-type-element-header-features-gcp-gdcv-for-vmware -->
### Features

| Feature task name | Description |
| --- | --- |
|[Kubernetes Audit Log](./features.md#kubernetes-audit-log)|Gather kubernetes audit logs and visualize resource modifications.|
|[Kubernetes Event Logs](./features.md#kubernetes-event-logs)|Gather kubernetes event logs and visualize these on the associated resource timeline.|
|[Kubernetes Node Logs](./features.md#kubernetes-node-logs)|Gather node components(e.g docker/container) logs. Log volume can be huge when the cluster has many nodes.|
|[Kubernetes container logs](./features.md#kubernetes-container-logs)|Gather stdout/stderr logs of containers on the cluster to visualize them on the timeline under an associated Pod. Log volume can be huge when the cluster has many Pods.|
|[OnPrem API logs](./features.md#onprem-api-logs)|Gather Anthos OnPrem audit log including cluster creation,deletion,enroll,unenroll and upgrades.|
|[Kubernetes Control plane component logs](./features.md#kubernetes-control-plane-component-logs)|Gather Kubernetes control plane component(e.g kube-scheduler, kube-controller-manager,api-server) logs|
<!-- END GENERATED PART: inspection-type-element-header-features-gcp-gdcv-for-vmware -->
