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

package googlecloudlogk8scontainer_impl

import (
	"context"
	"testing"
	"time"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

// TestLogIngester_ProcessLog tests the containerLogIngester.ProcessLog function.
func TestLogIngester_ProcessLog(t *testing.T) {
	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name: "successful container log ingestion",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityInfo,
				},
				&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
					Namespace:     "test-namespace",
					PodName:       "test-pod",
					ContainerName: "test-container",
					Message:       "test message",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("test message").
					HasSeverity(inspectioncore_contract.SeverityInfo).
					HasLogType(googlecloudlogk8scontainer_contract.LogTypeContainer).
					HasTimestamp(time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC))
			},
		},
	}

	ingester := &containerLogIngester{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs, err := ingester.ProcessLog(t.Context(), tc.input)
			if err != nil {
				t.Fatalf("ProcessLog() returned unexpected error: %v", err)
			}
			tc.assert(t, cs)
		})
	}
}

// TestLogToTimelineMapper_ProcessLogByGroup tests the containerLogLogToTimelineMapper.ProcessLogByGroup function.
func TestLogToTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
	apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
	namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "test-namespace")
	podTimeline := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-pod")
	expectedPath := commonlogk8saudit_contract.MustK8sContainerTimeline(ctx, podTimeline, "test-container")

	testCases := []struct {
		name     string
		inputLog *log.Log
		cluster  googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		assert   func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "simple container log mapping",
			inputLog: log.NewLogWithFieldSetsForTest(
				&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
					Namespace:     "test-namespace",
					PodName:       "test-pod",
					ContainerName: "test-container",
					Message:       "test message",
				},
			),
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath)
			},
		},
		{
			name: "container log with empty message",
			inputLog: log.NewLogWithFieldSetsForTest(
				&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
					Namespace:     "test-namespace",
					PodName:       "test-pod",
					ContainerName: "test-container",
					Message:       "",
				},
			),
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath)
			},
		},
	}

	mapper := &containerLogLogToTimelineMapper{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, googlecloudlogk8scontainer_contract.ClusterIdentityTaskID.Ref(), tc.cluster)

			cs, _, err := mapper.ProcessLogByGroup(ctx, tc.inputLog, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
			}
			tc.assert(t, ctx, cs)
		})
	}
}

// TestPodPhaseTimelineMapper_ProcessLogByGroup tests the containerLogPodPhaseTimelineMapper.ProcessLogByGroup function.
func TestPodPhaseTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewBuilder()
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
	apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
	namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "test-namespace")
	podPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-pod")
	bindingPath := commonlogk8saudit_contract.MustK8sSubresourceTimeline(ctx, podPath, "binding")
	expectedPath := mustPodPhaseTimelinePath(ctx, "test-cluster", "test-node", "test-namespace", "test-pod", "unknown")

	testCases := []struct {
		name                   string
		inputLogs              []*log.Log
		cluster                googlecloudk8scommon_contract.GoogleCloudClusterIdentity
		resourceRevisionResult inspectiontaskbase.TimelineMapperResult
		assert                 func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet)
	}{
		{
			name: "skipped because NodeName is empty",
			inputLogs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
					},
				),
			},
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if css[0] != nil {
					t.Errorf("expected cs to be nil, got %v", css[0])
				}
			},
		},
		{
			name: "mapped successfully (no audit log)",
			inputLogs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
					},
				),
			},
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, css[0]).
					HasRevision(expectedPath, &khifilev6.StagingRevision{
						ChangedTime: time.Unix(0, 0),
						Principal:   "N/A",
						VerbType:    nil,
						StateType:   commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
					})
			},
		},
		{
			name: "skipped because audit log has Pod resource timeline",
			inputLogs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
					},
				),
			},
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			resourceRevisionResult: inspectiontaskbase.TimelineMapperResult{
				Revisions: map[*khifilev6.TimelinePath]int{
					podPath: 1,
				},
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if css[0] != nil {
					t.Errorf("expected cs to be nil, got %v", css[0])
				}
			},
		},
		{
			name: "skipped because audit log has Pod binding timeline",
			inputLogs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
					},
				),
			},
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			resourceRevisionResult: inspectiontaskbase.TimelineMapperResult{
				Revisions: map[*khifilev6.TimelinePath]int{
					bindingPath: 1,
				},
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if css[0] != nil {
					t.Errorf("expected cs to be nil, got %v", css[0])
				}
			},
		},
		{
			name: "mapped once and skipped for subsequent logs with same node",
			inputLogs: []*log.Log{
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
					},
				),
				log.NewLogWithFieldSetsForTest(
					&googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
						Namespace:     "test-namespace",
						PodName:       "test-pod",
						ContainerName: "test-container",
					},
					&googlecloudlogk8scontainer_contract.GCPContainerLogNodeNameLabelFieldSet{
						NodeName: "test-node",
					},
					&log.CommonFieldSet{
						Timestamp: time.Date(2026, 5, 26, 12, 0, 1, 0, time.UTC),
					},
				),
			},
			cluster: googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			},
			assert: func(t *testing.T, ctx context.Context, css []*khifilev6.TimelineChangeSet) {
				if len(css) != 2 {
					t.Fatalf("expected 2 changesets, got %d", len(css))
				}
				testchangeset.AssertTimeline(t, css[0]).
					HasRevision(expectedPath, &khifilev6.StagingRevision{
						ChangedTime: time.Unix(0, 0),
						Principal:   "N/A",
						VerbType:    nil,
						StateType:   commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
					})
				if css[1] != nil {
					t.Errorf("expected second changeset to be nil, got %v", css[1])
				}
			},
		},
	}

	mapper := &containerLogPodPhaseTimelineMapper{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, googlecloudlogk8scontainer_contract.ClusterIdentityTaskID.Ref(), tc.cluster)
			ctx = tasktest.WithTaskResult(ctx, commonlogk8saudit_contract.ResourceRevisionLogToTimelineMapperTaskID.Ref(), tc.resourceRevisionResult)

			var css []*khifilev6.TimelineChangeSet
			state := ""
			for _, l := range tc.inputLogs {
				cs, nextState, err := mapper.ProcessLogByGroup(ctx, l, state)
				if err != nil {
					t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
				}
				css = append(css, cs)
				state = nextState
			}
			tc.assert(t, ctx, css)
		})
	}
}
