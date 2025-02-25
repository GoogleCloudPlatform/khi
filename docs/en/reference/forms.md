# Forms

> [!WARNING]
> 🚧 This reference document is under construction. 🚧

<!-- BEGIN GENERATED PART: form-element-header-cloud.google.com/input/project-id -->
## Project ID

The project ID containing logs of the cluster to query
<!-- END GENERATED PART: form-element-header-cloud.google.com/input/project-id -->
<!-- BEGIN GENERATED PART: form-used-feature-cloud.google.com/input/project-id -->
### Features using this parameter

Following feature tasks are depending on this parameter:


* [Kubernetes Audit Log](./features.md#kubernetes-audit-log)
* [Kubernetes Event Logs](./features.md#kubernetes-event-logs)
* [Kubernetes Node Logs](./features.md#kubernetes-node-logs)
* [Kubernetes container logs](./features.md#kubernetes-container-logs)
* [GKE Audit logs](./features.md#gke-audit-logs)
* [Compute API Logs](./features.md#compute-api-logs)
* [GCE Network Logs](./features.md#gce-network-logs)
* [MultiCloud API logs](./features.md#multicloud-api-logs)
* [Autoscaler Logs](./features.md#autoscaler-logs)
* [OnPrem API logs](./features.md#onprem-api-logs)
* [Kubernetes Control plane component logs](./features.md#kubernetes-control-plane-component-logs)
* [Node serial port logs](./features.md#node-serial-port-logs)
* [(Alpha) Composer / Airflow Scheduler](./features.md#alpha-composer--airflow-scheduler)
* [(Alpha) Cloud Composer / Airflow Worker](./features.md#alpha-cloud-composer--airflow-worker)
* [(Alpha) Composer / Airflow DagProcessorManager](./features.md#alpha-composer--airflow-dagprocessormanager)
<!-- END GENERATED PART: form-used-feature-cloud.google.com/input/project-id -->
<!-- BEGIN GENERATED PART: form-element-header-cloud.google.com/input/cluster-name -->
## Cluster name

The cluster name to gather logs.
<!-- END GENERATED PART: form-element-header-cloud.google.com/input/cluster-name -->
<!-- BEGIN GENERATED PART: form-used-feature-cloud.google.com/input/cluster-name -->
### Features using this parameter

Following feature tasks are depending on this parameter:


* [Kubernetes Audit Log](./features.md#kubernetes-audit-log)
* [Kubernetes Event Logs](./features.md#kubernetes-event-logs)
* [Kubernetes Node Logs](./features.md#kubernetes-node-logs)
* [Kubernetes container logs](./features.md#kubernetes-container-logs)
* [GKE Audit logs](./features.md#gke-audit-logs)
* [Compute API Logs](./features.md#compute-api-logs)
* [GCE Network Logs](./features.md#gce-network-logs)
* [MultiCloud API logs](./features.md#multicloud-api-logs)
* [Autoscaler Logs](./features.md#autoscaler-logs)
* [OnPrem API logs](./features.md#onprem-api-logs)
* [Kubernetes Control plane component logs](./features.md#kubernetes-control-plane-component-logs)
* [Node serial port logs](./features.md#node-serial-port-logs)
<!-- END GENERATED PART: form-used-feature-cloud.google.com/input/cluster-name -->
<!-- BEGIN GENERATED PART: form-element-header-cloud.google.com/input/duration -->
## Duration

The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)
<!-- END GENERATED PART: form-element-header-cloud.google.com/input/duration -->
<!-- BEGIN GENERATED PART: form-used-feature-cloud.google.com/input/duration -->
### Features using this parameter

Following feature tasks are depending on this parameter:


* [Kubernetes Audit Log](./features.md#kubernetes-audit-log)
* [Kubernetes Event Logs](./features.md#kubernetes-event-logs)
* [Kubernetes Node Logs](./features.md#kubernetes-node-logs)
* [Kubernetes container logs](./features.md#kubernetes-container-logs)
* [GKE Audit logs](./features.md#gke-audit-logs)
* [Compute API Logs](./features.md#compute-api-logs)
* [GCE Network Logs](./features.md#gce-network-logs)
* [MultiCloud API logs](./features.md#multicloud-api-logs)
* [Autoscaler Logs](./features.md#autoscaler-logs)
* [OnPrem API logs](./features.md#onprem-api-logs)
* [Kubernetes Control plane component logs](./features.md#kubernetes-control-plane-component-logs)
* [Node serial port logs](./features.md#node-serial-port-logs)
* [(Alpha) Composer / Airflow Scheduler](./features.md#alpha-composer--airflow-scheduler)
* [(Alpha) Cloud Composer / Airflow Worker](./features.md#alpha-cloud-composer--airflow-worker)
* [(Alpha) Composer / Airflow DagProcessorManager](./features.md#alpha-composer--airflow-dagprocessormanager)
<!-- END GENERATED PART: form-used-feature-cloud.google.com/input/duration -->
<!-- BEGIN GENERATED PART: form-element-header-cloud.google.com/input/end-time -->
## End time

The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.
<!-- END GENERATED PART: form-element-header-cloud.google.com/input/end-time -->
<!-- BEGIN GENERATED PART: form-used-feature-cloud.google.com/input/end-time -->
### Features using this parameter

Following feature tasks are depending on this parameter:


* [Kubernetes Audit Log](./features.md#kubernetes-audit-log)
* [Kubernetes Event Logs](./features.md#kubernetes-event-logs)
* [Kubernetes Node Logs](./features.md#kubernetes-node-logs)
* [Kubernetes container logs](./features.md#kubernetes-container-logs)
* [GKE Audit logs](./features.md#gke-audit-logs)
* [Compute API Logs](./features.md#compute-api-logs)
* [GCE Network Logs](./features.md#gce-network-logs)
* [MultiCloud API logs](./features.md#multicloud-api-logs)
* [Autoscaler Logs](./features.md#autoscaler-logs)
* [OnPrem API logs](./features.md#onprem-api-logs)
* [Kubernetes Control plane component logs](./features.md#kubernetes-control-plane-component-logs)
* [Node serial port logs](./features.md#node-serial-port-logs)
* [(Alpha) Composer / Airflow Scheduler](./features.md#alpha-composer--airflow-scheduler)
* [(Alpha) Cloud Composer / Airflow Worker](./features.md#alpha-cloud-composer--airflow-worker)
* [(Alpha) Composer / Airflow DagProcessorManager](./features.md#alpha-composer--airflow-dagprocessormanager)
<!-- END GENERATED PART: form-used-feature-cloud.google.com/input/end-time -->
<!-- BEGIN GENERATED PART: form-element-header-cloud.google.com/input/kinds -->
## Kind

The kinds of resources to gather logs. `@default` is a alias of set of kinds that frequently queried. Specify `@any` to query every kinds of resources
<!-- END GENERATED PART: form-element-header-cloud.google.com/input/kinds -->
<!-- BEGIN GENERATED PART: form-used-feature-cloud.google.com/input/kinds -->
### Features using this parameter

Following feature tasks are depending on this parameter:


* [Kubernetes Audit Log](./features.md#kubernetes-audit-log)
* [Compute API Logs](./features.md#compute-api-logs)
* [GCE Network Logs](./features.md#gce-network-logs)
* [Node serial port logs](./features.md#node-serial-port-logs)
<!-- END GENERATED PART: form-used-feature-cloud.google.com/input/kinds -->
<!-- BEGIN GENERATED PART: form-element-header-cloud.google.com/input/location -->
## Location


<!-- END GENERATED PART: form-element-header-cloud.google.com/input/location -->
<!-- BEGIN GENERATED PART: form-used-feature-cloud.google.com/input/location -->
### Features using this parameter

Following feature tasks are depending on this parameter:


* [(Alpha) Composer / Airflow Scheduler](./features.md#alpha-composer--airflow-scheduler)
* [(Alpha) Cloud Composer / Airflow Worker](./features.md#alpha-cloud-composer--airflow-worker)
* [(Alpha) Composer / Airflow DagProcessorManager](./features.md#alpha-composer--airflow-dagprocessormanager)
<!-- END GENERATED PART: form-used-feature-cloud.google.com/input/location -->
<!-- BEGIN GENERATED PART: form-element-header-cloud.google.com/input/namespaces -->
## Namespaces

The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.
<!-- END GENERATED PART: form-element-header-cloud.google.com/input/namespaces -->
<!-- BEGIN GENERATED PART: form-used-feature-cloud.google.com/input/namespaces -->
### Features using this parameter

Following feature tasks are depending on this parameter:


* [Kubernetes Audit Log](./features.md#kubernetes-audit-log)
* [Kubernetes Event Logs](./features.md#kubernetes-event-logs)
* [Compute API Logs](./features.md#compute-api-logs)
* [GCE Network Logs](./features.md#gce-network-logs)
* [Node serial port logs](./features.md#node-serial-port-logs)
<!-- END GENERATED PART: form-used-feature-cloud.google.com/input/namespaces -->
<!-- BEGIN GENERATED PART: form-element-header-cloud.google.com/input/node-name-filter -->
## Node names

A space-separated list of node name substrings used to collect node-related logs. If left blank, KHI gathers logs from all nodes in the cluster.
<!-- END GENERATED PART: form-element-header-cloud.google.com/input/node-name-filter -->
<!-- BEGIN GENERATED PART: form-used-feature-cloud.google.com/input/node-name-filter -->
### Features using this parameter

Following feature tasks are depending on this parameter:


* [Kubernetes Node Logs](./features.md#kubernetes-node-logs)
* [Node serial port logs](./features.md#node-serial-port-logs)
<!-- END GENERATED PART: form-used-feature-cloud.google.com/input/node-name-filter -->
<!-- BEGIN GENERATED PART: form-element-header-cloud.google.com/input/container-query-namespaces -->
## Namespaces(Container logs)

The namespace of Pods to gather container logs. Specify `@managed` to gather logs of system components.
<!-- END GENERATED PART: form-element-header-cloud.google.com/input/container-query-namespaces -->
<!-- BEGIN GENERATED PART: form-used-feature-cloud.google.com/input/container-query-namespaces -->
### Features using this parameter

Following feature tasks are depending on this parameter:


* [Kubernetes container logs](./features.md#kubernetes-container-logs)
<!-- END GENERATED PART: form-used-feature-cloud.google.com/input/container-query-namespaces -->
<!-- BEGIN GENERATED PART: form-element-header-cloud.google.com/input/container-query-podnames -->
## Pod names(Container logs)

The substring of Pod name to gather container logs. Specify `@any` to gather logs of all pods.
<!-- END GENERATED PART: form-element-header-cloud.google.com/input/container-query-podnames -->
<!-- BEGIN GENERATED PART: form-used-feature-cloud.google.com/input/container-query-podnames -->
### Features using this parameter

Following feature tasks are depending on this parameter:


* [Kubernetes container logs](./features.md#kubernetes-container-logs)
<!-- END GENERATED PART: form-used-feature-cloud.google.com/input/container-query-podnames -->
<!-- BEGIN GENERATED PART: form-element-header-cloud.google.com/input/component-names -->
## Control plane component names


<!-- END GENERATED PART: form-element-header-cloud.google.com/input/component-names -->
<!-- BEGIN GENERATED PART: form-used-feature-cloud.google.com/input/component-names -->
### Features using this parameter

Following feature tasks are depending on this parameter:


* [Kubernetes Control plane component logs](./features.md#kubernetes-control-plane-component-logs)
<!-- END GENERATED PART: form-used-feature-cloud.google.com/input/component-names -->
<!-- BEGIN GENERATED PART: form-element-header-cloud.google.com/input/composer/environment_name -->
## Composer Environment Name


<!-- END GENERATED PART: form-element-header-cloud.google.com/input/composer/environment_name -->
<!-- BEGIN GENERATED PART: form-used-feature-cloud.google.com/input/composer/environment_name -->
### Features using this parameter

Following feature tasks are depending on this parameter:


* [(Alpha) Composer / Airflow Scheduler](./features.md#alpha-composer--airflow-scheduler)
* [(Alpha) Cloud Composer / Airflow Worker](./features.md#alpha-cloud-composer--airflow-worker)
* [(Alpha) Composer / Airflow DagProcessorManager](./features.md#alpha-composer--airflow-dagprocessormanager)
<!-- END GENERATED PART: form-used-feature-cloud.google.com/input/composer/environment_name -->
