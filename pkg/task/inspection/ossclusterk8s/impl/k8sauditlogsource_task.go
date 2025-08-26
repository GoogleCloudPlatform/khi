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

package ossclusterk8s_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	ossclusterk8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/ossclusterk8s/contract"
)

// OSSK8sAuditLogSourceTask receives logs generated from the previous tasks specific to OSS audit log parsing and inject dependencies specific to this OSS inspection type.
var OSSK8sAuditLogSourceTask = inspectiontaskbase.NewInspectionTask(ossclusterk8s_contract.OSSK8sAuditLogSourceTaskID, []taskid.UntypedTaskReference{
	ossclusterk8s_contract.NonEventAuditLogFilterTaskID.Ref(),
}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (*commonlogk8saudit_contract.AuditLogParserLogSource, error) {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return nil, nil
	}
	logs := coretask.GetTaskResult(ctx, ossclusterk8s_contract.NonEventAuditLogFilterTaskID.Ref())

	return &commonlogk8saudit_contract.AuditLogParserLogSource{
		Logs:      logs,
		Extractor: &OSSJSONLAuditLogFieldExtractor{},
	}, nil
}, inspectioncore_contract.InspectionTypeLabel(ossclusterk8s_contract.InspectionTypeID))
