# Relationships

> [!WARNING]
> 🚧 This reference document is under construction. 🚧

KHI timelines are basically placed in the order of `Kind` -> `Namespace` -> `Resource name` -> `Subresource name`.
The relationship between its parent and children is usually interpretted as the order of its hierarchy, but some subresources are not actual kubernetes resources and it's associated with the parent timeline for convenience. Each timeline color meanings and type of logs put on them are different by this relationship.

<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipChild -->
## ![#CCCCCC](https://placehold.co/15x15/CCCCCC/CCCCCC.png)The default resource timeline
<!-- END GENERATED PART: relationship-element-header-RelationshipChild -->

![](./images/reference/default-timeline.png)

<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipChild-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipChild-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipChild-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#997700](https://placehold.co/15x15/997700/997700.png)Resource may be existing|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|This state indicates the resource exists at the time, but this existence is inferred from the other logs later. The detailed resource information is not available.|
|![#0000FF](https://placehold.co/15x15/0000FF/0000FF.png)Resource is existing|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|This state indicates the resource exists at the time|
|![#CC0000](https://placehold.co/15x15/CC0000/CC0000.png)Resource is deleted|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|This state indicates the resource is deleted at the time.|
|![#CC5500](https://placehold.co/15x15/CC5500/CC5500.png)Resource is under deleting with graceful period|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|This state indicates the resource is being deleted with grace period at the time.|
|![#4444ff](https://placehold.co/15x15/4444ff/4444ff.png)Resource is being provisioned|![#AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)gke_audit|This state indicates the resource is being provisioned. Currently this state is only used for cluster/nodepool status only.|

<!-- END GENERATED PART: relationship-element-header-RelationshipChild-revisions-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipChild-events-header -->
### Events

This timeline can have the following events.
<!-- END GENERATED PART: relationship-element-header-RelationshipChild-events-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipChild-events-table -->
|Source log|Description|
|---|---|
|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|An event that related to a resource but not changing the resource. This is often an error log for an operation to the resource.|
|![#3fb549](https://placehold.co/15x15/3fb549/3fb549.png)k8s_event|An event that related to a resource|
|![#0077CC](https://placehold.co/15x15/0077CC/0077CC.png)k8s_node|An event that related to a node resource|
|![#FFCC33](https://placehold.co/15x15/FFCC33/FFCC33.png)compute_api|An event that related to a compute resource|
|![#FF3333](https://placehold.co/15x15/FF3333/FF3333.png)control_plane_component|A log related to the timeline resource related to control plane component|
|![#FF5555](https://placehold.co/15x15/FF5555/FF5555.png)autoscaler|A log related to the Pod which triggered or prevented autoscaler|

<!-- END GENERATED PART: relationship-element-header-RelationshipChild-events-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipResourceCondition -->
## ![#4c29e8](https://placehold.co/15x15/4c29e8/4c29e8.png)Status condition field timeline

Timelines of this type have ![#4c29e8](https://placehold.co/15x15/4c29e8/4c29e8.png)`condition` chip on the left side of its timeline name.

<!-- END GENERATED PART: relationship-element-header-RelationshipResourceCondition -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipResourceCondition-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipResourceCondition-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipResourceCondition-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#004400](https://placehold.co/15x15/004400/004400.png)State is 'True'|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|The condition state is `True`. **This doesn't always mean a good status** (For example, `NetworkUnreachabel` condition on a Node means a bad condition when it is `True`)|
|![#EE4400](https://placehold.co/15x15/EE4400/EE4400.png)State is 'False'|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|The condition state is `False`. **This doesn't always mean a bad status** (For example, `NetworkUnreachabel` condition on a Node means a good condition when it is `False`)|
|![#663366](https://placehold.co/15x15/663366/663366.png)State is 'Unknown'|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|The condition state is `Unknown`|

<!-- END GENERATED PART: relationship-element-header-RelationshipResourceCondition-revisions-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipOperation -->
## ![#000000](https://placehold.co/15x15/000000/000000.png)Operation timeline

Timelines of this type have ![#000000](https://placehold.co/15x15/000000/000000.png)`operation` chip on the left side of its timeline name.

<!-- END GENERATED PART: relationship-element-header-RelationshipOperation -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipOperation-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipOperation-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipOperation-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#004400](https://placehold.co/15x15/004400/004400.png)Processing operation|![#FFCC33](https://placehold.co/15x15/FFCC33/FFCC33.png)compute_api|A long running operation is running|
|![#333333](https://placehold.co/15x15/333333/333333.png)Operation is finished|![#FFCC33](https://placehold.co/15x15/FFCC33/FFCC33.png)compute_api|An operation is finished at the time of left edge of this operation.|
|![#004400](https://placehold.co/15x15/004400/004400.png)Processing operation|![#AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)gke_audit|A long running operation is running|
|![#333333](https://placehold.co/15x15/333333/333333.png)Operation is finished|![#AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)gke_audit|An operation is finished at the time of left edge of this operation.|
|![#004400](https://placehold.co/15x15/004400/004400.png)Processing operation|![#33CCFF](https://placehold.co/15x15/33CCFF/33CCFF.png)network_api|A long running operation is running|
|![#333333](https://placehold.co/15x15/333333/333333.png)Operation is finished|![#33CCFF](https://placehold.co/15x15/33CCFF/33CCFF.png)network_api|An operation is finished at the time of left edge of this operation.|
|![#004400](https://placehold.co/15x15/004400/004400.png)Processing operation|![#AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)multicloud_api|A long running operation is running|
|![#333333](https://placehold.co/15x15/333333/333333.png)Operation is finished|![#AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)multicloud_api|An operation is finished at the time of left edge of this operation.|
|![#004400](https://placehold.co/15x15/004400/004400.png)Processing operation|![#AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)onprem_api|A long running operation is running|
|![#333333](https://placehold.co/15x15/333333/333333.png)Operation is finished|![#AA00FF](https://placehold.co/15x15/AA00FF/AA00FF.png)onprem_api|An operation is finished at the time of left edge of this operation.|

<!-- END GENERATED PART: relationship-element-header-RelationshipOperation-revisions-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipEndpointSlice -->
## ![#008000](https://placehold.co/15x15/008000/008000.png)Endpoint serving state timeline

Timelines of this type have ![#008000](https://placehold.co/15x15/008000/008000.png)`endpointslice` chip on the left side of its timeline name.

<!-- END GENERATED PART: relationship-element-header-RelationshipEndpointSlice -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipEndpointSlice-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipEndpointSlice-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipEndpointSlice-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#004400](https://placehold.co/15x15/004400/004400.png)Endpoint is ready|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|An endpoint associated with the parent resource is ready|
|![#EE4400](https://placehold.co/15x15/EE4400/EE4400.png)Endpoint is not ready|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|An endpoint associated with the parent resource is not ready. Traffic shouldn't be routed during this time.|
|![#fed700](https://placehold.co/15x15/fed700/fed700.png)Endpoint is being terminated|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|An endpoint associated with the parent resource is being terminated. New traffic shouldn't be routed to this endpoint during this time, but the endpoint can still have pending requests.|

<!-- END GENERATED PART: relationship-element-header-RelationshipEndpointSlice-revisions-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipContainer -->
## ![#fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)Container timeline

Timelines of this type have ![#fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)`container` chip on the left side of its timeline name.

<!-- END GENERATED PART: relationship-element-header-RelationshipContainer -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipContainer-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipContainer-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipContainer-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#997700](https://placehold.co/15x15/997700/997700.png)Waiting for starting container|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|The container is not started yet and waiting for something.(Example: Pulling images, mounting volumes ...etc)|
|![#EE4400](https://placehold.co/15x15/EE4400/EE4400.png)Container is not ready|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|The container is started but the readiness is not ready.|
|![#007700](https://placehold.co/15x15/007700/007700.png)Container is ready|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|The container is started and the readiness is ready|
|![#113333](https://placehold.co/15x15/113333/113333.png)Container exited with healthy exit code|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|The container is already terminated with successful exit code = 0|
|![#331111](https://placehold.co/15x15/331111/331111.png)Container exited with errornous exit code|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|The container is already terminated with errornous exit code != 0|

<!-- END GENERATED PART: relationship-element-header-RelationshipContainer-revisions-table -->

> [!TIP]
> Detailed container statuses are only available when your project enabled `DATA_WRITE` audit log for Kubernetes Engine API.
> Check [README](../../README.md) more details to configure `DATA_WRITE` audit log.

<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipContainer-events-header -->
### Events

This timeline can have the following events.
<!-- END GENERATED PART: relationship-element-header-RelationshipContainer-events-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipContainer-events-table -->
|Source log|Description|
|---|---|
|![#fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)k8s_container|A container log on stdout/etderr|
|![#0077CC](https://placehold.co/15x15/0077CC/0077CC.png)k8s_node|kubelet/containerd logs associated with the container|

<!-- END GENERATED PART: relationship-element-header-RelationshipContainer-events-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNodeComponent -->
## ![#0077CC](https://placehold.co/15x15/0077CC/0077CC.png)Node component timeline

Timelines of this type have ![#0077CC](https://placehold.co/15x15/0077CC/0077CC.png)`node-component` chip on the left side of its timeline name.

<!-- END GENERATED PART: relationship-element-header-RelationshipNodeComponent -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNodeComponent-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipNodeComponent-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNodeComponent-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#997700](https://placehold.co/15x15/997700/997700.png)Resource may be existing|![#0077CC](https://placehold.co/15x15/0077CC/0077CC.png)k8s_node|The component is infrred to be running because of the logs from it|
|![#0000FF](https://placehold.co/15x15/0000FF/0000FF.png)Resource is existing|![#0077CC](https://placehold.co/15x15/0077CC/0077CC.png)k8s_node|The component is running running. (Few node components supports this state because the parser knows logs on startup for specific components)|
|![#CC0000](https://placehold.co/15x15/CC0000/CC0000.png)Resource is deleted|![#0077CC](https://placehold.co/15x15/0077CC/0077CC.png)k8s_node|The component is no longer running. (Few node components supports this state because the parser knows logs on termination for specific components)|

<!-- END GENERATED PART: relationship-element-header-RelationshipNodeComponent-revisions-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNodeComponent-events-header -->
### Events

This timeline can have the following events.
<!-- END GENERATED PART: relationship-element-header-RelationshipNodeComponent-events-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNodeComponent-events-table -->
|Source log|Description|
|---|---|
|![#0077CC](https://placehold.co/15x15/0077CC/0077CC.png)k8s_node|A log from the component on the log|

<!-- END GENERATED PART: relationship-element-header-RelationshipNodeComponent-events-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipOwnerReference -->
## ![#33DD88](https://placehold.co/15x15/33DD88/33DD88.png)Owning children timeline

Timelines of this type have ![#33DD88](https://placehold.co/15x15/33DD88/33DD88.png)`owns` chip on the left side of its timeline name.

<!-- END GENERATED PART: relationship-element-header-RelationshipOwnerReference -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipOwnerReference-aliases-header -->
### Aliases

This timeline can have the following aliases.
<!-- END GENERATED PART: relationship-element-header-RelationshipOwnerReference-aliases-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipOwnerReference-aliases-table -->
|Aliased timeline|Source log|Description|
|---|---|---|
|![#CCCCCC](https://placehold.co/15x15/CCCCCC/CCCCCC.png)resource|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|This timeline shows the events and revisions of the owning resources.|

<!-- END GENERATED PART: relationship-element-header-RelationshipOwnerReference-aliases-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipPodBinding -->
## ![#FF8855](https://placehold.co/15x15/FF8855/FF8855.png)Pod binding timeline

Timelines of this type have ![#FF8855](https://placehold.co/15x15/FF8855/FF8855.png)`binds` chip on the left side of its timeline name.

<!-- END GENERATED PART: relationship-element-header-RelationshipPodBinding -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipPodBinding-aliases-header -->
### Aliases

This timeline can have the following aliases.
<!-- END GENERATED PART: relationship-element-header-RelationshipPodBinding-aliases-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipPodBinding-aliases-table -->
|Aliased timeline|Source log|Description|
|---|---|---|
|![#CCCCCC](https://placehold.co/15x15/CCCCCC/CCCCCC.png)resource|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|This timeline shows the binding subresources associated on a node|

<!-- END GENERATED PART: relationship-element-header-RelationshipPodBinding-aliases-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNetworkEndpointGroup -->
## ![#A52A2A](https://placehold.co/15x15/A52A2A/A52A2A.png)Network Endpoint Group timeline

Timelines of this type have ![#A52A2A](https://placehold.co/15x15/A52A2A/A52A2A.png)`neg` chip on the left side of its timeline name.

<!-- END GENERATED PART: relationship-element-header-RelationshipNetworkEndpointGroup -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNetworkEndpointGroup-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipNetworkEndpointGroup-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNetworkEndpointGroup-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#004400](https://placehold.co/15x15/004400/004400.png)State is 'True'|![#33CCFF](https://placehold.co/15x15/33CCFF/33CCFF.png)network_api|indicates the NEG is already attached to the Pod.|
|![#EE4400](https://placehold.co/15x15/EE4400/EE4400.png)State is 'False'|![#33CCFF](https://placehold.co/15x15/33CCFF/33CCFF.png)network_api|indicates the NEG is detached from the Pod|

<!-- END GENERATED PART: relationship-element-header-RelationshipNetworkEndpointGroup-revisions-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipManagedInstanceGroup -->
## ![#FF5555](https://placehold.co/15x15/FF5555/FF5555.png)Managed instance group timeline

Timelines of this type have ![#FF5555](https://placehold.co/15x15/FF5555/FF5555.png)`mig` chip on the left side of its timeline name.

<!-- END GENERATED PART: relationship-element-header-RelationshipManagedInstanceGroup -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipManagedInstanceGroup-events-header -->
### Events

This timeline can have the following events.
<!-- END GENERATED PART: relationship-element-header-RelationshipManagedInstanceGroup-events-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipManagedInstanceGroup-events-table -->
|Source log|Description|
|---|---|
|![#FF5555](https://placehold.co/15x15/FF5555/FF5555.png)autoscaler|Autoscaler logs associated to a MIG(e.g The mig was scaled up by the austoscaler)|

<!-- END GENERATED PART: relationship-element-header-RelationshipManagedInstanceGroup-events-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipControlPlaneComponent -->
## ![#FF5555](https://placehold.co/15x15/FF5555/FF5555.png)Control plane component timeline

Timelines of this type have ![#FF5555](https://placehold.co/15x15/FF5555/FF5555.png)`controlplane` chip on the left side of its timeline name.

<!-- END GENERATED PART: relationship-element-header-RelationshipControlPlaneComponent -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipControlPlaneComponent-events-header -->
### Events

This timeline can have the following events.
<!-- END GENERATED PART: relationship-element-header-RelationshipControlPlaneComponent-events-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipControlPlaneComponent-events-table -->
|Source log|Description|
|---|---|
|![#FF3333](https://placehold.co/15x15/FF3333/FF3333.png)control_plane_component|A log from the control plane component|

<!-- END GENERATED PART: relationship-element-header-RelationshipControlPlaneComponent-events-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipSerialPort -->
## ![#333333](https://placehold.co/15x15/333333/333333.png)Serialport log timeline

Timelines of this type have ![#333333](https://placehold.co/15x15/333333/333333.png)`serialport` chip on the left side of its timeline name.

<!-- END GENERATED PART: relationship-element-header-RelationshipSerialPort -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipSerialPort-events-header -->
### Events

This timeline can have the following events.
<!-- END GENERATED PART: relationship-element-header-RelationshipSerialPort-events-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipSerialPort-events-table -->
|Source log|Description|
|---|---|
|![#333333](https://placehold.co/15x15/333333/333333.png)serial_port|A serialport log from the node|

<!-- END GENERATED PART: relationship-element-header-RelationshipSerialPort-events-table -->
