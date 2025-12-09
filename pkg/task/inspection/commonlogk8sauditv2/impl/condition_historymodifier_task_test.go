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

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestConditionWalker(t *testing.T) {
	parentPath := resourcepath.ResourcePath{
		Path:               "core/v1#pod#default#nginx",
		ParentRelationship: enum.RelationshipChild,
	}
	conditionType := "Ready"

	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	commonFieldSet := &log.CommonFieldSet{
		Timestamp: baseTime,
	}
	k8sFieldSet := &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
		K8sOperation: &model.KubernetesObjectOperation{
			Verb: enum.RevisionVerbUpdate,
		},
		Principal: "user-1",
	}

	type step struct {
		name      string
		condition *model.K8sResourceStatusCondition
		want      *history.StagingResourceRevision
	}

	scenarios := []struct {
		name  string
		steps []step
	}{
		{
			name: "Standard Lifecycle",
			steps: []step{
				{
					name: "Initial Condition (TransitionTime)",
					condition: &model.K8sResourceStatusCondition{
						Type:               conditionType,
						Status:             "True",
						LastTransitionTime: baseTime.Format(time.RFC3339),
					},
					want: &history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						Body:       "lastTransitionTime: \"2024-01-01T00:00:00Z\"\nstatus: \"True\"\ntype: Ready\n",
						Partial:    false,
						Requestor:  "user-1",
						ChangeTime: baseTime,
						State:      enum.RevisionStateConditionTrue,
					},
				},
				{
					name: "No Change",
					condition: &model.K8sResourceStatusCondition{
						Type:               conditionType,
						Status:             "True",
						LastTransitionTime: baseTime.Format(time.RFC3339),
					},
					want: nil,
				},
				{
					name: "Status Change (TransitionTime)",
					condition: &model.K8sResourceStatusCondition{
						Type:               conditionType,
						Status:             "False",
						LastTransitionTime: baseTime.Add(1 * time.Hour).Format(time.RFC3339),
					},
					want: &history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						Body:       "lastTransitionTime: \"2024-01-01T01:00:00Z\"\nstatus: \"False\"\ntype: Ready\n",
						Partial:    false,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(1 * time.Hour),
						State:      enum.RevisionStateConditionFalse,
					},
				},
				{
					name: "Probe Time Change (ProbeLikeTime)",
					condition: &model.K8sResourceStatusCondition{
						Type:               conditionType,
						Status:             "False",
						LastTransitionTime: baseTime.Add(1 * time.Hour).Format(time.RFC3339),
						LastHeartbeatTime:  baseTime.Add(2 * time.Hour).Format(time.RFC3339),
					},
					want: &history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						Body:       "lastHeartbeatTime: \"2024-01-01T02:00:00Z\"\nlastTransitionTime: \"2024-01-01T01:00:00Z\"\nstatus: \"False\"\ntype: Ready\n",
						Partial:    false,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(2 * time.Hour),
						State:      enum.RevisionStateConditionFalse,
					},
				},
				{
					name: "No change on LastTransitionTime but changes on LastHeartbeatTime",
					condition: &model.K8sResourceStatusCondition{
						Type:               conditionType,
						Status:             "False",
						LastTransitionTime: baseTime.Add(1 * time.Hour).Format(time.RFC3339),
						LastHeartbeatTime:  baseTime.Add(3 * time.Hour).Format(time.RFC3339),
					},
					want: &history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						Body:       "lastHeartbeatTime: \"2024-01-01T03:00:00Z\"\nlastTransitionTime: \"2024-01-01T01:00:00Z\"\nstatus: \"False\"\ntype: Ready\n",
						Partial:    false,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(3 * time.Hour),
						State:      enum.RevisionStateConditionFalse,
					},
				},
				{
					name:      "Condition Removal",
					condition: nil,
					want: &history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						Body:       "",
						Partial:    false,
						Requestor:  "user-1",
						ChangeTime: baseTime,
						State:      enum.RevisionStateConditionNotGiven,
					},
				},
				{
					name:      "Condition Removal (Already Removed)",
					condition: nil,
					want:      nil,
				},
			},
		},
		{
			name: "patch conditions without the full status information",
			steps: []step{
				{
					name: "initial patch without status",
					condition: &model.K8sResourceStatusCondition{
						Type:               conditionType,
						LastTransitionTime: baseTime.Add(1 * time.Hour).Format(time.RFC3339),
					},
					want: &history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						Body:       "lastTransitionTime: \"2024-01-01T01:00:00Z\"\ntype: Ready\n",
						Partial:    false,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(1 * time.Hour),
						State:      enum.RevisionStateConditionNoAvailableInfo,
					},
				},
				{
					name: "patch without status, with heartbeat",
					condition: &model.K8sResourceStatusCondition{
						Type:               conditionType,
						LastTransitionTime: baseTime.Add(1 * time.Hour).Format(time.RFC3339),
						LastHeartbeatTime:  baseTime.Add(2 * time.Hour).Format(time.RFC3339),
					},
					want: &history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						Body:       "lastHeartbeatTime: \"2024-01-01T02:00:00Z\"\nlastTransitionTime: \"2024-01-01T01:00:00Z\"\ntype: Ready\n",
						Partial:    false,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(2 * time.Hour),
						State:      enum.RevisionStateConditionNoAvailableInfo,
					},
				},
				{
					name: "patch with status added",
					condition: &model.K8sResourceStatusCondition{
						Type:               conditionType,
						LastTransitionTime: baseTime.Add(3 * time.Hour).Format(time.RFC3339),
						LastHeartbeatTime:  baseTime.Add(2 * time.Hour).Format(time.RFC3339),
						Status:             "True",
					},
					want: &history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						Body:       "lastHeartbeatTime: \"2024-01-01T02:00:00Z\"\nlastTransitionTime: \"2024-01-01T03:00:00Z\"\nstatus: \"True\"\ntype: Ready\n",
						Partial:    false,
						Requestor:  "user-1",
						ChangeTime: baseTime.Add(3 * time.Hour),
						State:      enum.RevisionStateConditionTrue,
					},
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			walker := newConditionWalker(parentPath, conditionType)
			for _, tt := range scenario.steps {
				t.Run(tt.name, func(t *testing.T) {
					l := log.NewLogWithFieldSetsForTest()
					cs := history.NewChangeSet(l)
					walker.CheckAndRecord(commonFieldSet, k8sFieldSet, tt.condition, cs)

					if tt.want == nil {
						asserter := testchangeset.HasNoRevision{
							ResourcePath: resourcepath.Condition(parentPath, conditionType).Path,
						}
						asserter.Assert(t, cs)
					} else {
						asserter := testchangeset.HasRevision{
							ResourcePath: resourcepath.Condition(parentPath, conditionType).Path,
							WantRevision: *tt.want,
						}
						asserter.Assert(t, cs)
					}
				})
			}
		})
	}
}

func TestConditionHistoryModifierTask_Process(t *testing.T) {
	task := &conditionHistoryModifierTaskSetting{
		minimumDeltaTimeToCreateInferredCreationRevision: 10 * time.Second,
	}
	ctx := context.Background()
	parentPath := resourcepath.ResourcePath{
		Path:               "core/v1#pod#default#nginx",
		ParentRelationship: enum.RelationshipChild,
	}

	testCases := []struct {
		name         string
		pass         int
		yaml         string
		eventType    commonlogk8sauditv2_contract.ChangeEventType
		operation    enum.RevisionVerb
		timestamp    time.Time
		initialState *conditionHistoryModifierTaskState
		wantState    *conditionHistoryModifierTaskState
		asserters    []testchangeset.ChangeSetAsserter
	}{
		{
			name: "processFirstPass/Collect AvailableTypes",
			pass: 0,
			yaml: `
status:
  conditions:
  - type: Ready
    status: "True"
    lastTransitionTime: "2024-01-01T00:00:00Z"
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			initialState: &conditionHistoryModifierTaskState{
				AvailableTypes:   map[string]struct{}{},
				ConditionWalkers: map[string]*conditionWalker{},
			},
			wantState: &conditionHistoryModifierTaskState{
				AvailableTypes:   map[string]struct{}{"Ready": {}},
				ConditionWalkers: map[string]*conditionWalker{},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{}, // First pass doesn't generate revisions
				},
			},
		},
		{
			name: "processFirstPass/Nil Initial State",
			pass: 0,
			yaml: `
status:
  conditions:
  - type: Ready
    status: "True"
    lastTransitionTime: "2024-01-01T00:00:00Z"
`,
			eventType:    commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation:    enum.RevisionVerbUpdate,
			timestamp:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			initialState: nil,
			wantState: &conditionHistoryModifierTaskState{
				AvailableTypes:   map[string]struct{}{"Ready": {}},
				ConditionWalkers: map[string]*conditionWalker{},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{},
				},
			},
		},
		{
			name: "processFirstPass/New Condition Type",
			pass: 0,
			yaml: `
status:
  conditions:
  - type: Ready
    status: "True"
  - type: Scheduled
    status: "True"
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			initialState: &conditionHistoryModifierTaskState{
				AvailableTypes:   map[string]struct{}{"Ready": {}},
				ConditionWalkers: map[string]*conditionWalker{},
			},
			wantState: &conditionHistoryModifierTaskState{
				AvailableTypes:   map[string]struct{}{"Ready": {}, "Scheduled": {}},
				ConditionWalkers: map[string]*conditionWalker{},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{},
				},
			},
		},
		{
			name: "processSecondPass/Standard Update",
			pass: 1,
			yaml: `
status:
  conditions:
  - type: Ready
    status: "True"
    lastTransitionTime: "2024-01-01T00:00:00Z"
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			initialState: &conditionHistoryModifierTaskState{
				AvailableTypes:   map[string]struct{}{"Ready": {}},
				ConditionWalkers: map[string]*conditionWalker{},
			},
			wantState: &conditionHistoryModifierTaskState{
				AvailableTypes: map[string]struct{}{"Ready": {}},
				ConditionWalkers: map[string]*conditionWalker{
					"Ready": {
						parentResource:     parentPath,
						conditionType:      "Ready",
						lastStatus:         "True",
						lastTransitionTime: "2024-01-01T00:00:00Z",
					},
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.Condition(parentPath, "Ready").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateConditionTrue,
						ChangeTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						Requestor:  "user-1",
						Body:       "lastTransitionTime: \"2024-01-01T00:00:00Z\"\nstatus: \"True\"\ntype: Ready\n",
					},
				},
			},
		},
		{
			name: "processSecondPass/Inferred Creation",
			pass: 1,
			yaml: `
metadata:
  creationTimestamp: "2023-12-31T23:59:00Z"
status:
  conditions: []
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetCreation,
			operation: enum.RevisionVerbCreate,
			timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			initialState: &conditionHistoryModifierTaskState{
				AvailableTypes:   map[string]struct{}{"Ready": {}},
				ConditionWalkers: map[string]*conditionWalker{},
			},
			wantState: &conditionHistoryModifierTaskState{
				AvailableTypes: map[string]struct{}{"Ready": {}},
				ConditionWalkers: map[string]*conditionWalker{
					"Ready": {
						parentResource: parentPath,
						conditionType:  "Ready",
						lastStatus:     "n/a",
						minChangeTime:  testutil.P(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
					},
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.Condition(parentPath, "Ready").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateConditionNoAvailableInfo,
						ChangeTime: time.Date(2023, 12, 31, 23, 59, 0, 0, time.UTC),
						Requestor:  "user-1",
						Body:       "# Status information is not available. The creation time is not included in the log range.",
					},
				},
			},
		},
		{
			name: "processSecondPass/Deletion",
			pass: 1,
			yaml: `
status:
  conditions: []
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetDeletion,
			operation: enum.RevisionVerbDelete,
			timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			initialState: &conditionHistoryModifierTaskState{
				AvailableTypes:   map[string]struct{}{"Ready": {}},
				ConditionWalkers: map[string]*conditionWalker{},
			},
			wantState: &conditionHistoryModifierTaskState{
				AvailableTypes: map[string]struct{}{"Ready": {}},
				ConditionWalkers: map[string]*conditionWalker{
					"Ready": {
						parentResource: parentPath,
						conditionType:  "Ready",
						lastStatus:     "", // Reset() clears this
						minChangeTime:  testutil.P(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
					},
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.Condition(parentPath, "Ready").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateConditionNotGiven,
						ChangeTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						Requestor:  "user-1",
						Body:       "",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := mustParseYAML(t, tc.yaml)
			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{},
				&commonlogk8sauditv2_contract.K8sAuditLogFieldSet{},
			)
			commonFieldSet := log.MustGetFieldSet(l, &log.CommonFieldSet{})
			commonFieldSet.Timestamp = tc.timestamp
			k8sFieldSet := log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
			k8sFieldSet.K8sOperation = &model.KubernetesObjectOperation{Verb: tc.operation}
			k8sFieldSet.Principal = "user-1"

			event := commonlogk8sauditv2_contract.ResourceChangeEvent{
				Log:                   l,
				EventType:             tc.eventType,
				EventTargetBodyReader: reader,
				EventTargetResource: &commonlogk8sauditv2_contract.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "nginx",
				},
			}

			cs := history.NewChangeSet(l)
			nextState, err := task.Process(ctx, tc.pass, event, cs, nil, tc.initialState)
			if err != nil {
				t.Fatalf("Process(%d) failed: %v", tc.pass, err)
			}

			if diff := cmp.Diff(tc.wantState, nextState, cmp.AllowUnexported(conditionWalker{})); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}

			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}
