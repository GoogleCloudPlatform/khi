// Copyright 2025 Google LLC
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

var (
	pathProjectID        = structured.CompileFieldPath("resource.labels.project_id")
	pathClusterName      = structured.CompileFieldPath("resource.labels.cluster_name")
	pathDecision         = structured.CompileFieldPath("jsonPayload.decision")
	pathNoDecisionStatus = structured.CompileFieldPath("jsonPayload.noDecisionStatus")
	pathResultInfo       = structured.CompileFieldPath("jsonPayload.resultInfo")
)

type AutoscalerLogFieldSet struct {
	ProjectID     string
	ClusterName   string
	DecisionLog   *DecisionLog
	NoDecisionLog *NoDecisionStatusLog
	ResultInfoLog *ResultInfoLog
}

// ExtractAutoscalerLog extracts GKE autoscaler log fields from a NodeReader.
func ExtractAutoscalerLog(reader *structured.NodeReader) (AutoscalerLogFieldSet, error) {
	if mock, ok := structured.GetMock[AutoscalerLogFieldSet](reader); ok {
		return mock, nil
	}
	var result AutoscalerLogFieldSet
	result.ProjectID = reader.ReadStringOrDefault(pathProjectID, "unknown")
	result.ClusterName = reader.ReadStringOrDefault(pathClusterName, "")
	switch {
	case reader.Has(pathDecision):
		decisionLog, err := parseDecisionFromReader(reader)
		if err != nil {
			return result, fmt.Errorf("failed to parse decision log: %w", err)
		}
		result.DecisionLog = decisionLog
	case reader.Has(pathNoDecisionStatus):
		noDecisionLog, err := parseNoDecisionFromReader(reader)
		if err != nil {
			return result, fmt.Errorf("failed to parse noDecisionStatus log: %w", err)
		}
		result.NoDecisionLog = noDecisionLog
	case reader.Has(pathResultInfo):
		resultInfoLog, err := parseResultInfoFromReader(reader)
		if err != nil {
			return result, fmt.Errorf("failed to parse resultInfo log: %w", err)
		}
		result.ResultInfoLog = resultInfoLog
	}
	return result, nil
}
