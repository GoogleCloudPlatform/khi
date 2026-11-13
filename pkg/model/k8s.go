// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"fmt"
	"time"
)

// K8sResourceContainingStatus represents a Kubernetes resource with a `.status` field.
type K8sResourceContainingStatus struct {
	// Status holds the resource status section.
	Status *K8sResourceStatus `yaml:"status"`
}

// K8sResourceStatus represents the `.status` section of a Kubernetes resource.
type K8sResourceStatus struct {
	// Conditions lists the status conditions reported for the resource.
	Conditions []*K8sResourceStatusCondition `yaml:"conditions"`
}

// K8sDeleteRequest represents the payload of a Kubernetes delete request.
type K8sDeleteRequest struct {
	// Preconditions specifies constraints that must hold before deletion proceeds.
	Preconditions *K8sDeleteRequestPreconditions `yaml:"preconditions"`
}

// K8sDeleteRequestPreconditions holds preconditions for a Kubernetes delete request.
type K8sDeleteRequestPreconditions struct {
	// Uid specifies the expected UID of the resource being deleted.
	Uid string `yaml:"uid"`
}

// K8sResourceList represents a list response of Kubernetes resources.
type K8sResourceList struct {
	// Items contains the resources included in the list.
	Items []*K8sResourceWithMetadata `yaml:"items"`
}

// K8sResourceWithMetadata represents a Kubernetes resource containing object metadata.
type K8sResourceWithMetadata struct {
	// Metadata holds the standard Kubernetes object metadata.
	Metadata *K8sObjectMeta `yaml:"metadata"`
}

// OwnerReference identifies an owning Kubernetes object in metadata.
type OwnerReference struct {
	// APIVersion is the API version of the referent.
	APIVersion string `yaml:"apiVersion"`
	// Kind is the kind of the referent.
	Kind string `yaml:"kind"`
	// Name is the name of the referent.
	Name string `yaml:"name"`
	// UID is the UID of the referent.
	UID string `yaml:"uid"`
}

// K8sObjectMeta represents the standard Kubernetes object metadata fields.
type K8sObjectMeta struct {
	// Name is the name of the resource within its namespace.
	Name string `yaml:"name"`
	// Namespace is the namespace of the resource, if namespaced.
	Namespace string `yaml:"namespace"`
	// GenerateName is the prefix used to generate a unique name when Name is omitted.
	GenerateName string `yaml:"generateName"`
	// UID is the unique identifier assigned to the resource by Kubernetes.
	UID string `yaml:"uid"`
	// ResourceVersion identifies the server version of the resource.
	ResourceVersion string `yaml:"resourceVersion"`
	// Labels holds key-value labels attached to the resource.
	Labels map[string]string `yaml:"labels"`
	// Annotations holds key-value annotations attached to the resource.
	Annotations map[string]string `yaml:"annotations"`
	// OwnerReferences lists objects that own this resource.
	OwnerReferences []OwnerReference `yaml:"ownerReferences"`
	// Finalizers lists pending finalizer identifiers blocking resource deletion.
	Finalizers []string `yaml:"finalizers"`
}

// K8sResourceStatusCondition represents a single condition entry in a Kubernetes resource status.
type K8sResourceStatusCondition struct {
	// Type is the condition type name.
	Type string `yaml:"type"`
	// LastTransitionTime is the timestamp of the last status transition.
	LastTransitionTime string `yaml:"lastTransitionTime"`
	// LastHeartbeatTime is the timestamp of the last heartbeat update.
	LastHeartbeatTime string `yaml:"lastHeartbeatTime"`
	// LastProbeTime is the timestamp of the last probe check.
	LastProbeTime string `yaml:"lastProbeTime"`
	// Message provides a human-readable explanation for the last transition.
	Message string `yaml:"message"`
	// Status indicates the condition state (True, False, or Unknown).
	Status string `yaml:"status"`
	// Reason gives a programmatic reason identifier for the last transition.
	Reason string `yaml:"reason"`
}

// K8sTargetRef identifies a Kubernetes object referenced by an EndpointSlice endpoint.
type K8sTargetRef struct {
	// Kind is the kind of the referenced object.
	Kind string `yaml:"kind"`
	// Name is the name of the referenced object.
	Name string `yaml:"name"`
	// Namespace is the namespace of the referenced object.
	Namespace string `yaml:"namespace"`
	// Uid is the UID of the referenced object.
	Uid string `yaml:"uid"`
}

// EndpointSliceEndpointConditions represents the readiness and lifecycle flags of an EndpointSlice endpoint.
type EndpointSliceEndpointConditions struct {
	// Ready indicates whether the endpoint is ready to receive traffic.
	Ready bool `yaml:"ready"`
	// Serving indicates whether the endpoint is serving traffic regardless of termination state.
	Serving bool `yaml:"serving"`
	// Terminating indicates whether the endpoint is terminating.
	Terminating bool `yaml:"terminating"`
}

// EndpointSliceEndpoint represents a single network endpoint in an EndpointSlice.
type EndpointSliceEndpoint struct {
	// Addresses lists the IP addresses of the endpoint.
	Addresses []string `yaml:"addresses"`
	// Conditions holds the readiness and lifecycle conditions of the endpoint.
	Conditions *EndpointSliceEndpointConditions `yaml:"conditions"`
	// TargetRef identifies the Kubernetes object backing this endpoint.
	TargetRef *K8sTargetRef `yaml:"targetRef"`
	// NodeName is the name of the node hosting this endpoint.
	NodeName string `yaml:"nodeName"`
	// HostName is the hostname of the endpoint (may always be empty in GKE).
	HostName string `yaml:"hostName"`
}

// EndpointSlice represents a Kubernetes EndpointSlice resource.
type EndpointSlice struct {
	// Endpoints lists the endpoints belonging to this EndpointSlice.
	Endpoints []*EndpointSliceEndpoint `yaml:"endpoints"`
	// Metadata holds the EndpointSlice metadata.
	Metadata *K8sObjectMeta `yaml:"metadata"`
	// the other fields are not used.
}

// PodMetadata holds minimal Kubernetes Pod metadata.
type PodMetadata struct {
	// Uid is the UID of the Pod.
	Uid string `yaml:"uid"`
}

// Pod represents a minimal Kubernetes Pod resource.
type Pod struct {
	// Metadata holds the Pod metadata.
	Metadata *PodMetadata `yaml:"metadata"`
	// Status holds the Pod status.
	Status *PodStatus `yaml:"status"`
	// the other fields are not used.
}

// PodIP represents an IP address assigned to a Pod.
type PodIP struct {
	// IP is the IP address string.
	IP string `yaml:"ip"`
}

// PodStatus represents the status IP fields of a Kubernetes Pod.
type PodStatus struct {
	// HostIP is the IP address of the node hosting the Pod.
	HostIP string `yaml:"hostIP"`
	// PodIP is the primary IP address allocated to the Pod.
	PodIP string `yaml:"podIP"`
	// PodIPs lists all IP addresses allocated to the Pod.
	PodIPs []*PodIP `yaml:"podIPs"`
}

// ToMap converts non-empty fields of the condition into a string-keyed map.
func (c *K8sResourceStatusCondition) ToMap() map[string]any {
	result := map[string]any{}
	if c.Type != "" {
		result["type"] = c.Type
	}
	if c.LastTransitionTime != "" {
		result["lastTransitionTime"] = c.LastTransitionTime
	}
	if c.LastHeartbeatTime != "" {
		result["lastHeartbeatTime"] = c.LastHeartbeatTime
	}
	if c.LastProbeTime != "" {
		result["lastProbeTime"] = c.LastProbeTime
	}
	if c.Message != "" {
		result["message"] = c.Message
	}
	if c.Status != "" {
		result["status"] = c.Status
	}
	if c.Reason != "" {
		result["reason"] = c.Reason
	}
	return result
}

// ProbeLikeTime parses and returns LastProbeTime, falling back to LastHeartbeatTime when LastProbeTime is empty.
func (c *K8sResourceStatusCondition) ProbeLikeTime() (time.Time, error) {
	if c.LastProbeTime != "" {
		return time.Parse(time.RFC3339, c.LastProbeTime)
	}
	if c.LastHeartbeatTime != "" {
		return time.Parse(time.RFC3339, c.LastHeartbeatTime)
	}
	return time.Time{}, fmt.Errorf("lastProbeTime and lastHeartbeatTime are both empty")
}

// SameWith reports whether all condition flags match those of other.
func (c *EndpointSliceEndpointConditions) SameWith(other *EndpointSliceEndpointConditions) bool {
	return c.Ready == other.Ready && c.Serving == other.Serving && c.Terminating == other.Terminating
}
