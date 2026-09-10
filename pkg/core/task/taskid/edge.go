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

package taskid

// EdgeCardinality represents the multiplicity of the dependency.
type EdgeCardinality int

const (
	// CardinalityPointToPoint represents a 1:1 dependency targeting a specific single task.
	CardinalityPointToPoint EdgeCardinality = iota

	// CardinalityFanIn represents a 1:N aggregation dependency targeting all tasks matching a tag.
	CardinalityFanIn
)

// DependencyScope represents the search and binding scope for dependencies.
type DependencyScope int

const (
	// ScopeUnspecified indicates that no dependency scope was explicitly specified.
	ScopeUnspecified DependencyScope = iota

	// ScopeActiveGraph binds only to tasks already included in the active graph (lazy).
	// It never pulls upstream tasks into the graph by itself.
	ScopeActiveGraph

	// ScopeActiveFeatures pulls in producer tasks only if all of their upstream dependencies
	// merge into tasks that are already in the active graph.
	ScopeActiveFeatures

	// ScopeAll searches all available tasks and eagerly pulls them into the graph.
	ScopeAll
)

// DependencyDescriptor is the base interface that all dependency descriptors must implement.
// It has 0 internal package dependencies and consists only of standard Go types and taskid enums.
type DependencyDescriptor interface {
	// DescriptorCardinality returns whether this dependency is point-to-point or fan-in.
	DescriptorCardinality() EdgeCardinality
	// DescriptorScope returns the effective dependency resolution scope.
	DescriptorScope() DependencyScope
}

// PointToPointDescriptor is an interface for 1:1 dependencies targeting a single task by reference ID.
type PointToPointDescriptor interface {
	DependencyDescriptor
	// ReferenceID returns the target task's reference ID without any implementation hash.
	ReferenceID() string
}

// FanInDescriptor is an interface for 1:N aggregation dependencies targeting tasks matching a tag.
type FanInDescriptor interface {
	DependencyDescriptor
	// Tag returns the tag identifier to match against producer tasks.
	Tag() string
}

// DependencyConfig holds edge attribute configuration values applied by dependency options.
type DependencyConfig struct {
	// Cardinality specifies whether the dependency is a 1:1 reference or a 1:N fan-in reference.
	Cardinality EdgeCardinality
	// Scope specifies the task search and binding scope for resolving the dependency.
	Scope DependencyScope
}

// NewDefaultPointToPointConfig returns a DependencyConfig initialized for a standard point-to-point data dependency.
func NewDefaultPointToPointConfig() DependencyConfig {
	return DependencyConfig{
		Cardinality: CardinalityPointToPoint,
		Scope:       ScopeAll,
	}
}

// NewDefaultFanInConfig returns a DependencyConfig initialized for a standard fan-in data dependency.
func NewDefaultFanInConfig() DependencyConfig {
	return DependencyConfig{
		Cardinality: CardinalityFanIn,
		Scope:       ScopeActiveFeatures,
	}
}

// ReferenceOption is an option that can be applied to a point-to-point task reference.
type ReferenceOption interface {
	applyReference(*DependencyConfig)
}

// FanInOption is an option that can be applied to a fan-in tag reference.
type FanInOption interface {
	applyFanIn(*DependencyConfig)
}

var (
	_ ReferenceOption = DependencyScope(0)
	_ FanInOption     = DependencyScope(0)
)

func (s DependencyScope) applyReference(c *DependencyConfig) {
	if s == ScopeUnspecified {
		return
	}
	c.Scope = s
}

func (s DependencyScope) applyFanIn(c *DependencyConfig) {
	if s == ScopeAll {
		panic("ScopeAll is not supported for fan-in dependencies")
	}
	s.applyReference(c)
}

// ApplyReferenceOption applies a ReferenceOption to a DependencyConfig.
func ApplyReferenceOption(c *DependencyConfig, opt ReferenceOption) {
	opt.applyReference(c)
}

// ApplyFanInOption applies a FanInOption to a DependencyConfig.
func ApplyFanInOption(c *DependencyConfig, opt FanInOption) {
	opt.applyFanIn(c)
}

// TaskEdge represents a concrete directed edge in the resolved task DAG.
type TaskEdge struct {
	// SourceRefID is the reference ID of the upstream producer task.
	SourceRefID string
	// SourceImplID is the implementation ID of the upstream producer task.
	SourceImplID string
	// TargetImplID is the implementation ID of the downstream consumer task.
	TargetImplID string
	// Cardinality indicates whether this edge originated from a point-to-point or fan-in dependency.
	Cardinality EdgeCardinality
	// Tag is the tag identifier if this edge originated from a fan-in dependency.
	Tag string
	// Priority specifies the precedence weight of this edge during graph resolution and cycle pruning.
	Priority int
}
