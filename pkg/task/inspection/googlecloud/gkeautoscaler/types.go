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

package gkeautoscaler

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

var (
	pathAutoscalerDecision         = structured.CompileFieldPath("jsonPayload.decision")
	pathAutoscalerNoDecisionStatus = structured.CompileFieldPath("jsonPayload.noDecisionStatus")
	pathAutoscalerResultInfo       = structured.CompileFieldPath("jsonPayload.resultInfo")
)

// DecisionLog represents a jsonPayload.decision entry emitted when Cluster Autoscaler takes a scaling action.
type DecisionLog struct {
	// DecideTime is the timestamp when the autoscaler made the decision.
	DecideTime string `json:"decideTime" yaml:"decideTime"`
	// EventID is the unique identifier correlating the decision with its Result entry.
	EventID string `json:"eventId" yaml:"eventId"`
	// ScaleUp contains details when the decision adds nodes to existing node pools.
	ScaleUp *ScaleUpItem `json:"scaleUp,omitempty" yaml:"scaleUp,omitempty"`
	// ScaleDown contains details when the decision removes underutilized nodes.
	ScaleDown *ScaleDownItem `json:"scaleDown,omitempty" yaml:"scaleDown,omitempty"`
	// NodePoolCreated contains details when Node Auto-Provisioning creates node pools.
	NodePoolCreated *NodepoolCreatedItem `json:"nodePoolCreated,omitempty" yaml:"nodePoolCreated,omitempty"`
	// NodePoolDeleted contains details when Node Auto-Provisioning deletes empty node pools.
	NodePoolDeleted *NodepoolDeletedItem `json:"nodePoolDeleted,omitempty" yaml:"nodePoolDeleted,omitempty"`
}

// ScaleUpItem represents a scale-up decision made by Cluster Autoscaler.
// See https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-autoscaler-visibility#example_2 for schema details.
type ScaleUpItem struct {
	// IncreasedMigs lists the Managed Instance Groups whose target size was increased.
	IncreasedMigs []IncreasedMIGItem `json:"increasedMigs" yaml:"increasedMigs"`
	// TriggeringPods lists a sample of unschedulable pods that triggered the scale-up.
	TriggeringPods []PodItem `json:"triggeringPods" yaml:"triggeringPods"`
	// TriggeringPodsTotalCount is the total number of unschedulable pods that triggered the scale-up.
	TriggeringPodsTotalCount int `json:"triggeringPodsTotalCount" yaml:"triggeringPodsTotalCount"`
}

// ScaleDownItem represents a scale-down decision made by Cluster Autoscaler.
type ScaleDownItem struct {
	// NodesToBeRemoved lists the nodes selected for scale-down removal.
	NodesToBeRemoved []NodeToBeRemovedItem `json:"nodesToBeRemoved" yaml:"nodesToBeRemoved"`
}

// NodepoolCreatedItem represents a Node Auto-Provisioning decision that creates new node pools.
type NodepoolCreatedItem struct {
	// NodePools lists the node pools created by Node Auto-Provisioning.
	NodePools []NodepoolItem `json:"nodePools" yaml:"nodePools"`
	// TriggeringScaleUpId identifies the scale-up event that triggered node pool creation.
	TriggeringScaleUpId string `json:"triggeringScaleUpId" yaml:"triggeringScaleUpId"`
}

// NodepoolDeletedItem represents a Node Auto-Provisioning decision that deletes node pools.
type NodepoolDeletedItem struct {
	// NodePoolNames lists the names of the deleted node pools.
	NodePoolNames []string `json:"nodePoolNames" yaml:"nodePoolNames"`
}

// IncreasedMIGItem pairs a Managed Instance Group with the number of nodes requested for scale-up.
type IncreasedMIGItem struct {
	// Mig identifies the Managed Instance Group being scaled up.
	Mig MIGItem `json:"mig" yaml:"mig"`
	// RequestedNodes is the number of additional nodes requested in this MIG.
	RequestedNodes int `json:"requestedNodes" yaml:"requestedNodes"`
}

// MIGItem identifies a GCE Managed Instance Group belonging to a GKE node pool.
type MIGItem struct {
	// Name is the Managed Instance Group resource name.
	Name string `json:"name" yaml:"name"`
	// Nodepool is the name of the GKE node pool backed by this MIG.
	Nodepool string `json:"nodepool" yaml:"nodepool"`
	// Zone is the GCE zone where the MIG resides.
	Zone string `json:"zone" yaml:"zone"`
}

// PodItem identifies a Kubernetes pod and its owning controller referenced in autoscaler logs.
type PodItem struct {
	// Controller identifies the workload controller managing the pod.
	Controller ControllerItem `json:"controller" yaml:"controller"`
	// Name is the Kubernetes pod name.
	Name string `json:"name" yaml:"name"`
	// Namespace is the Kubernetes namespace of the pod.
	Namespace string `json:"namespace" yaml:"namespace"`
}

// ControllerItem identifies the Kubernetes controller owning a pod.
type ControllerItem struct {
	// ApiVersion is the API group/version of the controller resource.
	ApiVersion string `json:"apiVersion" yaml:"apiVersion"`
	// Kind is the resource kind of the controller (e.g., Deployment, StatefulSet).
	Kind string `json:"kind" yaml:"kind"`
	// Name is the name of the controller resource.
	Name string `json:"name" yaml:"name"`
}

// NodeToBeRemovedItem describes a node scheduled for removal during scale-down and its evicted pods.
type NodeToBeRemovedItem struct {
	// EvictedPods lists a sample of pods to be evicted from the node.
	EvictedPods []PodItem `json:"evictedPods" yaml:"evictedPods"`
	// EvictedPodsTotalCount is the total number of pods being evicted from the node.
	EvictedPodsTotalCount int `json:"evictedPodsTotalCount" yaml:"evictedPodsTotalCount"`
	// Node identifies the node selected for removal.
	Node NodeItem `json:"node" yaml:"node"`
}

// NodeItem describes a Kubernetes node and its utilization metrics in autoscaler logs.
type NodeItem struct {
	// CpuRatio is the CPU utilization percentage of the node.
	CpuRatio int `json:"cpuRatio" yaml:"cpuRatio"`
	// MemRatio is the memory utilization percentage of the node.
	MemRatio int `json:"memRatio" yaml:"memRatio"`
	// Mig identifies the Managed Instance Group that owns the node.
	Mig MIGItem `json:"mig" yaml:"mig"`
	// Name is the Kubernetes node name.
	Name string `json:"name" yaml:"name"`
}

// NodepoolItem describes a GKE node pool and its backing Managed Instance Groups.
type NodepoolItem struct {
	// Migs lists the Managed Instance Groups created for the node pool.
	Migs []MIGItem `json:"migs" yaml:"migs"`
	// Name is the GKE node pool name.
	Name string `json:"name" yaml:"name"`
}

// SkippedMIGItem records a Managed Instance Group that was skipped during scale-up evaluation.
type SkippedMIGItem struct {
	// Mig identifies the skipped Managed Instance Group.
	Mig MIGItem `json:"mig" yaml:"mig"`
	// Reason describes why the MIG was skipped.
	Reason ReasonItem `json:"reason" yaml:"reason"`
}

// ReasonItem represents a structured autoscaler message ID and its formatting parameters.
type ReasonItem struct {
	// MessageId is the stable identifier of the autoscaler reason message.
	MessageId string `json:"messageId" yaml:"messageId"`
	// Parameters holds positional substitution values for MessageId.
	Parameters []string `json:"parameters" yaml:"parameters"`
}

// NapFailureReasonItem represents why Node Auto-Provisioning could not provision a node pool.
type NapFailureReasonItem struct {
	// MessageId is the stable identifier of the NAP failure reason.
	MessageId string `json:"messageId" yaml:"messageId"`
	// Parameters holds positional substitution values for MessageId.
	Parameters []string `json:"parameters" yaml:"parameters"`
}

// UnhandledPodGroupItem describes a group of unschedulable pods that could not be accommodated by scale-up.
type UnhandledPodGroupItem struct {
	// NAPFailureReasons lists reasons why Node Auto-Provisioning could not handle the pod group.
	NAPFailureReasons []NapFailureReasonItem `json:"napFailureReasons" yaml:"napFailureReasons"`
	// PodGroup identifies the representative pod and total count in the group.
	PodGroup PodGroup `json:"podGroup" yaml:"podGroup"`
	// RejectedMigs lists candidate MIGs that rejected the pod group and why.
	RejectedMigs []RejectedMIGItem `json:"rejectedMigs" yaml:"rejectedMigs"`
}

// PodGroup summarizes a group of equivalent unschedulable pods.
type PodGroup struct {
	// SamplePod is a representative pod from the unschedulable group.
	SamplePod PodItem `json:"samplePod" yaml:"samplePod"`
	// TotalPodCount is the number of pods in the group.
	TotalPodCount int `json:"totalPodCount" yaml:"totalPodCount"`
}

// RejectedMIGItem records a Managed Instance Group that could not satisfy a pod group and the rejection reason.
type RejectedMIGItem struct {
	// Mig identifies the rejected Managed Instance Group.
	Mig MIGItem `json:"mig" yaml:"mig"`
	// Reason describes why the MIG could not schedule the pod group.
	Reason ReasonItem `json:"reason" yaml:"reason"`
}

// NoDecisionStatusLog represents a jsonPayload.noDecisionStatus entry explaining why no scaling action was taken.
type NoDecisionStatusLog struct {
	// MeasureTime is the timestamp when the autoscaler evaluated cluster state.
	MeasureTime string `json:"measureTime" yaml:"measureTime"`
	// NoScaleUp explains why unschedulable pods did not trigger a scale-up.
	NoScaleUp *NoScaleUpItem `json:"noScaleUp,omitempty" yaml:"noScaleUp,omitempty"`
	// NoScaleDown explains why underutilized nodes were not scaled down.
	NoScaleDown *NoScaleDownItem `json:"noScaleDown,omitempty" yaml:"noScaleDown,omitempty"`
}

// NoScaleUpItem contains details about skipped MIGs and unhandled pod groups when scale-up was not triggered.
type NoScaleUpItem struct {
	// SkippedMigs lists MIGs skipped during scale-up consideration.
	SkippedMigs []SkippedMIGItem `json:"skippedMigs" yaml:"skippedMigs"`
	// UnhandledPodGroups lists unschedulable pod groups that could not trigger a scale-up.
	UnhandledPodGroups []UnhandledPodGroupItem `json:"unhandledPodGroups" yaml:"unhandledPodGroups"`
	// UnhandledPodGroupsTotalCount is the total number of unhandled pod groups.
	UnhandledPodGroupsTotalCount int `json:"unhandledPodGroupsTotalCount" yaml:"unhandledPodGroupsTotalCount"`
}

// NoScaleDownItem contains details about nodes that could not be scaled down and why.
type NoScaleDownItem struct {
	// Nodes lists a sample of nodes blocked from scale-down.
	Nodes []NoScaleDownNodeItem `json:"nodes" yaml:"nodes"`
	// NodesTotalCount is the total number of nodes blocked from scale-down.
	NodesTotalCount int `json:"nodesTotalCount" yaml:"nodesTotalCount"`
	// Reason explains why the nodes were not scaled down.
	Reason ReasonItem `json:"reason" yaml:"reason"`
}

// NoScaleDownNodeItem wraps a node entry in a NoScaleDownItem.
type NoScaleDownNodeItem struct {
	// Node identifies the node that could not be scaled down.
	Node NodeItem `json:"node" yaml:"node"`
}

// ErrorMessageItem represents a structured error message from a Cluster Autoscaler Result.
type ErrorMessageItem struct {
	// MessageId is the stable identifier of the autoscaler error message.
	MessageId string `json:"messageId" yaml:"messageId"`
	// Parameters holds optional positional substitution values for MessageId.
	Parameters []string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// Result represents the outcome of a single autoscaler decision event.
type Result struct {
	// EventID links this result back to the corresponding DecisionLog event.
	EventID string `json:"eventId" yaml:"eventId"`
	// ErrorMsg describes the error if the autoscaler action failed, or nil on success.
	ErrorMsg *ErrorMessageItem `json:"errorMsg,omitempty" yaml:"errorMsg,omitempty"` // Pointer to allow for optional error
}

// ResultInfoLog represents a jsonPayload.resultInfo entry reporting outcomes of prior autoscaler decisions.
type ResultInfoLog struct {
	// MeasureTime is the timestamp when the results were reported.
	MeasureTime string `json:"measureTime" yaml:"measureTime"`
	// Results lists the per-event execution outcomes.
	Results []Result `json:"results" yaml:"results"`
}

// Id returns a unique ID used for deduping elements in the MIG array.
func (m MIGItem) Id() string {
	return fmt.Sprintf("%s/%s/%s", m.Nodepool, m.Zone, m.Name)
}

func parseDecisionFromReader(rootReader *structured.NodeReader) (*DecisionLog, error) {
	var result DecisionLog
	err := structured.ReadReflect(rootReader, pathAutoscalerDecision, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func parseNoDecisionFromReader(rootReader *structured.NodeReader) (*NoDecisionStatusLog, error) {
	var result NoDecisionStatusLog
	err := structured.ReadReflect(rootReader, pathAutoscalerNoDecisionStatus, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func parseResultInfoFromReader(rootReader *structured.NodeReader) (*ResultInfoLog, error) {
	var result ResultInfoLog
	err := structured.ReadReflect(rootReader, pathAutoscalerResultInfo, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}
