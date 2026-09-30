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

package tasktest

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
)

// InputValue is a value that Run gives to one input of the task under test.
// Create it with Given or GivenTag.
type InputValue interface {
	// dependency returns the dependency descriptor the value is given for.
	dependency() taskid.DependencyDescriptor
	// valueType returns the static type of the given value.
	valueType() reflect.Type
	// store writes the value into the harness state.
	store(state *harnessState)
}

// Given returns an InputValue that supplies value to the input declared with ref.
// Use it for inputs declared with coretask.Use or coretask.UseOptional.
func Given[T any](ref taskid.TaskReference[T], value T) InputValue {
	return givenValue[T]{ref: ref, value: value}
}

// GivenTag returns an InputValue that supplies values as the results of the producers of the tag referenced by ref.
// Use it for inputs declared with coretask.UseTag. The task reads the values in the given order.
func GivenTag[T any](ref coretask.TagReference[T], values ...T) InputValue {
	return givenTagValues[T]{ref: ref, values: values}
}

// Run validates inputs against the inputs declared by task and runs the task with them.
// It fails t when a required input is missing, when a value is given for an undeclared or ordering-only input,
// when the same input is given twice, or when a value cannot be read as the declared input type.
// Inputs declared with coretask.UseOptional read (zero, false) and tag inputs read an empty slice when no value is given.
func Run[T any](t testing.TB, ctx context.Context, task coretask.DefinedTask[T], inputs ...InputValue) (T, error) {
	t.Helper()
	if problems := validateInputs(task.Inputs(), inputs); len(problems) > 0 {
		t.Fatalf("task %s has invalid test inputs:\n%s", task.UntypedID(), strings.Join(problems, "\n"))
	}
	state := newHarnessState()
	for _, input := range inputs {
		input.store(state)
	}
	ctx = khictx.WithValue(ctx, core_contract.TaskImplementationIDContextKey, task.UntypedID())
	ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, state.results)
	ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, state.metadata)
	return task.Run(ctx)
}

// validateInputs compares the given values with the declared inputs and returns a description of each problem.
// Problems for given values come first in the given order, followed by missing required inputs in declaration order.
func validateInputs(specs []coretask.InputSpec, inputs []InputValue) []string {
	declared := make(map[string]coretask.InputSpec, len(specs))
	for _, spec := range specs {
		declared[inputKey(spec.Dependency)] = spec
	}
	var problems []string
	given := map[string]struct{}{}
	for _, input := range inputs {
		key := inputKey(input.dependency())
		if _, found := given[key]; found {
			problems = append(problems, fmt.Sprintf("input %s is given more than once", key))
			continue
		}
		given[key] = struct{}{}
		spec, found := declared[key]
		switch {
		case !found:
			problems = append(problems, fmt.Sprintf("undeclared input %s is given", key))
		case spec.Kind == coretask.InputKindOrdering:
			problems = append(problems, fmt.Sprintf("input %s is an ordering-only dependency and does not take a value", key))
		case !input.valueType().AssignableTo(spec.Dependency.ResultType()):
			problems = append(problems, fmt.Sprintf("input %s expects %s, but the given value is %s", key, spec.Dependency.ResultType(), input.valueType()))
		}
	}
	for _, spec := range specs {
		key := inputKey(spec.Dependency)
		if _, found := given[key]; spec.Kind == coretask.InputKindRequired && !found {
			problems = append(problems, fmt.Sprintf("missing required input %s (%s)", key, spec.Dependency.ResultType()))
		}
	}
	return problems
}

// inputKey identifies an input by the reference ID of a point-to-point dependency or the tag of a fan-in dependency.
func inputKey(dep taskid.DependencyDescriptor) string {
	switch d := dep.(type) {
	case taskid.PointToPointDescriptor:
		return "ref:" + d.ReferenceID()
	case taskid.FanInDescriptor:
		return "tag:" + d.Tag()
	default:
		panic(fmt.Sprintf("unsupported dependency descriptor %T", dep))
	}
}

// harnessState holds the task results and graph metadata that Run passes to the task under test.
type harnessState struct {
	results  *typedmap.TypedMap
	metadata *harnessGraphMetadata
}

func newHarnessState() *harnessState {
	return &harnessState{
		results: typedmap.NewTypedMap(),
		metadata: &harnessGraphMetadata{
			boundReferenceIDs:         map[string]struct{}{},
			producerReferenceIDsByTag: map[string][]string{},
		},
	}
}

// harnessGraphMetadata reports the given values as the only producers in the graph.
type harnessGraphMetadata struct {
	boundReferenceIDs         map[string]struct{}
	producerReferenceIDsByTag map[string][]string
}

var _ core_contract.TaskGraphMetadata = (*harnessGraphMetadata)(nil)

// IsBound implements core_contract.TaskGraphMetadata.
func (m *harnessGraphMetadata) IsBound(referenceID string) bool {
	_, found := m.boundReferenceIDs[referenceID]
	return found
}

// BoundReferenceIDsForTaskImplWithTag implements core_contract.TaskGraphMetadata.
// The harness runs a single task, so the producers do not depend on the task implementation ID.
func (m *harnessGraphMetadata) BoundReferenceIDsForTaskImplWithTag(taskImplementationID string, tag string) []string {
	return m.producerReferenceIDsByTag[tag]
}

type givenValue[T any] struct {
	ref   taskid.TaskReference[T]
	value T
}

var _ InputValue = givenValue[any]{}

func (g givenValue[T]) dependency() taskid.DependencyDescriptor {
	return g.ref
}

func (g givenValue[T]) valueType() reflect.Type {
	return reflect.TypeFor[T]()
}

func (g givenValue[T]) store(state *harnessState) {
	referenceID := g.ref.ReferenceIDString()
	typedmap.Set(state.results, typedmap.NewTypedKey[T](referenceID), g.value)
	state.metadata.boundReferenceIDs[referenceID] = struct{}{}
}

type givenTagValues[T any] struct {
	ref    coretask.TagReference[T]
	values []T
}

var _ InputValue = givenTagValues[any]{}

func (g givenTagValues[T]) dependency() taskid.DependencyDescriptor {
	return g.ref
}

func (g givenTagValues[T]) valueType() reflect.Type {
	return reflect.TypeFor[T]()
}

// store registers each value as the result of a synthetic producer of the tag.
func (g givenTagValues[T]) store(state *harnessState) {
	tag := g.ref.Tag()
	for i, value := range g.values {
		referenceID := fmt.Sprintf("tasktest.given/%s/%d", tag, i)
		typedmap.Set(state.results, typedmap.NewTypedKey[T](referenceID), value)
		state.metadata.producerReferenceIDsByTag[tag] = append(state.metadata.producerReferenceIDsByTag[tag], referenceID)
	}
}
