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
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8sevent_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8sevent/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GenerateK8sEventQuery generates a query for Kubernetes Event logs.
func GenerateK8sEventQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity, namespaceFilter *gcpqueryutil.SetFilterParseResult) string {
	return fmt.Sprintf(`log_id("events")
resource.labels.project_id="%s"
resource.labels.location="%s"
resource.labels.cluster_name="%s"
%s`, cluster.ProjectID, cluster.Location, cluster.NameWithClusterTypePrefix(), generateK8sEventNamespaceFilter(namespaceFilter))
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

type K8sEventListLogEntriesTaskSetting struct {
}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *K8sEventListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *K8sEventListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref(),
		googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *K8sEventListLogEntriesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		DefaultLogType: enum.LogTypeEvent,
		QueryName:      "Kubernetes Event Logs",
		ExampleQuery: GenerateK8sEventQuery(
			googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				Location:    "test-location",
				ClusterName: "test-cluster",
			},
			&gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped", "#namespaced"},
			},
		),
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *K8sEventListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	namespaceFilter := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref())
	return []string{GenerateK8sEventQuery(cluster, namespaceFilter)}, nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *K8sEventListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogk8sevent_contract.ListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *K8sEventListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*K8sEventListLogEntriesTaskSetting)(nil)

var ListLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&K8sEventListLogEntriesTaskSetting{})
