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
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestResourceRevisionLogToTimelineMapperTaskSettingV2_ProcessLog(t *testing.T) {
	testTime := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)

	// 1. Set up the mock Builder and construct comparison paths hierarchically.
	builder := khifilev6.NewBuilder()
	cluster := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})
	api := builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
	kind := builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
	ns := builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})

	parentPath := builder.TimelineAccumulator.GetPath(ns, khifilev6.PathSegment{Name: "test", Type: inspectioncore_contract.TimelineTypeResource})
	subresourcePath := builder.TimelineAccumulator.GetPath(parentPath, khifilev6.PathSegment{Name: "binding", Type: inspectioncore_contract.TimelineTypeSubresource})

	// Comparer for structured.Node using semantical YAML serializations to bypass unexported fields.
	nodeComparer := cmp.Comparer(func(a, b structured.Node) bool {
		if a == nil || b == nil {
			return a == b
		}
		aYAML, errA := structured.NewNodeReader(a).Serialize("", &structured.YAMLNodeSerializer{})
		bYAML, errB := structured.NewNodeReader(b).Serialize("", &structured.YAMLNodeSerializer{})
		if errA != nil || errB != nil {
			return false
		}
		return string(aYAML) == string(bYAML)
	})

	// Helper to parse YAML into structured.Node.
	parseYAML := func(yamlStr string) structured.Node {
		if yamlStr == "" {
			return nil
		}
		node, err := structured.FromYAML(yamlStr)
		if err != nil {
			t.Fatalf("failed to parse YAML: %v", err)
		}
		return node
	}

	testCases := []struct {
		name       string
		inputState *resourceRevisionLogToTimelineMapperStateV2
		verb       *pb.Verb
		bodyYAML   string
		role       string
		eventType  commonlogk8saudit_contract.ChangeEventTypeV2
		wantState  *resourceRevisionLogToTimelineMapperStateV2
		assert     func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node)
	}{
		{
			name:       "Create event",
			inputState: nil,
			verb:       commonlogk8saudit_contract.VerbCreate,
			bodyYAML: `metadata:
  uid: "test-uid"`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: false,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExisting,
					}, nodeComparer)
			},
		},
		{
			name: "Delete event without body",
			inputState: &resourceRevisionLogToTimelineMapperStateV2{
				PrevUID: "test-uid",
			},
			verb:      commonlogk8saudit_contract.VerbDelete,
			bodyYAML:  "",
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: false,
				DeletionStarted:      true,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbDelete,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted,
					}, nodeComparer)
			},
		},
		{
			name: "Delete event with graceful period > 0",
			inputState: &resourceRevisionLogToTimelineMapperStateV2{
				PrevUID: "test-uid",
			},
			verb: commonlogk8saudit_contract.VerbDelete,
			bodyYAML: `metadata:
  uid: "test-uid"
  deletionGracePeriodSeconds: 30
  deletionTimestamp: "2023-10-26T10:00:00Z"`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: false,
				DeletionStarted:      true,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbDelete,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceDeleting,
					}, nodeComparer)
			},
		},
		{
			name: "Delete event with graceful period = 0",
			inputState: &resourceRevisionLogToTimelineMapperStateV2{
				PrevUID: "test-uid",
			},
			verb: commonlogk8saudit_contract.VerbDelete,
			bodyYAML: `metadata:
  uid: "test-uid"
  deletionGracePeriodSeconds: 0
  deletionTimestamp: "2023-10-26T10:00:00Z"`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: true,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbDelete,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted,
					}, nodeComparer)
			},
		},
		{
			name: "Pod deletion with Failed phase",
			inputState: &resourceRevisionLogToTimelineMapperStateV2{
				PrevUID: "test-uid",
			},
			verb: commonlogk8saudit_contract.VerbDelete,
			bodyYAML: `apiVersion: v1
kind: Pod
metadata:
  uid: "test-uid"
status:
  phase: Failed`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: true,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbDelete,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted,
					}, nodeComparer)
			},
		},
		{
			name: "Recreation of resource",
			inputState: &resourceRevisionLogToTimelineMapperStateV2{
				PrevUID:              "old-uid",
				WasCompletelyRemoved: true,
			},
			verb: commonlogk8saudit_contract.VerbCreate,
			bodyYAML: `metadata:
  uid: "new-uid"`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Creation,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: false,
				DeletionStarted:      false,
				PrevUID:              "new-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExisting,
					}, nodeComparer)
			},
		},
		{
			name: "DeleteCollection with phase=Failed",
			inputState: &resourceRevisionLogToTimelineMapperStateV2{
				PrevUID: "test-uid",
			},
			verb: commonlogk8saudit_contract.VerbDeleteCollection,
			bodyYAML: `metadata:
  uid: "test-uid"
status:
  phase: Failed`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: true,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbDeleteCollection,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted,
					}, nodeComparer)
			},
		},
		{
			name:       "Inferred creation revision",
			inputState: nil,
			verb:       commonlogk8saudit_contract.VerbCreate,
			bodyYAML: `metadata:
  uid: "test-uid"
  creationTimestamp: "2023-10-26T09:59:00Z"`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Creation,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: false,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExisting,
					}, nodeComparer).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Date(2023, 10, 26, 9, 59, 0, 0, time.UTC),
						ResourceBody: nil,
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExisting,
					}, nodeComparer)
			},
		},
		{
			name: "Pod deletion without explicit signal",
			inputState: &resourceRevisionLogToTimelineMapperStateV2{
				PrevUID: "test-uid",
			},
			verb: commonlogk8saudit_contract.VerbDelete,
			bodyYAML: `metadata:
  uid: "test-uid"`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: false,
				DeletionStarted:      true,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbDelete,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceDeleting,
					}, nodeComparer)
			},
		},
		{
			name: "Patch during deletion",
			inputState: &resourceRevisionLogToTimelineMapperStateV2{
				PrevUID:         "test-uid",
				DeletionStarted: true,
			},
			verb: commonlogk8saudit_contract.VerbPatch,
			bodyYAML: `metadata:
  uid: "test-uid"`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: false,
				DeletionStarted:      true,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbPatch,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceDeleting,
					}, nodeComparer)
			},
		},
		{
			name: "Patch after deletion",
			inputState: &resourceRevisionLogToTimelineMapperStateV2{
				PrevUID:              "test-uid",
				WasCompletelyRemoved: true,
			},
			verb: commonlogk8saudit_contract.VerbPatch,
			bodyYAML: `metadata:
  uid: "test-uid"`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: true,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbPatch,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted,
					}, nodeComparer)
			},
		},
		{
			name: "deletionGracePeriodSeconds=0 but with finalizers",
			inputState: &resourceRevisionLogToTimelineMapperStateV2{
				PrevUID: "test-uid",
			},
			verb: commonlogk8saudit_contract.VerbPatch,
			bodyYAML: `metadata:
  uid: "test-uid"
  deletionGracePeriodSeconds: 0
  finalizers:
    - test-finalizer`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: false,
				DeletionStarted:      true,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbPatch,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceDeleting,
					}, nodeComparer)
			},
		},
		{
			name: "Deletion with finalizers",
			inputState: &resourceRevisionLogToTimelineMapperStateV2{
				PrevUID: "test-uid",
			},
			verb: commonlogk8saudit_contract.VerbDelete,
			bodyYAML: `metadata:
  uid: "test-uid"
  finalizers:
  - foregroundDeletion`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: false,
				DeletionStarted:      true,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(parentPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbDelete,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceDeleting,
					}, nodeComparer)
			},
		},
		{
			name: "DeleteCollection on already deleted resource",
			inputState: &resourceRevisionLogToTimelineMapperStateV2{
				PrevUID:              "test-uid",
				WasCompletelyRemoved: true,
			},
			verb: commonlogk8saudit_contract.VerbDeleteCollection,
			bodyYAML: `metadata: 
uid: "test-uid"`,
			role:      "target",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{
				WasCompletelyRemoved: true,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasNoRevision(parentPath)
			},
		},
		{
			name:       "SourceDeletion for subresource",
			inputState: nil,
			verb:       commonlogk8saudit_contract.VerbDelete,
			bodyYAML: `metadata:
  uid: "test-uid"`,
			role:      "source",
			eventType: commonlogk8saudit_contract.ChangeEventTypeV2Deletion,
			wantState: &resourceRevisionLogToTimelineMapperStateV2{},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, node structured.Node) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(subresourcePath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: node,
						Principal:    "admin",
						VerbType:     commonlogk8saudit_contract.VerbDelete,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted,
					}, nodeComparer)
			},
		},
	}

	mapperSetting := &ResourceRevisionLogToTimelineMapperTaskSettingV2{
		minimumDeltaTimeToCreateInferredCreationRevision: 5 * time.Second,
		kindsToWaitExactDeletionToDeterminDeletion: map[string]struct{}{
			"core/v1#pod": {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear comparison path builder states.
			builder = khifilev6.NewBuilder()
			cluster = builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})
			api = builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
			kind = builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
			ns = builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
			parentPath = builder.TimelineAccumulator.GetPath(ns, khifilev6.PathSegment{Name: "test", Type: inspectioncore_contract.TimelineTypeResource})
			subresourcePath = builder.TimelineAccumulator.GetPath(parentPath, khifilev6.PathSegment{Name: "binding", Type: inspectioncore_contract.TimelineTypeSubresource})

			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

			// Setup the Log and Mock Group Context dynamically for each test case.
			k8sFieldSet := &commonlogk8saudit_contract.K8sAuditLogFieldSet{
				Principal:    "admin",
				APIVersion:   "core/v1",
				PluralKind:   "pods",
				ResourceName: "test",
				Namespace:    "default",
				ClusterName:  "k8s",
				Verb:         tc.verb,
			}
			commonFs := &log.CommonFieldSet{
				Timestamp: testTime,
			}
			logObj := log.NewLogWithFieldSetsForTest(k8sFieldSet, commonFs)

			node := parseYAML(tc.bodyYAML)
			var nodeReader *structured.NodeReader
			if node != nil {
				nodeReader = structured.NewNodeReader(node)
			}

			// Setup the mock event GroupSet context.
			var sourceResource *commonlogk8saudit_contract.ResourceIdentity
			var targetResource *commonlogk8saudit_contract.ResourceIdentity

			if tc.role == "source" {
				sourceResource = &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "test",
				}
				targetResource = &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion:      "core/v1",
					Kind:            "pod",
					Namespace:       "default",
					Name:            "test",
					SubresourceName: "binding",
				}
			} else {
				targetResource = &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "test",
				}
			}

			groupSet := commonlogk8saudit_contract.RelatedGroupSet{
				Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
					"target": {
						Resource: targetResource,
						Logs: []*commonlogk8saudit_contract.ResourceManifestLog{
							{Log: logObj, ResourceBodyReader: nodeReader, ResourceBodyYAML: tc.bodyYAML},
						},
					},
				},
			}
			if sourceResource != nil {
				groupSet.Roles["source"] = &commonlogk8saudit_contract.ResourceManifestLogGroup{
					Resource: sourceResource,
					Logs: []*commonlogk8saudit_contract.ResourceManifestLog{
						{Log: logObj},
					},
				}
			}

			event := commonlogk8saudit_contract.MultiGroupLogEvent{
				Log:              logObj,
				GroupRole:        tc.role,
				ResourceIdentity: targetResource,
				EventType:        tc.eventType,
				GroupSet:         groupSet,
			}

			cs, nextState, err := mapperSetting.ProcessLog(ctx, event, tc.inputState)
			if err != nil {
				t.Fatalf("ProcessLog() failed: %v", err)
			}

			if diff := cmp.Diff(tc.wantState, nextState); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}

			tc.assert(t, cs, node)
		})
	}
}
