<!-- BEGIN GENERATED PART: feature-element-header-k8s_audit -->
## [Kubernetes Audit Log(v2)](#k8s_audit)

Visualize Kubernetes audit logs in GKE. 
This parser reveals how these resources are created,updated or deleted. 

<!-- END GENERATED PART: feature-element-header-k8s_audit -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-k8s_audit -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-k8s_audit -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-k8s_audit -->
#### ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit

**Sample used query**

```
resource.type="k8s_cluster"
resource.labels.cluster_name="gcp-cluster-name"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-- Invalid: none of the resources will be selected. Ignoreing kind filter.
-- Invalid: none of the resources will be selected. Ignoreing namespace filter.

```
<!-- END GENERATED PART: feature-element-depending-query-k8s_audit -->
<!-- BEGIN GENERATED PART: feature-element-header-k8s_event -->
## [Kubernetes Event Logs](#k8s_event)

Visualize Kubernetes event logs on GKE.
This parser shows events associated to K8s resources

<!-- END GENERATED PART: feature-element-header-k8s_event -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-k8s_event -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-k8s_event -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-k8s_event -->
#### ![3fb549](https://placehold.co/15x15/3fb549/3fb549.png)k8s_event

**Sample used query**

```
logName="projects/gcp-project-id/logs/events"
resource.labels.cluster_name="gcp-cluster-name"
-- Invalid: none of the resources will be selected. Ignoreing namespace filter.
```
<!-- END GENERATED PART: feature-element-depending-query-k8s_event -->
<!-- BEGIN GENERATED PART: feature-element-header-k8s_node -->
## [Kubernetes Node Logs](#k8s_node)

GKE worker node components logs mainly from kubelet,containerd and dockerd.

(WARNING)Log volume could be very large for long query duration or big cluster and can lead OOM. Please limit time range shorter.

<!-- END GENERATED PART: feature-element-header-k8s_node -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-k8s_node -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-k8s_node -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-k8s_node -->
#### ![0077CC](https://placehold.co/15x15/0077CC/0077CC.png)k8s_node

**Sample used query**

```
resource.type="k8s_node"
-logName="projects/gcp-project-id/logs/events"
resource.labels.cluster_name="gcp-cluster-name"
resource.labels.node_name:("gke-test-cluster-node-1" OR "gke-test-cluster-node-2")

```
<!-- END GENERATED PART: feature-element-depending-query-k8s_node -->
<!-- BEGIN GENERATED PART: feature-element-header-k8s_container -->
## [Kubernetes container logs](#k8s_container)

Container logs ingested from stdout/stderr of workload Pods. 

(WARNING)Log volume could be very large for long query duration or big cluster and can lead OOM. Please limit time range shorter or target namespace fewer.

<!-- END GENERATED PART: feature-element-header-k8s_container -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-k8s_container -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-k8s_container -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-k8s_container -->
#### ![fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)k8s_container

**Sample used query**

```
resource.type="k8s_container"
resource.labels.cluster_name="gcp-cluster-name"
-- Invalid: none of the resources will be selected. Ignoreing kind filter.
-- Invalid: none of the resources will be selected. Ignoreing kind filter.
```
<!-- END GENERATED PART: feature-element-depending-query-k8s_container -->
<!-- BEGIN GENERATED PART: feature-element-header-gke_audit -->
## [GKE Audit logs](#gke_audit)

GKE audit log including cluster creation,deletion and upgrades.

<!-- END GENERATED PART: feature-element-header-gke_audit -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-gke_audit -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-gke_audit -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-gke_audit -->
#### ![AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)gke_audit

**Sample used query**

```
resource.type=("gke_cluster" OR "gke_nodepool")
logName="projects/gcp-project-id/logs/cloudaudit.googleapis.com%2Factivity"
resource.labels.cluster_name="gcp-cluster-name"
```
<!-- END GENERATED PART: feature-element-depending-query-gke_audit -->
<!-- BEGIN GENERATED PART: feature-element-header-compute_api -->
## [Compute API Logs](#compute_api)

Compute API audit logs used for cluster related logs. This also visualize operations happened during the query time.

<!-- END GENERATED PART: feature-element-header-compute_api -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-compute_api -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-compute_api -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-compute_api -->
#### ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit

**Sample used query**

```
resource.type="k8s_cluster"
resource.labels.cluster_name="gcp-cluster-name"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-- Invalid: none of the resources will be selected. Ignoreing kind filter.
-- Invalid: none of the resources will be selected. Ignoreing namespace filter.

```
<!-- END GENERATED PART: feature-element-depending-query-compute_api -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-compute_api -->
#### ![FFCC33](https://placehold.co/15x15/FFCC33/FFCC33.png)compute_api

**Sample used query**

```
resource.type="gce_instance"
	-protoPayload.methodName:("list" OR "get" OR "watch")
	protoPayload.resourceName:(instances/gke-test-cluster-node-1 OR instances/gke-test-cluster-node-2)
	
```
<!-- END GENERATED PART: feature-element-depending-query-compute_api -->
<!-- BEGIN GENERATED PART: feature-element-header-gce_network -->
## [GCE Network Logs](#gce_network)

GCE network API audit log including NEG related audit logs to identify when the associated NEG was attached/detached.

<!-- END GENERATED PART: feature-element-header-gce_network -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-gce_network -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-gce_network -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-gce_network -->
#### ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit

**Sample used query**

```
resource.type="k8s_cluster"
resource.labels.cluster_name="gcp-cluster-name"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-- Invalid: none of the resources will be selected. Ignoreing kind filter.
-- Invalid: none of the resources will be selected. Ignoreing namespace filter.

```
<!-- END GENERATED PART: feature-element-depending-query-gce_network -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-gce_network -->
#### ![33CCFF](https://placehold.co/15x15/33CCFF/33CCFF.png)network_api

**Sample used query**

```
resource.type="gce_network"
-protoPayload.methodName:("list" OR "get" OR "watch")
protoPayload.resourceName:(networkEndpointGroups/neg-id-1 OR networkEndpointGroups/neg-id-2)

```
<!-- END GENERATED PART: feature-element-depending-query-gce_network -->
<!-- BEGIN GENERATED PART: feature-element-header-multicloud_api -->
## [MultiCloud API logs](#multicloud_api)

Anthos Multicloud audit log including cluster creation,deletion and upgrades.

<!-- END GENERATED PART: feature-element-header-multicloud_api -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-multicloud_api -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-multicloud_api -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-multicloud_api -->
#### ![AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)multicloud_api

**Sample used query**

```
resource.type="audited_resource"
resource.labels.service="gkemulticloud.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete")
protoPayload.resourceName:"awsClusters/cluster-foo"

```
<!-- END GENERATED PART: feature-element-depending-query-multicloud_api -->
<!-- BEGIN GENERATED PART: feature-element-header-autoscaler -->
## [Autoscaler Logs](#autoscaler)

Autoscaler logs including decision reasons why they scale up/down or why they didn't.
This log type also includes Node Auto Provisioner logs.

<!-- END GENERATED PART: feature-element-header-autoscaler -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-autoscaler -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-autoscaler -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-autoscaler -->
#### ![FF5555](https://placehold.co/15x15/FF5555/FF5555.png)autoscaler

**Sample used query**

```
resource.type="k8s_cluster"
resource.labels.project_id="gcp-project-id"
resource.labels.cluster_name="gcp-cluster-name"
-jsonPayload.status: ""
logName="projects/gcp-project-id/logs/container.googleapis.com%2Fcluster-autoscaler-visibility"
```
<!-- END GENERATED PART: feature-element-depending-query-autoscaler -->
<!-- BEGIN GENERATED PART: feature-element-header-onprem_api -->
## [OnPrem API logs](#onprem_api)

Anthos OnPrem audit log including cluster creation,deletion,enroll,unenroll and upgrades.

<!-- END GENERATED PART: feature-element-header-onprem_api -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-onprem_api -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-onprem_api -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-onprem_api -->
#### ![AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)onprem_api

**Sample used query**

```
resource.type="audited_resource"
resource.labels.service="gkeonprem.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete" OR "Enroll" OR "Unenroll")
protoPayload.resourceName:"baremetalClusters/my-cluster"

```
<!-- END GENERATED PART: feature-element-depending-query-onprem_api -->
<!-- BEGIN GENERATED PART: feature-element-header-k8s_control_plane_component -->
## [Kubernetes Control plane component logs](#k8s_control_plane_component)

Visualize Kubernetes control plane component logs on a cluster

<!-- END GENERATED PART: feature-element-header-k8s_control_plane_component -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-k8s_control_plane_component -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-k8s_control_plane_component -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-k8s_control_plane_component -->
#### ![FF3333](https://placehold.co/15x15/FF3333/FF3333.png)control_plane_component

**Sample used query**

```
resource.type="k8s_control_plane_component"
resource.labels.cluster_name="gcp-cluster-name"
resource.labels.project_id="gcp-project-id"
-sourceLocation.file="httplog.go"
-- Invalid: none of the controlplane component will be selected. Ignoreing component name filter.
```
<!-- END GENERATED PART: feature-element-depending-query-k8s_control_plane_component -->
<!-- BEGIN GENERATED PART: feature-element-header-serialport -->
## [Node serial port logs](#serialport)

Serial port logs of worker nodes. Serial port logging feature must be enabled on instances to query logs correctly.

<!-- END GENERATED PART: feature-element-header-serialport -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-serialport -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-serialport -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-serialport -->
#### ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit

**Sample used query**

```
resource.type="k8s_cluster"
resource.labels.cluster_name="gcp-cluster-name"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-- Invalid: none of the resources will be selected. Ignoreing kind filter.
-- Invalid: none of the resources will be selected. Ignoreing namespace filter.

```
<!-- END GENERATED PART: feature-element-depending-query-serialport -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-serialport -->
#### ![333333](https://placehold.co/15x15/333333/333333.png)serial_port

**Sample used query**

```
LOG_ID("serialconsole.googleapis.com%2Fserial_port_1_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_2_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_3_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_debug_output")

labels."compute.googleapis.com/resource_name"=("gke-test-cluster-node-1" OR "gke-test-cluster-node-2")

-- No node name substring filters are specified.
```
<!-- END GENERATED PART: feature-element-depending-query-serialport -->
<!-- BEGIN GENERATED PART: feature-element-header-airflow_schedule -->
## [(Alpha) Composer / Airflow Scheduler](#airflow_schedule)

Airflow Scheduler logs contain information related to the scheduling of TaskInstances, making it an ideal source for understanding the lifecycle of TaskInstances.

<!-- END GENERATED PART: feature-element-header-airflow_schedule -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-airflow_schedule -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-airflow_schedule -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-airflow_schedule -->
#### ![88AA55](https://placehold.co/15x15/88AA55/88AA55.png)composer_environment

**Sample used query**

```
TODO: add sample query
```
<!-- END GENERATED PART: feature-element-depending-query-airflow_schedule -->
<!-- BEGIN GENERATED PART: feature-element-header-airflow_worker -->
## [(Alpha) Cloud Composer / Airflow Worker](#airflow_worker)

Airflow Worker logs contain information related to the execution of TaskInstances. By including these logs, you can gain insights into where and how each TaskInstance was executed.

<!-- END GENERATED PART: feature-element-header-airflow_worker -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-airflow_worker -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-airflow_worker -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-airflow_worker -->
#### ![88AA55](https://placehold.co/15x15/88AA55/88AA55.png)composer_environment

**Sample used query**

```
TODO: add sample query
```
<!-- END GENERATED PART: feature-element-depending-query-airflow_worker -->
<!-- BEGIN GENERATED PART: feature-element-header-airflow_dag_processor -->
## [(Alpha) Composer / Airflow DagProcessorManager](#airflow_dag_processor)

The DagProcessorManager logs contain information for investigating the number of DAGs included in each Python file and the time it took to parse them. You can get information about missing DAGs and load.

<!-- END GENERATED PART: feature-element-header-airflow_dag_processor -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-airflow_dag_processor -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-airflow_dag_processor -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-airflow_dag_processor -->
#### ![88AA55](https://placehold.co/15x15/88AA55/88AA55.png)composer_environment

**Sample used query**

```
TODO: add sample query
```
<!-- END GENERATED PART: feature-element-depending-query-airflow_dag_processor -->
