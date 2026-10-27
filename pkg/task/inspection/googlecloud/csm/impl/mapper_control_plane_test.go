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

package csm_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	commoncsmcp "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/csmcp"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

func TestIngestCSMControlPlaneLog(t *testing.T) {
	testTime := time.Date(2026, time.May, 22, 12, 0, 0, 0, time.UTC)
	customTime := time.Date(2026, time.May, 22, 13, 0, 0, 0, time.UTC)
	testCases := []struct {
		name          string
		yaml          string
		wantSummary   string
		wantLogType   *pb.LogType
		wantSeverity  *pb.Severity
		wantTimestamp time.Time
	}{
		{
			name: "jsonPayload.message and severity info",
			yaml: `
timestamp: "2026-05-22T12:00:00Z"
severity: INFO
resource:
  type: istio_control_plane
  labels:
    project_id: "fleet-project"
    location: "us-central1"
    revision: "asm-managed"
jsonPayload:
  message: "ads: full sync complete"
`,
			wantSummary:   "ads: full sync complete",
			wantLogType:   commoncsmcp.LogTypeCSMCP,
			wantSeverity:  inspectioncore.SeverityInfo,
			wantTimestamp: testTime,
		},
		{
			name: "jsonPayload.timestamp overrides log timestamp",
			yaml: `
timestamp: "2026-05-22T12:00:00Z"
severity: INFO
resource:
  type: istio_control_plane
  labels:
    project_id: "fleet-project"
    location: "us-central1"
    revision: "asm-managed"
jsonPayload:
  message: "ADS: new delta connection for node:test-pod.default-1"
  timestamp: "2026-05-22T13:00:00Z"
`,
			wantSummary:   "ADS: new delta connection for node:test-pod.default-1",
			wantLogType:   commoncsmcp.LogTypeCSMCP,
			wantSeverity:  inspectioncore.SeverityInfo,
			wantTimestamp: customTime,
		},
		{
			name: "textPayload fallback and severity warning",
			yaml: `
timestamp: "2026-05-22T12:00:00Z"
severity: WARNING
resource:
  type: istio_control_plane
  labels:
    project_id: "fleet-project"
    location: "us-central1"
    revision: "asm-managed"
textPayload: "warning starting istiod"
`,
			wantSummary:   "warning starting istiod",
			wantLogType:   commoncsmcp.LogTypeCSMCP,
			wantSeverity:  inspectioncore.SeverityWarning,
			wantTimestamp: testTime,
		},
		{
			name: "missing severity",
			yaml: `
timestamp: "2026-05-22T12:00:00Z"
resource:
  type: istio_control_plane
jsonPayload:
  message: "no severity log"
`,
			wantSummary:   "no severity log",
			wantLogType:   commoncsmcp.LogTypeCSMCP,
			wantSeverity:  nil,
			wantTimestamp: testTime,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := testlog.New(testlog.YAML(tc.yaml)).MustBuildLogEntity()
			cs, err := ingestCSMControlPlaneLog(t.Context(), l)
			if err != nil {
				t.Fatalf("ingestCSMControlPlaneLog() error = %v", err)
			}
			assert := testchangeset.AssertLog(t, cs).
				HasLogType(tc.wantLogType).
				HasSummary(tc.wantSummary).
				HasTimestamp(tc.wantTimestamp)
			if tc.wantSeverity != nil {
				assert.HasSeverity(tc.wantSeverity)
			}
		})
	}
}

func TestGroupCSMControlPlaneLog(t *testing.T) {
	testCases := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "populated fields",
			yaml: `
resource:
  type: istio_control_plane
  labels:
    project_id: "fleet-project"
    location: "us-central1"
    revision: "asm-managed"
`,
			want: "fleet-project/us-central1/asm-managed",
		},
		{
			name: "empty fields",
			yaml: `
resource:
  type: istio_control_plane
`,
			want: "//",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := testlog.New(testlog.YAML(tc.yaml)).MustBuildLogEntity()
			got := groupCSMControlPlaneLog(t.Context(), l)
			if got != tc.want {
				t.Errorf("groupCSMControlPlaneLog() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMapCSMControlPlaneLog(t *testing.T) {
	testTime := time.Date(2026, time.May, 22, 12, 0, 0, 0, time.UTC)
	clusterIdentity := k8scommon.GoogleCloudClusterIdentity{
		ClusterName: "test-cluster",
		Location:    "us-central1",
	}

	testCases := []struct {
		name          string
		yaml          string
		prevGroupData *commoncsmcp.TimelineState
		wantNextConns map[string]bool
		assert        func(t *testing.T, b *khifilev6.Builder, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "unmapped control plane log maps only to Google Managed control plane timeline",
			yaml: `
timestamp: "2026-05-22T12:00:00Z"
resource:
  type: istio_control_plane
  labels:
    project_id: "fleet-project"
    location: "us-central1"
    revision: "asm-managed"
jsonPayload:
  message: "XDS: Pushing Services:10 ConnectedEndpoints:5 Version:2026-01-01T00:00:00Z/1"
`,
			assert: func(t *testing.T, b *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				cpPath := b.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "Google Managed", Type: gcpcommon.TimelineTypeGCPProject},
					khifilev6.PathSegment{Name: "Control Plane", Type: gcpcommon.TimelineTypeGCPResourceType},
					khifilev6.PathSegment{Name: "istio_control_plane", Type: gcpcommon.TimelineTypeGCPResource},
				)
				testchangeset.AssertTimeline(t, cs).HasEvent(cpPath)
			},
		},
		{
			name: "timeout writing and client closed connection logs map to control plane, Pod CSM CP, and connection timelines",
			yaml: `
timestamp: "2026-05-22T12:00:00Z"
resource:
  type: istio_control_plane
  labels:
    project_id: "fleet-project"
    location: "us-central1"
    revision: "asm-managed"
jsonPayload:
  message: "Client closed connection worker-pod-12345abcde-xyz12.example-ns-10"
`,
			wantNextConns: map[string]bool{"example-ns/worker-pod-12345abcde-xyz12/10": true},
			assert: func(t *testing.T, b *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				cpPath := b.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "Google Managed", Type: gcpcommon.TimelineTypeGCPProject},
					khifilev6.PathSegment{Name: "Control Plane", Type: gcpcommon.TimelineTypeGCPResourceType},
					khifilev6.PathSegment{Name: "istio_control_plane", Type: gcpcommon.TimelineTypeGCPResource},
				)
				podPath := b.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "pod", Type: inspectioncore.TimelineTypeKind},
					khifilev6.PathSegment{Name: "example-ns", Type: inspectioncore.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "worker-pod-12345abcde-xyz12", Type: inspectioncore.TimelineTypeResource},
				)
				csmcpPodPath := b.TimelineAccumulator.GetPath(podPath,
					khifilev6.PathSegment{Name: "CSM CP", Type: commoncsmcp.TimelineTypeCSMCPPodLog},
				)
				connPath := b.TimelineAccumulator.GetPath(podPath,
					khifilev6.PathSegment{Name: "connection-10", Type: commoncsmcp.TimelineTypeCSMCPConnection},
				)
				testchangeset.AssertTimeline(t, cs).
					HasEvent(cpPath).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						VerbType:     inspectioncore.VerbUnknown,
						StateType:    commoncsmcp.RevisionStateCSMCPConnectionTerminated,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
			},
		},
		{
			name: "xds connection log maps to control plane, pod CSM CP, and connection timelines",
			yaml: `
timestamp: "2026-05-22T12:00:00Z"
resource:
  type: istio_control_plane
  labels:
    project_id: "fleet-project"
    location: "us-central1"
    revision: "asm-managed"
jsonPayload:
  message: "ADS: new delta connection for node:test-pod.default-1"
`,
			wantNextConns: map[string]bool{"default/test-pod/1": true},
			assert: func(t *testing.T, b *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				cpPath := b.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "Google Managed", Type: gcpcommon.TimelineTypeGCPProject},
					khifilev6.PathSegment{Name: "Control Plane", Type: gcpcommon.TimelineTypeGCPResourceType},
					khifilev6.PathSegment{Name: "istio_control_plane", Type: gcpcommon.TimelineTypeGCPResource},
				)
				podPath := b.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "pod", Type: inspectioncore.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "test-pod", Type: inspectioncore.TimelineTypeResource},
				)
				csmcpPodPath := b.TimelineAccumulator.GetPath(podPath,
					khifilev6.PathSegment{Name: "CSM CP", Type: commoncsmcp.TimelineTypeCSMCPPodLog},
				)
				connPath := b.TimelineAccumulator.GetPath(podPath,
					khifilev6.PathSegment{Name: "connection-1", Type: commoncsmcp.TimelineTypeCSMCPConnection},
				)
				testchangeset.AssertTimeline(t, cs).
					HasEvent(cpPath).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						VerbType:     inspectioncore.VerbUnknown,
						StateType:    commoncsmcp.RevisionStateCSMCPConnectionConnected,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
			},
		},
		{
			name: "xds disconnection log without prior connection adds ConnectedLogNotFound and Terminated",
			yaml: `
timestamp: "2026-05-22T12:00:00Z"
resource:
  type: istio_control_plane
  labels:
    project_id: "fleet-project"
    location: "us-central1"
    revision: "asm-managed"
jsonPayload:
  message: 'ADS: "192.0.2.1:8080" test-pod.default-1 terminated'
`,
			wantNextConns: map[string]bool{"default/test-pod/1": true},
			assert: func(t *testing.T, b *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				cpPath := b.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "Google Managed", Type: gcpcommon.TimelineTypeGCPProject},
					khifilev6.PathSegment{Name: "Control Plane", Type: gcpcommon.TimelineTypeGCPResourceType},
					khifilev6.PathSegment{Name: "istio_control_plane", Type: gcpcommon.TimelineTypeGCPResource},
				)
				podPath := b.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-cluster", Type: inspectioncore.TimelineTypeK8sCluster},
					khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion},
					khifilev6.PathSegment{Name: "pod", Type: inspectioncore.TimelineTypeKind},
					khifilev6.PathSegment{Name: "default", Type: inspectioncore.TimelineTypeNamespace},
					khifilev6.PathSegment{Name: "test-pod", Type: inspectioncore.TimelineTypeResource},
				)
				csmcpPodPath := b.TimelineAccumulator.GetPath(podPath,
					khifilev6.PathSegment{Name: "CSM CP", Type: commoncsmcp.TimelineTypeCSMCPPodLog},
				)
				connPath := b.TimelineAccumulator.GetPath(podPath,
					khifilev6.PathSegment{Name: "connection-1", Type: commoncsmcp.TimelineTypeCSMCPConnection},
				)
				testchangeset.AssertTimeline(t, cs).
					HasEvent(cpPath).
					HasEvent(csmcpPodPath).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						VerbType:     inspectioncore.VerbUnknown,
						StateType:    commoncsmcp.RevisionStateCSMCPConnectionConnectedLogNotFound,
						ResourceBody: nil,
						Principal:    "csm-cp",
					}).
					HasRevision(connPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						VerbType:     inspectioncore.VerbUnknown,
						StateType:    commoncsmcp.RevisionStateCSMCPConnectionTerminated,
						ResourceBody: nil,
						Principal:    "csm-cp",
					})
			},
		},
		{
			name: "empty fields map to Google Managed control plane timeline",
			yaml: `
timestamp: "2026-05-22T12:00:00Z"
resource:
  type: istio_control_plane
`,
			assert: func(t *testing.T, b *khifilev6.Builder, cs *khifilev6.TimelineChangeSet) {
				wantPath := b.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "Google Managed", Type: gcpcommon.TimelineTypeGCPProject},
					khifilev6.PathSegment{Name: "Control Plane", Type: gcpcommon.TimelineTypeGCPResourceType},
					khifilev6.PathSegment{Name: "istio_control_plane", Type: gcpcommon.TimelineTypeGCPResource},
				)
				testchangeset.AssertTimeline(t, cs).HasEvent(wantPath)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := khifilev6.NewTestBuilder(id.NewGenerator())
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

			l := testlog.New(testlog.YAML(tc.yaml)).MustBuildLogEntity()
			cs, gotState, err := mapCSMControlPlaneLog(ctx, l, tc.prevGroupData, clusterIdentity)
			if err != nil {
				t.Fatalf("mapCSMControlPlaneLog() error = %v", err)
			}

			tc.assert(t, builder, cs)

			if tc.wantNextConns != nil {
				if gotState == nil {
					t.Fatalf("gotState is nil, want ObservedConnections %v", tc.wantNextConns)
				}
				if diff := cmp.Diff(tc.wantNextConns, gotState.ObservedConnections); diff != "" {
					t.Errorf("ObservedConnections mismatch (-want +got):\n%s", diff)
				}
			}

			dummyID := uint32(1)
			if err := builder.LogAccumulator.AddLog(&khifilev6.StagingLog{
				Log:      l,
				Severity: &pb.Severity{Id: &dummyID},
				LogType:  &pb.LogType{Id: &dummyID},
			}); err != nil {
				t.Fatalf("AddLog() failed: %v", err)
			}
			if err := cs.Flush(builder.TimelineAccumulator, builder.LogAccumulator); err != nil {
				t.Fatalf("Flush() failed: %v", err)
			}
		})
	}
}
