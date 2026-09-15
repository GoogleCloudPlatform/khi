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

package commonlogcsmcp_contract

import (
	"testing"
	"time"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestMapPodAndConnectionTimelines(t *testing.T) {
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	testTime := time.Date(2026, time.May, 22, 10, 0, 0, 0, time.UTC)

	clusterPath := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
	apiPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
	kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiPath, "pod")
	clientNsPath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindPath, "default")
	clientPodPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, clientNsPath, "test-pod")
	csmcpPodPath := MustCSMCPPodLogTimeline(ctx, clientPodPath)
	connPath := MustCSMCPConnectionTimeline(ctx, clientPodPath, "1")

	testCases := []struct {
		name                    string
		clusterName             string
		logMessage              string
		pods                    []PodIdentifier
		state                   *TimelineState
		wantObservedConnections map[string]bool
		assert                  func(*testing.T, *khifilev6.TimelineChangeSet)
	}{
		{
			name:        "new connection event initializes state and marks connected",
			clusterName: "test-cluster",
			logMessage:  "ADS: new delta connection for node:test-pod.default-1",
			pods: []PodIdentifier{
				{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
			},
			wantObservedConnections: map[string]bool{
				"default/test-pod/1": true,
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						VerbType:     inspectioncore_contract.VerbUnknown,
						StateType:    RevisionStateCSMCPConnectionConnected,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
			},
		},
		{
			name:        "disconnection event with previous connected state",
			clusterName: "test-cluster",
			logMessage:  `ADS: "192.0.2.1:41780" test-pod.default-1 terminated`,
			pods: []PodIdentifier{
				{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
			},
			state: &TimelineState{
				ObservedConnections: map[string]bool{"default/test-pod/1": true},
			},
			wantObservedConnections: map[string]bool{
				"default/test-pod/1": true,
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						VerbType:     inspectioncore_contract.VerbUnknown,
						StateType:    RevisionStateCSMCPConnectionTerminated,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
				if revisions := cs.GetRevisions(connPath); len(revisions) != 1 {
					t.Errorf("expected exactly 1 revision for %v, got %d", connPath, len(revisions))
				}
			},
		},
		{
			name:        "disconnection event with nil initial state adds ConnectedLogNotFound",
			clusterName: "test-cluster",
			logMessage:  `ADS: "192.0.2.1:41780" test-pod.default-1 terminated`,
			pods: []PodIdentifier{
				{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
			},
			state: nil,
			wantObservedConnections: map[string]bool{
				"default/test-pod/1": true,
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						VerbType:     inspectioncore_contract.VerbUnknown,
						StateType:    RevisionStateCSMCPConnectionConnectedLogNotFound,
						ResourceBody: nil,
						Principal:    "csm-cp",
					}).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						VerbType:     inspectioncore_contract.VerbUnknown,
						StateType:    RevisionStateCSMCPConnectionTerminated,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
			},
		},
		{
			name:        "disconnection event without previous connected state in existing state adds ConnectedLogNotFound",
			clusterName: "test-cluster",
			logMessage:  `ADS: "192.0.2.1:41780" test-pod.default-1 terminated`,
			pods: []PodIdentifier{
				{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
			},
			state: &TimelineState{
				ObservedConnections: map[string]bool{},
			},
			wantObservedConnections: map[string]bool{
				"default/test-pod/1": true,
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						VerbType:     inspectioncore_contract.VerbUnknown,
						StateType:    RevisionStateCSMCPConnectionConnectedLogNotFound,
						ResourceBody: nil,
						Principal:    "csm-cp",
					}).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						VerbType:     inspectioncore_contract.VerbUnknown,
						StateType:    RevisionStateCSMCPConnectionTerminated,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
			},
		},
		{
			name:        "push log without connection id does not create connection timeline",
			clusterName: "test-cluster",
			logMessage:  "CDS: PUSH request for node:test-pod.default",
			pods: []PodIdentifier{
				{Name: "test-pod", Namespace: "default", ConnectionID: ""},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(csmcpPodPath).
					HasNoRevision(connPath)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := &log.Log{Timestamp: testTime}
			cs := khifilev6.NewTimelineChangeSet(l)
			gotState := MapPodAndConnectionTimelines(ctx, cs, tc.clusterName, tc.logMessage, testTime, tc.pods, tc.state)
			tc.assert(t, cs)
			if tc.wantObservedConnections != nil {
				if gotState == nil {
					t.Fatalf("gotState is nil, want non-nil with %v", tc.wantObservedConnections)
				}
				if diff := cmp.Diff(tc.wantObservedConnections, gotState.ObservedConnections); diff != "" {
					t.Errorf("ObservedConnections mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}

	t.Run("sequential connection then disconnection preserves state and omits ConnectedLogNotFound", func(t *testing.T) {
		l1 := &log.Log{Timestamp: testTime}
		cs1 := khifilev6.NewTimelineChangeSet(l1)
		connPods := []PodIdentifier{
			{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
		}
		state := MapPodAndConnectionTimelines(ctx, cs1, "test-cluster", "ADS: new connection for node:test-pod.default-1", testTime, connPods, nil)

		termTime := testTime.Add(time.Minute)
		l2 := &log.Log{Timestamp: termTime}
		cs2 := khifilev6.NewTimelineChangeSet(l2)
		MapPodAndConnectionTimelines(ctx, cs2, "test-cluster", `ADS: "192.0.2.1:41780" test-pod.default-1 terminated`, termTime, connPods, state)

		testchangeset.AssertTimeline(t, cs2).
			HasEvent(csmcpPodPath).
			HasRevision(connPath, &khifilev6.StagingRevision{
				ChangedTime:  termTime,
				VerbType:     inspectioncore_contract.VerbUnknown,
				StateType:    RevisionStateCSMCPConnectionTerminated,
				ResourceBody: nil,
				Principal:    "csm-cp",
			})
		if revisions := cs2.GetRevisions(connPath); len(revisions) != 1 {
			t.Errorf("len(revisions) on disconnection when previously connected = %d, want 1", len(revisions))
		}
	})
}
