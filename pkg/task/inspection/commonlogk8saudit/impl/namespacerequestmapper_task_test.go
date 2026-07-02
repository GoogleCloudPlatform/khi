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

package commonlogk8saudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

// TestNamespaceRequestLogToTimelineMapperTaskSetting_ProcessLog tests that namespace-wide request logs
// stage timeline events directly on the namespace timeline.
func TestNamespaceRequestLogToTimelineMapperTaskSetting_ProcessLog(t *testing.T) {
	testTime := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)

	// Set up the mock Builder and construct comparison paths hierarchically at the top of the test.
	builder := khifilev6.NewBuilder()
	cluster := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})
	api := builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
	kind := builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
	nsPath := builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})

	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	testCases := []struct {
		name   string
		verb   *pb.Verb
		assert func(t *testing.T, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "DeleteCollection event",
			verb: commonlogk8saudit_contract.VerbDeleteCollection,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(nsPath)
			},
		},
	}

	mapperSetting := &namespaceRequestLogToTimelineMapperTaskSetting{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k8sFieldSet := &commonlogk8saudit_contract.K8sAuditLogFieldSet{
				Principal:    "admin",
				APIVersion:   "core/v1",
				PluralKind:   "pods",
				ResourceName: "",
				Namespace:    "default",
				ClusterName:  "k8s",
				Verb:         tc.verb,
			}
			commonFs := &log.CommonFieldSet{
				Timestamp: testTime,
			}
			logObj := log.NewLogWithFieldSetsForTest(k8sFieldSet, commonFs)

			targetResource := &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "",
			}

			groupSet := commonlogk8saudit_contract.RelatedGroupSet{
				Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
					"target": {
						Resource: targetResource,
						Logs: []*commonlogk8saudit_contract.ResourceManifestLog{
							{Log: logObj},
						},
					},
				},
			}

			event := commonlogk8saudit_contract.MultiGroupLogEvent{
				Log:              logObj,
				GroupRole:        "target",
				ResourceIdentity: targetResource,
				EventType:        commonlogk8saudit_contract.ChangeEventTypeModification,
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
