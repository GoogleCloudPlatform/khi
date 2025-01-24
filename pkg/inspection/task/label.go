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

package task

import common_task "github.com/GoogleCloudPlatform/khi/pkg/task"

const (
	InspectionTaskPrefix                 = common_task.KHISystemPrefix + "inspection/"
	LabelKeyInspectionFeatureFlag        = InspectionTaskPrefix + "feature"
	LabelKeyInspectionDefaultFeatureFlag = InspectionTaskPrefix + "default-feature"
	LabelKeyInspectionRequiredFlag       = InspectionTaskPrefix + "required"
	LabelKeyProgressReportable           = InspectionTaskPrefix + "progress-reportable"
	// A []string typed label of Definition. Task registry will filter task units by given inspection type at first.
	LabelKeyInspectionTypes  = InspectionTaskPrefix + "inspection-type"
	LabelKeyFeatureTaskTitle = InspectionTaskPrefix + "feature/title"

	LabelKeyFeatureTaskDescription = InspectionTaskPrefix + "feature/description"

	InspectionMainSubgraphName = InspectionTaskPrefix + "inspection-main"

	TaskModeDryRun = 1
	TaskModeRun    = 2
)

type ProgressReportableTaskLabelOptImpl struct{}

// Write implements task.LabelOpt.
func (i *ProgressReportableTaskLabelOptImpl) Write(label *common_task.LabelSet) {
	label.Set(LabelKeyProgressReportable, true)
}

var _ common_task.LabelOpt = (*ProgressReportableTaskLabelOptImpl)(nil)

// FeatureTaskLabelImpl is an implementation of task.LabelOpt.
// This annotate a task definition to be a feature in inspection.
type FeatureTaskLabelImpl struct {
	title            string
	description      string
	isDefaultFeature bool
}

func (ftl *FeatureTaskLabelImpl) Write(label *common_task.LabelSet) {
	label.Set(LabelKeyInspectionFeatureFlag, true)
	label.Set(LabelKeyFeatureTaskTitle, ftl.title)
	label.Set(LabelKeyFeatureTaskDescription, ftl.description)
	label.Set(LabelKeyInspectionDefaultFeatureFlag, ftl.isDefaultFeature)
}

func (ftl *FeatureTaskLabelImpl) WithDescription(description string) *FeatureTaskLabelImpl {
	ftl.description = description
	return ftl
}

var _ common_task.LabelOpt = (*FeatureTaskLabelImpl)(nil)

func FeatureTaskLabel(title string, description string, isDefaultFeature bool) *FeatureTaskLabelImpl {
	return &FeatureTaskLabelImpl{
		title:            title,
		description:      description,
		isDefaultFeature: isDefaultFeature,
	}
}

type InspectionTypeLabelImpl struct {
	inspectionTypes []string
}

// Write implements task.LabelOpt.
func (itl *InspectionTypeLabelImpl) Write(label *common_task.LabelSet) {
	label.Set(LabelKeyInspectionTypes, itl.inspectionTypes)
}

var _ common_task.LabelOpt = (*InspectionTypeLabelImpl)(nil)

// InspectionTypeLabel returns a LabelOpt to mark the task only to be used in the specified inspection types.
func InspectionTypeLabel(types ...string) *InspectionTypeLabelImpl {
	return &InspectionTypeLabelImpl{
		inspectionTypes: types,
	}
}

type RequriredTaskLabelImpl struct{}

func (r *RequriredTaskLabelImpl) Write(label *common_task.LabelSet) {
	label.Set(LabelKeyInspectionRequiredFlag, true)
}

// InspectionTypeLabel returns a LabelOpt to mark the task is always included in the result task graph.
func NewRequiredTaskLabel() *RequriredTaskLabelImpl {
	return &RequriredTaskLabelImpl{}
}
