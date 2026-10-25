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

package k8sevent_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8sevent"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// resourceUIDDiscoveryTask extracts resource UIDs from Kubernetes Event logs and registers them to ResourceUIDInventoryTask.
var resourceUIDDiscoveryTask = inspectiontaskbase.DefineInspectionTask(
	k8sevent.ResourceUIDDiscoveryTaskID,
	func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[k8saudit.UIDToResourceIdentity] {
		eventLogs := coretask.Use(b, k8sevent.ListLogEntriesTaskID.Ref())
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (k8saudit.UIDToResourceIdentity, error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return k8saudit.UIDToResourceIdentity{}, nil
			}

			result := k8saudit.UIDToResourceIdentity{}
			logs := eventLogs.Get(ctx)
			for _, l := range logs {
				fs, err := k8sevent.ExtractKubernetesEvent(l.NodeReader)
				if err == nil && fs.ResourceUID != "" && fs.Resource != "" && fs.ResourceKind != "" {
					result[fs.ResourceUID] = &k8saudit.ResourceIdentity{
						APIVersion: fs.APIVersion,
						Kind:       fs.ResourceKind,
						Namespace:  fs.Namespace,
						Name:       fs.Resource,
					}
				}
			}
			return result, nil
		}
	},
	coretask.ProvidesTag(k8saudit.TagResourceUIDDiscovery),
	coretask.WithFeatureGate(k8sevent.LogToTimelineMapperTaskID.Ref()),
)
