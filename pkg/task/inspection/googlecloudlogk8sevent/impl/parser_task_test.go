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

package googlecloudlogk8sevent_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogk8sevent_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8sevent/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestLogToTimelineMapperTask(t *testing.T) {
	// Initialize the shared Builder reference.
	builder := khifilev6.NewTestBuilder(id.NewGenerator())

	testCases := []struct {
		desc                    string
		input                   googlecloudlogk8sevent_contract.KubernetesEventFieldSet
		resourceIdentitiesByUID map[string]*commonlogk8saudit_contract.ResourceIdentity
		assert                  func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "namespaced resource event",
			input: googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
				ClusterName:  "test-cluster",
				APIVersion:   "apps/v1",
				ResourceKind: "deployment",
				Namespace:    "default",
				Resource:     "test-deployment",
				Reason:       "ScalingReplicaSet",
				Message:      "Scaled up replica set test-deployment-xyz to 3",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
				apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "apps/v1")
				kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "deployment")
				namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "default")
				expectedPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-deployment")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath).
					HasEventCount(1)
			},
		},
		{
			desc: "cluster-scoped resource event",
			input: googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
				ClusterName:  "test-cluster",
				APIVersion:   "core/v1",
				ResourceKind: "node",
				Namespace:    "cluster-scope",
				Resource:     "my-node",
				Reason:       "Starting",
				Message:      "Starting kubelet.",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
				apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
				kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "node")
				expectedPath := commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kindTimeline, "my-node")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath).
					HasEventCount(1)
			},
		},
		{
			desc: "empty resource name event (EventExporter fallback)",
			input: googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
				ProjectID:    "test-project",
				ClusterName:  "test-cluster",
				APIVersion:   "",
				ResourceKind: "",
				Namespace:    "",
				Resource:     "",
				Reason:       "ExporterStarted",
				Message:      "Event exporter started watching.",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, "test-project")
				clusterTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, projectTimeline, "test-cluster")
				otherTimeline := builder.TimelineAccumulator.GetPath(clusterTimeline, khifilev6.PathSegment{
					Name: "other",
					Type: googlecloudcommon_contract.TimelineTypeOtherGKEResources,
				})
				expectedPath := builder.TimelineAccumulator.GetPath(otherTimeline, khifilev6.PathSegment{
					Name: "event-exporter",
					Type: googlecloudlogk8sevent_contract.TimelineTypeEventExporter,
				})

				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath).
					HasEventCount(1)
			},
		},
		{
			desc: "event with UID in message matching another resource",
			input: googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
				ClusterName:  "test-cluster",
				APIVersion:   "batch/v1",
				ResourceKind: "job",
				Namespace:    "default",
				Resource:     "my-job",
				Reason:       "SuccessfulCreate",
				Message:      `Created pod: my-job-pod (UID: a1b2c3d4-e5f6-7890-abcd-ef0123456789)`,
			},
			resourceIdentitiesByUID: map[string]*commonlogk8saudit_contract.ResourceIdentity{
				"a1b2c3d4-e5f6-7890-abcd-ef0123456789": {
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "my-job-pod",
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
				jobAPIVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "batch/v1")
				jobKindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, jobAPIVersionTimeline, "job")
				jobNamespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, jobKindTimeline, "default")
				jobExpectedPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, jobNamespaceTimeline, "my-job")

				podAPIVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
				podKindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, podAPIVersionTimeline, "pod")
				podNamespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, podKindTimeline, "default")
				podExpectedPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, podNamespaceTimeline, "my-job-pod")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(jobExpectedPath).
					HasEvent(podExpectedPath).
					HasEventCount(2)
			},
		},
		{
			desc: "event with UID in message matching primary resource does not duplicate event",
			input: googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
				ClusterName:  "test-cluster",
				APIVersion:   "core/v1",
				ResourceKind: "pod",
				Namespace:    "default",
				Resource:     "my-pod",
				Reason:       "Killing",
				Message:      `Stopping container my-pod with uid 11112222-3333-4444-5555-666677778888`,
			},
			resourceIdentitiesByUID: map[string]*commonlogk8saudit_contract.ResourceIdentity{
				"11112222-3333-4444-5555-666677778888": {
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "my-pod",
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
				apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
				kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
				namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "default")
				expectedPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "my-pod")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath).
					HasEventCount(1)
			},
		},
		{
			desc: "event with multiple distinct UIDs and duplicate secondary UID occurrences",
			input: googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
				ClusterName:  "test-cluster",
				APIVersion:   "batch/v1",
				ResourceKind: "job",
				Namespace:    "default",
				Resource:     "my-job",
				Reason:       "SuccessfulCreate",
				Message:      `Created pod: pod-1 (UID: uid-1) and pod-2 (UID: uid-2). Duplicate ref: uid-1`,
			},
			resourceIdentitiesByUID: map[string]*commonlogk8saudit_contract.ResourceIdentity{
				"uid-1": {
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "pod-1",
				},
				"uid-2": {
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "pod-2",
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
				jobAPIVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "batch/v1")
				jobKindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, jobAPIVersionTimeline, "job")
				jobNamespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, jobKindTimeline, "default")
				jobExpectedPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, jobNamespaceTimeline, "my-job")

				pod1APIVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
				pod1KindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, pod1APIVersionTimeline, "pod")
				pod1NamespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, pod1KindTimeline, "default")
				pod1ExpectedPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, pod1NamespaceTimeline, "pod-1")

				pod2APIVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
				pod2KindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, pod2APIVersionTimeline, "pod")
				pod2NamespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, pod2KindTimeline, "default")
				pod2ExpectedPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, pod2NamespaceTimeline, "pod-2")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(jobExpectedPath).
					HasEvent(pod1ExpectedPath).
					HasEvent(pod2ExpectedPath).
					HasEventCount(3)
			},
		},
		{
			desc: "event with empty message stages primary resource only",
			input: googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
				ClusterName:  "test-cluster",
				APIVersion:   "core/v1",
				ResourceKind: "pod",
				Namespace:    "default",
				Resource:     "my-pod",
				Message:      "",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
				apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
				kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
				namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "default")
				expectedPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "my-pod")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath).
					HasEventCount(1)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := testlog.NewMockLog(tc.input)

			// Set up context with the same Builder reference.
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			finder := patternfinder.NewNaivePatternFinder[*commonlogk8saudit_contract.ResourceIdentity]()
			if tc.resourceIdentitiesByUID != nil {
				for k, v := range tc.resourceIdentitiesByUID {
					_ = finder.AddPattern(k, v)
				}
			}
			ctx = tasktest.WithTaskResult(ctx, commonlogk8saudit_contract.ResourceUIDPatternFinderTaskID.Ref(), finder)
			mapper := KubernetesEventTimelineMapper{}

			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup returned an unexpected error: %v", err)
			}

			tc.assert(t, ctx, cs)
		})
	}
}

func TestKubernetesEventLogIngester_ProcessLog(t *testing.T) {
	testCases := []struct {
		desc                    string
		input                   *log.Log
		resourceIdentitiesByUID map[string]*commonlogk8saudit_contract.ResourceIdentity
		assert                  func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			desc: "successful event log ingestion without UID",
			input: testlog.NewMockLog(
				time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC),
				inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityInfo,
				},
				googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
					Reason:  "Scheduled",
					Message: "Successfully assigned default/test-pod to node-1",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasTimestamp(time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)).
					HasSeverity(inspectioncore_contract.SeverityInfo).
					HasLogType(commonlogk8saudit_contract.LogTypeEvent).
					HasSummary("【Scheduled】Successfully assigned default/test-pod to node-1")
			},
		},
		{
			desc: "replaces resource UID in message with readable name",
			input: testlog.NewMockLog(
				time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC),
				googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
					Reason:  "SuccessfulCreate",
					Message: "Created pod: my-job-pod (UID: a1b2c3d4-e5f6-7890-abcd-ef0123456789)",
				},
			),
			resourceIdentitiesByUID: map[string]*commonlogk8saudit_contract.ResourceIdentity{
				"a1b2c3d4-e5f6-7890-abcd-ef0123456789": {
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "my-job-pod",
				},
			},
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("【SuccessfulCreate】Created pod: my-job-pod (UID: 【my-job-pod (Namespace: default, APIVersion: core/v1, Kind: pod)】)")
			},
		},
		{
			desc: "replaces multiple distinct UIDs and duplicate UID occurrences in message",
			input: testlog.NewMockLog(
				time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC),
				googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
					Reason:  "SyncLoop",
					Message: `Processing pod: a1b2c3d4-e5f6-7890-abcd-ef0123456789 and service: f9e8d7c6-b5a4-3210-fedc-ba9876543210 (retry for a1b2c3d4-e5f6-7890-abcd-ef0123456789)`,
				},
			),
			resourceIdentitiesByUID: map[string]*commonlogk8saudit_contract.ResourceIdentity{
				"a1b2c3d4-e5f6-7890-abcd-ef0123456789": {
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "my-job-pod",
				},
				"f9e8d7c6-b5a4-3210-fedc-ba9876543210": {
					APIVersion: "core/v1",
					Kind:       "service",
					Namespace:  "default",
					Name:       "my-service",
				},
			},
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("【SyncLoop】Processing pod: 【my-job-pod (Namespace: default, APIVersion: core/v1, Kind: pod)】 and service: 【my-service (Namespace: default, APIVersion: core/v1, Kind: service)】 (retry for 【my-job-pod (Namespace: default, APIVersion: core/v1, Kind: pod)】)")
			},
		},
		{
			desc: "event with empty message",
			input: testlog.NewMockLog(
				time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC),
				googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
					Reason:  "Scheduled",
					Message: "",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasTimestamp(time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)).
					HasSeverity(inspectioncore_contract.SeverityUnknown).
					HasLogType(commonlogk8saudit_contract.LogTypeEvent).
					HasSummary("【Scheduled】")
			},
		},
	}

	ingester := &KubernetesEventLogIngester{}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			finder := patternfinder.NewNaivePatternFinder[*commonlogk8saudit_contract.ResourceIdentity]()
			if tc.resourceIdentitiesByUID != nil {
				for k, v := range tc.resourceIdentitiesByUID {
					_ = finder.AddPattern(k, v)
				}
			}
			ctx := tasktest.WithTaskResult(t.Context(), commonlogk8saudit_contract.ResourceUIDPatternFinderTaskID.Ref(), finder)

			cs, err := ingester.ProcessLog(ctx, tc.input)
			if err != nil {
				t.Fatalf("ProcessLog() returned unexpected error: %v", err)
			}

			tc.assert(t, cs)
		})
	}
}
