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

package googlecloudlogcsm_impl

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

type CSMTrafficDirectorListLogEntryTaskSetting struct{}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (s *CSMTrafficDirectorListLogEntryTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	fleetProjectID := coretask.GetTaskResult(ctx, googlecloudlogcsm_contract.InputFleetProjectIDTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", fleetProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (s *CSMTrafficDirectorListLogEntryTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogcsm_contract.InputFleetProjectIDTaskID.Ref(),
		googlecloudlogcsm_contract.CSMClusterIdentifierTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (s *CSMTrafficDirectorListLogEntryTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeAudit,
		QueryName:      "CSM Traffic Director logs",
		ExampleQuery: `(log_id("cloudaudit.googleapis.com/activity") OR log_id("cloudaudit.googleapis.com/data_access"))
  protoPayload.resourceName: "gsmrsvd-XXXX" -- XXXX part will be generated from other log parsing result
  resource.labels.project_id="fleet-project"`,
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (s *CSMTrafficDirectorListLogEntryTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	fleetProjectID := coretask.GetTaskResult(ctx, googlecloudlogcsm_contract.InputFleetProjectIDTaskID.Ref())
	clusterIdentifiers := coretask.GetTaskResult(ctx, googlecloudlogcsm_contract.CSMClusterIdentifierTaskID.Ref())

	dryRunComment := ""
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		clusterIdentifiers = []string{"dummy"}
		dryRunComment = " -- The actual resource name selector will be generated from other logs in the middle of the pipeline."
	} else if len(clusterIdentifiers) == 0 {
		slog.InfoContext(ctx, "No CSM BackendServices found in inventory. Skipping Traffic Director log query.")
		return nil, nil
	}

	resourceNameFilter := ""
	if len(clusterIdentifiers) == 1 {
		resourceNameFilter = fmt.Sprintf(`protoPayload.resourceName:"gsmrsvd-%s"`, clusterIdentifiers[0])
	} else {
		quotedIdentifiers := make([]string, len(clusterIdentifiers))
		for i, id := range clusterIdentifiers {
			quotedIdentifiers[i] = fmt.Sprintf(`"gsmrsvd-%s"`, id)
		}
		resourceNameFilter = fmt.Sprintf(`protoPayload.resourceName:(%s)`, strings.Join(quotedIdentifiers, " OR "))
	}

	query := fmt.Sprintf(`(log_id("cloudaudit.googleapis.com/activity") OR log_id("cloudaudit.googleapis.com/data_access"))
%s%s
resource.labels.project_id="%s"`, resourceNameFilter, dryRunComment, fleetProjectID)

	return []string{query}, nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (s *CSMTrafficDirectorListLogEntryTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogcsm_contract.ListCSMTrafficDirectorLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (s *CSMTrafficDirectorListLogEntryTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*CSMTrafficDirectorListLogEntryTaskSetting)(nil)

var ListCSMTrafficDirectorLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&CSMTrafficDirectorListLogEntryTaskSetting{})
