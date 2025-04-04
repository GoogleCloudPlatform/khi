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

package upload

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/task/contextkey"
)

// GenerateUploadIDWithTaskContext generates the upload ID from form ID and task ID.
func GenerateUploadIDWithTaskContext(ctx context.Context, formId string) string {
	inspectionID := khictx.MustGetValue(ctx, inspection_task_contextkey.InspectionTaskInspectionID)
	taskID := khictx.MustGetValue(ctx, task_contextkey.TaskImplementationIDContextKey)
	return strings.ReplaceAll(fmt.Sprintf("%s_%s_%s", inspectionID, taskID.ReferenceIDString(), formId), "/", "_")
}
