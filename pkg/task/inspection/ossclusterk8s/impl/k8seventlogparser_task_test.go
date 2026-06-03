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

package ossclusterk8s_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	ossclusterk8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/ossclusterk8s/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestOSSK8sEventLogIngester_ProcessLog(t *testing.T) {
	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name: "successful event log ingestion",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC),
				},
				&ossclusterk8s_contract.OSSK8sEventFieldSet{
					Reason:  "Scheduled",
					Message: "Successfully assigned default/test-pod to node-1",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasTimestamp(time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)).
					HasLogType(commonlogk8saudit_contract.LogTypeEvent).
					HasSummary("【Scheduled】Successfully assigned default/test-pod to node-1")
			},
		},
	}

	ingester := &OSSK8sEventLogIngester{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()

			cs, err := ingester.ProcessLog(ctx, tc.input)
			if err != nil {
				t.Fatalf("ProcessLog() returned unexpected error: %v", err)
			}

			tc.assert(t, cs)
		})
	}
}

func TestOSSK8sEventTimelineMapper_ProcessLogByGroup(t *testing.T) {
	// Initialize the shared Builder reference.
	builder := khifilev6.NewBuilder()

	testCases := []struct {
		desc   string
		input  ossclusterk8s_contract.OSSK8sEventFieldSet
		assert func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "namespaced resource event",
			input: ossclusterk8s_contract.OSSK8sEventFieldSet{
				APIVersion:   "apps/v1",
				ResourceKind: "deployment",
				Namespace:    "default",
				Resource:     "test-deployment",
				Subresource:  "",
				Reason:       "ScalingReplicaSet",
				Message:      "Scaled up replica set test-deployment-xyz to 3",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "cluster")
				apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "apps/v1")
				kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "deployment")
				namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "default")
				expectedPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-deployment")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath)
			},
		},
		{
			desc: "namespaced subresource event",
			input: ossclusterk8s_contract.OSSK8sEventFieldSet{
				APIVersion:   "apps/v1",
				ResourceKind: "deployment",
				Namespace:    "default",
				Resource:     "test-deployment",
				Subresource:  "status",
				Reason:       "ScalingReplicaSet",
				Message:      "Scaled up replica set test-deployment-xyz to 3",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "cluster")
				apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "apps/v1")
				kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "deployment")
				namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "default")
				resourceTimeline := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-deployment")
				expectedPath := commonlogk8saudit_contract.MustK8sSubresourceTimeline(ctx, resourceTimeline, "status")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath)
			},
		},
		{
			desc: "cluster-scoped resource event",
			input: ossclusterk8s_contract.OSSK8sEventFieldSet{
				APIVersion:   "core/v1",
				ResourceKind: "node",
				Namespace:    "cluster-scope",
				Resource:     "my-node",
				Subresource:  "",
				Reason:       "Starting",
				Message:      "Starting kubelet.",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "cluster")
				apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
				kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "node")
				expectedPath := commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kindTimeline, "my-node")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(expectedPath)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(&tc.input)

			// Set up context with the same Builder reference.
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			mapper := OSSK8sEventTimelineMapper{}

			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup returned an unexpected error: %v", err)
			}

			tc.assert(t, ctx, cs)
		})
	}
}
