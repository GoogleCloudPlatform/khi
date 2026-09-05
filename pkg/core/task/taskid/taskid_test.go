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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewTaskReference(t *testing.T) {
	testCases := []struct {
		name            string
		id              string
		wantString      string
		wantReferenceID string
		expectPanic     bool
	}{
		{
			name:            "valid task reference ID",
			id:              "foo.bar",
			wantString:      "foo.bar",
			wantReferenceID: "foo.bar",
		},
		{
			name:        "invalid task reference ID containing hash",
			id:          "foo.bar#qux",
			expectPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected panic for %s, but did not panic", tc.name)
					}
				}()
			}

			ref := NewTaskReference[any](tc.id)
			if tc.expectPanic {
				return
			}

			if diff := cmp.Diff(tc.wantString, ref.String()); diff != "" {
				t.Errorf("String() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantReferenceID, ref.ReferenceIDString()); diff != "" {
				t.Errorf("ReferenceIDString() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTaskImplementationID(t *testing.T) {
	testCases := []struct {
		name                   string
		setup                  func() (UntypedTaskImplementationID, error)
		wantString             string
		wantReferenceIDString  string
		wantImplementationHash string
		expectPanic            bool
	}{
		{
			name: "NewDefaultImplementationID with valid ID",
			setup: func() (UntypedTaskImplementationID, error) {
				return NewDefaultImplementationID[string]("task.alpha"), nil
			},
			wantString:             "task.alpha#default",
			wantReferenceIDString:  "task.alpha",
			wantImplementationHash: "default",
		},
		{
			name: "NewDefaultImplementationID with hash in ID panics",
			setup: func() (UntypedTaskImplementationID, error) {
				return NewDefaultImplementationID[string]("task.alpha#invalid"), nil
			},
			expectPanic: true,
		},
		{
			name: "NewImplementationID with custom hash",
			setup: func() (UntypedTaskImplementationID, error) {
				baseRef := NewTaskReference[string]("task.beta")
				return NewImplementationID[string](baseRef, "custom-impl"), nil
			},
			wantString:             "task.beta#custom-impl",
			wantReferenceIDString:  "task.beta",
			wantImplementationHash: "custom-impl",
		},
		{
			name: "NewImplementationID with invalid hash containing hash symbol panics",
			setup: func() (UntypedTaskImplementationID, error) {
				baseRef := NewTaskReference[string]("task.beta")
				return NewImplementationID[string](baseRef, "custom#impl"), nil
			},
			expectPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected panic for %s, but did not panic", tc.name)
					}
				}()
			}

			implID, _ := tc.setup()
			if tc.expectPanic {
				return
			}

			if diff := cmp.Diff(tc.wantString, implID.String()); diff != "" {
				t.Errorf("String() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantReferenceIDString, implID.ReferenceIDString()); diff != "" {
				t.Errorf("ReferenceIDString() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantImplementationHash, implID.GetTaskImplementationHash()); diff != "" {
				t.Errorf("GetTaskImplementationHash() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantReferenceIDString, implID.GetUntypedReference().ReferenceIDString()); diff != "" {
				t.Errorf("GetUntypedReference().ReferenceIDString() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTaskReferenceDescriptor(t *testing.T) {
	testCases := []struct {
		name            string
		ref             UntypedTaskReference
		wantKind        EdgeKind
		wantCondition   EdgeCondition
		wantCardinality EdgeCardinality
		wantScope       DependencyScope
		wantRefID       string
	}{
		{
			name:            "default TaskImplementationID.Ref()",
			ref:             NewDefaultImplementationID[string]("foo.bar").Ref(),
			wantKind:        EdgeKindData,
			wantCondition:   ConditionRequired,
			wantCardinality: CardinalityPointToPoint,
			wantScope:       ScopeAll,
			wantRefID:       "foo.bar",
		},
		{
			name:            "Ref() with Optional",
			ref:             NewDefaultImplementationID[string]("foo.bar").Ref(Optional),
			wantKind:        EdgeKindData,
			wantCondition:   ConditionOptional,
			wantCardinality: CardinalityPointToPoint,
			wantScope:       ScopeActiveGraph,
			wantRefID:       "foo.bar",
		},
		{
			name:            "Ref() with Optional and FromActiveFeatures",
			ref:             NewDefaultImplementationID[string]("foo.bar").Ref(Optional, FromActiveFeatures),
			wantKind:        EdgeKindData,
			wantCondition:   ConditionOptional,
			wantCardinality: CardinalityPointToPoint,
			wantScope:       ScopeActiveFeatures,
			wantRefID:       "foo.bar",
		},
		{
			name:            "Ref() with OrderOnly",
			ref:             NewDefaultImplementationID[string]("foo.bar").Ref(OrderOnly),
			wantKind:        EdgeKindOrderOnly,
			wantCondition:   ConditionRequired,
			wantCardinality: CardinalityPointToPoint,
			wantScope:       ScopeAll,
			wantRefID:       "foo.bar",
		},
		{
			name:            "Ref() with Optional, FromActiveFeatures, and OrderOnly",
			ref:             NewDefaultImplementationID[string]("foo.bar").Ref(Optional, FromActiveFeatures, OrderOnly),
			wantKind:        EdgeKindOrderOnly,
			wantCondition:   ConditionOptional,
			wantCardinality: CardinalityPointToPoint,
			wantScope:       ScopeActiveFeatures,
			wantRefID:       "foo.bar",
		},
		{
			name:            "NewTaskReference with Optional",
			ref:             NewTaskReference[int]("baz.qux", Optional),
			wantKind:        EdgeKindData,
			wantCondition:   ConditionOptional,
			wantCardinality: CardinalityPointToPoint,
			wantScope:       ScopeActiveGraph,
			wantRefID:       "baz.qux",
		},
		{
			name:            "NewTaskReference with OrderOnly",
			ref:             NewTaskReference[int]("base.task", OrderOnly),
			wantKind:        EdgeKindOrderOnly,
			wantCondition:   ConditionRequired,
			wantCardinality: CardinalityPointToPoint,
			wantScope:       ScopeAll,
			wantRefID:       "base.task",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotKind := tc.ref.DescriptorKind()
			gotCondition := tc.ref.DescriptorCondition()
			gotCardinality := tc.ref.DescriptorCardinality()
			gotScope := tc.ref.DescriptorScope()
			gotRefID := tc.ref.ReferenceID()

			if diff := cmp.Diff(tc.wantKind, gotKind); diff != "" {
				t.Errorf("DescriptorKind() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantCondition, gotCondition); diff != "" {
				t.Errorf("DescriptorCondition() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantCardinality, gotCardinality); diff != "" {
				t.Errorf("DescriptorCardinality() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantScope, gotScope); diff != "" {
				t.Errorf("DescriptorScope() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantRefID, gotRefID); diff != "" {
				t.Errorf("ReferenceID() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTaskReferenceGetZeroValue(t *testing.T) {
	refInt := NewTaskReference[int]("int.task")
	if diff := cmp.Diff(0, refInt.GetZeroValue()); diff != "" {
		t.Errorf("GetZeroValue() mismatch (-want +got):\n%s", diff)
	}

	refStr := NewTaskReference[string]("str.task")
	if diff := cmp.Diff("", refStr.GetZeroValue()); diff != "" {
		t.Errorf("GetZeroValue() mismatch (-want +got):\n%s", diff)
	}
}
