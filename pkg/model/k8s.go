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

// A kubernetes resource `.status` field
type K8sResourceContainingStatus struct {
	Status *K8sResourceStatus `yaml:"status"`
}

type K8sResourceStatus struct {
	Conditions []*K8sResourceStatusCondition `yaml:"conditions"`
}

type K8sDeleteRequest struct {
	Preconditions *K8sDeleteRequestPreconditions `yaml:"preconditions"`
}

type K8sDeleteRequestPreconditions struct {
	Uid string `yaml:"uid"`
}

type K8sResourceList struct {
	Items []*K8sResourceWithMetadata `yaml:"items"`
}
type K8sResourceWithMetadata struct {
	Metadata *K8sObjectMeta `yaml:"metadata"`
}

type OwnerReference struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Name       string `yaml:"name"`
	UID        string `yaml:"uid"`
}

type K8sObjectMeta struct {
	Name            string            `yaml:"name"`
	Namespace       string            `yaml:"namespace"`
	GenerateName    string            `yaml:"generateName"`
	UID             string            `yaml:"uid"`
	ResourceVersion string            `yaml:"resourceVersion"`
	Labels          map[string]string `yaml:"labels"`
	Annotations     map[string]string `yaml:"annotations"`
	OwnerReferences []OwnerReference  `yaml:"ownerReferences"`
	Finalizers      []string          `yaml:"finalizers"`
}

type K8sResourceStatusCondition struct {
	Type               string `yaml:"type"`
	LastTransitionTime string `yaml:"lastTransitionTime"`
	LastHeartbeatTime  string `yaml:"lastHeartbeatTime"`
	LastProbeTime      string `yaml:"lastProbeTime"`
	Message            string `yaml:"message"`
	Status             string `yaml:"status"`
	Reason             string `yaml:"reason"`
}

type K8sTargetRef struct {
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Uid       string `yaml:"uid"`
}

type EndpointSliceEndpointConditions struct {
	Ready       bool `yaml:"ready"`
	Serving     bool `yaml:"serving"`
	Terminating bool `yaml:"terminating"`
}

type EndpointSliceEndpoint struct {
	Addresses  []string                         `yaml:"addresses"`
	Conditions *EndpointSliceEndpointConditions `yaml:"conditions"`
	TargetRef  *K8sTargetRef                    `yaml:"targetRef"`
	NodeName   string                           `yaml:"nodeName"`
	HostName   string                           `yaml:"hostName"` // This field maybe always empty in GKE
}

type EndpointSlice struct {
	Endpoints []*EndpointSliceEndpoint `yaml:"endpoints"`
	Metadata  *K8sObjectMeta           `yaml:"metadata"`
	// the other fields are not used.
}

func (c *EndpointSliceEndpointConditions) SameWith(other *EndpointSliceEndpointConditions) bool {
	return c.Ready == other.Ready && c.Serving == other.Serving && c.Terminating == other.Terminating
}

type PodMetadata struct {
	Uid string `yaml:"uid"`
}

type Pod struct {
	Metadata *PodMetadata `yaml:"metadata"`
	Status   *PodStatus   `yaml:"status"`
	// the other fields are not used.
}

type PodIP struct {
	IP string `yaml:"ip"`
}

type PodStatus struct {
	HostIP string   `yaml:"hostIP"`
	PodIP  string   `yaml:"podIP"`
	PodIPs []*PodIP `yaml:"podIPs"`
}
