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

// Binder collects the inputs of a task defined with Define.
// Inputs can only be declared while the bind function passed to Define is running.
type Binder struct {
	owner  taskid.UntypedTaskImplementationID
	specs  []InputSpec
	keys   map[string]struct{}
	sealed bool
}

func newBinder(owner taskid.UntypedTaskImplementationID) *Binder {
	return &Binder{
		owner: owner,
		keys:  map[string]struct{}{},
	}
}

// add registers an input. It panics when the dependency is nil, when the Binder is sealed or when the same input is declared twice,
// because all of them are programming errors that must fail at package initialization.
func (b *Binder) add(dep Dependency, kind InputKind) {
	if dep == nil {
		panic(fmt.Sprintf(`task %s declares a nil input. This may be caused because of initialization order issue of global variables.
Please define task IDs and types used in its type parameter in a different package.`, b.owner))
	}
	if b.sealed {
		panic(fmt.Sprintf("task %s declares input %s after its bind function returned; declare inputs in the bind function, not in the run function", b.owner, dependencyKey(dep)))
	}
	key := dependencyKey(dep)
	if _, found := b.keys[key]; found {
		panic(fmt.Sprintf("task %s declares input %s twice", b.owner, key))
	}
	b.keys[key] = struct{}{}
	b.specs = append(b.specs, InputSpec{Dependency: dep, Kind: kind})
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
// Use UseOptional for references with a narrower scope.
func Use[T any](b *Binder, ref taskid.TaskReference[T]) Input[T] {
	if ref.DescriptorScope() != taskid.ScopeAll {
		panic(fmt.Sprintf("task %s declares required input %s with a scope other than ScopeAll; use UseOptional for inputs whose producer may be absent", b.owner, ref.ReferenceIDString()))
	}
	b.add(ref, InputKindRequired)
	return Input[T]{owner: b.owner, ref: ref}
}

// UseOptional declares an input whose producer may be absent from the graph and returns the handle to read it.
// The reference must use a scope narrower than ScopeAll, such as ScopeActiveGraph or ScopeActiveFeatures.
func UseOptional[T any](b *Binder, ref taskid.TaskReference[T]) OptionalInput[T] {
	if ref.DescriptorScope() == taskid.ScopeAll {
		panic(fmt.Sprintf("task %s declares optional input %s with ScopeAll; pass a reference with a narrower scope such as FromActiveGraph, or use Use for a required input", b.owner, ref.ReferenceIDString()))
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
