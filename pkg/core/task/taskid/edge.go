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

// EdgeKind represents whether a dependency edge transfers data or only constrains execution ordering.
type EdgeKind int

const (
	// EdgeKindData indicates a data dependency where downstream consumes upstream output.
	// The upstream task result is retained in memory until all consumers finish.
	EdgeKindData EdgeKind = iota

	// EdgeKindOrderOnly indicates an execution order dependency without data consumption.
	// Downstream waits for upstream completion, but does not read upstream output,
	// allowing upstream memory to be reclaimed without waiting for this downstream.
	EdgeKindOrderOnly
)

// EdgeCondition represents whether a dependency edge is required or optional.
type EdgeCondition int

const (
	// ConditionRequired indicates a required dependency.
	// The upstream task is pulled into the task graph transitively; if missing, resolution fails.
	ConditionRequired EdgeCondition = iota

	// ConditionOptional indicates a conditional dependency.
	// The edge is activated only if upstream is already included in the graph by features or other required tasks.
	// It never pulls the upstream task into the graph by itself.
	ConditionOptional
)

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
	// DescriptorKind returns whether this dependency requires data or only order.
	DescriptorKind() EdgeKind
	// DescriptorCondition returns whether this dependency is required or optional.
	DescriptorCondition() EdgeCondition
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
	// Kind specifies whether the dependency edge transfers data or only enforces execution order.
	Kind EdgeKind
	// Condition specifies whether the dependency is required or optional.
	Condition EdgeCondition
	// Cardinality specifies whether the dependency is a 1:1 reference or a 1:N fan-in reference.
	Cardinality EdgeCardinality
	// Scope specifies the task search and binding scope for resolving the dependency.
	Scope DependencyScope
}

// ResolvedScope returns the effective DependencyScope, falling back to defaults if unspecified.
func (c DependencyConfig) ResolvedScope() DependencyScope {
	if c.Scope != ScopeUnspecified {
		return c.Scope
	}
	if c.Cardinality == CardinalityPointToPoint && c.Condition == ConditionRequired {
		return ScopeAll
	}
	return ScopeActiveGraph
}

// NewDefaultPointToPointConfig returns a DependencyConfig initialized for a standard required data dependency.
func NewDefaultPointToPointConfig() DependencyConfig {
	return DependencyConfig{
		Kind:        EdgeKindData,
		Condition:   ConditionRequired,
		Cardinality: CardinalityPointToPoint,
		Scope:       ScopeUnspecified,
	}
}

// NewDefaultFanInConfig returns a DependencyConfig initialized for a standard required fan-in data dependency.
func NewDefaultFanInConfig() DependencyConfig {
	return DependencyConfig{
		Kind:        EdgeKindData,
		Condition:   ConditionRequired,
		Cardinality: CardinalityFanIn,
		Scope:       ScopeUnspecified,
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

// OrderOnlyOption is an option that can be applied to both point-to-point and fan-in references.
type OrderOnlyOption interface {
	ReferenceOption
	FanInOption
}

// ScopeOption is an option that configures the resolution scope for either point-to-point or fan-in references.
type ScopeOption interface {
	ReferenceOption
	FanInOption
}

type optionalOption struct{}

func (optionalOption) applyReference(c *DependencyConfig) {
	c.Condition = ConditionOptional
}

// Optional configures the dependency condition to ConditionOptional.
// It can only be applied to point-to-point task references.
var Optional ReferenceOption = optionalOption{}

type orderOnlyOption struct{}

func (orderOnlyOption) applyReference(c *DependencyConfig) {
	c.Kind = EdgeKindOrderOnly
}

func (o orderOnlyOption) applyFanIn(c *DependencyConfig) {
	o.applyReference(c)
}

// OrderOnly configures the dependency kind to EdgeKindOrderOnly.
// It can be applied to both point-to-point references and fan-in references.
var OrderOnly OrderOnlyOption = orderOnlyOption{}

type scopeOption DependencyScope

func (s scopeOption) applyReference(c *DependencyConfig) {
	if c.Scope != ScopeUnspecified && c.Scope != DependencyScope(s) {
		panic("dependency scope is already set to a conflicting value")
	}
	c.Scope = DependencyScope(s)
}

func (s scopeOption) applyFanIn(c *DependencyConfig) {
	s.applyReference(c)
}

// FromActiveGraph configures the dependency scope to ScopeActiveGraph.
// It can be applied to both point-to-point and fan-in references.
var FromActiveGraph ScopeOption = scopeOption(ScopeActiveGraph)

// FromActiveFeatures configures the dependency scope to ScopeActiveFeatures.
// It can be applied to both point-to-point and fan-in references.
var FromActiveFeatures ScopeOption = scopeOption(ScopeActiveFeatures)

// FromAll configures the dependency scope to ScopeAll.
// It can be applied to both point-to-point and fan-in references.
var FromAll ScopeOption = scopeOption(ScopeAll)

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
	// TargetID is the implementation ID of the downstream consumer task.
	TargetID string
	// Kind indicates whether this edge transfers data or only execution order.
	Kind EdgeKind
	// Condition indicates whether this edge was required or optional.
	Condition EdgeCondition
	// Cardinality indicates whether this edge originated from a point-to-point or fan-in dependency.
	Cardinality EdgeCardinality
	// Tag is the tag identifier if this edge originated from a fan-in dependency.
	Tag string
}
