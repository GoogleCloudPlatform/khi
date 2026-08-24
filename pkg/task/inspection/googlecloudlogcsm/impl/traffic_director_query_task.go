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

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/logestimator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GenerateCSMTrafficDirectorStructuredQuery generates a structured query for CSM Traffic Director logs.
func GenerateCSMTrafficDirectorStructuredQuery(fleetProjectID string, clusterIdentifiers []string, isDryRun bool) *logestimator.StructuredLogQuery {
	if isDryRun {
		clusterIdentifiers = []string{"dummy"}
	}
	if len(clusterIdentifiers) == 0 {
		return nil
	}

	filters := []logestimator.LoggingMonitoringMatcher{
		logestimator.ResourceLabel("project_id", logestimator.Exact(fleetProjectID)),
		logestimator.LogID(logestimator.OneOf("cloudaudit.googleapis.com/activity", "cloudaudit.googleapis.com/data_access")),
	}

	switch {
	case isDryRun:
		filters = append(filters, logestimator.CustomFilter(`protoPayload.resourceName:"gsmrsvd-dummy" -- The actual resource name selector will be generated from other logs in the middle of the pipeline.`))
	case len(clusterIdentifiers) == 1:
		filters = append(filters, logestimator.CustomFilter(fmt.Sprintf(`protoPayload.resourceName:"gsmrsvd-%s"`, clusterIdentifiers[0])))
	default:
		quotedIdentifiers := make([]string, len(clusterIdentifiers))
		for i, id := range clusterIdentifiers {
			quotedIdentifiers[i] = fmt.Sprintf(`"gsmrsvd-%s"`, id)
		}
		filters = append(filters, logestimator.CustomFilter(fmt.Sprintf(`protoPayload.resourceName:(%s)`, strings.Join(quotedIdentifiers, " OR "))))
	}

	return &logestimator.StructuredLogQuery{
		Incomplete: fleetProjectID == "",
		Filters:    filters,
	}
}

// GenerateCSMTrafficDirectorQuery generates a query for CSM Traffic Director logs.
func GenerateCSMTrafficDirectorQuery(fleetProjectID string, clusterIdentifiers []string, isDryRun bool) string {
	sq := GenerateCSMTrafficDirectorStructuredQuery(fleetProjectID, clusterIdentifiers, isDryRun)
	if sq == nil {
		return ""
	}
	return sq.GenerateCloudLoggingQuery()
}

type CSMTrafficDirectorListLogEntryTaskSetting struct{}

// DefaultResourceNames implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (s *CSMTrafficDirectorListLogEntryTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	fleetProjectID := coretask.GetTaskResult(ctx, googlecloudlogcsm_contract.InputFleetProjectIDTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", fleetProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (s *CSMTrafficDirectorListLogEntryTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudlogcsm_contract.InputFleetProjectIDTaskID.Ref(),
		googlecloudlogcsm_contract.CSMClusterIdentifierTaskID.Ref(),
	}
}

// QueryName implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (s *CSMTrafficDirectorListLogEntryTaskSetting) QueryName() string {
	return "CSM Traffic Director logs"
}

// Queries implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (s *CSMTrafficDirectorListLogEntryTaskSetting) Queries(ctx context.Context) ([]*logestimator.StructuredLogQuery, error) {
	fleetProjectID := coretask.GetTaskResult(ctx, googlecloudlogcsm_contract.InputFleetProjectIDTaskID.Ref())
	clusterIdentifiers := coretask.GetTaskResult(ctx, googlecloudlogcsm_contract.CSMClusterIdentifierTaskID.Ref())
	taskMode := inspectioncore_contract.TaskModeRun
	if val, err := khictx.GetValue(ctx, inspectioncore_contract.InspectionTaskMode); err == nil {
		taskMode = val
	}
	isDryRun := taskMode == inspectioncore_contract.TaskModeDryRun

	sq := GenerateCSMTrafficDirectorStructuredQuery(fleetProjectID, clusterIdentifiers, isDryRun)
	if sq == nil {
		if !isDryRun {
			slog.InfoContext(ctx, "No CSM BackendServices found in inventory. Skipping Traffic Director log query.")
		}
		return nil, nil
	}

	return []*logestimator.StructuredLogQuery{sq}, nil
}

// TaskID implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (s *CSMTrafficDirectorListLogEntryTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogcsm_contract.ListCSMTrafficDirectorLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.StructuredListLogEntriesTaskSetting.
func (s *CSMTrafficDirectorListLogEntryTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 1, nil
}

var _ googlecloudcommon_contract.StructuredListLogEntriesTaskSetting = (*CSMTrafficDirectorListLogEntryTaskSetting)(nil)

var ListCSMTrafficDirectorLogEntriesTask = googlecloudcommon_contract.NewStructuredListLogEntriesTask(&CSMTrafficDirectorListLogEntryTaskSetting{})
