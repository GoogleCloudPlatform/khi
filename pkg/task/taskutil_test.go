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

package task

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/task/contextkey"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func TestWrapErrorWithTaskInformation(t *testing.T) {
	taskID := taskid.NewDefaultImplementationID[any]("foo.com/bar")

	ctx := context.Background()
	ctx = khictx.WithValue[taskid.UntypedTaskImplementationID](ctx, task_contextkey.TaskImplementationIDContextKey, taskID)

	originalErr := errors.New("original error message")

	wrappedErr := WrapErrorWithTaskInformation(ctx, originalErr)

	expectedTaskFragment := "An error occurred in task `foo.com/bar#default`"
	if !strings.Contains(wrappedErr.Error(), expectedTaskFragment) {
		t.Errorf("Expected wrapped error to contain task ID, got: %v", wrappedErr)
	}

	if !strings.Contains(wrappedErr.Error(), originalErr.Error()) {
		t.Errorf("Expected wrapped error to contain original error message, got: %v", wrappedErr)
	}

	if !errors.Is(wrappedErr, originalErr) {
		t.Error("errors.Is failed to identify the original error in the wrapped error")
	}
}

func TestGetTaskResult(t *testing.T) {
	// Create reference and task result map
	strRef := taskid.NewTaskReference[string]("test.string")
	nonExistentRef := taskid.NewTaskReference[bool]("test.nonexistent")

	// Prepare task result map
	taskResults := typedmap.NewTypedMap()
	typedmap.Set(taskResults, typedmap.NewTypedKey[string](strRef.ReferenceIDString()), "test-value")

	// Set up context with task results
	ctx := context.Background()
	ctx = khictx.WithValue(ctx, task_contextkey.TaskResultMapContextKey, taskResults)

	// We also need to set TaskImplementationIDContextKey for panic case to work properly
	taskID := taskid.NewDefaultImplementationID[any]("test.id")
	ctx = khictx.WithValue[taskid.UntypedTaskImplementationID](ctx, task_contextkey.TaskImplementationIDContextKey, taskID)

	t.Run("get string result", func(t *testing.T) {
		result := GetTaskResult(ctx, strRef)
		if result != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", result)
		}
	})

	t.Run("nonexistent result causes panic", func(t *testing.T) {
		// Setup recovery to catch expected panic
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic but none occurred")
			}
			// The panic value is a wrapped error, not a string
			if err, ok := r.(error); ok {
				if !strings.Contains(err.Error(), "test.nonexistent") {
					t.Errorf("Expected error message to contain reference ID, got: %v", err)
				}

				if !strings.Contains(err.Error(), "test.id#default") {
					t.Errorf("Expected error message to contain task ID, got: %v", err)
				}
			} else {
				t.Errorf("Expected panic value to be an error, got: %T", r)
			}
		}()

		// This should cause a panic
		_ = GetTaskResult(ctx, nonExistentRef)
	})
}
