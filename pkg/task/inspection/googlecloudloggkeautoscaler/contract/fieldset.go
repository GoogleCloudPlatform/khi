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
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

type AutoscalerLogFieldSet struct {
	DecisionLog   *DecisionLog
	NoDecisionLog *NoDecisionStatusLog
	ResultInfoLog *ResultInfoLog
}

// Kind implements log.FieldSet.
func (a *AutoscalerLogFieldSet) Kind() string {
	return "cluster_autoscaler"
}

var _ log.FieldSet = (*AutoscalerLogFieldSet)(nil)

type AutoscalerLogFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (a *AutoscalerLogFieldSetReader) FieldSetKind() string {
	return (&AutoscalerLogFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (a *AutoscalerLogFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	var result AutoscalerLogFieldSet
	if decisionLog, err := parseDecisionFromReader(reader); err == nil {
		result.DecisionLog = decisionLog
	}
	if noDecisionLog, err := parseNoDecisionFromReader(reader); err == nil {
		result.NoDecisionLog = noDecisionLog
	}
	if resultInfoLog, err := parseResultInfoFromReader(reader); err == nil {
		result.ResultInfoLog = resultInfoLog
	}
	return &result, nil

}

var _ log.FieldSetReader = (*AutoscalerLogFieldSetReader)(nil)
