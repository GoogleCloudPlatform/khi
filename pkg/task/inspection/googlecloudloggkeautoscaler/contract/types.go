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

package googlecloudloggkeautoscaler_contract

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
)

type DecisionLog struct {
	DecideTime      string               `json:"decideTime" yaml:"decideTime"`
	EventID         string               `json:"eventId" yaml:"eventId"`
	ScaleUp         *ScaleUpItem         `json:"scaleUp,omitempty" yaml:"scaleUp,omitempty"`
	ScaleDown       *ScaleDownItem       `json:"scaleDown,omitempty" yaml:"scaleDown,omitempty"`
	NodePoolCreated *NodepoolCreatedItem `json:"nodePoolCreated,omitempty" yaml:"nodePoolCreated,omitempty"`
	NodePoolDeleted *NodepoolDeletedItem `json:"nodePoolDeleted,omitempty" yaml:"nodePoolDeleted,omitempty"`
}

// https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-autoscaler-visibility#example_2
type ScaleUpItem struct {
	IncreasedMigs            []IncreasedMIGItem `json:"increasedMigs" yaml:"increasedMigs"`
	TriggeringPods           []PodItem          `json:"triggeringPods" yaml:"triggeringPods"`
	TriggeringPodsTotalCount int                `json:"triggeringPodsTotalCount" yaml:"triggeringPodsTotalCount"`
}

type ScaleDownItem struct {
	NodesToBeRemoved []NodeToBeRemovedItem `json:"nodesToBeRemoved" yaml:"nodesToBeRemoved"`
}

type NodepoolCreatedItem struct {
	NodePools           []NodepoolItem `json:"nodePools" yaml:"nodePools"`
	TriggeringScaleUpId string         `json:"triggeringScaleUpId" yaml:"triggeringScaleUpId"`
}

type NodepoolDeletedItem struct {
	NodePoolNames []string `json:"nodePoolNames" yaml:"nodePoolNames"`
}

type IncreasedMIGItem struct {
	Mig            MIGItem `json:"mig" yaml:"mig"`
	RequestedNodes int     `json:"requestedNodes" yaml:"requestedNodes"`
}

type MIGItem struct {
	Name     string `json:"name" yaml:"name"`
	Nodepool string `json:"nodepool" yaml:"nodepool"`
	Zone     string `json:"zone" yaml:"zone"`
}

type PodItem struct {
	Controller ControllerItem `json:"controller" yaml:"controller"`
	Name       string         `json:"name" yaml:"name"`
	Namespace  string         `json:"namespace" yaml:"namespace"`
}

type ControllerItem struct {
	ApiVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
	Name       string `json:"name" yaml:"name"`
}

type NodeToBeRemovedItem struct {
	EvictedPods           []PodItem `json:"evictedPods" yaml:"evictedPods"`
	EvictedPodsTotalCount int       `json:"evictedPodsTotalCount" yaml:"evictedPodsTotalCount"`
	Node                  NodeItem  `json:"node" yaml:"node"`
}

type NodeItem struct {
	CpuRatio int     `json:"cpuRatio" yaml:"cpuRatio"`
	MemRatio int     `json:"memRatio" yaml:"memRatio"`
	Mig      MIGItem `json:"mig" yaml:"mig"`
	Name     string  `json:"name" yaml:"name"`
}

type NodepoolItem struct {
	Migs []MIGItem `json:"migs" yaml:"migs"`
	Name string    `json:"name" yaml:"name"`
}

type SkippedMIGItem struct {
	Mig    MIGItem    `json:"mig" yaml:"mig"`
	Reason ReasonItem `json:"reason" yaml:"reason"`
}

type ReasonItem struct {
	MessageId  string   `json:"messageId" yaml:"messageId"`
	Parameters []string `json:"parameters" yaml:"parameters"`
}

type NapFailureReasonItem struct {
	MessageId  string   `json:"messageId" yaml:"messageId"`
	Parameters []string `json:"parameters" yaml:"parameters"`
}

type UnhandledPodGroupItem struct {
	NAPFailureReasons []NapFailureReasonItem `json:"napFailureReasons" yaml:"napFailureReasons"`
	PodGroup          PodGroup               `json:"podGroup" yaml:"podGroup"`
	RejectedMigs      []RejectedMIGItem      `json:"rejectedMigs" yaml:"rejectedMigs"`
}

type PodGroup struct {
	SamplePod     PodItem `json:"samplePod" yaml:"samplePod"`
	TotalPodCount int     `json:"totalPodCount" yaml:"totalPodCount"`
}

type RejectedMIGItem struct {
	Mig    MIGItem    `json:"mig" yaml:"mig"`
	Reason ReasonItem `json:"reason" yaml:"reason"`
}

type NoDecisionStatusLog struct {
	MeasureTime string           `json:"measureTime" yaml:"measureTime"`
	NoScaleUp   *NoScaleUpItem   `json:"noScaleUp,omitempty" yaml:"noScaleUp,omitempty"`
	NoScaleDown *NoScaleDownItem `json:"noScaleDown,omitempty" yaml:"noScaleDown,omitempty"`
}

type NoScaleUpItem struct {
	SkippedMigs                  []SkippedMIGItem        `json:"skippedMigs" yaml:"skippedMigs"`
	UnhandledPodGroups           []UnhandledPodGroupItem `json:"unhandledPodGroups" yaml:"unhandledPodGroups"`
	UnhandledPodGroupsTotalCount int                     `json:"unhandledPodGroupsTotalCount" yaml:"unhandledPodGroupsTotalCount"`
}

type NoScaleDownItem struct {
	Nodes           []NoScaleDownNodeItem `json:"nodes" yaml:"nodes"`
	NodesTotalCount int                   `json:"nodesTotalCount" yaml:"nodesTotalCount"`
	Reason          ReasonItem            `json:"reason" yaml:"reason"`
}

type NoScaleDownNodeItem struct {
	Node NodeItem `json:"node" yaml:"node"`
}

type ErrorMessageItem struct {
	MessageId  string   `json:"messageId" yaml:"messageId"`
	Parameters []string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type Result struct {
	EventID  string            `json:"eventId" yaml:"eventId"`
	ErrorMsg *ErrorMessageItem `json:"errorMsg,omitempty" yaml:"errorMsg,omitempty"` // Pointer to allow for optional error
}

type ResultInfoLog struct {
	MeasureTime string   `json:"measureTime" yaml:"measureTime"`
	Results     []Result `json:"results" yaml:"results"`
}

// Unique ID used for deduping elements in mig array
func (m MIGItem) Id() string {
	return fmt.Sprintf("%s/%s/%s", m.Nodepool, m.Zone, m.Name)
}

func parseDecisionFromReader(rootReader *structured.NodeReader) (*DecisionLog, error) {
	var result DecisionLog
	err := structured.ReadReflect(rootReader, "jsonPayload.decision", &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func parseNoDecisionFromReader(rootReader *structured.NodeReader) (*NoDecisionStatusLog, error) {
	var result NoDecisionStatusLog
	err := structured.ReadReflect(rootReader, "jsonPayload.noDecisionStatus", &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func parseResultInfoFromReader(rootReader *structured.NodeReader) (*ResultInfoLog, error) {
	var result ResultInfoLog
	err := structured.ReadReflect(rootReader, "jsonPayload.resultInfo", &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}
