// Copyright 2025 Google LLC
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
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	"github.com/google/go-cmp/cmp"
)

type mockGraphMetadata struct {
	boundTasks        map[string]bool
	boundTasksWithTag map[string][]string
}

func (m *mockGraphMetadata) IsBound(referenceID string) bool {
	if m.boundTasks == nil {
		return false
	}
	return m.boundTasks[referenceID]
}

func (m *mockGraphMetadata) BoundReferenceIDsWithTag(tag string) []string {
	if m.boundTasksWithTag == nil {
		return nil
	}
	return m.boundTasksWithTag[tag]
}

var _ core_contract.TaskGraphMetadata = (*mockGraphMetadata)(nil)

func TestWrapErrorWithTaskInformation(t *testing.T) {
	taskID := taskid.NewDefaultImplementationID[any]("foo.com/bar")

	testCases := []struct {
		name                 string
		originalErr          error
		expectedTaskFragment string
	}{
		{
			name:                 "wraps error with task implementation ID",
			originalErr:          errors.New("original error message"),
			expectedTaskFragment: "An error occurred in task `foo.com/bar#default`",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = khictx.WithValue[taskid.UntypedTaskImplementationID](ctx, core_contract.TaskImplementationIDContextKey, taskID)

			wrappedErr := WrapErrorWithTaskInformation(ctx, tc.originalErr)

			if !strings.Contains(wrappedErr.Error(), tc.expectedTaskFragment) {
				t.Errorf("expected wrapped error to contain task ID, got: %v", wrappedErr)
			}
			if !strings.Contains(wrappedErr.Error(), tc.originalErr.Error()) {
				t.Errorf("expected wrapped error to contain original error message, got: %v", wrappedErr)
			}
			if !errors.Is(wrappedErr, tc.originalErr) {
				t.Error("errors.Is failed to identify the original error in the wrapped error")
			}
		})
	}
}

func TestGetTaskResult(t *testing.T) {
	strRef := taskid.NewTaskReference[string]("test.string")
	orderOnlyRef := taskid.NewTaskReference[string]("test.order_only", taskid.OrderOnly)
	nonExistentRef := taskid.NewTaskReference[bool]("test.nonexistent")
	taskID := taskid.NewDefaultImplementationID[any]("test.id")

	testCases := []struct {
		name         string
		setupCtx     func(ctx context.Context) context.Context
		targetRef    taskid.TaskReference[string]
		want         string
		wantErrMatch string
	}{
		{
			name: "success with declared dependency",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				typedmap.Set(taskResults, typedmap.NewTypedKey[string](strRef.ReferenceIDString()), "test-value")
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{strRef})
				return ctx
			},
			targetRef: strRef,
			want:      "test-value",
		},
		{
			name: "success when task dependencies context key is omitted",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				typedmap.Set(taskResults, typedmap.NewTypedKey[string](strRef.ReferenceIDString()), "test-value")
				return khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
			},
			targetRef: strRef,
			want:      "test-value",
		},
		{
			name: "panic when dependency is not declared in task dependencies",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				typedmap.Set(taskResults, typedmap.NewTypedKey[string](strRef.ReferenceIDString()), "test-value")
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{})
				return ctx
			},
			targetRef:    strRef,
			wantErrMatch: "undeclared task dependency access",
		},
		{
			name: "panic when dependency is declared as order-only",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				typedmap.Set(taskResults, typedmap.NewTypedKey[string](orderOnlyRef.ReferenceIDString()), "test-value")
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{orderOnlyRef})
				return ctx
			},
			targetRef:    orderOnlyRef,
			wantErrMatch: "cannot get task result for order-only dependency",
		},
		{
			name: "panic when result is missing in result map and lists available results",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				typedmap.Set(taskResults, typedmap.NewTypedKey[string]("other.task"), "other-val")
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{nonExistentRef})
				return ctx
			},
			targetRef:    taskid.NewTaskReference[string]("test.nonexistent"),
			wantErrMatch: "Available task results:\n* other.task",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = khictx.WithValue[taskid.UntypedTaskImplementationID](ctx, core_contract.TaskImplementationIDContextKey, taskID)
			ctx = tc.setupCtx(ctx)

			if tc.wantErrMatch != "" {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected panic containing %q, but none occurred", tc.wantErrMatch)
						return
					}
					msg := ""
					if err, ok := r.(error); ok {
						msg = err.Error()
					} else if s, ok := r.(string); ok {
						msg = s
					} else {
						t.Fatalf("unexpected panic type: %T", r)
					}
					if !strings.Contains(msg, tc.wantErrMatch) {
						t.Errorf("expected panic message to contain %q, got: %v", tc.wantErrMatch, msg)
					}
				}()
			}

			got := GetTaskResult(ctx, tc.targetRef)
			if tc.wantErrMatch == "" {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("GetTaskResult() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetOptionalTaskResult(t *testing.T) {
	strRef := taskid.NewTaskReference[string]("test.string", taskid.Optional)
	orderOnlyRef := taskid.NewTaskReference[string]("test.order_only", taskid.OrderOnly, taskid.Optional)
	taskID := taskid.NewDefaultImplementationID[any]("test.id")

	testCases := []struct {
		name         string
		setupCtx     func(ctx context.Context) context.Context
		targetRef    taskid.TaskReference[string]
		wantVal      string
		wantOk       bool
		wantErrMatch string
	}{
		{
			name: "success found result when bound in graph metadata",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				typedmap.Set(taskResults, typedmap.NewTypedKey[string](strRef.ReferenceIDString()), "opt-bound-value")
				meta := &mockGraphMetadata{
					boundTasks: map[string]bool{strRef.ReferenceIDString(): true},
				}
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, meta)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{strRef})
				return ctx
			},
			targetRef: strRef,
			wantVal:   "opt-bound-value",
			wantOk:    true,
		},
		{
			name: "not found when not bound in graph metadata",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				meta := &mockGraphMetadata{
					boundTasks: map[string]bool{strRef.ReferenceIDString(): false},
				}
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, meta)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{strRef})
				return ctx
			},
			targetRef: strRef,
			wantVal:   "",
			wantOk:    false,
		},
		{
			name: "panic when bound in graph metadata but missing from result map",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				meta := &mockGraphMetadata{
					boundTasks: map[string]bool{strRef.ReferenceIDString(): true},
				}
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, meta)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{strRef})
				return ctx
			},
			targetRef:    strRef,
			wantErrMatch: "was bound in DAG but result is missing",
		},
		{
			name: "panic when task graph metadata is missing",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{strRef})
				return ctx
			},
			targetRef:    strRef,
			wantErrMatch: "value not found for key: khi.google.com/task-graph-metadata",
		},
		{
			name: "panic when dependency is not declared in task dependencies",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				meta := &mockGraphMetadata{
					boundTasks: map[string]bool{strRef.ReferenceIDString(): true},
				}
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, meta)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{})
				return ctx
			},
			targetRef:    strRef,
			wantErrMatch: "undeclared task dependency access",
		},
		{
			name: "panic when order-only dependency",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				meta := &mockGraphMetadata{
					boundTasks: map[string]bool{orderOnlyRef.ReferenceIDString(): true},
				}
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, meta)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{orderOnlyRef})
				return ctx
			},
			targetRef:    orderOnlyRef,
			wantErrMatch: "cannot get task result for order-only dependency",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = khictx.WithValue[taskid.UntypedTaskImplementationID](ctx, core_contract.TaskImplementationIDContextKey, taskID)
			ctx = tc.setupCtx(ctx)

			if tc.wantErrMatch != "" {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected panic containing %q, but none occurred", tc.wantErrMatch)
						return
					}
					msg := ""
					if err, ok := r.(error); ok {
						msg = err.Error()
					} else if s, ok := r.(string); ok {
						msg = s
					} else {
						t.Fatalf("unexpected panic type: %T", r)
					}
					if !strings.Contains(msg, tc.wantErrMatch) {
						t.Errorf("expected panic message to contain %q, got: %v", tc.wantErrMatch, msg)
					}
				}()
			}

			val, ok := GetOptionalTaskResult(ctx, tc.targetRef)
			if tc.wantErrMatch == "" {
				if diff := cmp.Diff(tc.wantOk, ok); diff != "" {
					t.Errorf("GetOptionalTaskResult() ok mismatch (-want %v +got %v):\n%s", tc.wantOk, ok, diff)
				}
				if diff := cmp.Diff(tc.wantVal, val); diff != "" {
					t.Errorf("GetOptionalTaskResult() value mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetTaskResultsWithTag(t *testing.T) {
	tag := NewTag[string]("test/tag")
	tagRef := tag.Ref()
	orderOnlyTagRef := tag.Ref(taskid.OrderOnly)
	taskID := taskid.NewDefaultImplementationID[any]("test.consumer")

	testCases := []struct {
		name         string
		setupCtx     func(ctx context.Context) context.Context
		targetRef    TagReference[string]
		want         []string
		wantErrMatch string
	}{
		{
			name: "aggregates multiple producers in deterministic order",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				typedmap.Set(taskResults, typedmap.NewTypedKey[string]("p1"), "apple")
				typedmap.Set(taskResults, typedmap.NewTypedKey[string]("p2"), "banana")
				meta := &mockGraphMetadata{
					boundTasksWithTag: map[string][]string{tag.ID(): {"p1", "p2"}},
				}
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, meta)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{tagRef})
				return ctx
			},
			targetRef: tagRef,
			want:      []string{"apple", "banana"},
		},
		{
			name: "returns empty slice when no producers bound",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				meta := &mockGraphMetadata{
					boundTasksWithTag: map[string][]string{tag.ID(): {}},
				}
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, meta)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{tagRef})
				return ctx
			},
			targetRef: tagRef,
			want:      []string{},
		},
		{
			name: "returns empty slice when tag has nil bound tasks",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				meta := &mockGraphMetadata{
					boundTasksWithTag: nil,
				}
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, meta)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{tagRef})
				return ctx
			},
			targetRef: tagRef,
			want:      []string{},
		},
		{
			name: "panic when producer result missing",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				meta := &mockGraphMetadata{
					boundTasksWithTag: map[string][]string{tag.ID(): {"missing-p"}},
				}
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, meta)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{tagRef})
				return ctx
			},
			targetRef:    tagRef,
			wantErrMatch: "task missing-p providing tag test/tag result missing",
		},
		{
			name: "panic when tag reference undeclared",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				meta := &mockGraphMetadata{
					boundTasksWithTag: map[string][]string{tag.ID(): {}},
				}
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, meta)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{})
				return ctx
			},
			targetRef:    tagRef,
			wantErrMatch: "undeclared task dependency access",
		},
		{
			name: "panic when tag reference is order-only",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				meta := &mockGraphMetadata{
					boundTasksWithTag: map[string][]string{tag.ID(): {}},
				}
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue[core_contract.TaskGraphMetadata](ctx, core_contract.TaskGraphMetadataContextKey, meta)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{orderOnlyTagRef})
				return ctx
			},
			targetRef:    orderOnlyTagRef,
			wantErrMatch: "cannot get task result for order-only dependency",
		},
		{
			name: "panic when task graph metadata is missing",
			setupCtx: func(ctx context.Context) context.Context {
				taskResults := typedmap.NewTypedMap()
				ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskResults)
				ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, []taskid.DependencyDescriptor{tagRef})
				return ctx
			},
			targetRef:    tagRef,
			wantErrMatch: "value not found for key: khi.google.com/task-graph-metadata",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = khictx.WithValue[taskid.UntypedTaskImplementationID](ctx, core_contract.TaskImplementationIDContextKey, taskID)
			ctx = tc.setupCtx(ctx)

			if tc.wantErrMatch != "" {
				defer func() {
					r := recover()
					if r == nil {
						t.Errorf("expected panic containing %q, but none occurred", tc.wantErrMatch)
						return
					}
					msg := ""
					if err, ok := r.(error); ok {
						msg = err.Error()
					} else if s, ok := r.(string); ok {
						msg = s
					} else {
						t.Fatalf("unexpected panic type: %T", r)
					}
					if !strings.Contains(msg, tc.wantErrMatch) {
						t.Errorf("expected panic message to contain %q, got: %v", tc.wantErrMatch, msg)
					}
				}()
			}

			got := GetTaskResultsWithTag(ctx, tc.targetRef)
			if tc.wantErrMatch == "" {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("GetTaskResultsWithTag() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
