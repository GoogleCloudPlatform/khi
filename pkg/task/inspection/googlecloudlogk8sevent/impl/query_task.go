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

package googlecloudlogk8sevent_impl

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8sevent_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8sevent/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GenerateK8sEventQuery generates a query for Kubernetes Event logs.
func GenerateK8sEventQuery(clusterName string, projectId string, namespaceFilter *gcpqueryutil.SetFilterParseResult) string {
	return fmt.Sprintf(`logName="projects/%s/logs/events"
resource.labels.cluster_name="%s"
%s`, projectId, clusterName, generateK8sEventNamespaceFilter(namespaceFilter))
}

func generateK8sEventNamespaceFilter(filter *gcpqueryutil.SetFilterParseResult) string {
	if filter.ValidationError != "" {
		return fmt.Sprintf(`-- Failed to generate namespace filter due to the validation error "%s"`, filter.ValidationError)
	}
	if filter.SubtractMode {
		return "-- Unsupported operation"
	} else {
		hasClusterScope := slices.Contains(filter.Additives, "#cluster-scoped")
		hasNamespacedScope := slices.Contains(filter.Additives, "#namespaced")
		if hasClusterScope && hasNamespacedScope {
			return "-- No namespace filter"
		}
		if !hasClusterScope && hasNamespacedScope {
			return `jsonPayload.involvedObject.namespace:"" -- ignore events in k8s object with namespace`
		}
		if hasClusterScope && !hasNamespacedScope {
			if len(filter.Additives) == 1 {
				return `-jsonPayload.involvedObject.namespace:"" -- ignore events in k8s object with namespace`
			}
			namespaceContains := []string{}
			for _, additive := range filter.Additives {
				if strings.HasPrefix(additive, "#") {
					continue
				}
				namespaceContains = append(namespaceContains, additive)
			}
			return fmt.Sprintf(`(jsonPayload.involvedObject.namespace=(%s) OR NOT (jsonPayload.involvedObject.namespace:""))`, strings.Join(namespaceContains, " OR "))
		}
		if len(filter.Additives) == 0 {
			return `-- Invalid: none of the resources will be selected. Ignoreing namespace filter.`
		}
		return fmt.Sprintf(`jsonPayload.involvedObject.namespace=(%s)`, strings.Join(filter.Additives, " OR "))
	}
}

// GKEK8sEventLogQueryTask defines a task that queries Kubernetes Event logs from Cloud Logging.
var GKEK8sEventLogQueryTask = googlecloudcommon_contract.NewCloudLoggingListLogTask(googlecloudlogk8sevent_contract.GKEK8sEventLogQueryTaskID, "K8s event logs", enum.LogTypeEvent, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
	googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref(),
}, &googlecloudcommon_contract.ProjectIDDefaultResourceNamesGenerator{}, func(ctx context.Context, i inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	namespaceFilter := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref())

	return []string{GenerateK8sEventQuery(clusterName, projectID, namespaceFilter)}, nil
}, GenerateK8sEventQuery(
	"gcp-cluster-name",
	"gcp-project-id",
	&gcpqueryutil.SetFilterParseResult{
		Additives: []string{"#cluster-scoped", "#namespaced"},
	},
))
