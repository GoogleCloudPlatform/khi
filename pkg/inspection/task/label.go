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

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	common_task "github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

//TODO: move task label related constants to ./label

const (
	InspectionTaskPrefix = common_task.KHISystemPrefix + "inspection/"

	InspectionMainSubgraphName = InspectionTaskPrefix + "inspection-main"
)

var InspectionMainSubgraphInitTaskID = taskid.NewDefaultImplementationID[any](InspectionMainSubgraphName + "-init")
var InspectionMainSubgraphDoneTaskID = taskid.NewDefaultImplementationID[any](InspectionMainSubgraphName + "-done")

var (
	LabelKeyInspectionFeatureFlag        = common_task.NewTaskLabelKey[bool](InspectionTaskPrefix + "feature")
	LabelKeyInspectionDefaultFeatureFlag = common_task.NewTaskLabelKey[bool](InspectionTaskPrefix + "default-feature")
	LabelKeyInspectionRequiredFlag       = common_task.NewTaskLabelKey[bool](InspectionTaskPrefix + "required")
	LabelKeyProgressReportable           = common_task.NewTaskLabelKey[bool](InspectionTaskPrefix + "progress-reportable")
	LabelKeyInspectionTypes              = common_task.NewTaskLabelKey[[]string](InspectionTaskPrefix + "inspection-type")
	LabelKeyFeatureTaskTitle             = common_task.NewTaskLabelKey[string](InspectionTaskPrefix + "feature/title")
	LabelKeyFeatureTaskTargetLogType     = common_task.NewTaskLabelKey[enum.LogType](InspectionTaskPrefix + "feature/log-type")
	LabelKeyFeatureTaskDescription       = common_task.NewTaskLabelKey[string](InspectionTaskPrefix + "feature/description")
)

type ProgressReportableTaskLabelOptImpl struct{}

// Write implements task.LabelOpt.
func (i *ProgressReportableTaskLabelOptImpl) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, LabelKeyProgressReportable, true)
}

var _ common_task.LabelOpt = (*ProgressReportableTaskLabelOptImpl)(nil)

// FeatureTaskLabelImpl is an implementation of task.LabelOpt.
// This annotate a task definition to be a feature in inspection.
type FeatureTaskLabelImpl struct {
	title            string
	description      string
	logType          enum.LogType
	isDefaultFeature bool
}

func (ftl *FeatureTaskLabelImpl) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, LabelKeyInspectionFeatureFlag, true)
	typedmap.Set(label, LabelKeyFeatureTaskTargetLogType, ftl.logType)
	typedmap.Set(label, LabelKeyFeatureTaskTitle, ftl.title)
	typedmap.Set(label, LabelKeyFeatureTaskDescription, ftl.description)
	typedmap.Set(label, LabelKeyInspectionDefaultFeatureFlag, ftl.isDefaultFeature)
}

func (ftl *FeatureTaskLabelImpl) WithDescription(description string) *FeatureTaskLabelImpl {
	ftl.description = description
	return ftl
}

var _ common_task.LabelOpt = (*FeatureTaskLabelImpl)(nil)

func FeatureTaskLabel(title string, description string, logType enum.LogType, isDefaultFeature bool) *FeatureTaskLabelImpl {
	return &FeatureTaskLabelImpl{
		title:            title,
		description:      description,
		logType:          logType,
		isDefaultFeature: isDefaultFeature,
	}
}

type InspectionTypeLabelImpl struct {
	inspectionTypes []string
}

// Write implements task.LabelOpt.
func (itl *InspectionTypeLabelImpl) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, LabelKeyInspectionTypes, itl.inspectionTypes)
}

var _ common_task.LabelOpt = (*InspectionTypeLabelImpl)(nil)

// InspectionTypeLabel returns a LabelOpt to mark the task only to be used in the specified inspection types.
func InspectionTypeLabel(types ...string) *InspectionTypeLabelImpl {
	return &InspectionTypeLabelImpl{
		inspectionTypes: types,
	}
}

type RequriredTaskLabelImpl struct{}

func (r *RequriredTaskLabelImpl) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, LabelKeyInspectionRequiredFlag, true)
}

// InspectionTypeLabel returns a LabelOpt to mark the task is always included in the result task graph.
func NewRequiredTaskLabel() *RequriredTaskLabelImpl {
	return &RequriredTaskLabelImpl{}
}
