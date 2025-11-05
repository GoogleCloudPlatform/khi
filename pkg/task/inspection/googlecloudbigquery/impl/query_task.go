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

package googlecloudbigquery_impl

import (
	"context"
	"fmt"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	googlecloudbigquery_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudbigquery/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func GenerateBigQueryJobCompletedFilter(projectId string) string {
	return fmt.Sprintf(`
resource.labels.project_id="%s"
resource.type="bigquery_resource"
protoPayload.serviceData.jobCompletedEvent.eventName="query_job_completed"
`, projectId)
}

var BigQueryJobCompletedTask = googlecloudcommon_contract.NewCloudLoggingListLogTask(googlecloudbigquery_contract.BigQueryCompletedEventQueryID, "BigQuery CompletedEvent logs", enum.LogTypeBigQueryResource, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
}, &googlecloudcommon_contract.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	projectId := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	return []string{GenerateBigQueryJobCompletedFilter(projectId)}, nil
}, GenerateBigQueryJobCompletedFilter("google-cloud-project-id"))
