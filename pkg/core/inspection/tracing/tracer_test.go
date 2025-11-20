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

package tracing

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewInspectionTraceInterceptor(t *testing.T) {
	// Setup OpenTelemetry with a memory exporter to verify spans
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test-tracer")

	interceptor := NewInspectionTraceInterceptor(tracer)

	// Create a mock TaskRunner
	mockRunner := &mockTaskRunner{}

	// Create a context with the mock TaskRunner
	ctx := context.Background()
	ctx = context.WithValue(ctx, inspectioncore_contract.TaskRunner, mockRunner)
	ctx = context.WithValue(ctx, inspectioncore_contract.InspectionTaskInspectionID, "test-inspection-id")
	ctx = context.WithValue(ctx, inspectioncore_contract.InspectionTaskRunID, "test-run-id")
	ctx = context.WithValue(ctx, inspectioncore_contract.InspectionTaskMode, inspectioncore_contract.TaskModeRun)

	req := &inspectioncore_contract.InspectionRequest{}

	// Execute the interceptor
	err := interceptor(ctx, req, func(ctx context.Context) error {
		// Simulate inspection execution
		return nil
	})

	if err != nil {
		t.Fatalf("Interceptor returned error: %v", err)
	}

	// Verify that the interceptor was added to the TaskRunner
	if len(mockRunner.interceptors) != 1 {
		t.Errorf("Expected 1 interceptor to be added to TaskRunner, got %d", len(mockRunner.interceptors))
	}

	// Verify spans
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Errorf("Expected 1 span (root span), got %d", len(spans))
	} else if spans[0].Name != "inspection-test-inspection-id" {
		t.Errorf("Expected span name 'inspection-test-inspection-id', got '%s'", spans[0].Name)
	}

	// Test the added task interceptor
	taskInterceptor := mockRunner.interceptors[0]
	mockTask := &mockTask{
		id: taskid.NewDefaultImplementationID[any]("test-task"),
	}

	_, err = taskInterceptor(ctx, mockTask, func(ctx context.Context) (any, error) {
		return "result", nil
	})
	if err != nil {
		t.Fatalf("Task interceptor returned error: %v", err)
	}

	spans = exporter.GetSpans()
	if len(spans) != 2 {
		t.Errorf("Expected 2 spans (root + task), got %d", len(spans))
	} else {
		// Check if the second span is the task span
		taskSpan := spans[1]
		if taskSpan.Name != "test-task#default" {
			t.Errorf("Expected task span name 'test-task#default', got '%s'", taskSpan.Name)
		}
	}
}

type mockTaskRunner struct {
	interceptors []coretask.Interceptor
}

func (m *mockTaskRunner) Run(ctx context.Context) error {
	return nil
}

func (m *mockTaskRunner) Wait() <-chan interface{} {
	return nil
}

func (m *mockTaskRunner) Result() (*typedmap.ReadonlyTypedMap, error) {
	return nil, nil
}

func (m *mockTaskRunner) Tasks() []coretask.UntypedTask {
	return nil
}

func (m *mockTaskRunner) AddInterceptor(interceptor coretask.Interceptor) {
	m.interceptors = append(m.interceptors, interceptor)
}

type mockTask struct {
	id taskid.UntypedTaskImplementationID
}

func (m *mockTask) UntypedID() taskid.UntypedTaskImplementationID {
	return m.id
}

func (m *mockTask) Dependencies() []taskid.UntypedTaskReference {
	return nil
}

func (m *mockTask) UntypedRun(ctx context.Context) (any, error) {
	return nil, nil
}

func (m *mockTask) Labels() *typedmap.ReadonlyTypedMap {
	return typedmap.NewTypedMap().AsReadonly()
}
