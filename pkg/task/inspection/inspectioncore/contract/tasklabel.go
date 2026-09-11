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
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

// LabelSelector represents a set of labels to match against target resources/features.
type LabelSelector map[string]string

// Match returns true if all keys defined in the selector are present in the target with matching values.
func (s LabelSelector) Match(target map[string]string) bool {
	for k, v := range s {
		if tv, ok := target[k]; !ok || tv != v {
			return false
		}
	}
	return true
}

const (
	// InspectionTypeLabelKeyLogSource is the label key for the log source of the inspection.
	// Expected values of this label key: "cloud_logging", "jsonl_upload", etc.
	InspectionTypeLabelKeyLogSource = "khi.google.com/log_source"

	// InspectionTypeLabelKeyEnvironment is the label key for the environment where the target product is running.
	// Expected values of this label key: "googlecloud", "onprem", "oss", etc.
	InspectionTypeLabelKeyEnvironment = "khi.google.com/environment"

	// InspectionTypeLabelKeyBasePlatform is the label key for the base platform of the cluster.
	// Expected values of this label key: "kubernetes", etc.
	InspectionTypeLabelKeyBasePlatform = "khi.google.com/base_platform"
)

var (
	LabelKeyInspectionFeatureFlag        = coretask.NewTaskLabelKey[bool](InspectionTaskPrefix + "feature")
	LabelKeyInspectionDefaultFeatureFlag = coretask.NewTaskLabelKey[bool](InspectionTaskPrefix + "default-feature")
	LabelKeyProgressReportable           = coretask.NewTaskLabelKey[bool](InspectionTaskPrefix + "progress-reportable")
	// LabelKeyInspectionTypeLabelSelector is a task label key used to specify target inspection types using a label selector.
	LabelKeyInspectionTypeLabelSelector = coretask.NewTaskLabelKey[LabelSelector](InspectionTaskPrefix + "inspection-type-selector")
	LabelKeyFeatureTaskTitle            = coretask.NewTaskLabelKey[string](InspectionTaskPrefix + "feature/title")
	LabelKeyFeatureTaskDescription      = coretask.NewTaskLabelKey[string](InspectionTaskPrefix + "feature/description")
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
// This annotates a task to be a feature in inspection for v6 format.
type FeatureTaskLabelImpl struct {
	title            string
	description      string
	featureOrder     int
	isDefaultFeature bool
}

// Write implements task.LabelOpt.
func (ftl *FeatureTaskLabelImpl) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, LabelKeyInspectionFeatureFlag, true)
	typedmap.Set(label, LabelKeyFeatureTaskTitle, ftl.title)
	typedmap.Set(label, LabelKeyFeatureTaskDescription, ftl.description)
	typedmap.Set(label, coretask.LabelKeyTaskDescription, ftl.description)
	typedmap.Set(label, LabelKeyFeatureTaskOrder, ftl.featureOrder)
	typedmap.Set(label, LabelKeyInspectionDefaultFeatureFlag, ftl.isDefaultFeature)
}

var _ coretask.LabelOpt = (*FeatureTaskLabelImpl)(nil)

// FeatureTaskLabel returns a LabelOpt to mark the task as a feature in the inspection for v6 format.
func FeatureTaskLabel(title string, description string, featureOrder int, isDefaultFeature bool) *FeatureTaskLabelImpl {
	return &FeatureTaskLabelImpl{
		title:            title,
		description:      description,
		featureOrder:     featureOrder,
		isDefaultFeature: isDefaultFeature,
	}
}

type InspectionTypeLabelSelectorImpl struct {
	selector LabelSelector
}

// Write implements task.LabelOpt.
func (itl *InspectionTypeLabelSelectorImpl) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, LabelKeyInspectionTypeLabelSelector, itl.selector)
}

var _ coretask.LabelOpt = (*InspectionTypeLabelSelectorImpl)(nil)

// InspectionTypeLabelSelector returns a LabelOpt to mark the task to match against InspectionType labels using the given selector.
func InspectionTypeLabelSelector(selector map[string]string) *InspectionTypeLabelSelectorImpl {
	return &InspectionTypeLabelSelectorImpl{
		selector: LabelSelector(selector),
	}
}
