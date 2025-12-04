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

func TestResourceRevisionHistoryModifierTask_Process(t *testing.T) {
	testTime := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)

	testCases := []struct {
		name                   string
		inputState             *resourceRevisionHistoryModifierState
		verb                   enum.RevisionVerb
		targetResourceBodyYAML string
		eventType              commonlogk8sauditv2_contract.ChangeEventType
		wantState              *resourceRevisionHistoryModifierState
		subResourceName        string
		asserters              []testchangeset.ChangeSetAsserter
	}{
		{
			name:       "Create event",
			inputState: nil,
			verb:       enum.RevisionVerbCreate,
			targetResourceBodyYAML: `metadata:
  uid: "test-uid"`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: false,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateExisting,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"`,
					},
				},
			},
		},
		{
			name: "Delete event without body",
			inputState: &resourceRevisionHistoryModifierState{
				PrevUID: "test-uid",
			},
			verb:                   enum.RevisionVerbDelete,
			targetResourceBodyYAML: "",
			eventType:              commonlogk8sauditv2_contract.ChangeEventTypeTargetDeletion,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: false,
				DeletionStarted:      true,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleted,
						Requestor:  "",
						ChangeTime: testTime,
						Body:       "",
					},
				},
			},
		},
		{
			name: "Delete event with graceful period > 0",
			inputState: &resourceRevisionHistoryModifierState{
				PrevUID: "test-uid",
			},
			verb: enum.RevisionVerbDelete,
			targetResourceBodyYAML: `metadata:
  uid: "test-uid"
  deletionGracePeriodSeconds: 30
  deletionTimestamp: "2023-10-26T10:00:00Z"`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetDeletion,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: false,
				DeletionStarted:      true,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleting,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"
  deletionGracePeriodSeconds: 30
  deletionTimestamp: "2023-10-26T10:00:00Z"`,
					},
				},
			},
		},
		{
			name: "Delete event with graceful period = 0",
			inputState: &resourceRevisionHistoryModifierState{
				PrevUID: "test-uid",
			},
			verb: enum.RevisionVerbDelete,
			targetResourceBodyYAML: `metadata:
  uid: "test-uid"
  deletionGracePeriodSeconds: 0
  deletionTimestamp: "2023-10-26T10:00:00Z"`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetDeletion,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: true,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleted,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"
  deletionGracePeriodSeconds: 0
  deletionTimestamp: "2023-10-26T10:00:00Z"`,
					},
				},
			},
		},
		{
			name: "Pod deletion with Failed phase",
			inputState: &resourceRevisionHistoryModifierState{
				PrevUID: "test-uid",
			},
			verb: enum.RevisionVerbDelete,
			targetResourceBodyYAML: `apiVersion: v1
kind: Pod
metadata:
  uid: "test-uid"
status:
  phase: Failed`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetDeletion,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: true,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleted,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `apiVersion: v1
kind: Pod
metadata:
  uid: "test-uid"
status:
  phase: Failed`,
					},
				},
			},
		},
		{
			name: "Recreation of resource",
			inputState: &resourceRevisionHistoryModifierState{
				PrevUID:              "old-uid",
				WasCompletelyRemoved: true,
			},
			verb: enum.RevisionVerbCreate,
			targetResourceBodyYAML: `metadata:
  uid: "new-uid"`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: false,
				DeletionStarted:      false,
				PrevUID:              "new-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateExisting,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "new-uid"`,
					},
				},
			},
		},
		{
			name: "DeleteCollection with phase=Failed",
			inputState: &resourceRevisionHistoryModifierState{
				PrevUID: "test-uid",
			},
			verb: enum.RevisionVerbDeleteCollection,
			targetResourceBodyYAML: `metadata:
  uid: "test-uid"
status:
  phase: Failed`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetDeletion,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: true,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDeleteCollection,
						State:      enum.RevisionStateDeleted,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"
status:
  phase: Failed`,
					},
				},
			},
		},
		{
			name:       "Inferred creation revision",
			inputState: nil,
			verb:       enum.RevisionVerbCreate,
			targetResourceBodyYAML: `metadata:
  uid: "test-uid"
  creationTimestamp: "2023-10-26T09:59:00Z"`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: false,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateExisting,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"
  creationTimestamp: "2023-10-26T09:59:00Z"`,
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateInferred,
						Requestor:  "N/A",
						ChangeTime: time.Date(2023, 10, 26, 9, 59, 0, 0, time.UTC),
						Body:       "# Resource creation seems to happen at the creationTime written in the later log, but the creation request wasn't found during the queried log period.",
					},
				},
			},
		},
		{
			name: "Pod deletion without explicit signal",
			inputState: &resourceRevisionHistoryModifierState{
				PrevUID: "test-uid",
			},
			verb: enum.RevisionVerbDelete,
			targetResourceBodyYAML: `metadata:
  uid: "test-uid"`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetDeletion,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: false,
				DeletionStarted:      true,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleting,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"`,
					},
				},
			},
		},
		{
			name: "Patch during deletion",
			inputState: &resourceRevisionHistoryModifierState{
				PrevUID:         "test-uid",
				DeletionStarted: true,
			},
			verb: enum.RevisionVerbPatch,
			targetResourceBodyYAML: `metadata:
  uid: "test-uid"`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: false,
				DeletionStarted:      true,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbPatch,
						State:      enum.RevisionStateDeleting,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"`,
					},
				},
			},
		},
		{
			name: "Patch after deletion",
			inputState: &resourceRevisionHistoryModifierState{
				PrevUID:              "test-uid",
				WasCompletelyRemoved: true,
			},
			verb: enum.RevisionVerbPatch,
			targetResourceBodyYAML: `metadata:
  uid: "test-uid"`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: true,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbPatch,
						State:      enum.RevisionStateDeleted,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"`,
					},
				},
			},
		},
		{
			name: "deletionGracePeriodSeconds=0 but with finalizers",
			inputState: &resourceRevisionHistoryModifierState{
				PrevUID: "test-uid",
			},
			verb: enum.RevisionVerbPatch,
			targetResourceBodyYAML: `metadata:
  uid: "test-uid"
  deletionGracePeriodSeconds: 0
  finalizers:
    - test-finalizer`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: false,
				DeletionStarted:      true,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbPatch,
						State:      enum.RevisionStateDeleting,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"
  deletionGracePeriodSeconds: 0
  finalizers:
    - test-finalizer`,
					},
				},
			},
		},
		{
			name: "Deletion with finalizers",
			inputState: &resourceRevisionHistoryModifierState{
				PrevUID: "test-uid",
			},
			verb: enum.RevisionVerbDelete,
			targetResourceBodyYAML: `metadata:
  uid: "test-uid"
  finalizers:
  - foregroundDeletion`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetDeletion,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: false,
				DeletionStarted:      true,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleting,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"
  finalizers:
  - foregroundDeletion`,
					},
				},
			},
		},
		{
			name: "DeleteCollection on already deleted resource",
			inputState: &resourceRevisionHistoryModifierState{
				PrevUID:              "test-uid",
				WasCompletelyRemoved: true,
			},
			verb: enum.RevisionVerbDeleteCollection,
			targetResourceBodyYAML: `metadata:
  uid: "test-uid"`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetDeletion,
			wantState: &resourceRevisionHistoryModifierState{
				WasCompletelyRemoved: true,
				DeletionStarted:      false,
				PrevUID:              "test-uid",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasNoRevision{
					ResourcePath: "core/v1#pod#default#test",
				},
			},
		},
		{
			name:            "SourceDeletion for subresource",
			inputState:      nil,
			verb:            enum.RevisionVerbDelete,
			subResourceName: "binding",
			targetResourceBodyYAML: `metadata:
  uid: "test-uid"`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeSourceDeletion,
			wantState: &resourceRevisionHistoryModifierState{},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#pod#default#test#binding",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleted,
						Requestor:  "",
						ChangeTime: testTime,
						Body: `metadata:
  uid: "test-uid"`,
					},
				},
			},
		},
	}

	task := &ResourceRevisionHistoryModifierTaskSetting{
		minimumDeltaTimeToCreateInferredCreationRevision: 5 * time.Second,
		kindsToWaitExactDeletionToDeterminDeletion: map[string]struct{}{
			"core/v1#pod": {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := newTestK8sAuditLogFieldSet(tc.verb, "core/v1", "pods")
			commonFs := &log.CommonFieldSet{
				Timestamp: testTime,
			}
			logObj := log.NewLogWithFieldSetsForTest(fs, commonFs)

			var reader *structured.NodeReader
			if tc.targetResourceBodyYAML != "" {
				node, err := structured.FromYAML(tc.targetResourceBodyYAML)
				if err != nil {
					t.Fatalf("failed to parse resource body: %v", err)
				}
				reader = structured.NewNodeReader(node)
			}

			var sourceResource *commonlogk8sauditv2_contract.ResourceIdentity
			var targetResource *commonlogk8sauditv2_contract.ResourceIdentity
			if tc.subResourceName != "" {
				sourceResource = &commonlogk8sauditv2_contract.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "test",
				}
				targetResource = &commonlogk8sauditv2_contract.ResourceIdentity{
					APIVersion:      "core/v1",
					Kind:            "pod",
					Namespace:       "default",
					Name:            "test",
					SubresourceName: tc.subResourceName,
				}
			} else {
				targetResource = &commonlogk8sauditv2_contract.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "test",
				}
			}

			event := commonlogk8sauditv2_contract.ResourceChangeEvent{
				Log:                   logObj,
				EventType:             tc.eventType,
				EventTargetBodyReader: reader,
				EventTargetBodyYAML:   tc.targetResourceBodyYAML,
				EventSourceResource:   sourceResource,
				EventTargetResource:   targetResource,
			}
			cs := history.NewChangeSet(logObj)
			gotState, err := task.Process(context.Background(), 0, event, cs, nil, tc.inputState)
			if err != nil {
				t.Fatalf("Process failed: %v", err)
			}

			if diff := cmp.Diff(tc.wantState, gotState); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}

			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}
