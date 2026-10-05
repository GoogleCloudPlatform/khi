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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// InputKind classifies how a task declared an input through the Binder.
type InputKind int

const (
	// InputKindRequired is an input declared with Use. The producer must be in the graph.
	InputKindRequired InputKind = iota
	// InputKindOptional is an input declared with UseOptional. The producer may be absent from the graph.
	InputKindOptional
	// InputKindTag is a fan-in input declared with UseTag over all producers of a tag.
	InputKindTag
	// InputKindOrdering is a dependency declared with After. The task waits for it but never reads its value.
	InputKindOrdering
)

// String returns a human readable name of the input kind.
// It panics for values other than the declared InputKind constants because they indicate a programming error.
func (k InputKind) String() string {
	switch k {
	case InputKindRequired:
		return "required"
	case InputKindOptional:
		return "optional"
	case InputKindTag:
		return "tag"
	case InputKindOrdering:
		return "ordering"
	default:
		panic(fmt.Sprintf("unknown InputKind %d", int(k)))
	}
}

// InputSpec describes one input declared on a Binder.
type InputSpec struct {
	// Dependency is the dependency descriptor registered in the task graph for this input.
	Dependency Dependency
	// Kind is how the input was declared.
	Kind InputKind
}

// precedence ranks how strongly a kind binds the task to its producer when the same input is declared more than once.
// Required and tag never share a key because references and tags use different key prefixes, so they can share a rank.
func (k InputKind) precedence() int {
	switch k {
	case InputKindRequired, InputKindTag:
		return 2
	case InputKindOptional:
		return 1
	case InputKindOrdering:
		return 0
	default:
		panic(fmt.Sprintf("unknown InputKind %d", int(k)))
	}
}

// Binder collects the inputs of a task defined with Define.
// Inputs can only be declared while the bind function passed to Define is running.
type Binder struct {
	owner taskid.UntypedTaskImplementationID
	specs []InputSpec
	// keys maps the dependency key of each declared input to its index in specs.
	keys   map[string]int
	sealed bool
}

func newBinder(owner taskid.UntypedTaskImplementationID) *Binder {
	return &Binder{
		owner: owner,
		keys:  map[string]int{},
	}
}

// add registers an input. Declaring the same input more than once merges the declarations into one spec,
// so that a wrapper and the bind function it wraps can both declare an input they read.
// It panics when the Binder is sealed or when the declarations disagree on the result type,
// because both are programming errors that must fail at package initialization.
// Bind-time panic messages omit the task ID because Define prefixes them with it.
func (b *Binder) add(dep Dependency, kind InputKind) {
	key := dependencyKey(dep)
	if b.sealed {
		panic(fmt.Sprintf("task %s: declares input %s after its bind function returned; declare inputs in the bind function, not in the run function", b.owner, key))
	}
	i, found := b.keys[key]
	if !found {
		b.keys[key] = len(b.specs)
		b.specs = append(b.specs, InputSpec{Dependency: dep, Kind: kind})
		return
	}
	existing := b.specs[i]
	if existing.Dependency.ResultType() != dep.ResultType() {
		panic(fmt.Sprintf("declares input %s with conflicting result types %s and %s", key, existing.Dependency.ResultType(), dep.ResultType()))
	}
	b.specs[i] = mergeInputSpecs(existing, InputSpec{Dependency: dep, Kind: kind})
}

// mergeInputSpecs merges two declarations of the same input into the one that binds the task most strongly.
// It keeps the kind with the higher precedence and the dependency with the broader scope.
func mergeInputSpecs(a, b InputSpec) InputSpec {
	merged := a
	if b.Kind.precedence() > merged.Kind.precedence() {
		merged.Kind = b.Kind
	}
	// DependencyScope constants are declared from the narrowest to the broadest scope.
	if b.Dependency.DescriptorScope() > merged.Dependency.DescriptorScope() {
		merged.Dependency = b.Dependency
	}
	// An ordering dependency with ScopeAll pulls the producer into the graph, so an optional read of it is effectively required.
	if merged.Kind == InputKindOptional && merged.Dependency.DescriptorScope() == taskid.ScopeAll {
		merged.Kind = InputKindRequired
	}
	return merged
}

// dependencies returns the dependency descriptors of all declared inputs in declaration order.
func (b *Binder) dependencies() []Dependency {
	deps := make([]Dependency, 0, len(b.specs))
	for _, spec := range b.specs {
		deps = append(deps, spec.Dependency)
	}
	return deps
}

// Use declares a required input and returns the handle to read its value.
// The reference must use ScopeAll, the default scope, so that the producer is always pulled into the graph.
// Use UseOptional for references with a narrower scope. Define prefixes the panic for a wrong scope with the task ID.
func Use[T any](b *Binder, ref taskid.TaskReference[T]) Input[T] {
	if ref.DescriptorScope() != taskid.ScopeAll {
		panic(fmt.Sprintf("declares required input %s with a scope other than ScopeAll; use UseOptional for inputs whose producer may be absent", ref.ReferenceIDString()))
	}
	b.add(ref, InputKindRequired)
	return Input[T]{owner: b.owner, ref: ref}
}

// UseOptional declares an input whose producer may be absent from the graph and returns the handle to read it.
// The reference must use a scope narrower than ScopeAll, such as ScopeActiveGraph or ScopeActiveFeatures.
// Define prefixes the panic for a wrong scope with the task ID.
func UseOptional[T any](b *Binder, ref taskid.TaskReference[T]) OptionalInput[T] {
	if ref.DescriptorScope() == taskid.ScopeAll {
		panic(fmt.Sprintf("declares optional input %s with ScopeAll; pass a reference with a narrower scope such as FromActiveGraph, or use Use for a required input", ref.ReferenceIDString()))
	}
	b.add(ref, InputKindOptional)
	return OptionalInput[T]{owner: b.owner, ref: ref}
}

// UseTag declares a fan-in input over all producers of a tag and returns the handle to read their results.
func UseTag[T any](b *Binder, ref TagReference[T]) TagInput[T] {
	b.add(ref, InputKindTag)
	return TagInput[T]{owner: b.owner, ref: ref}
}

// After declares a dependency that the task only waits for. It intentionally returns no handle,
// so the task cannot read the value and the declaration cannot become an unused input.
func After(b *Binder, dep Dependency) {
	b.add(dep, InputKindOrdering)
}
