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

package inspectioncore_contract

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

var (
	LabelKeyInspectionFeatureFlag        = coretask.NewTaskLabelKey[bool](InspectionTaskPrefix + "feature")
	LabelKeyInspectionDefaultFeatureFlag = coretask.NewTaskLabelKey[bool](InspectionTaskPrefix + "default-feature")
	LabelKeyInspectionRequiredFlag       = coretask.NewTaskLabelKey[bool](InspectionTaskPrefix + "required")
	LabelKeyProgressReportable           = coretask.NewTaskLabelKey[bool](InspectionTaskPrefix + "progress-reportable")
	LabelKeyInspectionTypes              = coretask.NewTaskLabelKey[[]string](InspectionTaskPrefix + "inspection-type")
	LabelKeyFeatureTaskTitle             = coretask.NewTaskLabelKey[string](InspectionTaskPrefix + "feature/title")
	LabelKeyFeatureTaskTargetLogType     = coretask.NewTaskLabelKey[enum.LogType](InspectionTaskPrefix + "feature/log-type")
	LabelKeyFeatureTaskDescription       = coretask.NewTaskLabelKey[string](InspectionTaskPrefix + "feature/description")
	// LabelKeyFeatureTaskOrder is a label key of an integer assigned for a feature task. Feature task with smaller order is placed at the top of the feature task list.
	LabelKeyFeatureTaskOrder = coretask.NewTaskLabelKey[int](InspectionTaskPrefix + "feature/order")
)

type ProgressReportableTaskLabelOptImpl struct{}

// Write implements task.LabelOpt.
func (i *ProgressReportableTaskLabelOptImpl) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, LabelKeyProgressReportable, true)
}

var _ coretask.LabelOpt = (*ProgressReportableTaskLabelOptImpl)(nil)

// FeatureTaskLabelImpl is an implementation of task.LabelOpt.
// This annotate a task to be a feature in inspection.
type FeatureTaskLabelImpl struct {
	title            string
	description      string
	logType          enum.LogType
	featureOrder     int
	isDefaultFeature bool
	inspectionTypes  []string
}

func (ftl *FeatureTaskLabelImpl) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, LabelKeyInspectionFeatureFlag, true)
	typedmap.Set(label, LabelKeyFeatureTaskTargetLogType, ftl.logType)
	typedmap.Set(label, LabelKeyFeatureTaskTitle, ftl.title)
	typedmap.Set(label, LabelKeyFeatureTaskDescription, ftl.description)
	typedmap.Set(label, LabelKeyFeatureTaskOrder, ftl.featureOrder)
	typedmap.Set(label, LabelKeyInspectionDefaultFeatureFlag, ftl.isDefaultFeature)
	typedmap.Set(label, LabelKeyInspectionTypes, ftl.inspectionTypes)
}

func (ftl *FeatureTaskLabelImpl) WithDescription(description string) *FeatureTaskLabelImpl {
	ftl.description = description
	return ftl
}

var _ coretask.LabelOpt = (*FeatureTaskLabelImpl)(nil)

func FeatureTaskLabel(title string, description string, logType enum.LogType, featureOrder int, isDefaultFeature bool, inspectionTypes ...string) *FeatureTaskLabelImpl {
	for i, t := range inspectionTypes {
		if t == "" {
			panic(fmt.Sprintf(`Invalid inspection type at index at #%d. Empty inspection type was given to FeatureTaskLabel function. This may be caused because of initialization order issue of global variables.
Please define task IDs and types used in its type parameter in a different package.`, i))
		}
	}
	return &FeatureTaskLabelImpl{
		title:            title,
		description:      description,
		logType:          logType,
		featureOrder:     featureOrder,
		isDefaultFeature: isDefaultFeature,
		inspectionTypes:  inspectionTypes,
	}
}

type InspectionTypeLabelImpl struct {
	inspectionTypes []string
}

// Write implements task.LabelOpt.
func (itl *InspectionTypeLabelImpl) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, LabelKeyInspectionTypes, itl.inspectionTypes)
}

var _ coretask.LabelOpt = (*InspectionTypeLabelImpl)(nil)

// InspectionTypeLabel returns a LabelOpt to mark the task only to be used in the specified inspection types.
// This label must not be used in the feature task. Use the FeatureTaskLabel in feature tasks.
func InspectionTypeLabel(types ...string) *InspectionTypeLabelImpl {
	for i, t := range types {
		if t == "" {
			panic(fmt.Sprintf(`Invalid inspection type at index at #%d. Empty inspection type was given to InspectionTypeLabel function. This may be caused because of initialization order issue of global variables.
Please define task IDs and types used in its type parameter in a different package.`, i))
		}
	}
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
