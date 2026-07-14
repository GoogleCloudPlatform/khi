// Copyright 2026 Google LLC
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

package inspectioncore_impl

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// JobModeCommandTaskID defines the unique ID of the JobModeCommandTask.
var JobModeCommandTaskID = taskid.NewDefaultImplementationID[any](inspectioncore_contract.InspectionTaskPrefix + "job-command")

// JobModeCommandTask calculates the job mode command example and populates it into the metadata map.
var JobModeCommandTask = inspectiontaskbase.NewInspectionTask(
	JobModeCommandTaskID,
	[]taskid.UntypedTaskReference{},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (any, error) {
		metadataSet := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionRunMetadata)
		jobMetadata, found := typedmap.Get(metadataSet, inspectionmetadata.JobModeCommandMetadataKey)
		if !found {
			return nil, fmt.Errorf("job command metadata not found")
		}

		enabledFeatures, err := khictx.GetValue(ctx, inspectioncore_contract.InspectionTaskEnabledFeatures)
		if err != nil {
			return nil, err
		}

		inspectionType := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskType)

		taskInput := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskInput)

		formFields, found := typedmap.Get(metadataSet, inspectionmetadata.FormFieldSetMetadataKey)
		if !found {
			return nil, fmt.Errorf("form field metadata not found")
		}
		fileFieldIDs := formFields.GetFileFieldIDs()

		command, err := GenerateJobModeCommand(inspectionType, enabledFeatures, taskInput, fileFieldIDs)
		if err != nil {
			return nil, err
		}

		jobMetadata.SetCommand(command)

		return nil, nil
	},
)

// GenerateJobModeCommand formats a copy-pasteable command to execute KHI in job mode.
func GenerateJobModeCommand(inspectionType string, enabledFeatures []string, taskInput map[string]any, fileFieldIDs []string) (string, error) {
	// Create a copy to avoid mutating the original features array.
	featuresCopy := make([]string, len(enabledFeatures))
	copy(featuresCopy, enabledFeatures)
	slices.Sort(featuresCopy)
	featuresStr := strings.Join(featuresCopy, ",")

	values := taskInput
	if len(fileFieldIDs) > 0 {
		values = maps.Clone(taskInput)
		if values == nil {
			values = map[string]any{}
		}
		for _, id := range fileFieldIDs {
			values[id] = "path/to/file"
		}
	}
	var valuesStr string
	if len(values) > 0 {
		valuesBytes, err := json.MarshalIndent(values, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal inspection values: %w", err)
		}
		valuesStr = strings.ReplaceAll(string(valuesBytes), "'", "'\\''")
	}

	command := fmt.Sprintf(`./khi \
  --job-mode \
  --job-inspection-type="%s" \
  --job-inspection-features="%s" \
  --job-inspection-values='%s' \
  --job-export-destination="output.khi"`,
		inspectionType,
		featuresStr,
		valuesStr,
	)
	return command, nil
}
