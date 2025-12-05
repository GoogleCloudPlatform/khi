// Copyright 2025 Google LLC
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

package commonlogk8sauditv2_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestPodPhaseTask_Process(t *testing.T) {
	testTime := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)

	type step struct {
		verb             enum.RevisionVerb
		resourceBodyYAML string
		eventType        commonlogk8sauditv2_contract.ChangeEventType
	}

	testCases := []struct {
		name         string
		targetPass   int
		initialState *podPhaseTaskState
		steps        []step
		wantState    *podPhaseTaskState
		asserters    []testchangeset.ChangeSetAsserter
	}{
		{
			name:       "Standard Pod Lifecycle - Pass 0",
			targetPass: 0,
			steps: []step{
				{
					verb: enum.RevisionVerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Pending`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation,
				},
				{
					verb: enum.RevisionVerbPatch,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Running`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
				lastPhase: "",
				lastNode:  "",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasNoRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#default/test[test-uid]",
				},
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
					verb: enum.RevisionVerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Pending`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation,
				},
				{
					verb: enum.RevisionVerbPatch,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Running`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
				lastPhase: "Running",
				lastNode:  "node-1",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#default/test[test-uid]",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStatePodPhasePending,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Pending`,
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#default/test[test-uid]",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbPatch,
						State:      enum.RevisionStatePodPhaseRunning,
						Requestor:  "",
						ChangeTime: testTime.Add(1 * time.Second),
						Body: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Running`,
					},
				},
			},
		},
		{
			name:       "Pod scheduled later - Pass 0",
			targetPass: 0,
			steps: []step{
				{
					verb: enum.RevisionVerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: ""
status:
  phase: Pending`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation,
				},
				{
					verb: enum.RevisionVerbPatch,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Pending`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
				lastPhase: "",
				lastNode:  "",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasNoRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#default/test[test-uid]",
				},
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
					verb: enum.RevisionVerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: ""
status:
  phase: Pending`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation,
				},
				{
					verb: enum.RevisionVerbPatch,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
spec:
  nodeName: "node-1"
status:
  phase: Pending`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
				lastPhase: "Pending",
				lastNode:  "node-1",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#default/test[test-uid]",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStatePodPhasePending,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"
spec:
  nodeName: ""
status:
  phase: Pending`,
					},
				},
			},
		},
		{
			name:       "Missing NodeName - Pass 0",
			targetPass: 0,
			steps: []step{
				{
					verb: enum.RevisionVerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
status:
  phase: Pending`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{},
				lastPhase:        "",
				lastNode:         "",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasNoRevision{
					ResourcePath: "core/v1#pod#default#test#phase",
				},
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
					verb: enum.RevisionVerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
status:
  phase: Pending`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{},
				lastPhase:        "",
				lastNode:         "",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasNoRevision{
					ResourcePath: "core/v1#pod#default#test#phase",
				},
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
					verb: enum.RevisionVerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
status:
  phase: Pending`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation,
				},
				{
					// Binding resource creation
					verb: enum.RevisionVerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
status:
  phase: Pending`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeSourceCreation,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
				lastPhase: "Pending",
				lastNode:  "node-1",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#default/test[test-uid]",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStatePodPhasePending,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"
status:
  phase: Pending`,
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#default/test[test-uid]",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStatePodPhaseScheduled,
						Requestor:  "",
						ChangeTime: testTime.Add(1 * time.Second),
						Body: `metadata:
  uid: "test-uid"
status:
  phase: Pending`,
					},
				},
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
					verb: enum.RevisionVerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
  creationTimestamp: "2023-10-26T09:00:00Z"
status:
  phase: Running`,
					eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation,
				},
			},
			wantState: &podPhaseTaskState{
				uidToNodeNameMap: map[string]string{
					"test-uid": "node-1",
				},
				lastPhase: "Running",
				lastNode:  "node-1",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#default/test[test-uid]",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUnknown,
						State:      enum.RevisionStatePodPhaseUnknown,
						Requestor:  "N/A",
						ChangeTime: testTime.Add(-1 * time.Hour),
						Body:       `# Pod exists during this period but no body information available`,
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#default/test[test-uid]",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStatePodPhaseRunning,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"
  creationTimestamp: "2023-10-26T09:00:00Z"
status:
  phase: Running`,
					},
				},
			},
		},
	}

	taskSetting := &podPhaseTaskSetting{
		minimumDeltaTimeToCreateInferredCreationRevision: 5 * time.Second,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			commonFs := &log.CommonFieldSet{
				Timestamp: testTime,
			}
			// Initial logObj for ChangeSet creation (not used for events)
			fs := newTestK8sAuditLogFieldSet(enum.RevisionVerbCreate, "core/v1", "pods")
			logObj := log.NewLogWithFieldSetsForTest(fs, commonFs)

			var state *podPhaseTaskState = tc.initialState
			var err error

			cs := history.NewChangeSet(logObj)
			for i, s := range tc.steps {
				fs := newTestK8sAuditLogFieldSet(s.verb, "core/v1", "pods")
				stepTime := testTime.Add(time.Duration(i) * time.Second)
				commonFs := &log.CommonFieldSet{
					Timestamp: stepTime,
				}
				logObj := log.NewLogWithFieldSetsForTest(fs, commonFs)
				var reader *structured.NodeReader
				if s.resourceBodyYAML != "" {
					node, err := structured.FromYAML(s.resourceBodyYAML)
					if err != nil {
						t.Fatalf("failed to parse resource body: %v", err)
					}
					reader = structured.NewNodeReader(node)
				}

				event := commonlogk8sauditv2_contract.ResourceChangeEvent{
					Log:                   logObj,
					EventType:             s.eventType,
					EventTargetBodyReader: reader,
					EventTargetBodyYAML:   s.resourceBodyYAML,
					EventTargetResource: &commonlogk8sauditv2_contract.ResourceIdentity{
						APIVersion: "core/v1",
						Kind:       "pod",
						Namespace:  "default",
						Name:       "test",
					},
				}
				state, err = taskSetting.Process(context.Background(), tc.targetPass, event, cs, nil, state)
				if err != nil {
					t.Fatalf("Process pass %d failed: %v", tc.targetPass, err)
				}
			}

			if diff := cmp.Diff(tc.wantState, state, cmp.AllowUnexported(podPhaseTaskState{})); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}

			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}
