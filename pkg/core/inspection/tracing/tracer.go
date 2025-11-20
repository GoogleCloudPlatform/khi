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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func NewInspectionTraceInterceptor(tracer trace.Tracer) coreinspection.InspectionInterceptor {
	return func(ctx context.Context, req *inspectioncore_contract.InspectionRequest, next func(context.Context) error) error {
		inspectionID := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskInspectionID)
		runID := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskRunID)
		runner := khictx.MustGetValue(ctx, inspectioncore_contract.TaskRunner)
		mode := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskMode)
		ctx = khictx.WithValue(ctx, inspectioncore_contract.TracingActive, true)

		ctx, span := tracer.Start(ctx, fmt.Sprintf("inspection-%s", inspectionID), trace.WithAttributes(
			attribute.String("inspection_id", inspectionID),
			attribute.String("run_id", runID),
			attribute.String("mode", inspectioncore_contract.TaskModeToString(mode)),
		))
		defer span.End()

		runner.AddInterceptor(func(ctx context.Context, task coretask.UntypedTask, next func(context.Context) (any, error)) (any, error) {
			ctx, span := tracer.Start(ctx, task.UntypedID().String(), trace.WithAttributes(
				attribute.String("task_id", task.UntypedID().String()),
			))
			defer span.End()

			result, err := next(ctx)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
			return result, err
		})

		err := next(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
}
