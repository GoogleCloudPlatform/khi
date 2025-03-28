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

package taskid

import (
	"fmt"
	"strings"
)

// TaskImplementationID represents a unique identifier for a specific task implementation.
// It is formatted as "ReferenceID#ImplementationHash" where ReferenceID identifies the type of task
// and ImplementationHash identifies the specific implementation of that task type.

// TaskReference is an ID used to refer to tasks in dependencies. It consists of
// only the ReferenceID part without the implementation hash suffix.

// For example, "foo.bar#qux" is a TaskImplementationID where "foo.bar" is the ReferenceID
// and "qux" is the ImplementationHash. Multiple implementations can share the same ReferenceID
// but must produce the same type of output (e.g., "foo.bar#qux" and "foo.bar#baz").

// This dual-ID system allows tasks to depend on a common interface (via TaskReference)
// while the actual implementation used (TaskImplementationID) can vary based on context.

// UntypedTaskReference defines the interface for task references without type information.
// This allows the task system to handle references generically when exact types are not needed.
type UntypedTaskReference interface {
	// String returns the string representation of the reference ID.
	String() string
	// ReferenceIDString returns the reference ID portion without any implementation hash.
	ReferenceIDString() string
}

// TaskReference defines a typed reference to a task that produces a specific result type.
// The type parameter ensures that dependencies between tasks maintain type safety.
type TaskReference[TaskResult any] interface {
	UntypedTaskReference
	// GetZeroValue returns a zero value of the TaskResult type.
	// This is used to maintain type safety by ensuring TaskReference[A] and TaskReference[B]
	// are considered different types when A and B are different.
	GetZeroValue() TaskResult
}

// UntypedTaskImplementationID defines the interface for task implementation IDs
// without type information. This allows the task system to handle task IDs
// generically when exact types are not needed.
type UntypedTaskImplementationID interface {
	// String returns the full string representation of the task implementation ID (ReferenceID#ImplementationHash).
	String() string
	// ReferenceIDString returns only the reference ID portion without the implementation hash.
	ReferenceIDString() string
	// GetTaskImplementationHash returns the implementation-specific hash part of the ID.
	GetTaskImplementationHash() string
	// GetUntypedReference returns the reference ID associated with this implementation ID.
	GetUntypedReference() UntypedTaskReference
}

// TaskImplementationID defines a typed implementation ID for a task that produces a specific result type.
// The type parameter ensures that implementations maintain type safety with their references.
type TaskImplementationID[TaskResult any] interface {
	UntypedTaskImplementationID
	// GetTaskReference returns the typed reference associated with this implementation ID.
	GetTaskReference() TaskReference[TaskResult]
}

// taskReferenceImpl implements the TaskReference interface for a specific result type.
type taskReferenceImpl[TaskResult any] struct {
	id string
}

// String returns the string representation of the reference ID.
func (t taskReferenceImpl[TaskResult]) String() string {
	return t.id
}

// GetZeroValue returns a zero value of the TaskResult type.
// This is used to distinguish between TaskReference[A] and TaskReference[B] at the type level.
func (t taskReferenceImpl[TaskResult]) GetZeroValue() TaskResult {
	return *new(TaskResult)
}

// taskImplementationIDImpl implements the TaskImplementationID interface for a specific result type.
type taskImplementationIDImpl[TaskResult any] struct {
	referenceId        string
	implementationHash string
}

// String returns the full string representation of the implementation ID in the format "referenceId#implementationHash".
func (t taskImplementationIDImpl[TaskResult]) String() string {
	return t.referenceId + "#" + t.implementationHash
}

// GetTaskReference returns a TaskReference associated with this implementation ID.
// This allows accessing the reference from the implementation, enabling type-safe dependencies.
func (t taskImplementationIDImpl[TaskResult]) GetTaskReference() TaskReference[TaskResult] {
	return taskReferenceImpl[TaskResult]{id: t.referenceId}
}

// ReferenceIDString returns the reference ID portion of the task reference.
// For taskReferenceImpl, this is the same as String().
func (t taskReferenceImpl[TaskResult]) ReferenceIDString() string {
	return t.String()
}

// ReferenceIDString returns only the reference ID portion of the implementation ID, without the hash.
func (t taskImplementationIDImpl[TaskResult]) ReferenceIDString() string {
	return t.referenceId
}

// GetTaskImplementationHash returns the implementation-specific hash part of the ID.
// This distinguishes between different implementations of the same reference.
func (t taskImplementationIDImpl[TaskResult]) GetTaskImplementationHash() string {
	return t.implementationHash
}

// GetUntypedReference returns the reference ID associated with this implementation ID as an UntypedTaskReference.
// This allows working with references without knowledge of their specific result types.
func (t taskImplementationIDImpl[TaskResult]) GetUntypedReference() UntypedTaskReference {
	return t.GetTaskReference()
}

// NewTaskReference creates a new TaskReference with the specified ID.
// This function is used to create references to tasks that can be used in dependencies.
// The ID cannot contain '#' as it would be confused with an implementation hash.
// Typically used to define the interface of a task that other tasks can depend on.
func NewTaskReference[TaskResult any](id string) TaskReference[TaskResult] {
	if strings.Contains(id, "#") {
		panic(fmt.Sprintf("reference id %s is invalid. It cannot contain '#' in reference ID\nThis is likely a bug in the KHI task implementation or an incorrect ID was provided in the taskid definition.\nPlease report a bug at https://github.com/GoogleCloudPlatform/khi/issues", id))
	}
	return taskReferenceImpl[TaskResult]{id: id}
}

// NewDefaultImplementationID creates a new TaskImplementationID with the "default" implementation hash.
// This is a convenience function for creating standard task implementations.
// The ID cannot contain '#' as the function will append "#default" to create the full implementation ID.
// Typically used when there is only one common implementation of a task reference.
func NewDefaultImplementationID[TaskResult any](id string) TaskImplementationID[TaskResult] {
	if strings.Contains(id, "#") {
		panic(fmt.Sprintf("task id %s is invalid. It cannot contain '#' in NewDefaultImplementationID. Use NewImplementationID instead to use a custom implementation hash.\nThis is likely a bug in the KHI task implementation or an incorrect ID was provided in the taskid definition.\nPlease report a bug at https://github.com/GoogleCloudPlatform/khi/issues", id))
	}
	return taskImplementationIDImpl[TaskResult]{referenceId: id, implementationHash: "default"}
}

// NewImplementationID creates a new TaskImplementationID with a custom implementation hash.
// This function is used when multiple different implementations of the same task reference are needed.
// The implementation hash distinguishes between different implementations that produce the same type of result.
// For example, a log parser task could have different implementations for different log formats,
// but all implementations would share the same TaskReference.
func NewImplementationID[TaskResult any](baseReference TaskReference[TaskResult], implementationHash string) TaskImplementationID[TaskResult] {
	if strings.Contains(implementationHash, "#") {
		panic(fmt.Sprintf("implementation hash %s is invalid. It cannot contain '#' in NewImplementationID.\nThis is likely a bug in the KHI task implementation or an incorrect ID was provided in the taskid definition.\nPlease report a bug at https://github.com/GoogleCloudPlatform/khi/issues", implementationHash))
	}
	return taskImplementationIDImpl[TaskResult]{referenceId: baseReference.String(), implementationHash: implementationHash}
}

// ReinterpretTaskReference casts UntypedTaskReference to TaskReference[T]. Use this with caution.
func ReinterpretTaskReference[T any](ref UntypedTaskReference) TaskReference[T] {
	return ref.(TaskReference[T])
}

// ReinterpretTaskImplementationID casts UntypedImplementationID to TaskImplementationID[T]. Use this with caution.
func ReinterpretTaskImplementationID[T any](id UntypedTaskImplementationID) TaskImplementationID[T] {
	return id.(TaskImplementationID[T])
}
