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

package coretask

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

var _ LabelOpt = (*taskResultRetentionLabelOptImpl)(nil)

// TaskLabelKey is a key of labels given to task.
type TaskLabelKey[LabelValueType any] = typedmap.TypedKey[LabelValueType]

// LabelOpt implementations wraps setting values to the task albels.
type LabelOpt interface {
	// Write applies the label option to the given TypedMap.
	Write(labels *typedmap.TypedMap)
}

type selectionPrioirtyLabelOpt struct {
	priority int
}

// labelValueOpt stores a label value associating to a label key.
type labelValueOpt[T any] struct {
	labelKey TaskLabelKey[T]
	value    T
}

type requiredTaskLabelImpl struct{}

type taskResultRetentionLabelOptImpl struct {
	retain bool
}

// NewTaskLabelKey returns the key used in labels with type annotation.
func NewTaskLabelKey[T any](key string) TaskLabelKey[T] {
	return typedmap.NewTypedKey[T](key)
}

// NewLabelSet constructs the LabelSet by applying the given label options.
func NewLabelSet(labelOpts ...LabelOpt) *typedmap.ReadonlyTypedMap {
	typedMap := typedmap.NewTypedMap()
	for _, lo := range labelOpts {
		lo.Write(typedMap)
	}
	return typedMap.AsReadonly()
}

// WithSelectionPriority returns a LabelOpt that sets the task selection priority.
func WithSelectionPriority(priority int) LabelOpt {
	return &selectionPrioirtyLabelOpt{
		priority: priority,
	}
}

// WithLabelValue creates a LabelOpt to store a single value associated to a label key.
func WithLabelValue[T any](labelKey TaskLabelKey[T], value T) LabelOpt {
	return &labelValueOpt[T]{
		labelKey: labelKey,
		value:    value,
	}
}

// NewRequiredTaskLabel returns a LabelOpt to mark the task is always included in the result task graph.
func NewRequiredTaskLabel() *requiredTaskLabelImpl {
	return &requiredTaskLabelImpl{}
}

// WithFeatureGate sets the feature gate task reference for a task.
// When a task with a feature gate is referenced via ScopeActiveFeatures, it is pulled into the graph
// only if it is already in the graph or its feature gate task is present in the active graph.
// Tasks without WithFeatureGate are pulled in unconditionally when referenced via ScopeActiveFeatures.
func WithFeatureGate(featureTaskRef taskid.UntypedTaskReference) LabelOpt {
	if featureTaskRef == nil || featureTaskRef.ReferenceIDString() == "" {
		panic("WithFeatureGate: featureTaskRef must not be nil or empty")
	}
	return WithLabelValue(LabelKeyFeatureGateTaskRef, featureTaskRef)
}

// NewTaskResultRetentionLabel returns a LabelOpt to specify whether the task result should be retained after all dependent tasks finish.
func NewTaskResultRetentionLabel(retain bool) LabelOpt {
	return &taskResultRetentionLabelOptImpl{
		retain: retain,
	}
}

// WithTitle returns a LabelOpt to attach a human-readable title to the task.
func WithTitle(title string) LabelOpt {
	return WithLabelValue(LabelKeyTaskTitle, title)
}

// WithDescription returns a LabelOpt to attach a human-readable description to the task.
func WithDescription(description string) LabelOpt {
	return WithLabelValue(LabelKeyTaskDescription, description)
}

func (s *selectionPrioirtyLabelOpt) Write(ls *typedmap.TypedMap) {
	typedmap.Set(ls, LabelKeyTaskSelectionPriority, s.priority)
}

// Write implements LabelOpt interface.
func (o *labelValueOpt[T]) Write(labels *typedmap.TypedMap) {
	typedmap.Set(labels, o.labelKey, o.value)
}

func (r *requiredTaskLabelImpl) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, LabelKeyRequiredTask, true)
}

// Write implements LabelOpt.
func (r *taskResultRetentionLabelOptImpl) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, LabelKeyTaskResultRetention, r.retain)
}
