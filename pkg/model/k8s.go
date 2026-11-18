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
