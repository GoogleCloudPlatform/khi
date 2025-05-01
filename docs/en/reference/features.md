# Features

> [!WARNING]
> 🚧 This reference document is under construction. 🚧

The output timelnes of KHI is formed in the `feature tasks`. A feature may depends on parameters, other log query.
User will select features on the 2nd menu of the dialog after clicking `New inspection` button.

<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com//feature/audit-parser-v2 -->
## Kubernetes Audit Log

Gather kubernetes audit logs and visualize resource modifications.

<!-- END GENERATED PART: feature-element-header-cloud.google.com//feature/audit-parser-v2 -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com//feature/audit-parser-v2 -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Kind](./forms.md#kind)|The kinds of resources to gather logs. `@default` is a alias of set of kinds that frequently queried. Specify `@any` to query every kinds of resources|
|[Namespaces](./forms.md#namespaces)|The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.|
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Cluster name](./forms.md#cluster-name)|The cluster name to gather logs.|
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com//feature/audit-parser-v2 -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com//feature/audit-parser-v2 -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
|![CCCCCC](https://placehold.co/15x15/CCCCCC/CCCCCC.png)[The default resource timeline](./relationships.md#the-default-resource-timeline)|resource|A default timeline recording the history of Kubernetes resources|
|![4c29e8](https://placehold.co/15x15/4c29e8/4c29e8.png)[Status condition field timeline](./relationships.md#status-condition-field-timeline)|condition|A timeline showing the state changes on `.status.conditions` of the parent resource|
|![008000](https://placehold.co/15x15/008000/008000.png)[Endpoint serving state timeline](./relationships.md#endpoint-serving-state-timeline)|endpointslice|A timeline indicates the status of endpoint related to the parent resource(Pod or Service)|
|![fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)[Container timeline](./relationships.md#container-timeline)|container|A timline of a container included in the parent timeline of a Pod|
|![33DD88](https://placehold.co/15x15/33DD88/33DD88.png)[Owning children timeline](./relationships.md#owning-children-timeline)|owns||
|![FF8855](https://placehold.co/15x15/FF8855/FF8855.png)[Pod binding timeline](./relationships.md#pod-binding-timeline)|binds||

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com//feature/audit-parser-v2 -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com//feature/audit-parser-v2 -->
### Target log type

**![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit**

Sample query:

```ada
resource.type="k8s_cluster"
resource.labels.cluster_name="gcp-cluster-name"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
protoPayload.methodName=~"\.(deployments|replicasets|pods|nodes)\."
-- No namespace filter

```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com//feature/audit-parser-v2 -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com//feature/audit-parser-v2 -->
### Inspection types

This feature is supported in the following inspection types.

* [Google Kubernetes Engine](./inspection-type.md#google-kubernetes-engine)
* [Cloud Composer](./inspection-type.md#cloud-composer)
* [GKE on AWS(Anthos on AWS)](./inspection-type.md#gke-on-awsanthos-on-aws)
* [GKE on Azure(Anthos on Azure)](./inspection-type.md#gke-on-azureanthos-on-azure)
* [GDCV for Baremetal(GKE on Baremetal, Anthos on Baremetal)](./inspection-type.md#gdcv-for-baremetalgke-on-baremetal-anthos-on-baremetal)
* [GDCV for VMWare(GKE on VMWare, Anthos on VMWare)](./inspection-type.md#gdcv-for-vmwaregke-on-vmware-anthos-on-vmware)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com//feature/audit-parser-v2 -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/event-parser -->
## Kubernetes Event Logs

Gather kubernetes event logs and visualize these on the associated resource timeline.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/event-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/event-parser -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Namespaces](./forms.md#namespaces)|The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.|
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Cluster name](./forms.md#cluster-name)|The cluster name to gather logs.|
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/event-parser -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/event-parser -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
|![CCCCCC](https://placehold.co/15x15/CCCCCC/CCCCCC.png)[The default resource timeline](./relationships.md#the-default-resource-timeline)|resource|A default timeline recording the history of Kubernetes resources|

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/event-parser -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/feature/event-parser -->
### Target log type

**![3fb549](https://placehold.co/15x15/3fb549/3fb549.png)k8s_event**

Sample query:

```ada
logName="projects/gcp-project-id/logs/events"
resource.labels.cluster_name="gcp-cluster-name"
-- No namespace filter
```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/feature/event-parser -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/event-parser -->
### Inspection types

This feature is supported in the following inspection types.

* [Google Kubernetes Engine](./inspection-type.md#google-kubernetes-engine)
* [Cloud Composer](./inspection-type.md#cloud-composer)
* [GKE on AWS(Anthos on AWS)](./inspection-type.md#gke-on-awsanthos-on-aws)
* [GKE on Azure(Anthos on Azure)](./inspection-type.md#gke-on-azureanthos-on-azure)
* [GDCV for Baremetal(GKE on Baremetal, Anthos on Baremetal)](./inspection-type.md#gdcv-for-baremetalgke-on-baremetal-anthos-on-baremetal)
* [GDCV for VMWare(GKE on VMWare, Anthos on VMWare)](./inspection-type.md#gdcv-for-vmwaregke-on-vmware-anthos-on-vmware)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/event-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/nodelog-parser -->
## Kubernetes Node Logs

Gather node components(e.g docker/container) logs. Log volume can be huge when the cluster has many nodes.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/nodelog-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/nodelog-parser -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Node names](./forms.md#node-names)|A space-separated list of node name substrings used to collect node-related logs. If left blank, KHI gathers logs from all nodes in the cluster.|
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Cluster name](./forms.md#cluster-name)|The cluster name to gather logs.|
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/nodelog-parser -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/nodelog-parser -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
|![CCCCCC](https://placehold.co/15x15/CCCCCC/CCCCCC.png)[The default resource timeline](./relationships.md#the-default-resource-timeline)|resource|A default timeline recording the history of Kubernetes resources|
|![fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)[Container timeline](./relationships.md#container-timeline)|container|A timline of a container included in the parent timeline of a Pod|
|![0077CC](https://placehold.co/15x15/0077CC/0077CC.png)[Node component timeline](./relationships.md#node-component-timeline)|node-component|A component running inside of the parent timeline of a Node|

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/nodelog-parser -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/feature/nodelog-parser -->
### Target log type

**![0077CC](https://placehold.co/15x15/0077CC/0077CC.png)k8s_node**

Sample query:

```ada
resource.type="k8s_node"
-logName="projects/gcp-project-id/logs/events"
resource.labels.cluster_name="gcp-cluster-name"
resource.labels.node_name:("gke-test-cluster-node-1" OR "gke-test-cluster-node-2")

```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/feature/nodelog-parser -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/nodelog-parser -->
### Inspection types

This feature is supported in the following inspection types.

* [Google Kubernetes Engine](./inspection-type.md#google-kubernetes-engine)
* [Cloud Composer](./inspection-type.md#cloud-composer)
* [GKE on AWS(Anthos on AWS)](./inspection-type.md#gke-on-awsanthos-on-aws)
* [GKE on Azure(Anthos on Azure)](./inspection-type.md#gke-on-azureanthos-on-azure)
* [GDCV for Baremetal(GKE on Baremetal, Anthos on Baremetal)](./inspection-type.md#gdcv-for-baremetalgke-on-baremetal-anthos-on-baremetal)
* [GDCV for VMWare(GKE on VMWare, Anthos on VMWare)](./inspection-type.md#gdcv-for-vmwaregke-on-vmware-anthos-on-vmware)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/nodelog-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/container-parser -->
## Kubernetes container logs

Gather stdout/stderr logs of containers on the cluster to visualize them on the timeline under an associated Pod. Log volume can be huge when the cluster has many Pods.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/container-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/container-parser -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Namespaces(Container logs)](./forms.md#namespacescontainer-logs)|The namespace of Pods to gather container logs. Specify `@managed` to gather logs of system components.|
|[Pod names(Container logs)](./forms.md#pod-namescontainer-logs)|The substring of Pod name to gather container logs. Specify `@any` to gather logs of all pods.|
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Cluster name](./forms.md#cluster-name)|The cluster name to gather logs.|
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/container-parser -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/container-parser -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
|![fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)[Container timeline](./relationships.md#container-timeline)|container|A timline of a container included in the parent timeline of a Pod|

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/container-parser -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/feature/container-parser -->
### Target log type

**![fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)k8s_container**

Sample query:

```ada
resource.type="k8s_container"
resource.labels.cluster_name="gcp-cluster-name"
resource.labels.namespace_name=("default")
-resource.labels.pod_name:("nginx-" OR "redis")
```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/feature/container-parser -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/container-parser -->
### Inspection types

This feature is supported in the following inspection types.

* [Google Kubernetes Engine](./inspection-type.md#google-kubernetes-engine)
* [Cloud Composer](./inspection-type.md#cloud-composer)
* [GKE on AWS(Anthos on AWS)](./inspection-type.md#gke-on-awsanthos-on-aws)
* [GKE on Azure(Anthos on Azure)](./inspection-type.md#gke-on-azureanthos-on-azure)
* [GDCV for Baremetal(GKE on Baremetal, Anthos on Baremetal)](./inspection-type.md#gdcv-for-baremetalgke-on-baremetal-anthos-on-baremetal)
* [GDCV for VMWare(GKE on VMWare, Anthos on VMWare)](./inspection-type.md#gdcv-for-vmwaregke-on-vmware-anthos-on-vmware)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/container-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/gke-audit-parser -->
## GKE Audit logs

Gather GKE audit log to show creation/upgrade/deletion of logs cluster/nodepool

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/gke-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/gke-audit-parser -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Cluster name](./forms.md#cluster-name)|The cluster name to gather logs.|
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/gke-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/gke-audit-parser -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
|![CCCCCC](https://placehold.co/15x15/CCCCCC/CCCCCC.png)[The default resource timeline](./relationships.md#the-default-resource-timeline)|resource|A default timeline recording the history of Kubernetes resources|
|![000000](https://placehold.co/15x15/000000/000000.png)[Operation timeline](./relationships.md#operation-timeline)|operation|A timeline showing long running operation status related to the parent resource|

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/gke-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/feature/gke-audit-parser -->
### Target log type

**![AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)gke_audit**

Sample query:

```ada
resource.type=("gke_cluster" OR "gke_nodepool")
logName="projects/gcp-project-id/logs/cloudaudit.googleapis.com%2Factivity"
resource.labels.cluster_name="gcp-cluster-name"
```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/feature/gke-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/gke-audit-parser -->
### Inspection types

This feature is supported in the following inspection types.

* [Google Kubernetes Engine](./inspection-type.md#google-kubernetes-engine)
* [Cloud Composer](./inspection-type.md#cloud-composer)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/gke-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/compute-api-parser -->
## Compute API Logs

Gather Compute API audit logs to show the timings of the provisioning of resources(e.g creating/deleting GCE VM,mounting Persistent Disk...etc) on associated timelines.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/compute-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/compute-api-parser -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Kind](./forms.md#kind)|The kinds of resources to gather logs. `@default` is a alias of set of kinds that frequently queried. Specify `@any` to query every kinds of resources|
|[Namespaces](./forms.md#namespaces)|The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.|
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Cluster name](./forms.md#cluster-name)|The cluster name to gather logs.|
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/compute-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/compute-api-parser -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
|![CCCCCC](https://placehold.co/15x15/CCCCCC/CCCCCC.png)[The default resource timeline](./relationships.md#the-default-resource-timeline)|resource|A default timeline recording the history of Kubernetes resources|
|![000000](https://placehold.co/15x15/000000/000000.png)[Operation timeline](./relationships.md#operation-timeline)|operation|A timeline showing long running operation status related to the parent resource|

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/compute-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/feature/compute-api-parser -->
### Target log type

**![FFCC33](https://placehold.co/15x15/FFCC33/FFCC33.png)compute_api**

Sample query:

```ada
resource.type="gce_instance"
-protoPayload.methodName:("list" OR "get" OR "watch")
protoPayload.resourceName:(instances/gke-test-cluster-node-1 OR instances/gke-test-cluster-node-2)

```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/feature/compute-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-indirect-query-header-cloud.google.com/feature/compute-api-parser -->
### Dependent queries

Following log queries are used with this feature.

* ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit
<!-- END GENERATED PART: feature-element-depending-indirect-query-header-cloud.google.com/feature/compute-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/compute-api-parser -->
### Inspection types

This feature is supported in the following inspection types.

* [Google Kubernetes Engine](./inspection-type.md#google-kubernetes-engine)
* [Cloud Composer](./inspection-type.md#cloud-composer)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/compute-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/network-api-parser -->
## GCE Network Logs

Gather GCE Network API logs to visualize statuses of Network Endpoint Groups(NEG)

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/network-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/network-api-parser -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Kind](./forms.md#kind)|The kinds of resources to gather logs. `@default` is a alias of set of kinds that frequently queried. Specify `@any` to query every kinds of resources|
|[Namespaces](./forms.md#namespaces)|The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.|
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Cluster name](./forms.md#cluster-name)|The cluster name to gather logs.|
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/network-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/network-api-parser -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
|![000000](https://placehold.co/15x15/000000/000000.png)[Operation timeline](./relationships.md#operation-timeline)|operation|A timeline showing long running operation status related to the parent resource|
|![A52A2A](https://placehold.co/15x15/A52A2A/A52A2A.png)[Network Endpoint Group timeline](./relationships.md#network-endpoint-group-timeline)|neg||

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/network-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/feature/network-api-parser -->
### Target log type

**![33CCFF](https://placehold.co/15x15/33CCFF/33CCFF.png)network_api**

Sample query:

```ada
resource.type="gce_network"
-protoPayload.methodName:("list" OR "get" OR "watch")
protoPayload.resourceName:(networkEndpointGroups/neg-id-1 OR networkEndpointGroups/neg-id-2)

```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/feature/network-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-indirect-query-header-cloud.google.com/feature/network-api-parser -->
### Dependent queries

Following log queries are used with this feature.

* ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit
<!-- END GENERATED PART: feature-element-depending-indirect-query-header-cloud.google.com/feature/network-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/network-api-parser -->
### Inspection types

This feature is supported in the following inspection types.

* [Google Kubernetes Engine](./inspection-type.md#google-kubernetes-engine)
* [Cloud Composer](./inspection-type.md#cloud-composer)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/network-api-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/multicloud-audit-parser -->
## MultiCloud API logs

Gather Anthos Multicloud audit log including cluster creation,deletion and upgrades.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/multicloud-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/multicloud-audit-parser -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Cluster name](./forms.md#cluster-name)|The cluster name to gather logs.|
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/multicloud-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/multicloud-audit-parser -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
|![000000](https://placehold.co/15x15/000000/000000.png)[Operation timeline](./relationships.md#operation-timeline)|operation|A timeline showing long running operation status related to the parent resource|

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/multicloud-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/feature/multicloud-audit-parser -->
### Target log type

**![AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)multicloud_api**

Sample query:

```ada
resource.type="audited_resource"
resource.labels.service="gkemulticloud.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete")
protoPayload.resourceName:"awsClusters/cluster-foo"

```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/feature/multicloud-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/multicloud-audit-parser -->
### Inspection types

This feature is supported in the following inspection types.

* [GKE on AWS(Anthos on AWS)](./inspection-type.md#gke-on-awsanthos-on-aws)
* [GKE on Azure(Anthos on Azure)](./inspection-type.md#gke-on-azureanthos-on-azure)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/multicloud-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/autoscaler-parser -->
## Autoscaler Logs

Gather logs related to cluster autoscaler behavior to show them on the timelines of resources related to the autoscaler decision.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/autoscaler-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/autoscaler-parser -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Cluster name](./forms.md#cluster-name)|The cluster name to gather logs.|
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/autoscaler-parser -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/autoscaler-parser -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
|![CCCCCC](https://placehold.co/15x15/CCCCCC/CCCCCC.png)[The default resource timeline](./relationships.md#the-default-resource-timeline)|resource|A default timeline recording the history of Kubernetes resources|
|![FF5555](https://placehold.co/15x15/FF5555/FF5555.png)[Managed instance group timeline](./relationships.md#managed-instance-group-timeline)|mig||

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/autoscaler-parser -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/feature/autoscaler-parser -->
### Target log type

**![FF5555](https://placehold.co/15x15/FF5555/FF5555.png)autoscaler**

Sample query:

```ada
resource.type="k8s_cluster"
resource.labels.project_id="gcp-project-id"
resource.labels.cluster_name="gcp-cluster-name"
-jsonPayload.status: ""
logName="projects/gcp-project-id/logs/container.googleapis.com%2Fcluster-autoscaler-visibility"
```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/feature/autoscaler-parser -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/autoscaler-parser -->
### Inspection types

This feature is supported in the following inspection types.

* [Google Kubernetes Engine](./inspection-type.md#google-kubernetes-engine)
* [Cloud Composer](./inspection-type.md#cloud-composer)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/autoscaler-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/onprem-audit-parser -->
## OnPrem API logs

Gather Anthos OnPrem audit log including cluster creation,deletion,enroll,unenroll and upgrades.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/onprem-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/onprem-audit-parser -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Cluster name](./forms.md#cluster-name)|The cluster name to gather logs.|
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/onprem-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/onprem-audit-parser -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
|![000000](https://placehold.co/15x15/000000/000000.png)[Operation timeline](./relationships.md#operation-timeline)|operation|A timeline showing long running operation status related to the parent resource|

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/onprem-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/feature/onprem-audit-parser -->
### Target log type

**![AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)onprem_api**

Sample query:

```ada
resource.type="audited_resource"
resource.labels.service="gkeonprem.googleapis.com"
resource.labels.method:("Update" OR "Create" OR "Delete" OR "Enroll" OR "Unenroll")
protoPayload.resourceName:"baremetalClusters/my-cluster"

```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/feature/onprem-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/onprem-audit-parser -->
### Inspection types

This feature is supported in the following inspection types.

* [GDCV for Baremetal(GKE on Baremetal, Anthos on Baremetal)](./inspection-type.md#gdcv-for-baremetalgke-on-baremetal-anthos-on-baremetal)
* [GDCV for VMWare(GKE on VMWare, Anthos on VMWare)](./inspection-type.md#gdcv-for-vmwaregke-on-vmware-anthos-on-vmware)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/onprem-audit-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/controlplane-component-parser -->
## Kubernetes Control plane component logs

Gather Kubernetes control plane component(e.g kube-scheduler, kube-controller-manager,api-server) logs

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/controlplane-component-parser -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/controlplane-component-parser -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Control plane component names](./forms.md#control-plane-component-names)||
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Cluster name](./forms.md#cluster-name)|The cluster name to gather logs.|
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/controlplane-component-parser -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/controlplane-component-parser -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
|![CCCCCC](https://placehold.co/15x15/CCCCCC/CCCCCC.png)[The default resource timeline](./relationships.md#the-default-resource-timeline)|resource|A default timeline recording the history of Kubernetes resources|
|![FF5555](https://placehold.co/15x15/FF5555/FF5555.png)[Control plane component timeline](./relationships.md#control-plane-component-timeline)|controlplane||

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/controlplane-component-parser -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/feature/controlplane-component-parser -->
### Target log type

**![FF3333](https://placehold.co/15x15/FF3333/FF3333.png)control_plane_component**

Sample query:

```ada
resource.type="k8s_control_plane_component"
resource.labels.cluster_name="gcp-cluster-name"
resource.labels.project_id="gcp-project-id"
-sourceLocation.file="httplog.go"
-- No component name filter
```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/feature/controlplane-component-parser -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/controlplane-component-parser -->
### Inspection types

This feature is supported in the following inspection types.

* [Google Kubernetes Engine](./inspection-type.md#google-kubernetes-engine)
* [Cloud Composer](./inspection-type.md#cloud-composer)
* [GKE on AWS(Anthos on AWS)](./inspection-type.md#gke-on-awsanthos-on-aws)
* [GKE on Azure(Anthos on Azure)](./inspection-type.md#gke-on-azureanthos-on-azure)
* [GDCV for Baremetal(GKE on Baremetal, Anthos on Baremetal)](./inspection-type.md#gdcv-for-baremetalgke-on-baremetal-anthos-on-baremetal)
* [GDCV for VMWare(GKE on VMWare, Anthos on VMWare)](./inspection-type.md#gdcv-for-vmwaregke-on-vmware-anthos-on-vmware)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/controlplane-component-parser -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/feature/serialport -->
## Node serial port logs

Gather serialport logs of GKE nodes. This helps detailed investigation on VM bootstrapping issue on GKE node.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/feature/serialport -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/serialport -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Kind](./forms.md#kind)|The kinds of resources to gather logs. `@default` is a alias of set of kinds that frequently queried. Specify `@any` to query every kinds of resources|
|[Namespaces](./forms.md#namespaces)|The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources.|
|[Node names](./forms.md#node-names)|A space-separated list of node name substrings used to collect node-related logs. If left blank, KHI gathers logs from all nodes in the cluster.|
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Cluster name](./forms.md#cluster-name)|The cluster name to gather logs.|
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/feature/serialport -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/serialport -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
|![333333](https://placehold.co/15x15/333333/333333.png)[Serialport log timeline](./relationships.md#serialport-log-timeline)|serialport||

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/feature/serialport -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/feature/serialport -->
### Target log type

**![333333](https://placehold.co/15x15/333333/333333.png)serial_port**

Sample query:

```ada
LOG_ID("serialconsole.googleapis.com%2Fserial_port_1_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_2_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_3_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_debug_output")

labels."compute.googleapis.com/resource_name"=("gke-test-cluster-node-1" OR "gke-test-cluster-node-2")

-- No node name substring filters are specified.
```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/feature/serialport -->
<!-- BEGIN GENERATED PART: feature-element-depending-indirect-query-header-cloud.google.com/feature/serialport -->
### Dependent queries

Following log queries are used with this feature.

* ![000000](https://placehold.co/15x15/000000/000000.png)k8s_audit
<!-- END GENERATED PART: feature-element-depending-indirect-query-header-cloud.google.com/feature/serialport -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/serialport -->
### Inspection types

This feature is supported in the following inspection types.

* [Google Kubernetes Engine](./inspection-type.md#google-kubernetes-engine)
* [Cloud Composer](./inspection-type.md#cloud-composer)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/feature/serialport -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/composer/scheduler -->
## (Alpha) Composer / Airflow Scheduler

Airflow Scheduler logs contain information related to the scheduling of TaskInstances, making it an ideal source for understanding the lifecycle of TaskInstances.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/composer/scheduler -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/composer/scheduler -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Location](./forms.md#location)||
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Composer Environment Name](./forms.md#composer-environment-name)||
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/composer/scheduler -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/composer/scheduler -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/composer/scheduler -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/composer/scheduler -->
### Target log type

**![88AA55](https://placehold.co/15x15/88AA55/88AA55.png)composer_environment**

Sample query:

```ada
resource.type="cloud_composer_environment"
resource.labels.environment_name="sample-composer-environment"
log_name=projects/test-project/logs/airflow-scheduler
```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/composer/scheduler -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/composer/scheduler -->
### Inspection types

This feature is supported in the following inspection types.

* [Cloud Composer](./inspection-type.md#cloud-composer)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/composer/scheduler -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/composer/worker -->
## (Alpha) Cloud Composer / Airflow Worker

Airflow Worker logs contain information related to the execution of TaskInstances. By including these logs, you can gain insights into where and how each TaskInstance was executed.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/composer/worker -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/composer/worker -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Location](./forms.md#location)||
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Composer Environment Name](./forms.md#composer-environment-name)||
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/composer/worker -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/composer/worker -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/composer/worker -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/composer/worker -->
### Target log type

**![88AA55](https://placehold.co/15x15/88AA55/88AA55.png)composer_environment**

Sample query:

```ada
resource.type="cloud_composer_environment"
resource.labels.environment_name="sample-composer-environment"
log_name=projects/test-project/logs/airflow-worker
```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/composer/worker -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/composer/worker -->
### Inspection types

This feature is supported in the following inspection types.

* [Cloud Composer](./inspection-type.md#cloud-composer)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/composer/worker -->
<!-- BEGIN GENERATED PART: feature-element-header-cloud.google.com/composer/dagprocessor -->
## (Alpha) Composer / Airflow DagProcessorManager

The DagProcessorManager logs contain information for investigating the number of DAGs included in each Python file and the time it took to parse them. You can get information about missing DAGs and load.

<!-- END GENERATED PART: feature-element-header-cloud.google.com/composer/dagprocessor -->
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-cloud.google.com/composer/dagprocessor -->
### Parameters

|Parameter name|Description|
|:-:|---|
|[Location](./forms.md#location)||
|[Project ID](./forms.md#project-id)|The project ID containing logs of the cluster to query|
|[Composer Environment Name](./forms.md#composer-environment-name)||
|[End time](./forms.md#end-time)|The endtime of the time range to gather logs.  The start time of the time range will be this endtime subtracted with the duration parameter.|
|[Duration](./forms.md#duration)|The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)|
<!-- END GENERATED PART: feature-element-depending-form-header-cloud.google.com/composer/dagprocessor -->
<!-- BEGIN GENERATED PART: feature-element-output-timelines-cloud.google.com/composer/dagprocessor -->
### Output timelines

This feature can generates following timeline relationship of timelines.

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|

<!-- END GENERATED PART: feature-element-output-timelines-cloud.google.com/composer/dagprocessor -->
<!-- BEGIN GENERATED PART: feature-element-target-query-cloud.google.com/composer/dagprocessor -->
### Target log type

**![88AA55](https://placehold.co/15x15/88AA55/88AA55.png)composer_environment**

Sample query:

```ada
resource.type="cloud_composer_environment"
resource.labels.environment_name="sample-composer-environment"
log_name=projects/test-project/logs/dag-processor-manager
```

<!-- END GENERATED PART: feature-element-target-query-cloud.google.com/composer/dagprocessor -->
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-cloud.google.com/composer/dagprocessor -->
### Inspection types

This feature is supported in the following inspection types.

* [Cloud Composer](./inspection-type.md#cloud-composer)
<!-- END GENERATED PART: feature-element-available-inspection-type-cloud.google.com/composer/dagprocessor -->
