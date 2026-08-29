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

package taskrecord

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// sanitizeTaskReferenceForFileName converts a task reference ID into a valid file name.
func sanitizeTaskReferenceForFileName(refID string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		":", "_",
		"\\", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(refID)
}

// saveRecordedTaskResults serializes and saves captured task results to the specified fixture directory.
func saveRecordedTaskResults(fixtureDir string, results map[string]any) error {
	if err := os.MkdirAll(fixtureDir, 0755); err != nil {
		return fmt.Errorf("failed to create fixture directory %s: %w", fixtureDir, err)
	}

	for refID, val := range results {
		if val == nil {
			continue
		}
		codec := GetCodec(reflect.TypeOf(val))
		data, err := codec.Serialize(val)
		if err != nil {
			return fmt.Errorf("failed to serialize task %s: %w", refID, err)
		}

		filePath := filepath.Join(fixtureDir, sanitizeTaskReferenceForFileName(refID)+".json")
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write fixture file %s: %w", filePath, err)
		}
	}
	return nil
}

// newRecordInspectionInterceptor creates an InspectionInterceptor that intercepts and collects
// the output of the specified recorded tasks during Job execution.
func newRecordInspectionInterceptor(recordedTasks []taskid.UntypedTaskReference, cancelFunc func(), onRecordComplete func(map[string]any)) coreinspection.InspectionInterceptor {
	targetSet := make(map[string]struct{}, len(recordedTasks))
	for _, ref := range recordedTasks {
		targetSet[ref.ReferenceIDString()] = struct{}{}
	}

	var mu sync.Mutex
	recordedResults := make(map[string]any)

	return func(ctx context.Context, req *inspectioncore_contract.InspectionRequest, next func(context.Context) error) error {
		runner := khictx.MustGetValue(ctx, inspectioncore_contract.TaskRunner)

		runner.AddInterceptor(func(taskCtx context.Context, task coretask.UntypedTask, taskNext func(context.Context) (any, error)) (any, error) {
			result, err := taskNext(taskCtx)
			if err != nil {
				return result, err
			}

			refID := task.UntypedID().GetUntypedReference().ReferenceIDString()
			mu.Lock()
			if _, isTarget := targetSet[refID]; isTarget {
				recordedResults[refID] = result
				if len(recordedResults) == len(targetSet) {
					// All targets recorded.
					if onRecordComplete != nil {
						onRecordComplete(recordedResults)
					}
					if cancelFunc != nil {
						cancelFunc()
					}
				}
			}
			mu.Unlock()

			return result, nil
		})

		return next(ctx)
	}
}
