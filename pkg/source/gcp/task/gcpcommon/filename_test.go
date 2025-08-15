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

package gcpcommon

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"

	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestHeaderSuggestedFileNameTask(t *testing.T) {
	testCases := []struct {
		Name              string
		ClusterName       string
		StartTime         time.Time
		EndTime           time.Time
		SuggestedFileName string
	}{
		{
			Name:              "normal case",
			ClusterName:       "test-cluster",
			StartTime:         time.Date(2023, time.January, 1, 10, 0, 0, 0, time.UTC),
			EndTime:           time.Date(2023, time.January, 1, 11, 0, 0, 0, time.UTC),
			SuggestedFileName: "test-cluster-2023_01_01_1000-2023_01_01_1100.khi",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			inspectiontest.RunInspectionTask(ctx, HeaderSuggestedFileNameTask, inspection_contract.TaskModeRun, map[string]any{},
				tasktest.NewTaskDependencyValuePair(gcp_task.InputClusterNameTaskID.Ref(), tc.ClusterName),
				tasktest.NewTaskDependencyValuePair(gcp_task.InputStartTimeTaskID.Ref(), tc.StartTime),
				tasktest.NewTaskDependencyValuePair(gcp_task.InputEndTimeTaskID.Ref(), tc.EndTime),
			)

			metadata := khictx.MustGetValue(ctx, inspection_contract.InspectionRunMetadata)
			header, found := typedmap.Get(metadata, inspectionmetadata.HeaderMetadataKey)
			if !found {
				t.Fatalf("header metadata not found")
			}

			if header.SuggestedFileName != tc.SuggestedFileName {
				t.Fatalf("suggested file name mismatch. expected: %s, got: %s", tc.SuggestedFileName, header.SuggestedFileName)
			}
		})
	}
}
