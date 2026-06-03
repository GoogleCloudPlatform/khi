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

func TestPodPhaseTask_ProcessLog(t *testing.T) {
	testTime := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)

	// 1. Set up the mock Builder and construct comparison paths hierarchically.
	builder := khifilev6.NewBuilder()
	ctxForPath := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
	podPhasePath := MustPodPhaseTimelinePath(ctxForPath, "k8s", "node-1", "default", "test", "test-uid")

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

	type step struct {
		verb             *pb.Verb
		resourceBodyYAML string
		role             string
		eventType        commonlogk8saudit_contract.ChangeEventTypeV2
	}

	testCases := []struct {
		name         string
		targetPass   int
		initialState *podPhaseTaskState
		steps        []step
		wantState    *podPhaseTaskState
		assert       func(t *testing.T, cs *khifilev6.TimelineChangeSet, podPhasePath *khifilev6.TimelinePath, nodeComparer cmp.Option)
	}{
		{
			name:       "Standard Pod Lifecycle - Pass 0",
			targetPass: 0,
			steps: []step{
				{
					verb: commonlogk8saudit_contract.VerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Pending`,
					role:      "pod",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Creation,
				},
				{
					verb: commonlogk8saudit_contract.VerbPatch,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Running`,
					role:      "pod",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
				lastPhase: "",
				lastNode:  "",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, podPhasePath *khifilev6.TimelinePath, nodeComparer cmp.Option) {
				testchangeset.AssertTimeline(t, cs).
					HasNoRevision(podPhasePath)
			},
		},
		{
			name:       "Standard Pod Lifecycle - Pass 1",
			targetPass: 1,
			initialState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
			},
			steps: []step{
				{
					verb: commonlogk8saudit_contract.VerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Pending`,
					role:      "pod",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Creation,
				},
				{
					verb: commonlogk8saudit_contract.VerbPatch,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Running`,
					role:      "pod",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
				lastPhase: "Running",
				lastNode:  "node-1",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, podPhasePath *khifilev6.TimelinePath, nodeComparer cmp.Option) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(podPhasePath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						ResourceBody: parseYAML(`metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Pending`),
						Principal: "admin",
						VerbType:  commonlogk8saudit_contract.VerbCreate,
						StateType: commonlogk8saudit_contract.RevisionStatePodPhasePending,
					}, nodeComparer).
					HasRevision(podPhasePath, &khifilev6.StagingRevision{
						ChangedTime: testTime.Add(1 * time.Second),
						ResourceBody: parseYAML(`metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Running`),
						Principal: "admin",
						VerbType:  commonlogk8saudit_contract.VerbPatch,
						StateType: commonlogk8saudit_contract.RevisionStatePodPhaseRunning,
					}, nodeComparer)
			},
		},
		{
			name:       "Pod scheduled later - Pass 0",
			targetPass: 0,
			steps: []step{
				{
					verb: commonlogk8saudit_contract.VerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: ""
status:
  phase: Pending`,
					role:      "pod",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Creation,
				},
				{
					verb: commonlogk8saudit_contract.VerbPatch,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Pending`,
					role:      "pod",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
				lastPhase: "",
				lastNode:  "",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, podPhasePath *khifilev6.TimelinePath, nodeComparer cmp.Option) {
				testchangeset.AssertTimeline(t, cs).
					HasNoRevision(podPhasePath)
			},
		},
		{
			name:       "Pod scheduled later - Pass 1",
			targetPass: 1,
			initialState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
			},
			steps: []step{
				{
					verb: commonlogk8saudit_contract.VerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: ""
status:
  phase: Pending`,
					role:      "pod",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Creation,
				},
				{
					verb: commonlogk8saudit_contract.VerbPatch,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Pending`,
					role:      "pod",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Modification,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
				lastPhase: "Pending",
				lastNode:  "node-1",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, podPhasePath *khifilev6.TimelinePath, nodeComparer cmp.Option) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(podPhasePath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						ResourceBody: parseYAML(`metadata:
  uid: "test-uid"
spec:
  nodeName: ""
status:
  phase: Pending`),
						Principal: "admin",
						VerbType:  commonlogk8saudit_contract.VerbCreate,
						StateType: commonlogk8saudit_contract.RevisionStatePodPhasePending,
					}, nodeComparer)
			},
		},
		{
			name:       "Missing NodeName - Pass 0",
			targetPass: 0,
			steps: []step{
				{
					verb: commonlogk8saudit_contract.VerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
status:
  phase: Pending`,
					role:      "pod",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Creation,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{},
				lastPhase:        "",
				lastNode:         "",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, podPhasePath *khifilev6.TimelinePath, nodeComparer cmp.Option) {
				testchangeset.AssertTimeline(t, cs).
					HasNoRevision(podPhasePath)
			},
		},
		{
			name:       "Missing NodeName - Pass 1",
			targetPass: 1,
			initialState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{},
			},
			steps: []step{
				{
					verb: commonlogk8saudit_contract.VerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
status:
  phase: Pending`,
					role:      "pod",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Creation,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{},
				lastPhase:        "",
				lastNode:         "",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, podPhasePath *khifilev6.TimelinePath, nodeComparer cmp.Option) {
				testchangeset.AssertTimeline(t, cs).
					HasNoRevision(podPhasePath)
			},
		},
		{
			name:       "Pod scheduled by binding - Pass 1",
			targetPass: 1,
			initialState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
			},
			steps: []step{
				{
					verb: commonlogk8saudit_contract.VerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
status:
  phase: Pending`,
					role:      "pod",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Creation,
				},
				{
					// Binding resource creation
					verb: commonlogk8saudit_contract.VerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
status:
  phase: Pending`,
					role:      "binding",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Creation,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
				lastPhase: "Pending",
				lastNode:  "node-1",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, podPhasePath *khifilev6.TimelinePath, nodeComparer cmp.Option) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(podPhasePath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						ResourceBody: parseYAML(`metadata:
  uid: "test-uid"
status:
  phase: Pending`),
						Principal: "admin",
						VerbType:  commonlogk8saudit_contract.VerbCreate,
						StateType: commonlogk8saudit_contract.RevisionStatePodPhasePending,
					}, nodeComparer).
					HasRevision(podPhasePath, &khifilev6.StagingRevision{
						ChangedTime: testTime.Add(1 * time.Second),
						ResourceBody: parseYAML(`metadata:
  uid: "test-uid"
status:
  phase: Pending`),
						Principal: "admin",
						VerbType:  commonlogk8saudit_contract.VerbCreate,
						StateType: commonlogk8saudit_contract.RevisionStatePodPhaseScheduled,
					}, nodeComparer)
			},
		},
		{
			name:       "Pod creation log is missing but complemented from creationTime - Pass 1",
			targetPass: 1,
			initialState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
			},
			steps: []step{
				{
					verb: commonlogk8saudit_contract.VerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
  creationTimestamp: "2023-10-26T09:00:00Z"
status:
  phase: Running`,
					role:      "pod",
					eventType: commonlogk8saudit_contract.ChangeEventTypeV2Creation,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
				lastPhase: "Running",
				lastNode:  "node-1",
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, podPhasePath *khifilev6.TimelinePath, nodeComparer cmp.Option) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(podPhasePath, &khifilev6.StagingRevision{
						ChangedTime:  testTime.Add(-1 * time.Hour),
						ResourceBody: nil,
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStatePodPhaseUnknown,
					}, nodeComparer).
					HasRevision(podPhasePath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						ResourceBody: parseYAML(`metadata:
  uid: "test-uid"
  creationTimestamp: "2023-10-26T09:00:00Z"
status:
  phase: Running`),
						Principal: "admin",
						VerbType:  commonlogk8saudit_contract.VerbCreate,
						StateType: commonlogk8saudit_contract.RevisionStatePodPhaseRunning,
					}, nodeComparer)
			},
		},
	}

	taskSetting := &podPhaseLogToTimelineMapperTaskSettingV2{
		minimumDeltaTimeToCreateInferredCreationRevision: 5 * time.Second,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset the builder for each test case to clear internal state.
			builder = khifilev6.NewBuilder()
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			podPhasePath = MustPodPhaseTimelinePath(ctx, "k8s", "node-1", "default", "test", "test-uid")

			var state = tc.initialState
			var err error

			// Create a dummy initial log for NewTimelineChangeSet
			dummyFs := &commonlogk8saudit_contract.K8sAuditLogFieldSet{
				Principal:    "admin",
				APIVersion:   "core/v1",
				PluralKind:   "pods",
				ResourceName: "test",
				Namespace:    "default",
				ClusterName:  "k8s",
				Verb:         commonlogk8saudit_contract.VerbCreate,
			}
			dummyCommon := &log.CommonFieldSet{
				Timestamp: testTime,
			}
			dummyLog := log.NewLogWithFieldSetsForTest(dummyFs, dummyCommon)

			accumulatedCS := khifilev6.NewTimelineChangeSet(dummyLog)

			for i, s := range tc.steps {
				k8sFieldSet := &commonlogk8saudit_contract.K8sAuditLogFieldSet{
					Principal:    "admin",
					APIVersion:   "core/v1",
					PluralKind:   "pods",
					ResourceName: "test",
					Namespace:    "default",
					ClusterName:  "k8s",
					Verb:         s.verb,
				}
				stepTime := testTime.Add(time.Duration(i) * time.Second)
				commonFs := &log.CommonFieldSet{
					Timestamp: stepTime,
				}
				logObj := log.NewLogWithFieldSetsForTest(k8sFieldSet, commonFs)

				node := parseYAML(s.resourceBodyYAML)
				var nodeReader *structured.NodeReader
				if node != nil {
					nodeReader = structured.NewNodeReader(node)
				}

				var sourceResource *commonlogk8saudit_contract.ResourceIdentity
				var targetResource *commonlogk8saudit_contract.ResourceIdentity

				if s.role == "binding" {
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
						"pod": {
							Resource: targetResource,
							Logs: []*commonlogk8saudit_contract.ResourceManifestLog{
								{Log: logObj, ResourceBodyReader: nodeReader, ResourceBodyYAML: s.resourceBodyYAML},
							},
						},
					},
				}
				if sourceResource != nil {
					groupSet.Roles["binding"] = &commonlogk8saudit_contract.ResourceManifestLogGroup{
						Resource: sourceResource,
						Logs: []*commonlogk8saudit_contract.ResourceManifestLog{
							{Log: logObj},
						},
					}
				}

				event := commonlogk8saudit_contract.MultiGroupLogEvent{
					Log:              logObj,
					GroupRole:        s.role,
					ResourceIdentity: targetResource,
					EventType:        s.eventType,
					GroupSet:         groupSet,
				}

				if tc.targetPass == 0 {
					state, err = taskSetting.PreProcessLog(ctx, 0, event, state)
					if err != nil {
						t.Fatalf("PreProcessLog failed: %v", err)
					}
				} else {
					var stepCS *khifilev6.TimelineChangeSet
					stepCS, state, err = taskSetting.ProcessLog(ctx, event, state)
					if err != nil {
						t.Fatalf("ProcessLog failed: %v", err)
					}
					if stepCS != nil {
						for path, ev := range stepCS.Events {
							accumulatedCS.Events[path] = ev
						}
						for path, revs := range stepCS.Revisions {
							accumulatedCS.Revisions[path] = append(accumulatedCS.Revisions[path], revs...)
						}
					}
				}
			}

			if diff := cmp.Diff(tc.wantState, state, cmp.AllowUnexported(podPhaseTaskState{})); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}

			if tc.targetPass != 0 && tc.assert != nil {
				tc.assert(t, accumulatedCS, podPhasePath, nodeComparer)
			}
		})
	}
}
