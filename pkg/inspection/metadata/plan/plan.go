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

package plan

import (
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var InspectionPlanMetadataKey = "plan"

type InspectionPlan struct {
	TaskGraph string `json:"taskGraph"`
}

// Labels implements metadata.Metadata.
func (*InspectionPlan) Labels() *task.LabelSet {
	return task.NewLabelSet(metadata.IncludeInDryRunResult(), metadata.IncludeInRunResult())
}

// ToSerializable implements metadata.Metadata.
func (p *InspectionPlan) ToSerializable() interface{} {
	return p
}

var _ metadata.Metadata = (*InspectionPlan)(nil)

type InspectionPlanMetadataFactory struct{}

// Instanciate implements metadata.MetadataFactory.
func (i *InspectionPlanMetadataFactory) Instanciate() metadata.Metadata {
	return &InspectionPlan{}
}

// InspectionPlanMetadataFactory implements metadata.MetadataFactory
var _ metadata.MetadataFactory = (*InspectionPlanMetadataFactory)(nil)
