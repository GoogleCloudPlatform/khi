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

package googlecloudlogcsmcp_impl

import (
	"testing"
	"time"

	inspectiontaskbasetest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbasetest"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	commonlogcsmcp_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogcsmcp/contract"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogcsmcp_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsmcp/contract"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

func TestIstiodLogFilterTask(t *testing.T) {
	testCases := []inspectiontaskbasetest.FilterTaskTestCase{
		{
			Description: "istiod discovery container log",
			Log: testlog.New(
				testlog.YAML(`
resource:
  labels:
    container_name: discovery
    pod_name: istiod-asm-1234
`),
			).MustBuildLogEntity(),
			WantIncluded: true,
		},
		{
			Description: "other container in istiod pod",
			Log: testlog.New(
				testlog.YAML(`
resource:
  labels:
    container_name: istio-proxy
    pod_name: istiod-asm-1234
`),
			).MustBuildLogEntity(),
			WantIncluded: false,
		},
		{
			Description: "discovery container in other pod",
			Log: testlog.New(
				testlog.YAML(`
resource:
  labels:
    container_name: discovery
    pod_name: other-pod
`),
			).MustBuildLogEntity(),
			WantIncluded: false,
		},
		{
			Description: "empty log",
			Log: testlog.New(
				testlog.YAML(``),
			).MustBuildLogEntity(),
			WantIncluded: false,
		},
	}

	inspectiontaskbasetest.AssertFilterTask(t, IstiodLogFilterTask, googlecloudlogk8scontainer_contract.ListLogEntriesTaskID.Ref(), testCases)
}

func TestCSMCPTimelineMapper_ProcessLogByGroup(t *testing.T) {
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	ctx = tasktest.WithTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(), googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ClusterName: "test-cluster",
		Location:    "test-location",
	})

	testTime := time.Date(2026, time.May, 22, 10, 0, 0, 0, time.UTC)

	clusterPath := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
	apiPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
	kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiPath, "pod")

	// Client Pod timeline paths
	clientNsPath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindPath, "default")
	clientPodPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, clientNsPath, "test-pod")
	csmcpPodPath := commonlogcsmcp_contract.MustCSMCPPodLogTimeline(ctx, clientPodPath)
	connPath := commonlogcsmcp_contract.MustCSMCPConnectionTimeline(ctx, clientPodPath, "1")

	customTime := time.Date(2026, time.May, 22, 11, 0, 0, 0, time.UTC)

	testCases := []struct {
		name            string
		logMessage      string
		pods            []commonlogcsmcp_contract.PodIdentifier
		prevGroupData   *commonlogcsmcp_contract.TimelineState
		customTimestamp *time.Time
		wantNextConns   map[string]bool
		assert          func(*testing.T, *khifilev6.TimelineChangeSet)
	}{
		{
			name:       "new delta connection log",
			logMessage: "ADS: new delta connection for node:test-pod.default-1",
			pods: []commonlogcsmcp_contract.PodIdentifier{
				{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
			},
			wantNextConns: map[string]bool{"default/test-pod/1": true},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						VerbType:     inspectioncore.VerbUnknown,
						StateType:    commonlogcsmcp_contract.RevisionStateCSMCPConnectionConnected,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
			},
		},
		{
			name:            "custom timestamp overrides log timestamp",
			logMessage:      "ADS: new connection for node:test-pod.default-1",
			customTimestamp: &customTime,
			pods: []commonlogcsmcp_contract.PodIdentifier{
				{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
			},
			wantNextConns: map[string]bool{"default/test-pod/1": true},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  customTime,
						VerbType:     inspectioncore.VerbUnknown,
						StateType:    commonlogcsmcp_contract.RevisionStateCSMCPConnectionConnected,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
			},
		},
		{
			name:       "new standard connection log",
			logMessage: "ADS: new connection for node:test-pod.default-1",
			pods: []commonlogcsmcp_contract.PodIdentifier{
				{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
			},
			wantNextConns: map[string]bool{"default/test-pod/1": true},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						VerbType:     inspectioncore.VerbUnknown,
						StateType:    commonlogcsmcp_contract.RevisionStateCSMCPConnectionConnected,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
			},
		},
		{
			name:       "terminated log with previous connected state",
			logMessage: `ADS: "192.0.2.1:41780" test-pod.default-1 terminated`,
			pods: []commonlogcsmcp_contract.PodIdentifier{
				{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
			},
			prevGroupData: &commonlogcsmcp_contract.TimelineState{
				ObservedConnections: map[string]bool{"default/test-pod/1": true},
			},
			wantNextConns: map[string]bool{"default/test-pod/1": true},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						VerbType:     inspectioncore.VerbUnknown,
						StateType:    commonlogcsmcp_contract.RevisionStateCSMCPConnectionTerminated,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
				if revisions := cs.GetRevisions(connPath); len(revisions) != 1 {
					t.Errorf("expected exactly 1 revision for %v, got %d", connPath, len(revisions))
				}
			},
		},
		{
			name:       "terminated log without previous connected state (nil prevGroupData)",
			logMessage: `ADS: "192.0.2.1:41780" test-pod.default-1 terminated`,
			pods: []commonlogcsmcp_contract.PodIdentifier{
				{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
			},
			prevGroupData: nil,
			wantNextConns: map[string]bool{"default/test-pod/1": true},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						VerbType:     inspectioncore.VerbUnknown,
						StateType:    commonlogcsmcp_contract.RevisionStateCSMCPConnectionConnectedLogNotFound,
						ResourceBody: nil,
						Principal:    "csm-cp",
					}).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						VerbType:     inspectioncore.VerbUnknown,
						StateType:    commonlogcsmcp_contract.RevisionStateCSMCPConnectionTerminated,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
			},
		},
		{
			name:       "terminated log without previous connected state in existing state",
			logMessage: `ADS: "192.0.2.1:41780" test-pod.default-1 terminated`,
			pods: []commonlogcsmcp_contract.PodIdentifier{
				{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
			},
			prevGroupData: &commonlogcsmcp_contract.TimelineState{
				ObservedConnections: map[string]bool{},
			},
			wantNextConns: map[string]bool{"default/test-pod/1": true},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						VerbType:     inspectioncore.VerbUnknown,
						StateType:    commonlogcsmcp_contract.RevisionStateCSMCPConnectionConnectedLogNotFound,
						ResourceBody: nil,
						Principal:    "csm-cp",
					}).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						VerbType:     inspectioncore.VerbUnknown,
						StateType:    commonlogcsmcp_contract.RevisionStateCSMCPConnectionTerminated,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
			},
		},
		{
			name:       "push log without connection id",
			logMessage: "CDS: PUSH request for node:test-pod.default resources:87",
			pods: []commonlogcsmcp_contract.PodIdentifier{
				{Name: "test-pod", Namespace: "default", ConnectionID: ""},
			},
			wantNextConns: nil,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(csmcpPodPath).
					HasNoRevision(connPath)
			},
		},
		{
			name:          "non-xds log from discovery container",
			logMessage:    "info Starting Istiod control plane...",
			pods:          nil,
			wantNextConns: nil,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasNoEvent(csmcpPodPath).
					HasNoRevision(connPath)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := googlecloudlogcsmcp_contract.FieldSet{
				Timestamp: tc.customTimestamp,
				Message:   tc.logMessage,
				Pods:      tc.pods,
			}
			logObj := testlog.NewMockLog(testTime, fs)

			mapper := &csmcpTimelineMapper{}
			cs, nextState, err := mapper.ProcessLogByGroup(ctx, logObj, tc.prevGroupData)
			if err != nil {
				t.Fatalf("ProcessLogByGroup() failed: %v", err)
			}

			if tc.assert != nil {
				tc.assert(t, cs)
			}

			if tc.wantNextConns != nil {
				if nextState == nil {
					t.Fatalf("ProcessLogByGroup() returned nil nextState, want non-nil")
				}
				if diff := cmp.Diff(tc.wantNextConns, nextState.ObservedConnections); diff != "" {
					t.Errorf("ProcessLogByGroup() nextState mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCSMCPTimelineMapper_ProcessLogByGroup_ClusterNameFallback(t *testing.T) {
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	ctx = tasktest.WithTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(), googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ClusterName: "",
	})

	testTime := time.Date(2026, time.May, 22, 10, 0, 0, 0, time.UTC)
	clusterPath := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "fallback-cluster")
	apiPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
	kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiPath, "pod")
	clientNsPath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindPath, "default")
	clientPodPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, clientNsPath, "test-pod")
	csmcpPodPath := commonlogcsmcp_contract.MustCSMCPPodLogTimeline(ctx, clientPodPath)

	fs := googlecloudlogcsmcp_contract.FieldSet{
		ClusterName: "fallback-cluster",
		Message:     "ADS: new connection for node:test-pod.default-1",
		Pods: []commonlogcsmcp_contract.PodIdentifier{
			{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
		},
	}
	logObj := testlog.NewMockLog(testTime, fs)

	mapper := &csmcpTimelineMapper{}
	cs, _, err := mapper.ProcessLogByGroup(ctx, logObj, nil)
	if err != nil {
		t.Fatalf("ProcessLogByGroup() failed: %v", err)
	}

	testchangeset.AssertTimeline(t, cs).HasEvent(csmcpPodPath)
}

func TestCSMCPTimelineMapper_ProcessLogByGroup_SequentialProcessing(t *testing.T) {
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	ctx = tasktest.WithTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(), googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ClusterName: "test-cluster",
		Location:    "test-location",
	})

	testTime1 := time.Date(2026, time.May, 22, 10, 0, 0, 0, time.UTC)
	testTime2 := time.Date(2026, time.May, 22, 10, 5, 0, 0, time.UTC)

	clusterPath := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
	apiPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
	kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiPath, "pod")
	clientNsPath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindPath, "default")
	clientPodPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, clientNsPath, "test-pod")
	csmcpPodPath := commonlogcsmcp_contract.MustCSMCPPodLogTimeline(ctx, clientPodPath)
	connPath := commonlogcsmcp_contract.MustCSMCPConnectionTimeline(ctx, clientPodPath, "1")

	mapper := &csmcpTimelineMapper{}

	// Log 1: connected
	fs1 := googlecloudlogcsmcp_contract.FieldSet{
		Message: "ADS: new connection for node:test-pod.default-1",
		Pods: []commonlogcsmcp_contract.PodIdentifier{
			{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
		},
	}
	log1 := testlog.NewMockLog(testTime1, fs1)
	cs1, state1, err := mapper.ProcessLogByGroup(ctx, log1, nil)
	if err != nil {
		t.Fatalf("ProcessLogByGroup() for log1 failed: %v", err)
	}
	testchangeset.AssertTimeline(t, cs1).
		HasEvent(csmcpPodPath).
		HasRevision(connPath, &khifilev6.StagingRevision{
			ChangedTime:  testTime1,
			VerbType:     inspectioncore.VerbUnknown,
			StateType:    commonlogcsmcp_contract.RevisionStateCSMCPConnectionConnected,
			ResourceBody: nil,
			Principal:    "csm-cp",
		})

	// Log 2: terminated, passing state1 from log1
	fs2 := googlecloudlogcsmcp_contract.FieldSet{
		Message: `ADS: "192.0.2.1:41780" test-pod.default-1 terminated`,
		Pods: []commonlogcsmcp_contract.PodIdentifier{
			{Name: "test-pod", Namespace: "default", ConnectionID: "1"},
		},
	}
	log2 := testlog.NewMockLog(testTime2, fs2)
	cs2, _, err := mapper.ProcessLogByGroup(ctx, log2, state1)
	if err != nil {
		t.Fatalf("ProcessLogByGroup() for log2 failed: %v", err)
	}

	// Should have exactly 1 revision (Terminated), without ConnectedLogNotFound synthetic revision
	testchangeset.AssertTimeline(t, cs2).
		HasEvent(csmcpPodPath).
		HasRevision(connPath, &khifilev6.StagingRevision{
			ChangedTime:  testTime2,
			VerbType:     inspectioncore.VerbUnknown,
			StateType:    commonlogcsmcp_contract.RevisionStateCSMCPConnectionTerminated,
			ResourceBody: nil,
			Principal:    "csm-cp",
		})
	if revisions := cs2.GetRevisions(connPath); len(revisions) != 1 {
		t.Errorf("expected exactly 1 revision for %v, got %d", connPath, len(revisions))
	}
}
