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

// TaskImplementationId is a unique value associated with each task implementation.
// TaskReferenceId is an ID that can be used to refer tasks in the dependencies and does not
// have a task suffix.

// For example, "foo.bar" can be used as both a TaskImplementationId and a
// TaskReferenceId, but "foo.bar#qux" cannot be used as a TaskReferenceId because it
// has a suffix.

// TODO: rewrite comments above.

type UntypedTaskReference interface {
	String() string
	ReferenceIDString() string
}

type TaskReference[TaskResult any] interface {
	UntypedTaskReference
	// GetZeroValue is only needed to make sure TaskReference[A] != TaskReference[B]
	GetZeroValue() TaskResult
}

type UntypedTaskImplementationID interface {
	String() string
	ReferenceIDString() string
	GetTaskImplementationHash() string
	GetUntypedReference() UntypedTaskReference
}

type TaskImplementationID[TaskResult any] interface {
	UntypedTaskImplementationID
	GetTaskReference() TaskReference[TaskResult]
}

type taskReferenceImpl[TaskResult any] struct {
	id string
}

func (t taskReferenceImpl[TaskResult]) String() string {
	return t.id
}

func (t taskReferenceImpl[TaskResult]) GetZeroValue() TaskResult {
	return *new(TaskResult)
}

type taskImplementationIDImpl[TaskResult any] struct {
	referenceId        string
	implementationHash string
}

func (t taskImplementationIDImpl[TaskResult]) String() string {
	return t.referenceId + "#" + t.implementationHash
}

func (t taskImplementationIDImpl[TaskResult]) GetTaskReference() TaskReference[TaskResult] {
	return taskReferenceImpl[TaskResult]{id: t.referenceId}
}

func (t taskReferenceImpl[TaskResult]) ReferenceIDString() string {
	return t.String()
}

func (t taskImplementationIDImpl[TaskResult]) ReferenceIDString() string {
	return t.referenceId
}

func (t taskImplementationIDImpl[TaskResult]) GetTaskImplementationHash() string {
	return t.implementationHash
}

func (t taskImplementationIDImpl[TaskResult]) GetUntypedReference() UntypedTaskReference {
	return t.GetTaskReference()
}

func NewTaskReference[TaskResult any](id string) TaskReference[TaskResult] {
	if strings.Contains(id, "#") {
		panic(fmt.Sprintf("reference id %s is invalid. It cannot contain '#' in reference ID", id))
	}
	return taskReferenceImpl[TaskResult]{id: id}
}

func NewDefaultImplementationID[TaskResult any](id string) TaskImplementationID[TaskResult] {
	if strings.Contains(id, "#") {
		panic(fmt.Sprintf("task id %s is invalid. It cannot contain '#' on NewDefaultImplementationID. Use NewImplementationID instead to use a custom implementation hash.", id))
	}
	return taskImplementationIDImpl[TaskResult]{referenceId: id, implementationHash: "default"}
}

func NewImplementationID[TaskResult any](baseReference TaskReference[TaskResult], implementationHash string) TaskImplementationID[TaskResult] {
	return taskImplementationIDImpl[TaskResult]{referenceId: baseReference.String(), implementationHash: implementationHash}
}
