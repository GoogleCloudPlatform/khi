// Copyright 2026 Google LLC
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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// LabelKeyProvidedTagPrefix is the prefix used for tag labels on tasks.
const LabelKeyProvidedTagPrefix = KHISystemPrefix + "provided-tag/"

// LabelKeyProvidedTag generates a TaskLabelKey to record a provided tag on a task.
func LabelKeyProvidedTag(tagID string) TaskLabelKey[bool] {
	return NewTaskLabelKey[bool](LabelKeyProvidedTagPrefix + tagID)
}

// Tag represents a strongly-typed tag identifier that groups task outputs of type TaskResult.
type Tag[TaskResult any] struct {
	id string
}

// NewTag creates a new typed tag with the given identifier.
func NewTag[TaskResult any](id string) Tag[TaskResult] {
	if strings.TrimSpace(id) == "" {
		panic("tag id must not be empty")
	}
	return Tag[TaskResult]{id: id}
}

// ID returns the string identifier of the tag.
func (t Tag[TaskResult]) ID() string {
	return t.id
}

// String returns the string representation of the tag.
func (t Tag[TaskResult]) String() string {
	return t.id
}

// Ref creates a typed dependency reference to tasks providing this tag with optional configurations.
func (t Tag[TaskResult]) Ref(opts ...taskid.FanInOption) TagReference[TaskResult] {
	return NewTagReference[TaskResult](t.id, opts...)
}

type providesTagLabelOpt[TaskResult any] struct {
	tag Tag[TaskResult]
}

func (p *providesTagLabelOpt[TaskResult]) Write(labels *typedmap.TypedMap) {
	typedmap.Set(labels, LabelKeyProvidedTag(p.tag.ID()), true)
}

var _ LabelOpt = (*providesTagLabelOpt[any])(nil)

// ProvidesTag returns a LabelOpt declaring that the task provides the given typed tag.
func ProvidesTag[TaskResult any](tag Tag[TaskResult]) LabelOpt {
	return &providesTagLabelOpt[TaskResult]{tag: tag}
}
