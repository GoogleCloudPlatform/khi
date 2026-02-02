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
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestEndpointLogToTimelineMapperTask_Process(t *testing.T) {
	task := &endpointResourceLogToTimelineMapperTaskSetting{}
	ctx := context.Background()
	timestamp := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name         string
		pass         int
		yaml         string
		eventType    commonlogk8sauditv2_contract.ChangeEventType
		operation    enum.RevisionVerb
		initialState *endpointResourceLogToTimelineMapperState
		wantState    *endpointResourceLogToTimelineMapperState
		asserters    []testchangeset.ChangeSetAsserter
	}{
		{
			name: "Pass 0: Collect Service Name",
			pass: 0,
			yaml: `
metadata:
  ownerReferences:
  - kind: Service
    name: my-service
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]enum.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]enum.RevisionState{},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{WantResourcePaths: []string{}},
			},
		},
		{
			name: "Pass 0: Collect Pod Identity",
			pass: 0,
			yaml: `
endpoints:
- targetRef:
    kind: Pod
    name: my-pod
    namespace: default
    uid: pod-uid-1
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]enum.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {
						uid:       "pod-uid-1",
						name:      "my-pod",
						namespace: "default",
					},
				},
				lastStates: map[string]enum.RevisionState{},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{WantResourcePaths: []string{}},
			},
		},
		{
			name: "Pass 1: Standard Update (Ready)",
			pass: 1,
			yaml: `
endpoints:
- conditions:
    ready: true
  targetRef:
    kind: Pod
    name: my-pod
    namespace: default
    uid: pod-uid-1
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]enum.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]enum.RevisionState{
					"pod-uid-1": enum.RevisionStateEndpointReady,
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.PodEndpointSlice("default", "my-endpoint", "default", "my-pod").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateEndpointReady,
						ChangeTime: timestamp,
						Requestor:  "user-1",
						Body:       "conditions:\n  ready: true\ntargetRef:\n  kind: Pod\n  name: my-pod\n  namespace: default\n  uid: pod-uid-1\n",
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.EndpointSliceChildPod("default", "my-endpoint", "default", "my-pod").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateEndpointReady,
						ChangeTime: timestamp,
						Requestor:  "user-1",
						Body:       "conditions:\n  ready: true\ntargetRef:\n  kind: Pod\n  name: my-pod\n  namespace: default\n  uid: pod-uid-1\n",
					},
				},
			},
		},
		{
			name: "Pass 1: Standard Update (Terminating)",
			pass: 1,
			yaml: `
endpoints:
- conditions:
    terminating: true
  targetRef:
    kind: Pod
    name: my-pod
    namespace: default
    uid: pod-uid-1
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]enum.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]enum.RevisionState{
					"pod-uid-1": enum.RevisionStateEndpointTerminating,
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.PodEndpointSlice("default", "my-endpoint", "default", "my-pod").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateEndpointTerminating,
						ChangeTime: timestamp,
						Requestor:  "user-1",
						Body:       "conditions:\n  terminating: true\ntargetRef:\n  kind: Pod\n  name: my-pod\n  namespace: default\n  uid: pod-uid-1\n",
					},
				},
			},
		},
		{
			name: "Pass 1: Standard Update (Unready)",
			pass: 1,
			yaml: `
endpoints:
- conditions:
    ready: false
  targetRef:
    kind: Pod
    name: my-pod
    namespace: default
    uid: pod-uid-1
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]enum.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]enum.RevisionState{
					"pod-uid-1": enum.RevisionStateEndpointUnready,
				},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.PodEndpointSlice("default", "my-endpoint", "default", "my-pod").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateEndpointUnready,
						ChangeTime: timestamp,
						Requestor:  "user-1",
						Body:       "conditions:\n  ready: false\ntargetRef:\n  kind: Pod\n  name: my-pod\n  namespace: default\n  uid: pod-uid-1\n",
					},
				},
			},
		},
		{
			name: "Pass 1: Service State (Ready)",
			pass: 1,
			yaml: `
endpoints:
- conditions:
    ready: true
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]enum.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]enum.RevisionState{},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.ServiceEndpointSlice("default", "my-endpoint", "my-service").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateEndpointReady,
						ChangeTime: timestamp,
						Requestor:  "user-1",
						Body:       "\nendpoints:\n- conditions:\n    ready: true\n",
					},
				},
			},
		},
		{
			name: "Pass 1: Service State (Terminating)",
			pass: 1,
			yaml: `
endpoints:
- conditions:
    terminating: true
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]enum.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]enum.RevisionState{},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.ServiceEndpointSlice("default", "my-endpoint", "my-service").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateEndpointTerminating,
						ChangeTime: timestamp,
						Requestor:  "user-1",
						Body:       "\nendpoints:\n- conditions:\n    terminating: true\n",
					},
				},
			},
		},
		{
			name: "Pass 1: Service State (Unready)",
			pass: 1,
			yaml: `
endpoints:
- conditions:
    ready: false
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]enum.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]enum.RevisionState{},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.ServiceEndpointSlice("default", "my-endpoint", "my-service").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateEndpointUnready,
						ChangeTime: timestamp,
						Requestor:  "user-1",
						Body:       "\nendpoints:\n- conditions:\n    ready: false\n",
					},
				},
			},
		},
		{
			name: "Pass 1: Endpoint Removal (Implicit)",
			pass: 1,
			yaml: `
endpoints: []
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]enum.RevisionState{
					"pod-uid-1": enum.RevisionStateEndpointReady,
				},
			},
			wantState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]enum.RevisionState{}, // Removed
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.PodEndpointSlice("default", "my-endpoint", "default", "my-pod").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbUpdate,
						State:      enum.RevisionStateDeleted,
						ChangeTime: timestamp,
						Requestor:  "user-1",
						Body:       "",
					},
				},
			},
		},
		{
			name: "Pass 1: Target Deletion",
			pass: 1,
			yaml: `
metadata:
  name: my-endpoint
`,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetDeletion,
			operation: enum.RevisionVerbDelete,
			initialState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]enum.RevisionState{
					"pod-uid-1": enum.RevisionStateEndpointReady,
				},
			},
			wantState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{"my-service": {}},
				foundPods: map[string]*podIdentity{
					"pod-uid-1": {uid: "pod-uid-1", name: "my-pod", namespace: "default"},
				},
				lastStates: map[string]enum.RevisionState{}, // Removed
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.PodEndpointSlice("default", "my-endpoint", "default", "my-pod").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleted,
						ChangeTime: timestamp,
						Requestor:  "user-1",
						Body:       "",
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: resourcepath.ServiceEndpointSlice("default", "my-endpoint", "my-service").Path,
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleted,
						ChangeTime: timestamp,
						Requestor:  "user-1",
						Body:       "",
					},
				},
			},
		},
		{
			name:      "Pass 0: No EndpointSlice body",
			pass:      0,
			eventType: commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
			operation: enum.RevisionVerbUpdate,
			initialState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]enum.RevisionState{},
			},
			wantState: &endpointResourceLogToTimelineMapperState{
				serviceNames: map[string]struct{}{},
				foundPods:    map[string]*podIdentity{},
				lastStates:   map[string]enum.RevisionState{},
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{WantResourcePaths: []string{}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var reader *structured.NodeReader
			if tc.yaml != "" {
				reader = mustParseYAML(t, tc.yaml)
			}
			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{},
				&commonlogk8sauditv2_contract.K8sAuditLogFieldSet{},
			)
			commonFieldSet := log.MustGetFieldSet(l, &log.CommonFieldSet{})
			commonFieldSet.Timestamp = timestamp
			k8sFieldSet := log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
			k8sFieldSet.K8sOperation = &model.KubernetesObjectOperation{Verb: tc.operation}
			k8sFieldSet.Principal = "user-1"

			event := commonlogk8sauditv2_contract.ResourceChangeEvent{
				Log:                   l,
				EventType:             tc.eventType,
				EventTargetBodyReader: reader,
				EventTargetResource: &commonlogk8sauditv2_contract.ResourceIdentity{
					APIVersion: "discovery.k8s.io/v1",
					Kind:       "endpointslice",
					Namespace:  "default",
					Name:       "my-endpoint",
				},
				EventTargetBodyYAML: tc.yaml,
			}

			cs := history.NewChangeSet(l)
			nextState, err := task.Process(ctx, tc.pass, event, cs, nil, tc.initialState)
			if err != nil {
				t.Fatalf("Process(%d) failed: %v", tc.pass, err)
			}

			if diff := cmp.Diff(tc.wantState, nextState, cmp.AllowUnexported(endpointResourceLogToTimelineMapperState{}, podIdentity{})); diff != "" {
				t.Errorf("state mismatch (-want +got):\n%s", diff)
			}

			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}
