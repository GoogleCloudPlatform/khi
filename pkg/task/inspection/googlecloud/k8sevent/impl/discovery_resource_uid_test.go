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
	"testing"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8sevent"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

// TestResourceUIDDiscoveryTask tests resourceUIDDiscoveryTask.
func TestResourceUIDDiscoveryTask(t *testing.T) {
	tests := []struct {
		name     string
		logs     []*log.Log
		taskMode inspectioncore.InspectionTaskModeType
		want     k8saudit.UIDToResourceIdentity
	}{
		{
			name: "valid event logs with involvedObject uid",
			logs: []*log.Log{
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					APIVersion:   "core/v1",
					ResourceKind: "pod",
					Namespace:    "default",
					Resource:     "test-pod-1",
					ResourceUID:  "8b04c24f-b9f7-49dd-ab67-f55568b07817",
				}),
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					APIVersion:   "apps/v1",
					ResourceKind: "deployment",
					Namespace:    "kube-system",
					Resource:     "test-deployment",
					ResourceUID:  "d3a7ae3c-7108-4ab2-ae02-b6d7ed7a21d8",
				}),
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					APIVersion:   "core/v1",
					ResourceKind: "pod",
					Namespace:    "default",
					Resource:     "test-pod-1",
					ResourceUID:  "8b04c24f-b9f7-49dd-ab67-f55568b07817",
				}),
			},
			taskMode: inspectioncore.TaskModeRun,
			want: k8saudit.UIDToResourceIdentity{
				"8b04c24f-b9f7-49dd-ab67-f55568b07817": {
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "test-pod-1",
				},
				"d3a7ae3c-7108-4ab2-ae02-b6d7ed7a21d8": {
					APIVersion: "apps/v1",
					Kind:       "deployment",
					Namespace:  "kube-system",
					Name:       "test-deployment",
				},
			},
		},
		{
			name: "logs missing uid or resource identity fields are ignored",
			logs: []*log.Log{
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					APIVersion:   "core/v1",
					ResourceKind: "pod",
					Namespace:    "default",
					Resource:     "test-pod-no-uid",
					ResourceUID:  "",
				}),
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					APIVersion:   "core/v1",
					ResourceKind: "pod",
					Namespace:    "default",
					Resource:     "",
					ResourceUID:  "8b04c24f-b9f7-49dd-ab67-f55568b07817",
				}),
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					APIVersion:   "core/v1",
					ResourceKind: "",
					Namespace:    "default",
					Resource:     "test-pod-no-kind",
					ResourceUID:  "d3a7ae3c-7108-4ab2-ae02-b6d7ed7a21d8",
				}),
			},
			taskMode: inspectioncore.TaskModeRun,
			want:     k8saudit.UIDToResourceIdentity{},
		},
		{
			name: "dry run returns empty map",
			logs: []*log.Log{
				testlog.NewMockLog(k8sevent.KubernetesEventFieldSet{
					APIVersion:   "core/v1",
					ResourceKind: "pod",
					Namespace:    "default",
					Resource:     "test-pod-1",
					ResourceUID:  "8b04c24f-b9f7-49dd-ab67-f55568b07817",
				}),
			},
			taskMode: inspectioncore.TaskModeDryRun,
			want:     k8saudit.UIDToResourceIdentity{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			result, _, err := inspectiontest.Run(t, ctx, resourceUIDDiscoveryTask, tc.taskMode, map[string]any{},
				tasktest.Given(k8sevent.ListLogEntriesTaskID.Ref(), tc.logs),
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, result); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
