<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipChild -->
## [The default resource timeline](#RelationshipChild)
<!-- END GENERATED PART: relationship-element-header-RelationshipChild -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipChild-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipChild-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipChild-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#0000FF](https://placehold.co/15x15/0000FF/0000FF.png)Resource is existing|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|This state indicates the resource exits at the time|
|![#CC0000](https://placehold.co/15x15/CC0000/CC0000.png)Resource is deleted|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|This state indicates the resource is deleted at the time.|
|![#CC5500](https://placehold.co/15x15/CC5500/CC5500.png)Resource is under deleting with graceful period|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit|This state indicates the resource is being deleted with grace period at the time.|

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

<!-- END GENERATED PART: relationship-element-header-RelationshipChild-events-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipResourceCondition -->
## [![#4c29e8](https://placehold.co/15x15/4c29e8/4c29e8.png) condition - Status condition field timeline](#RelationshipResourceCondition)
<!-- END GENERATED PART: relationship-element-header-RelationshipResourceCondition -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipResourceCondition-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipResourceCondition-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipResourceCondition-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#004400](https://placehold.co/15x15/004400/004400.png)State is 'True'|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit||
|![#EE4400](https://placehold.co/15x15/EE4400/EE4400.png)State is 'False'|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit||
|![#663366](https://placehold.co/15x15/663366/663366.png)State is 'Unknown'|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit||

<!-- END GENERATED PART: relationship-element-header-RelationshipResourceCondition-revisions-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipOperation -->
## [![#000000](https://placehold.co/15x15/000000/000000.png) operation - Operation timeline](#RelationshipOperation)
<!-- END GENERATED PART: relationship-element-header-RelationshipOperation -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipOperation-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipOperation-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipOperation-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#004400](https://placehold.co/15x15/004400/004400.png)Processing operation|![#FFCC33](https://placehold.co/15x15/FFCC33/FFCC33.png)compute_api||
|![#333333](https://placehold.co/15x15/333333/333333.png)Operation is finished|![#FFCC33](https://placehold.co/15x15/FFCC33/FFCC33.png)compute_api||

<!-- END GENERATED PART: relationship-element-header-RelationshipOperation-revisions-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipEndpointSlice -->
## [![#008000](https://placehold.co/15x15/008000/008000.png) endpointslice - Endpoint serving state timeline](#RelationshipEndpointSlice)
<!-- END GENERATED PART: relationship-element-header-RelationshipEndpointSlice -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipEndpointSlice-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipEndpointSlice-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipEndpointSlice-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#004400](https://placehold.co/15x15/004400/004400.png)Endpoint is ready|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit||
|![#EE4400](https://placehold.co/15x15/EE4400/EE4400.png)Endpoint is not ready|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit||
|![#fed700](https://placehold.co/15x15/fed700/fed700.png)Endpoint is being terminated|![#000000](https://placehold.co/15x15/000000/000000.png)k8s_audit||

<!-- END GENERATED PART: relationship-element-header-RelationshipEndpointSlice-revisions-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipContainer -->
## [![#fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png) container - Container timeline](#RelationshipContainer)
<!-- END GENERATED PART: relationship-element-header-RelationshipContainer -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipContainer-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipContainer-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipContainer-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#997700](https://placehold.co/15x15/997700/997700.png)Waiting for starting container|![#fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)k8s_container||
|![#EE4400](https://placehold.co/15x15/EE4400/EE4400.png)Container is not ready|![#fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)k8s_container||
|![#007700](https://placehold.co/15x15/007700/007700.png)Container is ready|![#fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)k8s_container||
|![#113333](https://placehold.co/15x15/113333/113333.png)Container exited with healthy exit code|![#fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)k8s_container||
|![#331111](https://placehold.co/15x15/331111/331111.png)Container exited with errornous exit code|![#fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)k8s_container||

<!-- END GENERATED PART: relationship-element-header-RelationshipContainer-revisions-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipContainer-events-header -->
### Events

This timeline can have the following events.
<!-- END GENERATED PART: relationship-element-header-RelationshipContainer-events-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipContainer-events-table -->
|Source log|Description|
|---|---|
|![#fe9bab](https://placehold.co/15x15/fe9bab/fe9bab.png)k8s_container||
|![#0077CC](https://placehold.co/15x15/0077CC/0077CC.png)k8s_node||

<!-- END GENERATED PART: relationship-element-header-RelationshipContainer-events-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNodeComponent -->
## [![#0077CC](https://placehold.co/15x15/0077CC/0077CC.png) node-component - Node component timeline](#RelationshipNodeComponent)
<!-- END GENERATED PART: relationship-element-header-RelationshipNodeComponent -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNodeComponent-events-header -->
### Events

This timeline can have the following events.
<!-- END GENERATED PART: relationship-element-header-RelationshipNodeComponent-events-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNodeComponent-events-table -->
|Source log|Description|
|---|---|
|![#0077CC](https://placehold.co/15x15/0077CC/0077CC.png)k8s_node||

<!-- END GENERATED PART: relationship-element-header-RelationshipNodeComponent-events-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipOwnerReference -->
## [![#33DD88](https://placehold.co/15x15/33DD88/33DD88.png) owns - Owning children timeline](#RelationshipOwnerReference)
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
## [![#FF8855](https://placehold.co/15x15/FF8855/FF8855.png) binds - Pod binding timeline](#RelationshipPodBinding)
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
## [![#A52A2A](https://placehold.co/15x15/A52A2A/A52A2A.png) neg - NEG timeline](#RelationshipNetworkEndpointGroup)
<!-- END GENERATED PART: relationship-element-header-RelationshipNetworkEndpointGroup -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNetworkEndpointGroup-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-RelationshipNetworkEndpointGroup-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipNetworkEndpointGroup-revisions-table -->
|State|Source log|Description|
|---|---|---|
|![#004400](https://placehold.co/15x15/004400/004400.png)State is 'True'|![#33CCFF](https://placehold.co/15x15/33CCFF/33CCFF.png)network_api||
|![#EE4400](https://placehold.co/15x15/EE4400/EE4400.png)State is 'False'|![#33CCFF](https://placehold.co/15x15/33CCFF/33CCFF.png)network_api||

<!-- END GENERATED PART: relationship-element-header-RelationshipNetworkEndpointGroup-revisions-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipManagedInstanceGroup -->
## [![#FF5555](https://placehold.co/15x15/FF5555/FF5555.png) mig - Managed instance group timeline](#RelationshipManagedInstanceGroup)
<!-- END GENERATED PART: relationship-element-header-RelationshipManagedInstanceGroup -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipManagedInstanceGroup-events-header -->
### Events

This timeline can have the following events.
<!-- END GENERATED PART: relationship-element-header-RelationshipManagedInstanceGroup-events-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipManagedInstanceGroup-events-table -->
|Source log|Description|
|---|---|
|![#FF5555](https://placehold.co/15x15/FF5555/FF5555.png)autoscaler||

<!-- END GENERATED PART: relationship-element-header-RelationshipManagedInstanceGroup-events-table -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipControlPlaneComponent -->
## [![#FF5555](https://placehold.co/15x15/FF5555/FF5555.png) controlplane - Control plane component timeline](#RelationshipControlPlaneComponent)
<!-- END GENERATED PART: relationship-element-header-RelationshipControlPlaneComponent -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipControlPlaneComponent-events-header -->
### Events

This timeline can have the following events.
<!-- END GENERATED PART: relationship-element-header-RelationshipControlPlaneComponent-events-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-RelationshipControlPlaneComponent-events-table -->
|Source log|Description|
|---|---|
|![#FF3333](https://placehold.co/15x15/FF3333/FF3333.png)control_plane_component||

<!-- END GENERATED PART: relationship-element-header-RelationshipControlPlaneComponent-events-table -->
