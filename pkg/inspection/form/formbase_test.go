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

package form

import (
	"testing"

	form_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func TestNewFormTaskBuilderBase(t *testing.T) {
	testID := "test-id"
	testPriority := 5
	testLabel := "Test Label"

	builder := NewFormTaskBuilderBase(taskid.NewDefaultImplementationID[string](testID), testPriority, testLabel)

	if builder.id.ReferenceIDString() != testID {
		t.Errorf("Expected id to be %s, got %s", testID, builder.id)
	}
	if builder.priority != testPriority {
		t.Errorf("Expected priority to be %d, got %d", testPriority, builder.priority)
	}
	if builder.label != testLabel {
		t.Errorf("Expected label to be %s, got %s", testLabel, builder.label)
	}
	if len(builder.dependencies) != 0 {
		t.Errorf("Expected dependencies to be an empty slice, got %v", builder.dependencies)
	}
	if builder.description != "" {
		t.Errorf("Expected description to be empty, got %s", builder.description)
	}
}

func TestFormTaskBuilderBase_WithDescription(t *testing.T) {
	builder := NewFormTaskBuilderBase(taskid.NewDefaultImplementationID[string]("test-id"), 1, "Test Label")
	testDescription := "Test Description"

	result := builder.WithDescription(testDescription)

	if builder.description != testDescription {
		t.Errorf("Expected description to be %s, got %s", testDescription, builder.description)
	}

	if result != &builder {
		t.Errorf("Expected method to return the builder pointer, got a different pointer")
	}
}

func TestFormTaskBuilderBase_WithDependencies(t *testing.T) {
	builder := NewFormTaskBuilderBase(taskid.NewDefaultImplementationID[string]("test-id"), 1, "Test Label")
	testDependencies := []taskid.UntypedTaskReference{taskid.NewTaskReference[string]("dep1"), taskid.NewTaskReference[string]("dep2")}

	result := builder.WithDependencies(testDependencies)

	if len(builder.dependencies) != len(testDependencies) {
		t.Errorf("Expected dependencies length to be %d, got %d", len(testDependencies), len(builder.dependencies))
	}

	for i, dep := range testDependencies {
		if builder.dependencies[i] != dep {
			t.Errorf("Expected dependency at index %d to be %s, got %s", i, dep, builder.dependencies[i])
		}
	}

	if result != &builder {
		t.Errorf("Expected method to return the builder pointer, got a different pointer")
	}
}

func TestFormTaskBuilderBase_SetupBaseFormField(t *testing.T) {
	testID := "test-id"
	testPriority := 5
	testLabel := "Test Label"
	testDescription := "Test Description"

	builder := NewFormTaskBuilderBase(taskid.NewDefaultImplementationID[string](testID), testPriority, testLabel)
	builder.WithDescription(testDescription)

	field := &form_metadata.ParameterFormFieldBase{}
	builder.SetupBaseFormField(field)

	if field.ID != testID {
		t.Errorf("Expected field ID to be %s, got %s", testID, field.ID)
	}
	if field.Priority != testPriority {
		t.Errorf("Expected field Priority to be %d, got %d", testPriority, field.Priority)
	}
	if field.Label != testLabel {
		t.Errorf("Expected field Label to be %s, got %s", testLabel, field.Label)
	}
	if field.Description != testDescription {
		t.Errorf("Expected field Description to be %s, got %s", testDescription, field.Description)
	}
}
