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

// TaskLabelKey is a key of labels given to task.
type TaskLabelKey[LabelValueType any] = typedmap.TypedKey[LabelValueType]

// NewTaskLabelKey returns the key used in labels with type annotation,
func NewTaskLabelKey[T any](key string) TaskLabelKey[T] {
	return typedmap.NewTypedKey[T](key)
}

// Construct the LabelSet with required fields.
func NewLabelSet(labelOpts ...LabelOpt) *typedmap.ReadonlyTypedMap {
	typedMap := typedmap.NewTypedMap()
	for _, lo := range labelOpts {
		lo.Write(typedMap)
	}
	return typedMap.AsReadonly()
}

// LabelOpt implementations wraps setting values to the task albels.
type LabelOpt interface {
	Write(labels *typedmap.TypedMap)
}

type selectionPrioirtyLabelOpt struct {
	priority int
}

func (s *selectionPrioirtyLabelOpt) Write(ls *typedmap.TypedMap) {
	typedmap.Set(ls, LabelKeyTaskSelectionPriority, s.priority)
}

func WithSelectionPriority(priority int) LabelOpt {
	return &selectionPrioirtyLabelOpt{
		priority: priority,
	}
}

// labelValueOpt stores a label value associating to a label key.
type labelValueOpt[T any] struct {
	labelKey TaskLabelKey[T]
	value    T
}

// Write implements LabelOpt interface
func (o *labelValueOpt[T]) Write(labels *typedmap.TypedMap) {
	typedmap.Set(labels, o.labelKey, o.value)
}

// WithLabelValue creates a LabelOpt to store a single value associated to a label key.
func WithLabelValue[T any](labelKey TaskLabelKey[T], value T) LabelOpt {
	return &labelValueOpt[T]{
		labelKey: labelKey,
		value:    value,
	}
}

// FromLabels creates a list of LabelOpt to clone the set of labels from a task to the other.
func FromLabels(labels *typedmap.ReadonlyTypedMap) []LabelOpt {
	result := make([]LabelOpt, 0)
	for _, key := range labels.Keys() {
		labelKey := typedmap.NewTypedKey[any](key)
		value, found := typedmap.Get(labels, labelKey)
		if !found {
			panic("unreachable")
		}
		result = append(result, WithLabelValue(labelKey, value))
	}
	return result
}

type requiredTaskLabelImpl struct{}

func (r *requiredTaskLabelImpl) Write(label *typedmap.TypedMap) {
	typedmap.Set(label, LabelKeyRequiredTask, true)
}

// InspectionTypeLabel returns a LabelOpt to mark the task is always included in the result task graph.
func NewRequiredTaskLabel() *requiredTaskLabelImpl {
	return &requiredTaskLabelImpl{}
}

type withSubsequentTaskRef struct {
	additionalSubsequentTaskRefs []taskid.UntypedTaskReference
}

// NewSubsequentTaskRefsTaskLabel returns a LabelOpt to add subsequent task to the current task.
func NewSubsequentTaskRefsTaskLabel(refs ...taskid.UntypedTaskReference) LabelOpt {
	return &withSubsequentTaskRef{
		additionalSubsequentTaskRefs: refs,
	}
}

// Write implements LabelOpt.
func (w *withSubsequentTaskRef) Write(labels *typedmap.TypedMap) {
	subsequentTasks := typedmap.GetOrDefault(labels, LabelKeySubsequentTaskRefs, []taskid.UntypedTaskReference{})
	for _, additional := range w.additionalSubsequentTaskRefs {
		found := false
		for _, alreadyIncluded := range subsequentTasks {
			if additional.ReferenceIDString() == alreadyIncluded.ReferenceIDString() {
				found = true
				break
			}
		}
		if !found {
			subsequentTasks = append(subsequentTasks, additional)
		}
	}
	typedmap.Set(labels, LabelKeySubsequentTaskRefs, subsequentTasks)
}

var _ LabelOpt = (*withSubsequentTaskRef)(nil)
