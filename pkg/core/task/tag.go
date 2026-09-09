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

// DefaultTagPriority is the default priority assigned to a provided tag when not explicitly specified.
const DefaultTagPriority = 100

// LabelKeyProvidedTagPriorityPrefix is the prefix used for tag priority labels on tasks.
const LabelKeyProvidedTagPriorityPrefix = KHISystemPrefix + "provided-tag-priority/"

// LabelKeyProvidedTag returns a TaskLabelKey to record a provided tag on a task.
func LabelKeyProvidedTag(tagID string) TaskLabelKey[bool] {
	return NewTaskLabelKey[bool](LabelKeyProvidedTagPrefix + tagID)
}

// LabelKeyProvidedTagPriority returns a TaskLabelKey to record the priority of a provided tag on a task.
func LabelKeyProvidedTagPriority(tagID string) TaskLabelKey[int] {
	return NewTaskLabelKey[int](LabelKeyProvidedTagPriorityPrefix + tagID)
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

// ProvidesTagOption is an option to configure tag provision attributes on a task.
type ProvidesTagOption interface {
	applyProvidesTag(*providesTagConfig)
}

type providesTagConfig struct {
	priority int
}

type withTagPriorityOption struct {
	priority int
}

func (w *withTagPriorityOption) applyProvidesTag(c *providesTagConfig) {
	c.priority = w.priority
}

var _ ProvidesTagOption = (*withTagPriorityOption)(nil)

// WithTagPriority returns a ProvidesTagOption specifying the priority weight of the provided tag during graph resolution and cycle pruning.
// Lower numerical values indicate higher precedence (e.g. 10 is higher priority than 100).
// The default priority when unspecified is DefaultTagPriority (100).
func WithTagPriority(priority int) ProvidesTagOption {
	return &withTagPriorityOption{priority: priority}
}

type providesTagLabelOpt[TaskResult any] struct {
	tag      Tag[TaskResult]
	priority int
}

func (p *providesTagLabelOpt[TaskResult]) Write(labels *typedmap.TypedMap) {
	typedmap.Set(labels, LabelKeyProvidedTag(p.tag.ID()), true)
	typedmap.Set(labels, LabelKeyProvidedTagPriority(p.tag.ID()), p.priority)
}

var _ LabelOpt = (*providesTagLabelOpt[any])(nil)

// ProvidesTag returns a LabelOpt declaring that the task provides the given typed tag.
// Optional ProvidesTagOptions (such as WithTagPriority) can be specified to adjust tag attributes.
func ProvidesTag[TaskResult any](tag Tag[TaskResult], opts ...ProvidesTagOption) LabelOpt {
	cfg := providesTagConfig{
		priority: DefaultTagPriority,
	}
	for _, opt := range opts {
		opt.applyProvidesTag(&cfg)
	}
	return &providesTagLabelOpt[TaskResult]{
		tag:      tag,
		priority: cfg.priority,
	}
}
