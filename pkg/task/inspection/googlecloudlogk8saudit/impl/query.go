// Copyright 2024 Google LLC
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

package googlecloudlogk8saudit_impl

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

var GCPK8sAuditLogListLogEntriesTask = googlecloudcommon_contract.NewListLogEntriesTask(&GCPK8sAuditLogListLogEntriesTaskSetting{})

type GCPK8sAuditLogListLogEntriesTaskSetting struct{}

// DefaultResourceNames implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *GCPK8sAuditLogListLogEntriesTaskSetting) DefaultResourceNames(ctx context.Context) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	return []string{fmt.Sprintf("projects/%s", cluster.ProjectID)}, nil
}

// Dependencies implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *GCPK8sAuditLogListLogEntriesTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref(),
		googlecloudk8scommon_contract.InputKindFilterTaskID.Ref(),
		googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref(),
	}
}

// Description implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *GCPK8sAuditLogListLogEntriesTaskSetting) Description() *googlecloudcommon_contract.ListLogEntriesTaskDescription {
	return &googlecloudcommon_contract.ListLogEntriesTaskDescription{
		QueryName:      "K8s audit logs",
		DefaultLogType: enum.LogTypeAudit,
		ExampleQuery: GenerateK8sAuditQuery(
			googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ProjectID:   "test-project",
				Location:    "test-location",
				ClusterName: "test-cluster",
			},
			&gcpqueryutil.SetFilterParseResult{
				Additives: []string{"deployments", "replicasets", "pods", "nodes"},
			},
			&gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped", "#namespaced"},
			},
		),
	}
}

// LogFilters implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *GCPK8sAuditLogListLogEntriesTaskSetting) LogFilters(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]string, error) {
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIndentityTaskID.Ref())
	kindFilter := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputKindFilterTaskID.Ref())
	namespaceFilter := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref())

	return []string{GenerateK8sAuditQuery(cluster, kindFilter, namespaceFilter)}, nil
}

// TaskID implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *GCPK8sAuditLogListLogEntriesTaskSetting) TaskID() taskid.TaskImplementationID[[]*log.Log] {
	return googlecloudlogk8saudit_contract.GCPK8sAuditLogListLogEntriesTaskID
}

// TimePartitionCount implements googlecloudcommon_contract.ListLogEntriesTaskSetting.
func (k *GCPK8sAuditLogListLogEntriesTaskSetting) TimePartitionCount(ctx context.Context) (int, error) {
	return 10, nil
}

var _ googlecloudcommon_contract.ListLogEntriesTaskSetting = (*GCPK8sAuditLogListLogEntriesTaskSetting)(nil)

// GenerateK8sAuditQuery constructs a Google Cloud Logging query string for fetching
// Kubernetes audit logs based on cluster name, kind filters, and namespace filters.
func GenerateK8sAuditQuery(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity, auditKindFilter *gcpqueryutil.SetFilterParseResult, namespaceFilter *gcpqueryutil.SetFilterParseResult) string {
	return fmt.Sprintf(`resource.type="k8s_cluster"
resource.labels.project_id="%s"
resource.labels.location="%s"
resource.labels.cluster_name="%s"
protoPayload.methodName: ("create" OR "update" OR "patch" OR "delete")
%s
%s
`, cluster.ProjectID, cluster.Location, cluster.NameWithClusterTypePrefix(), generateAuditKindFilter(auditKindFilter), generateK8sAuditNamespaceFilter(namespaceFilter))
}

// generateAuditKindFilter creates a log filter snippet for Kubernetes resource kinds
// based on the parsed filter result.
func generateAuditKindFilter(filter *gcpqueryutil.SetFilterParseResult) string {
	if filter.ValidationError != "" {
		return fmt.Sprintf(`-- Failed to generate kind filter due to the validation error "%s"`, filter.ValidationError)
	}
	if filter.SubtractMode {
		if len(filter.Subtractives) == 0 {
			return "-- No kind filter"
		}
		return fmt.Sprintf(`-protoPayload.methodName=~"\.(%s)\."`, strings.Join(filter.Subtractives, "|"))
	} else {
		if len(filter.Additives) == 0 {
			return `-- Invalid: none of the resources will be selected. Ignoreing kind filter.`
		}
		return fmt.Sprintf(`protoPayload.methodName=~"\.(%s)\."`, strings.Join(filter.Additives, "|"))
	}
}

// generateK8sAuditNamespaceFilter creates a log filter snippet for Kubernetes namespaces
// based on the parsed filter result.
func generateK8sAuditNamespaceFilter(filter *gcpqueryutil.SetFilterParseResult) string {
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
			return `protoPayload.resourceName:"namespaces/"`
		}
		if hasClusterScope && !hasNamespacedScope {
			if len(filter.Additives) == 1 { // 1 is used for #cluster-scope
				return `-protoPayload.resourceName:"/namespaces/"`
			}
			resourceNameContains := []string{}
			for _, additive := range filter.Additives {
				if strings.HasPrefix(additive, "#") {
					continue
				}
				resourceNameContains = append(resourceNameContains, fmt.Sprintf(`"/namespaces/%s"`, additive))
			}
			return fmt.Sprintf(`(protoPayload.resourceName:(%s) OR NOT (protoPayload.resourceName:"/namespaces/"))`, strings.Join(resourceNameContains, " OR "))
		}
		if len(filter.Additives) == 0 {
			return `-- Invalid: none of the resources will be selected. Ignoreing namespace filter.`
		}
		resourceNameContains := []string{}
		for _, additive := range filter.Additives {
			resourceNameContains = append(resourceNameContains, fmt.Sprintf(`"/namespaces/%s"`, additive))
		}
		return fmt.Sprintf(`protoPayload.resourceName:(%s)`, strings.Join(resourceNameContains, " OR "))
	}
}
