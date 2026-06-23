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

package googlecloudlogcsm_impl

import (
	"testing"
	"time"
	"unique"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestCSMTrafficDirectorLogIngester_ProcessLog(t *testing.T) {
	testCases := []struct {
		desc        string
		inputAudit  *googlecloudcommon_contract.GCPAuditLogFieldSet
		wantSummary string
	}{
		{
			desc: "immediate operation CreateMesh",
			inputAudit: &googlecloudcommon_contract.GCPAuditLogFieldSet{
				MethodName:     "google.cloud.networkservices.v1.NetworkServices.CreateMesh",
				OperationFirst: true,
				OperationLast:  true,
			},
			wantSummary: "Succeeded: google.cloud.networkservices.v1.NetworkServices.CreateMesh",
		},
		{
			desc: "long running operation CreateMesh started",
			inputAudit: &googlecloudcommon_contract.GCPAuditLogFieldSet{
				MethodName:     "google.cloud.networkservices.v1.NetworkServices.CreateMesh",
				OperationFirst: true,
				OperationLast:  false,
			},
			wantSummary: "Start: google.cloud.networkservices.v1.NetworkServices.CreateMesh",
		},
		{
			desc: "long running operation CreateMesh finished",
			inputAudit: &googlecloudcommon_contract.GCPAuditLogFieldSet{
				MethodName:     "google.cloud.networkservices.v1.NetworkServices.CreateMesh",
				OperationFirst: false,
				OperationLast:  true,
			},
			wantSummary: "Succeeded: google.cloud.networkservices.v1.NetworkServices.CreateMesh",
		},
	}

	ingester := googlecloudcommon_contract.NewGCPOperationLogIngester(googlecloudlogcsm_contract.CSMTrafficDirectorFieldSetReaderTaskID.Ref(), googlecloudlogcsm_contract.LogTypeCSMAccessLog)
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)},
				tc.inputAudit,
			)
			cs, err := ingester.ProcessLog(t.Context(), l)
			if err != nil {
				t.Fatalf("ProcessLog() failed: %v", err)
			}
			testchangeset.AssertLog(t, cs).
				HasSummary(tc.wantSummary).
				HasLogType(googlecloudlogcsm_contract.LogTypeCSMAccessLog)
		})
	}
}

func TestCSMTrafficDirectorLogToTimelineMapper_ProcessLogByGroup(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)

	createReader := func(t *testing.T, data map[string]any) *structured.NodeReader {
		if data == nil {
			return nil
		}
		node, err := structured.FromGoValue(data, &structured.AlphabeticalGoMapKeyOrderProvider{})
		if err != nil {
			t.Fatalf("failed to create node: %v", err)
		}
		return structured.NewNodeReader(node)
	}

	tests := []struct {
		name           string
		auditFieldSets []*googlecloudcommon_contract.GCPAuditLogFieldSet
		assert         func(t *testing.T, builder *khifilev6.Builder, results []*khifilev6.TimelineChangeSet)
	}{
		{
			name: "Mesh creation log",
			auditFieldSets: []*googlecloudcommon_contract.GCPAuditLogFieldSet{
				{
					ProjectID:      "test-project",
					MethodName:     "google.cloud.networkservices.v1.NetworkServices.CreateMesh",
					ResourceName:   "projects/test-project/locations/global/meshes/test-mesh",
					PrincipalEmail: "user@example.com",
					OperationFirst: true,
					OperationLast:  true,
					Response:       createReader(t, map[string]any{"name": "test-mesh", "description": "test"}),
				},
			},
			assert: func(t *testing.T, builder *khifilev6.Builder, results []*khifilev6.TimelineChangeSet) {
				wantMeshPath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "test-project", Type: googlecloudcommon_contract.TimelineTypeGCPProject},
					khifilev6.PathSegment{Name: "meshes", Type: googlecloudcommon_contract.TimelineTypeGCPResourceType},
					khifilev6.PathSegment{Name: "test-mesh", Type: googlecloudcommon_contract.TimelineTypeGCPResource},
				)

				testchangeset.AssertTimeline(t, results[0]).
					HasEvent(wantMeshPath)
			},
		},
		{
			name: "Long running operation sequence",
			auditFieldSets: []*googlecloudcommon_contract.GCPAuditLogFieldSet{
				{
					ProjectID:      "p",
					OperationID:    "op1",
					MethodName:     "google.cloud.networkservices.v1.NetworkServices.CreateMesh",
					ResourceName:   "projects/p/locations/global/meshes/m1",
					PrincipalEmail: "u@e.c",
					OperationFirst: true,
					OperationLast:  false,
					Request:        createReader(t, map[string]any{"description": "init"}),
				},
				{
					ProjectID:      "p",
					OperationID:    "op1",
					MethodName:     "google.cloud.networkservices.v1.NetworkServices.CreateMesh",
					ResourceName:   "projects/p/locations/global/meshes/m1",
					PrincipalEmail: "u@e.c",
					OperationFirst: false,
					OperationLast:  true,
				},
			},
			assert: func(t *testing.T, builder *khifilev6.Builder, results []*khifilev6.TimelineChangeSet) {
				wantMeshPath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "p", Type: googlecloudcommon_contract.TimelineTypeGCPProject},
					khifilev6.PathSegment{Name: "meshes", Type: googlecloudcommon_contract.TimelineTypeGCPResourceType},
					khifilev6.PathSegment{Name: "m1", Type: googlecloudcommon_contract.TimelineTypeGCPResource},
				)
				wantOpPath := builder.TimelineAccumulator.GetPath(nil,
					khifilev6.PathSegment{Name: "p", Type: googlecloudcommon_contract.TimelineTypeGCPProject},
					khifilev6.PathSegment{Name: "meshes", Type: googlecloudcommon_contract.TimelineTypeGCPResourceType},
					khifilev6.PathSegment{Name: "m1", Type: googlecloudcommon_contract.TimelineTypeGCPResource},
					khifilev6.PathSegment{Name: "CreateMesh-op1", Type: googlecloudcommon_contract.TimelineTypeOperation},
				)

				reqNode := createReader(t, map[string]any{"description": "init"}).Node

				// First log (starting)
				testchangeset.AssertTimeline(t, results[0]).
					HasRevision(wantMeshPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExisting,
						Principal:    "u@e.c",
						ResourceBody: structured.NewStandardScalarNode("description: init\n"),
						ChangedTime:  now,
					}, cmp.AllowUnexported(structured.StandardScalarNode[string]{})).
					HasRevision(wantOpPath, &khifilev6.StagingRevision{
						VerbType:     googlecloudcommon_contract.VerbOperationStart,
						StateType:    googlecloudcommon_contract.RevisionStateOperationStarted,
						Principal:    "u@e.c",
						ResourceBody: reqNode,
						ChangedTime:  now,
					}, cmp.AllowUnexported(
						structured.StandardMapNode{},
						structured.StandardScalarNode[string]{},
						structured.StandardScalarNode[any]{},
						structured.StandardSequenceNode{},
						unique.Handle[string]{},
					))

				// Second log (finished)
				testchangeset.AssertTimeline(t, results[1]).
					HasNoRevision(wantMeshPath).
					HasRevision(wantOpPath, &khifilev6.StagingRevision{
						VerbType:     googlecloudcommon_contract.VerbOperationFinish,
						StateType:    googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:    "u@e.c",
						ResourceBody: nil,
						ChangedTime:  now.Add(time.Second),
					})
			},
		},
	}

	mapper := &CSMTrafficDirectorLogToTimelineMapper{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			builder := khifilev6.NewBuilder()
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, googlecloudlogcsm_contract.ClusterIdentityTaskID.Ref(), googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "test-cluster",
			})

			tracker := googlecloudcommon_contract.NewGCPOperationTracker()
			var results []*khifilev6.TimelineChangeSet

			for i, auditFieldSet := range tc.auditFieldSets {
				logTime := now.Add(time.Duration(i) * time.Second)
				l := log.NewLogWithFieldSetsForTest(
					&log.CommonFieldSet{Timestamp: logTime},
					auditFieldSet,
				)

				cs, _, err := mapper.ProcessLogByGroup(ctx, l, tracker)
				if err != nil {
					t.Fatalf("log %d: unexpected error: %v", i, err)
				}
				results = append(results, cs)
			}

			tc.assert(t, builder, results)
		})
	}
}

func TestParseGCPResource(t *testing.T) {
	tests := []struct {
		name             string
		resourceName     string
		wantResourceType string
		wantResourceName string
	}{
		{
			name:             "standard resource path",
			resourceName:     "projects/12345678/locations/global/meshes/my-mesh",
			wantResourceType: "meshes",
			wantResourceName: "my-mesh",
		},
		{
			name:             "empty input",
			resourceName:     "",
			wantResourceType: "unknown",
			wantResourceName: "unknown",
		},
		{
			name:             "unknown input",
			resourceName:     "unknown",
			wantResourceType: "unknown",
			wantResourceName: "unknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotName := parseGCPResource(tc.resourceName)
			if gotType != tc.wantResourceType || gotName != tc.wantResourceName {
				t.Errorf("parseGCPResource(%q) = (%q, %q), want (%q, %q)",
					tc.resourceName, gotType, gotName, tc.wantResourceType, tc.wantResourceName)
			}
		})
	}
}
