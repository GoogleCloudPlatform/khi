# Features

The output timelnes of KHI is formed in the `feature tasks`. A feature may depends on parameters, other log query.
User will select features on the 2nd menu of the dialog after clicking `New inspection` button.

<!-- BEGIN GENERATED PART: feature-element-header-k8s_audit -->
## [Kubernetes Audit Log](#k8s_audit)

Visualize Kubernetes audit logs in GKE. 
This parser reveals how these resources are created,updated or deleted. 

<!-- END GENERATED PART: feature-element-header-k8s_audit -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-k8s_audit -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Kind|The kinds of resources to gather logs. `@default` is a alias of set of kinds that frequently queried. Specify `@any` to query every kinds of resources|
|Namespaces|The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.|
|Project ID|The project ID containing the logs of cluster to query|
|Cluster name|The cluster name to gather logs.|
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-k8s_audit -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-k8s_audit -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|
|![CCCCCC](https://placehold.co/15x15/CCCCCC/CCCCCC.png)[The default resource timeline](./relationships.md#RelationshipChild)|resource|
|![4c29e8](https://placehold.co/15x15/4c29e8/4c29e8.png)[Status condition field timeline](./relationships.md#RelationshipResourceCondition)|condition|
|![008000](https://placehold.co/15x15/008000/008000.png)[Endpoint serving state timeline](./relationships.md#RelationshipEndpointSlice)|endpointslice|
|![33DD88](https://placehold.co/15x15/33DD88/33DD88.png)[Owning children timeline](./relationships.md#RelationshipOwnerReference)|owns|
|![FF8855](https://placehold.co/15x15/FF8855/FF8855.png)[Pod binding timeline](./relationships.md#RelationshipPodBinding)|binds|

<!-- END GENERATED PART: feature-element-output-timelines-k8s_audit -->
<!-- BEGIN GENERATED PART: feature-element-target-query-k8s_audit -->
### Target log type

**![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit**

Sample query:

```
resource.type="k8s_cluster"
resource.labels.cluster_name="gcp-cluster-name"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
protoPayload.methodName=~"\.(deployments|replicasets|pods|nodes)\."
-- No namespace filter

```

<!-- END GENERATED PART: feature-element-target-query-k8s_audit -->
<!-- BEGIN GENERATED PART: feature-element-header-k8s_event -->
## [Kubernetes Event Logs](#k8s_event)

Visualize Kubernetes event logs on GKE.
This parser shows events associated to K8s resources

<!-- END GENERATED PART: feature-element-header-k8s_event -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-k8s_event -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Namespaces|The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.|
|Project ID|The project ID containing the logs of cluster to query|
|Cluster name|The cluster name to gather logs.|
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-k8s_event -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-k8s_event -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|
|![CCCCCC](https://placehold.co/15x15/CCCCCC/CCCCCC.png)[The default resource timeline](./relationships.md#RelationshipChild)|resource|

<!-- END GENERATED PART: feature-element-output-timelines-k8s_event -->
<!-- BEGIN GENERATED PART: feature-element-target-query-k8s_event -->
### Target log type

**![3fb549](https://placehold.co/15x15/3fb549/3fb549.png)k8s_event**

Sample query:

```
logName="projects/gcp-project-id/logs/events"
resource.labels.cluster_name="gcp-cluster-name"
-- No namespace filter
```

<!-- END GENERATED PART: feature-element-target-query-k8s_event -->
<!-- BEGIN GENERATED PART: feature-element-header-k8s_node -->
## [Kubernetes Node Logs](#k8s_node)

GKE worker node components logs mainly from kubelet,containerd and dockerd.

(WARNING)Log volume could be very large for long query duration or big cluster and can lead OOM. Please limit time range shorter.

<!-- END GENERATED PART: feature-element-header-k8s_node -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-k8s_node -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Node names||
|Project ID|The project ID containing the logs of cluster to query|
|Cluster name|The cluster name to gather logs.|
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-k8s_node -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-k8s_node -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|
|![fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)[Container timeline](./relationships.md#RelationshipContainer)|container|
|![0077CC](https://placehold.co/15x15/0077CC/0077CC.png)[Node component timeline](./relationships.md#RelationshipNodeComponent)|node-component|

<!-- END GENERATED PART: feature-element-output-timelines-k8s_node -->
<!-- BEGIN GENERATED PART: feature-element-target-query-k8s_node -->
### Target log type

**![0077CC](https://placehold.co/15x15/0077CC/0077CC.png)k8s_node**

Sample query:

```
resource.type="k8s_node"
-logName="projects/gcp-project-id/logs/events"
resource.labels.cluster_name="gcp-cluster-name"
resource.labels.node_name:("gke-test-cluster-node-1" OR "gke-test-cluster-node-2")

```

<!-- END GENERATED PART: feature-element-target-query-k8s_node -->
<!-- BEGIN GENERATED PART: feature-element-header-k8s_container -->
## [Kubernetes container logs](#k8s_container)

Container logs ingested from stdout/stderr of workload Pods. 

(WARNING)Log volume could be very large for long query duration or big cluster and can lead OOM. Please limit time range shorter or target namespace fewer.

<!-- END GENERATED PART: feature-element-header-k8s_container -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-k8s_container -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Namespaces(Container logs)||
|Pod names(Container logs)||
|Project ID|The project ID containing the logs of cluster to query|
|Cluster name|The cluster name to gather logs.|
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-k8s_container -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-k8s_container -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|
|![fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)[Container timeline](./relationships.md#RelationshipContainer)|container|

<!-- END GENERATED PART: feature-element-output-timelines-k8s_container -->
<!-- BEGIN GENERATED PART: feature-element-target-query-k8s_container -->
### Target log type

**![fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)k8s_container**

Sample query:

```
resource.type="k8s_container"
resource.labels.cluster_name="gcp-cluster-name"
resource.labels.namespace_name=("default")
-resource.labels.pod_name:("nginx-" OR "redis")
```

<!-- END GENERATED PART: feature-element-target-query-k8s_container -->
<!-- BEGIN GENERATED PART: feature-element-header-gke_audit -->
## [GKE Audit logs](#gke_audit)

GKE audit log including cluster creation,deletion and upgrades.

<!-- END GENERATED PART: feature-element-header-gke_audit -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-gke_audit -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Project ID|The project ID containing the logs of cluster to query|
|Cluster name|The cluster name to gather logs.|
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-gke_audit -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-gke_audit -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|

<!-- END GENERATED PART: feature-element-output-timelines-gke_audit -->
<!-- BEGIN GENERATED PART: feature-element-target-query-gke_audit -->
### Target log type

**![AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)gke_audit**

Sample query:

```
resource.type=("gke_cluster" OR "gke_nodepool")
logName="projects/gcp-project-id/logs/cloudaudit.googleapis.com%2Factivity"
resource.labels.cluster_name="gcp-cluster-name"
```

<!-- END GENERATED PART: feature-element-target-query-gke_audit -->
<!-- BEGIN GENERATED PART: feature-element-header-compute_api -->
## [Compute API Logs](#compute_api)

Compute API audit logs used for cluster related logs. This also visualize operations happened during the query time.

<!-- END GENERATED PART: feature-element-header-compute_api -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-compute_api -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Kind|The kinds of resources to gather logs. `@default` is a alias of set of kinds that frequently queried. Specify `@any` to query every kinds of resources|
|Namespaces|The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.|
|Project ID|The project ID containing the logs of cluster to query|
|Cluster name|The cluster name to gather logs.|
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-compute_api -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-compute_api -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|
|![000000](https://placehold.co/15x15/000000/000000.png)[Operation timeline](./relationships.md#RelationshipOperation)|operation|

<!-- END GENERATED PART: feature-element-output-timelines-compute_api -->
<!-- BEGIN GENERATED PART: feature-element-target-query-compute_api -->
### Target log type

**![FFCC33](https://placehold.co/15x15/FFCC33/FFCC33.png)compute_api**

Sample query:

```
resource.type="gce_instance"
-protoPayload.methodName:("list" OR "get" OR "watch")
protoPayload.resourceName:(instances/gke-test-cluster-node-1 OR instances/gke-test-cluster-node-2)

```

<!-- END GENERATED PART: feature-element-target-query-compute_api -->
<!-- BEGIN GENERATED PART: feature-element-depending-indirect-query-header-compute_api -->
### Dependent queries

Following log queries are used with this feature.

* ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit
<!-- END GENERATED PART: feature-element-depending-indirect-query-header-compute_api -->
<!-- BEGIN GENERATED PART: feature-element-header-gce_network -->
## [GCE Network Logs](#gce_network)

GCE network API audit log including NEG related audit logs to identify when the associated NEG was attached/detached.

<!-- END GENERATED PART: feature-element-header-gce_network -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-gce_network -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Kind|The kinds of resources to gather logs. `@default` is a alias of set of kinds that frequently queried. Specify `@any` to query every kinds of resources|
|Namespaces|The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.|
|Project ID|The project ID containing the logs of cluster to query|
|Cluster name|The cluster name to gather logs.|
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-gce_network -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-gce_network -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|
|![A52A2A](https://placehold.co/15x15/A52A2A/A52A2A.png)[NEG timeline](./relationships.md#RelationshipNetworkEndpointGroup)|neg|

<!-- END GENERATED PART: feature-element-output-timelines-gce_network -->
<!-- BEGIN GENERATED PART: feature-element-target-query-gce_network -->
### Target log type

**![33CCFF](https://placehold.co/15x15/33CCFF/33CCFF.png)network_api**

Sample query:

```
resource.type="gce_network"
-protoPayload.methodName:("list" OR "get" OR "watch")
protoPayload.resourceName:(networkEndpointGroups/neg-id-1 OR networkEndpointGroups/neg-id-2)

```

<!-- END GENERATED PART: feature-element-target-query-gce_network -->
<!-- BEGIN GENERATED PART: feature-element-depending-indirect-query-header-gce_network -->
### Dependent queries

Following log queries are used with this feature.

* ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit
<!-- END GENERATED PART: feature-element-depending-indirect-query-header-gce_network -->
<!-- BEGIN GENERATED PART: feature-element-header-multicloud_api -->
## [MultiCloud API logs](#multicloud_api)

Anthos Multicloud audit log including cluster creation,deletion and upgrades.

<!-- END GENERATED PART: feature-element-header-multicloud_api -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-multicloud_api -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Project ID|The project ID containing the logs of cluster to query|
|Cluster name|The cluster name to gather logs.|
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-multicloud_api -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-multicloud_api -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|

<!-- END GENERATED PART: feature-element-output-timelines-multicloud_api -->
<!-- BEGIN GENERATED PART: feature-element-target-query-multicloud_api -->
### Target log type

**![AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)multicloud_api**

Sample query:

```
resource.type="audited_resource"
resource.labels.service="gkemulticloud.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete")
protoPayload.resourceName:"awsClusters/cluster-foo"

```

<!-- END GENERATED PART: feature-element-target-query-multicloud_api -->
<!-- BEGIN GENERATED PART: feature-element-header-autoscaler -->
## [Autoscaler Logs](#autoscaler)

Autoscaler logs including decision reasons why they scale up/down or why they didn't.
This log type also includes Node Auto Provisioner logs.

<!-- END GENERATED PART: feature-element-header-autoscaler -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-autoscaler -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Project ID|The project ID containing the logs of cluster to query|
|Cluster name|The cluster name to gather logs.|
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-autoscaler -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-autoscaler -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|
|![FF5555](https://placehold.co/15x15/FF5555/FF5555.png)[Managed instance group timeline](./relationships.md#RelationshipManagedInstanceGroup)|mig|

<!-- END GENERATED PART: feature-element-output-timelines-autoscaler -->
<!-- BEGIN GENERATED PART: feature-element-target-query-autoscaler -->
### Target log type

**![FF5555](https://placehold.co/15x15/FF5555/FF5555.png)autoscaler**

Sample query:

```
resource.type="k8s_cluster"
resource.labels.project_id="gcp-project-id"
resource.labels.cluster_name="gcp-cluster-name"
-jsonPayload.status: ""
logName="projects/gcp-project-id/logs/container.googleapis.com%2Fcluster-autoscaler-visibility"
```

<!-- END GENERATED PART: feature-element-target-query-autoscaler -->
<!-- BEGIN GENERATED PART: feature-element-header-onprem_api -->
## [OnPrem API logs](#onprem_api)

Anthos OnPrem audit log including cluster creation,deletion,enroll,unenroll and upgrades.

<!-- END GENERATED PART: feature-element-header-onprem_api -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-onprem_api -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Project ID|The project ID containing the logs of cluster to query|
|Cluster name|The cluster name to gather logs.|
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-onprem_api -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-onprem_api -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|

<!-- END GENERATED PART: feature-element-output-timelines-onprem_api -->
<!-- BEGIN GENERATED PART: feature-element-target-query-onprem_api -->
### Target log type

**![AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)onprem_api**

Sample query:

```
resource.type="audited_resource"
resource.labels.service="gkeonprem.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete" OR "Enroll" OR "Unenroll")
protoPayload.resourceName:"baremetalClusters/my-cluster"

```

<!-- END GENERATED PART: feature-element-target-query-onprem_api -->
<!-- BEGIN GENERATED PART: feature-element-header-k8s_control_plane_component -->
## [Kubernetes Control plane component logs](#k8s_control_plane_component)

Visualize Kubernetes control plane component logs on a cluster

<!-- END GENERATED PART: feature-element-header-k8s_control_plane_component -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-k8s_control_plane_component -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Control plane component names||
|Project ID|The project ID containing the logs of cluster to query|
|Cluster name|The cluster name to gather logs.|
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-k8s_control_plane_component -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-k8s_control_plane_component -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|
|![FF5555](https://placehold.co/15x15/FF5555/FF5555.png)[Control plane component timeline](./relationships.md#RelationshipControlPlaneComponent)|controlplane|

<!-- END GENERATED PART: feature-element-output-timelines-k8s_control_plane_component -->
<!-- BEGIN GENERATED PART: feature-element-target-query-k8s_control_plane_component -->
### Target log type

**![FF3333](https://placehold.co/15x15/FF3333/FF3333.png)control_plane_component**

Sample query:

```
resource.type="k8s_control_plane_component"
resource.labels.cluster_name="gcp-cluster-name"
resource.labels.project_id="gcp-project-id"
-sourceLocation.file="httplog.go"
-- No component name filter
```

<!-- END GENERATED PART: feature-element-target-query-k8s_control_plane_component -->
<!-- BEGIN GENERATED PART: feature-element-header-serialport -->
## [Node serial port logs](#serialport)

Serial port logs of worker nodes. Serial port logging feature must be enabled on instances to query logs correctly.

<!-- END GENERATED PART: feature-element-header-serialport -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-serialport -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Kind|The kinds of resources to gather logs. `@default` is a alias of set of kinds that frequently queried. Specify `@any` to query every kinds of resources|
|Namespaces|The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.|
|Node names||
|Project ID|The project ID containing the logs of cluster to query|
|Cluster name|The cluster name to gather logs.|
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-serialport -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-serialport -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|

<!-- END GENERATED PART: feature-element-output-timelines-serialport -->
<!-- BEGIN GENERATED PART: feature-element-target-query-serialport -->
### Target log type

**![333333](https://placehold.co/15x15/333333/333333.png)serial_port**

Sample query:

```
LOG_ID("serialconsole.googleapis.com%2Fserial_port_1_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_2_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_3_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_debug_output")

labels."compute.googleapis.com/resource_name"=("gke-test-cluster-node-1" OR "gke-test-cluster-node-2")

-- No node name substring filters are specified.
```

<!-- END GENERATED PART: feature-element-target-query-serialport -->
<!-- BEGIN GENERATED PART: feature-element-depending-indirect-query-header-serialport -->
### Dependent queries

Following log queries are used with this feature.

* ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit
<!-- END GENERATED PART: feature-element-depending-indirect-query-header-serialport -->
<!-- BEGIN GENERATED PART: feature-element-header-airflow_schedule -->
## [(Alpha) Composer / Airflow Scheduler](#airflow_schedule)

Airflow Scheduler logs contain information related to the scheduling of TaskInstances, making it an ideal source for understanding the lifecycle of TaskInstances.

<!-- END GENERATED PART: feature-element-header-airflow_schedule -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-airflow_schedule -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Location||
|Project ID|The project ID containing the logs of cluster to query|
|Composer Environment Name||
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-airflow_schedule -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-airflow_schedule -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|

<!-- END GENERATED PART: feature-element-output-timelines-airflow_schedule -->
<!-- BEGIN GENERATED PART: feature-element-target-query-airflow_schedule -->
### Target log type

**![88AA55](https://placehold.co/15x15/88AA55/88AA55.png)composer_environment**

Sample query:

```
TODO: add sample query
```

<!-- END GENERATED PART: feature-element-target-query-airflow_schedule -->
<!-- BEGIN GENERATED PART: feature-element-header-airflow_worker -->
## [(Alpha) Cloud Composer / Airflow Worker](#airflow_worker)

Airflow Worker logs contain information related to the execution of TaskInstances. By including these logs, you can gain insights into where and how each TaskInstance was executed.

<!-- END GENERATED PART: feature-element-header-airflow_worker -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-airflow_worker -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Location||
|Project ID|The project ID containing the logs of cluster to query|
|Composer Environment Name||
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-airflow_worker -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-airflow_worker -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|

<!-- END GENERATED PART: feature-element-output-timelines-airflow_worker -->
<!-- BEGIN GENERATED PART: feature-element-target-query-airflow_worker -->
### Target log type

**![88AA55](https://placehold.co/15x15/88AA55/88AA55.png)composer_environment**

Sample query:

```
TODO: add sample query
```

<!-- END GENERATED PART: feature-element-target-query-airflow_worker -->
<!-- BEGIN GENERATED PART: feature-element-header-airflow_dag_processor -->
## [(Alpha) Composer / Airflow DagProcessorManager](#airflow_dag_processor)

The DagProcessorManager logs contain information for investigating the number of DAGs included in each Python file and the time it took to parse them. You can get information about missing DAGs and load.

<!-- END GENERATED PART: feature-element-header-airflow_dag_processor -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-airflow_dag_processor -->
### Parameters

|Parameter name|Description|
|:-:|---|
|Location||
|Project ID|The project ID containing the logs of cluster to query|
|Composer Environment Name||
|End time|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|Duration|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-airflow_dag_processor -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-airflow_dag_processor -->
### Output timelines

|Timeline type|Short name on chip|
|:-:|:-:|

<!-- END GENERATED PART: feature-element-output-timelines-airflow_dag_processor -->
<!-- BEGIN GENERATED PART: feature-element-target-query-airflow_dag_processor -->
### Target log type

**![88AA55](https://placehold.co/15x15/88AA55/88AA55.png)composer_environment**

Sample query:

```
TODO: add sample query
```

<!-- END GENERATED PART: feature-element-target-query-airflow_dag_processor -->
