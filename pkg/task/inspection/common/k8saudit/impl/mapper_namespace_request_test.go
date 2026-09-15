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

package k8saudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

// TestNamespaceRequestLogToTimelineMapperTaskSetting_ProcessLog tests that namespace-wide request logs
// stage timeline events directly on the namespace timeline.
func TestNamespaceRequestLogToTimelineMapperTaskSetting_ProcessLog(t *testing.T) {
	testTime := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)

	// Set up the mock Builder and construct comparison paths hierarchically at the top of the test.
	builder := khifilev6.NewTestBuilder(id.NewGenerator())
	cluster := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore.TimelineTypeK8sCluster})
	api := builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion})
	kind := builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore.TimelineTypeKind})
	nsPath := builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore.TimelineTypeNamespace})

	ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

	testCases := []struct {
		name   string
		verb   *pb.Verb
		assert func(t *testing.T, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "DeleteCollection event",
			verb: k8saudit.VerbDeleteCollection,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(nsPath)
			},
		},
	}

	mapperSetting := &namespaceRequestLogToTimelineMapperTaskSetting{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logObj := testlog.NewMockLog(
				testTime,
				k8saudit.K8sAuditLogFieldSet{
					Principal:    "admin",
					APIVersion:   "core/v1",
					PluralKind:   "pods",
					ResourceName: "",
					Namespace:    "default",
					ClusterName:  "k8s",
					Verb:         tc.verb,
				},
			)

			targetResource := &k8saudit.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "",
			}

			groupSet := k8saudit.RelatedGroupSet{
				Roles: map[string]*k8saudit.ResourceManifestLogGroup{
					"target": {
						Resource: targetResource,
						Logs: []*k8saudit.ResourceManifestLog{
							{Log: logObj},
						},
					},
				},
			}

			event := k8saudit.MultiGroupLogEvent{
				Log:              logObj,
				GroupRole:        "target",
				ResourceIdentity: targetResource,
				EventType:        k8saudit.ChangeEventTypeModification,
				GroupSet:         groupSet,
			}

			cs, _, err := mapperSetting.ProcessLog(ctx, event, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLog() failed: %v", err)
			}

			tc.assert(t, cs)
		})
	}
}
