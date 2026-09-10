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
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// Dependency represents any task dependency descriptor, such as a point-to-point reference or a fan-in tag reference.
type Dependency = taskid.DependencyDescriptor

var (
	// FromActiveFeatures configures the dependency scope to ScopeActiveFeatures.
	FromActiveFeatures = taskid.ScopeActiveFeatures
	// FromActiveGraph configures the dependency scope to ScopeActiveGraph.
	FromActiveGraph = taskid.ScopeActiveGraph
	// FromAll configures the dependency scope to ScopeAll.
	FromAll = taskid.ScopeAll
)

// TagReference defines a typed reference to an aggregation of tasks providing a specific tag.
type TagReference[TaskResult any] interface {
	taskid.FanInDescriptor
	// GetZeroValue returns a zero value of the TaskResult type to preserve type safety.
	GetZeroValue() TaskResult
}

type tagReferenceImpl[TaskResult any] struct {
	tag    string
	config taskid.DependencyConfig
}

var _ TagReference[any] = (*tagReferenceImpl[any])(nil)
var _ taskid.FanInDescriptor = (*tagReferenceImpl[any])(nil)

// DescriptorCardinality returns that this dependency is a fan-in aggregation.
func (t *tagReferenceImpl[TaskResult]) DescriptorCardinality() taskid.EdgeCardinality {
	return t.config.Cardinality
}

// DescriptorScope returns the effective dependency resolution scope.
func (t *tagReferenceImpl[TaskResult]) DescriptorScope() taskid.DependencyScope {
	return t.config.Scope
}

// Tag returns the tag name to match producer tasks.
func (t *tagReferenceImpl[TaskResult]) Tag() string {
	return t.tag
}

// GetZeroValue returns a zero value of the TaskResult type.
func (t *tagReferenceImpl[TaskResult]) GetZeroValue() TaskResult {
	var zero TaskResult
	return zero
}

// NewTagReference creates a new TagReference for the specified tag and options.
func NewTagReference[TaskResult any](tag string, opts ...taskid.FanInOption) TagReference[TaskResult] {
	cfg := taskid.NewDefaultFanInConfig()
	for _, opt := range opts {
		taskid.ApplyFanInOption(&cfg, opt)
	}
	return &tagReferenceImpl[TaskResult]{
		tag:    tag,
		config: cfg,
	}
}
