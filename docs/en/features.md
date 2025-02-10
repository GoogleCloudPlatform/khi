<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com//feature/audit-parser-v2 -->
## [Kubernetes Audit Log(v2)](#cloud.google.com//feature/audit-parser-v2)

Visualize Kubernetes audit logs in GKE. 
This parser reveals how these resources are created,updated or deleted. 

<!-- END GENERATED PART: feature-element-header-cloud.google.com//feature/audit-parser-v2 -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com//feature/audit-parser-v2 -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com//feature/audit-parser-v2 -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com//feature/audit-parser-v2 -->
#### ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit

**Sample used query**

```
resource.type="k8s_cluster"
resource.labels.cluster_name="gcp-cluster-name"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-- Invalid: none of the resources will be selected. Ignoreing kind filter.
-- Invalid: none of the resources will be selected. Ignoreing namespace filter.

```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com//feature/audit-parser-v2 -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/event-parser -->
## [Kubernetes Event Logs](#cloud.google.com/feature/event-parser)

Visualize Kubernetes event logs on GKE.
This parser shows events associated to K8s resources

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/event-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/event-parser -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/event-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/event-parser -->
#### ![3fb549](https://placehold.co/15x15/3fb549/3fb549.png)k8s_event

**Sample used query**

```
logName="projects/gcp-project-id/logs/events"
resource.labels.cluster_name="gcp-cluster-name"
-- Invalid: none of the resources will be selected. Ignoreing namespace filter.
```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/event-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/nodelog-parser -->
## [Kubernetes Node Logs](#cloud.google.com/feature/nodelog-parser)

GKE worker node components logs mainly from kubelet,containerd and dockerd.

(WARNING)Log volume could be very large for long query duration or big cluster and can lead OOM. Please limit time range shorter.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/nodelog-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/nodelog-parser -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/nodelog-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/nodelog-parser -->
#### ![0077CC](https://placehold.co/15x15/0077CC/0077CC.png)k8s_node

**Sample used query**

```
resource.type="k8s_node"
-logName="projects/gcp-project-id/logs/events"
resource.labels.cluster_name="gcp-cluster-name"
resource.labels.node_name:("gke-test-cluster-node-1" OR "gke-test-cluster-node-2")

```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/nodelog-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/container-parser -->
## [Kubernetes container logs](#cloud.google.com/feature/container-parser)

Container logs ingested from stdout/stderr of workload Pods. 

(WARNING)Log volume could be very large for long query duration or big cluster and can lead OOM. Please limit time range shorter or target namespace fewer.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/container-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/container-parser -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/container-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/container-parser -->
#### ![fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)k8s_container

**Sample used query**

```
resource.type="k8s_container"
resource.labels.cluster_name="gcp-cluster-name"
-- Invalid: none of the resources will be selected. Ignoreing kind filter.
-- Invalid: none of the resources will be selected. Ignoreing kind filter.
```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/container-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/gke-audit-parser -->
## [GKE Audit logs](#cloud.google.com/feature/gke-audit-parser)

GKE audit log including cluster creation,deletion and upgrades.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/gke-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/gke-audit-parser -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/gke-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/gke-audit-parser -->
#### ![AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)gke_audit

**Sample used query**

```
resource.type=("gke_cluster" OR "gke_nodepool")
logName="projects/gcp-project-id/logs/cloudaudit.googleapis.com%2Factivity"
resource.labels.cluster_name="gcp-cluster-name"
```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/gke-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/compute-api-parser -->
## [Compute API Logs](#cloud.google.com/feature/compute-api-parser)

Compute API audit logs used for cluster related logs. This also visualize operations happened during the query time.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/compute-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/compute-api-parser -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/compute-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/compute-api-parser -->
#### ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit

**Sample used query**

```
resource.type="k8s_cluster"
resource.labels.cluster_name="gcp-cluster-name"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-- Invalid: none of the resources will be selected. Ignoreing kind filter.
-- Invalid: none of the resources will be selected. Ignoreing namespace filter.

```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/compute-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/compute-api-parser -->
#### ![FFCC33](https://placehold.co/15x15/FFCC33/FFCC33.png)compute_api

**Sample used query**

```
resource.type="gce_instance"
	-protoPayload.methodName:("list" OR "get" OR "watch")
	protoPayload.resourceName:(instances/gke-test-cluster-node-1 OR instances/gke-test-cluster-node-2)
	
```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/compute-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/network-api-parser -->
## [GCE Network Logs](#cloud.google.com/feature/network-api-parser)

GCE network API audit log including NEG related audit logs to identify when the associated NEG was attached/detached.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/network-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/network-api-parser -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/network-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/network-api-parser -->
#### ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit

**Sample used query**

```
resource.type="k8s_cluster"
resource.labels.cluster_name="gcp-cluster-name"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-- Invalid: none of the resources will be selected. Ignoreing kind filter.
-- Invalid: none of the resources will be selected. Ignoreing namespace filter.

```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/network-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/network-api-parser -->
#### ![33CCFF](https://placehold.co/15x15/33CCFF/33CCFF.png)network_api

**Sample used query**

```
resource.type="gce_network"
-protoPayload.methodName:("list" OR "get" OR "watch")
protoPayload.resourceName:(networkEndpointGroups/neg-id-1 OR networkEndpointGroups/neg-id-2)

```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/network-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/multicloud-audit-parser -->
## [MultiCloud API logs](#cloud.google.com/feature/multicloud-audit-parser)

Anthos Multicloud audit log including cluster creation,deletion and upgrades.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/multicloud-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/multicloud-audit-parser -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/multicloud-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/multicloud-audit-parser -->
#### ![AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)multicloud_api

**Sample used query**

```
resource.type="audited_resource"
resource.labels.service="gkemulticloud.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete")
protoPayload.resourceName:"awsClusters/cluster-foo"

```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/multicloud-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/autoscaler-parser -->
## [Autoscaler Logs](#cloud.google.com/feature/autoscaler-parser)

Autoscaler logs including decision reasons why they scale up/down or why they didn't.
This log type also includes Node Auto Provisioner logs.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/autoscaler-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/autoscaler-parser -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/autoscaler-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/autoscaler-parser -->
#### ![FF5555](https://placehold.co/15x15/FF5555/FF5555.png)autoscaler

**Sample used query**

```
resource.type="k8s_cluster"
resource.labels.project_id="gcp-project-id"
resource.labels.cluster_name="gcp-cluster-name"
-jsonPayload.status: ""
logName="projects/gcp-project-id/logs/container.googleapis.com%2Fcluster-autoscaler-visibility"
```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/autoscaler-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/onprem-audit-parser -->
## [OnPrem API logs](#cloud.google.com/feature/onprem-audit-parser)

Anthos OnPrem audit log including cluster creation,deletion,enroll,unenroll and upgrades.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/onprem-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/onprem-audit-parser -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/onprem-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/onprem-audit-parser -->
#### ![AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)onprem_api

**Sample used query**

```
resource.type="audited_resource"
resource.labels.service="gkeonprem.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete" OR "Enroll" OR "Unenroll")
protoPayload.resourceName:"baremetalClusters/my-cluster"

```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/onprem-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/controlplane-component-parser -->
## [Kubernetes Control plane component logs](#cloud.google.com/feature/controlplane-component-parser)

Visualize Kubernetes control plane component logs on a cluster

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/controlplane-component-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/controlplane-component-parser -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/controlplane-component-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/controlplane-component-parser -->
#### ![FF3333](https://placehold.co/15x15/FF3333/FF3333.png)control_plane_component

**Sample used query**

```
resource.type="k8s_control_plane_component"
resource.labels.cluster_name="gcp-cluster-name"
resource.labels.project_id="gcp-project-id"
-sourceLocation.file="httplog.go"
-- Invalid: none of the controlplane component will be selected. Ignoreing component name filter.
```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/controlplane-component-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/serialport -->
## [Node serial port logs](#cloud.google.com/feature/serialport)

Serial port logs of worker nodes. Serial port logging feature must be enabled on instances to query logs correctly.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/serialport -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/serialport -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/feature/serialport -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/serialport -->
#### ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit

**Sample used query**

```
resource.type="k8s_cluster"
resource.labels.cluster_name="gcp-cluster-name"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
-- Invalid: none of the resources will be selected. Ignoreing kind filter.
-- Invalid: none of the resources will be selected. Ignoreing namespace filter.

```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/serialport -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/feature/serialport -->
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
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/feature/serialport -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/composer/scheduler -->
## [(Alpha) Composer / Airflow Scheduler](#cloud.google.com/composer/scheduler)

Airflow Scheduler logs contain information related to the scheduling of TaskInstances, making it an ideal source for understanding the lifecycle of TaskInstances.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/composer/scheduler -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/composer/scheduler -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/composer/scheduler -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/composer/scheduler -->
#### ![88AA55](https://placehold.co/15x15/88AA55/88AA55.png)composer_environment

**Sample used query**

```
TODO: add sample query
```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/composer/scheduler -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/composer/worker -->
## [(Alpha) Cloud Composer / Airflow Worker](#cloud.google.com/composer/worker)

Airflow Worker logs contain information related to the execution of TaskInstances. By including these logs, you can gain insights into where and how each TaskInstance was executed.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/composer/worker -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/composer/worker -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/composer/worker -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/composer/worker -->
#### ![88AA55](https://placehold.co/15x15/88AA55/88AA55.png)composer_environment

**Sample used query**

```
TODO: add sample query
```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/composer/worker -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/composer/dagprocessor -->
## [(Alpha) Composer / Airflow DagProcessorManager](#cloud.google.com/composer/dagprocessor)

The DagProcessorManager logs contain information for investigating the number of DAGs included in each Python file and the time it took to parse them. You can get information about missing DAGs and load.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/composer/dagprocessor -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-cloud.google.com/composer/dagprocessor -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-cloud.google.com/composer/dagprocessor -->
<!-- BEGIN GENERATED PART: feature-element-depending-query-cloud.google.com/composer/dagprocessor -->
#### ![88AA55](https://placehold.co/15x15/88AA55/88AA55.png)composer_environment

**Sample used query**

```
TODO: add sample query
```
<!-- END GENERATED PART: feature-element-depending-query-cloud.google.com/composer/dagprocessor -->
