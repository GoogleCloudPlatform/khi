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

package autoscaler

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure"
)

type decision struct {
	DecideTime      string           `json:"decideTime"`
	EventID         string           `json:"eventId"`
	ScaleUp         *scaleUp         `json:"scaleUp"`
	ScaleDown       *scaleDown       `json:"scaleDown"`
	NodePoolCreated *nodePoolCreated `json:"nodePoolCreated"`
	NodePoolDeleted *nodePoolDeleted `json:"nodePoolDeleted"`
}

// https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-autoscaler-visibility#example_2
type scaleUp struct {
	IncreasedMigs            []increasedMig `json:"increasedMigs"`
	TriggeringPods           []pod          `json:"triggeringPods"`
	TriggeringPodsTotalCount int            `json:"triggeringPodsTotalCount"`
}

type scaleDown struct {
	NodesToBeRemoved []nodeToBeRemoved `json:"nodesToBeRemoved"`
}

type nodePoolCreated struct {
	NodePools           []nodepool `json:"nodePools"`
	TriggeringScaleUpId string     `json:"triggeringScaleUpId"`
}

type nodePoolDeleted struct {
	NodePoolNames []string `json:"nodePoolNames"`
}

type increasedMig struct {
	Mig            mig `json:"mig"`
	RequestedNodes int `json:"requestedNodes"`
}

type mig struct {
	Name     string `json:"name"`
	Nodepool string `json:"nodepool"`
	Zone     string `json:"zone"`
}

type pod struct {
	Controller controller `json:"controller"`
	Name       string     `json:"name"`
	Namespace  string     `json:"namespace"`
}

type controller struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

type nodeToBeRemoved struct {
	EvictedPods           []pod `json:"evictedPods"`
	EvictedPodsTotalCount int   `json:"evictedPodsTotalCount"`
	Node                  node  `json:"node"`
}

type node struct {
	CpuRatio int    `json:"cpuRatio"`
	MemRatio int    `json:"memRatio"`
	Mig      mig    `json:"mig"`
	Name     string `json:"name"`
}

type nodepool struct {
	Migs []mig  `json:"migs"`
	Name string `json:"name"`
}

type skippedMig struct {
	Mig    mig    `json:"mig"`
	Reason reason `json:"reason"`
}

type reason struct {
	MessageId  string   `json:"messageId"`
	Parameters []string `json:"parameters"`
}

type napFailureReason struct {
	MessageId  string   `json:"messageId"`
	Parameters []string `json:"parameters"`
}

type unhandledPodGroup struct {
	NAPFailureReasons []napFailureReason `json:"napFailureReasons"`
	PodGroup          podGroup           `json:"podGroup"`
	RejectedMigs      []rejectedMig      `json:"rejectedMigs"`
}

type podGroup struct {
	SamplePod     pod `json:"samplePod"`
	TotalPodCount int `json:"totalPodCount"`
}

type rejectedMig struct {
	Mig    mig    `json:"mig"`
	Reason reason `json:"reason"`
}

type noDecisionStatus struct {
	MeasureTime string       `json:"measureTime"`
	NoScaleUp   *noScaleUp   `json:"noScaleUp"`
	NoScaleDown *noScaleDown `json:"noScaleDown"`
}

type noScaleUp struct {
	SkippedMigs                  []skippedMig        `json:"skippedMigs"`
	UnhandledPodGroups           []unhandledPodGroup `json:"unhandledPodGroups"`
	UnhandledPodGroupsTotalCount int                 `json:"unhandledPodGroupsTotalCount"`
}

type noScaleDown struct {
	Nodes           []noScaleDownNode `json:"nodes"`
	NodesTotalCount int               `json:"nodesTotalCount"`
	Reason          reason            `json:"reason"`
}

type noScaleDownNode struct {
	Node node `json:"node"`
}

type errorMsg struct {
	MessageId  string   `json:"messageId"`
	Parameters []string `json:"parameters"`
}

type result struct {
	EventID  string    `json:"eventId"`
	ErrorMsg *errorMsg `json:"errorMsg"` // Pointer to allow for optional error
}

type resultInfo struct {
	MeasureTime string   `json:"measureTime"`
	Results     []result `json:"results"`
}

// Unique ID used for deduping elements in mig array
func (m mig) Id() string {
	return fmt.Sprintf("%s/%s/%s", m.Nodepool, m.Zone, m.Name)
}

func parseDecisionFromReader(decisionReader *structure.Reader) (*decision, error) {
	var result decision
	err := decisionReader.ReadReflect("", &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func parseNoDecisionFromReader(noDecisionReader *structure.Reader) (*noDecisionStatus, error) {
	var result noDecisionStatus
	err := noDecisionReader.ReadReflect("", &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func parseResultInfoFromReader(resultInfoReader *structure.Reader) (*resultInfo, error) {
	var result resultInfo
	err := resultInfoReader.ReadReflect("", &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}
