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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	"github.com/google/go-cmp/cmp"
)

func newTestK8sAuditLogFieldSet(verb enum.RevisionVerb, apiVersion string, pluralKind string) *commonlogk8sauditv2_contract.K8sAuditLogFieldSet {
	return &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
		K8sOperation: &model.KubernetesObjectOperation{
			Verb:       verb,
			APIVersion: apiVersion,
			PluralKind: pluralKind,
			Namespace:  "default",
			Name:       "test",
		},
	}
}

func TestLifeTimeTrackerTask(t *testing.T) {
	testCases := []struct {
		desc                     string
		resourceBodyYAML         string
		inputK8sAuditLogFieldSet *commonlogk8sauditv2_contract.K8sAuditLogFieldSet
		prevState                *lifeTimeTrackerGroupState
		wantState                *lifeTimeTrackerGroupState
		wantResourceCreated      bool
		wantResourceDeleted      bool
	}{
		{
			desc:                     "create",
			inputK8sAuditLogFieldSet: newTestK8sAuditLogFieldSet(enum.RevisionVerbCreate, "core/v1", "pods"),
			prevState:                nil,
			wantState:                &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: false},
			wantResourceCreated:      true,
			wantResourceDeleted:      false,
		},
		{
			desc:                     "delete without body",
			inputK8sAuditLogFieldSet: newTestK8sAuditLogFieldSet(enum.RevisionVerbDelete, "core/v1", "pods"),
			prevState:                &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: false},
			wantState:                &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: true},
			wantResourceCreated:      false,
			wantResourceDeleted:      true,
		},
		{
			desc:                     "delete with body (graceful period > 0)",
			inputK8sAuditLogFieldSet: newTestK8sAuditLogFieldSet(enum.RevisionVerbDelete, "core/v1", "pods"),
			resourceBodyYAML: `
metadata:
  deletionGracePeriodSeconds: 30
  deletionTimestamp: "2023-10-26T10:00:00Z"
status:
  phase: Running
`,
			prevState:           &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: false},
			wantState:           &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: true},
			wantResourceCreated: false,
			wantResourceDeleted: false,
		},
		{
			desc:                     "delete with body (graceful period = 0)",
			inputK8sAuditLogFieldSet: newTestK8sAuditLogFieldSet(enum.RevisionVerbDelete, "core/v1", "pods"),
			resourceBodyYAML: `
metadata:
  deletionGracePeriodSeconds: 0
  deletionTimestamp: "2023-10-26T10:00:00Z"
status:
  phase: Succeeded
`,
			prevState:           &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: false},
			wantState:           &lifeTimeTrackerGroupState{WasCompletelyRemoved: true, DeletionStarted: false},
			wantResourceCreated: false,
			wantResourceDeleted: true,
		},
		{
			desc:                     "delete collection pod (Running)",
			inputK8sAuditLogFieldSet: newTestK8sAuditLogFieldSet(enum.RevisionVerbDeleteCollection, "core/v1", "pods"),
			resourceBodyYAML: `
status:
  phase: Running
`,
			prevState:           &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: false},
			wantState:           &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: true},
			wantResourceCreated: false,
			wantResourceDeleted: false,
		},
		{
			desc:                     "delete collection pod (Succeeded)",
			inputK8sAuditLogFieldSet: newTestK8sAuditLogFieldSet(enum.RevisionVerbDeleteCollection, "core/v1", "pods"),
			resourceBodyYAML: `
status:
  phase: Succeeded
`,
			prevState:           &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: false},
			wantState:           &lifeTimeTrackerGroupState{WasCompletelyRemoved: true, DeletionStarted: false}, // metadata.deletionGracefulSeconds wouldn't be set for non Running pods
			wantResourceCreated: false,
			wantResourceDeleted: true,
		},
		{
			desc:                     "update with finalizers (deletion started)",
			inputK8sAuditLogFieldSet: newTestK8sAuditLogFieldSet(enum.RevisionVerbUpdate, "core/v1", "pods"),
			resourceBodyYAML: `
metadata:
  deletionTimestamp: "2023-10-26T10:00:00Z"
  finalizers:
    - example.com/finalizer
`,
			prevState:           &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: true},
			wantState:           &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: true},
			wantResourceCreated: false,
			wantResourceDeleted: false,
		},
		{
			desc:                     "patch on deleted resource",
			inputK8sAuditLogFieldSet: newTestK8sAuditLogFieldSet(enum.RevisionVerbPatch, "core/v1", "pods"),
			prevState:                &lifeTimeTrackerGroupState{WasCompletelyRemoved: true, DeletionStarted: false},
			wantState:                &lifeTimeTrackerGroupState{WasCompletelyRemoved: true, DeletionStarted: false},
			wantResourceCreated:      false,
			wantResourceDeleted:      false,
		},
		{
			desc:                     "patch on deleting resource",
			inputK8sAuditLogFieldSet: newTestK8sAuditLogFieldSet(enum.RevisionVerbPatch, "core/v1", "pods"),
			prevState:                &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: true},
			wantState:                &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: true},
			wantResourceCreated:      false,
			wantResourceDeleted:      false,
		},
		{
			desc:                     "re-create after deletion",
			inputK8sAuditLogFieldSet: newTestK8sAuditLogFieldSet(enum.RevisionVerbCreate, "core/v1", "pods"),
			prevState:                &lifeTimeTrackerGroupState{WasCompletelyRemoved: true, DeletionStarted: false},
			resourceBodyYAML: `
metadata:
  uid: 12345678-1234-1234-1234-1234567890ab
`,
			wantState:           &lifeTimeTrackerGroupState{WasCompletelyRemoved: false, DeletionStarted: false, PrevUID: "12345678-1234-1234-1234-1234567890ab"},
			wantResourceCreated: true,
			wantResourceDeleted: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(tc.inputK8sAuditLogFieldSet)
			var reader *structured.NodeReader
			if tc.resourceBodyYAML != "" {
				node, err := structured.FromYAML(tc.resourceBodyYAML)
				if err != nil {
					t.Fatalf("failed to parse resource body: %v", err)
				}
				reader = structured.NewNodeReader(node)
			}
			lifeTimeTracker := &lifeTimeTrackerTaskSetting{}
			resourceLog := &commonlogk8sauditv2_contract.ResourceManifestLog{
				Log:                l,
				ResourceBodyReader: reader,
				ResourceBodyYAML:   tc.resourceBodyYAML,
				ResourceCreated:    false,
				ResourceDeleted:    false,
			}
			gotState, err := lifeTimeTracker.DetectLifetimeLogEvent(t.Context(), resourceLog, tc.prevState)
			if err != nil {
				t.Fatalf("failed to detect lifetime log event: %v", err)
			}
			if diff := cmp.Diff(gotState, tc.wantState); diff != "" {
				t.Errorf("state mismatch (-want +got): %s", diff)
			}
			if resourceLog.ResourceCreated != tc.wantResourceCreated {
				t.Errorf("resource created mismatch (-want +got): %v", resourceLog.ResourceCreated)
			}
			if resourceLog.ResourceDeleted != tc.wantResourceDeleted {
				t.Errorf("resource deleted mismatch (-want +got): %v", resourceLog.ResourceDeleted)
			}
		})
	}
}

func TestLifeTimeTrackerTask_Scenarios(t *testing.T) {
	type step struct {
		verb                enum.RevisionVerb
		resourceBodyYAML    string
		wantDeletionStarted bool
		wantResourceDeleted bool
		wantResourceCreated bool
	}

	testCases := []struct {
		name                                        string
		apiVersion                                  string
		pluralKind                                  string
		kindsToWaitExactDeletionToDetermineDeletion map[string]struct{}
		steps                                       []step
	}{
		{
			name:       "Pod deletion scenario",
			apiVersion: "core/v1",
			pluralKind: "pods",
			kindsToWaitExactDeletionToDetermineDeletion: map[string]struct{}{
				"core/v1#pod": {},
			},
			steps: []step{
				{
					verb: enum.RevisionVerbCreate,
					resourceBodyYAML: `apiVersion: v1
kind: Pod
metadata:
  uid: "test-uid"`,
					wantResourceCreated: true,
					wantDeletionStarted: false,
					wantResourceDeleted: false,
				},
				{
					verb: enum.RevisionVerbUpdate,
					resourceBodyYAML: `apiVersion: v1
kind: Pod
metadata:
  uid: "test-uid"`,
					wantDeletionStarted: false,
					wantResourceDeleted: false,
				},
				{
					// DeleteCollection is not a deletion event, but it marks the resource as being deleted for Pods because the actual deletion should be reported from the node.
					verb: enum.RevisionVerbDeleteCollection,
					resourceBodyYAML: `apiVersion: v1
kind: Pod
metadata:
  uid: "test-uid"`,
					wantDeletionStarted: true,
					wantResourceDeleted: false,
				},
				{
					verb: enum.RevisionVerbPatch,
					resourceBodyYAML: `apiVersion: v1
kind: Pod
metadata:
  uid: "test-uid"`,
					wantDeletionStarted: true,
					wantResourceDeleted: false,
				},
				{
					verb: enum.RevisionVerbDelete,
					resourceBodyYAML: `apiVersion: v1
kind: Pod
metadata:
  deletionGracePeriodSeconds: 0
  deletionTimestamp: "2023-10-26T10:00:00Z"
status:
  phase: Running
`,
					wantDeletionStarted: false,
					wantResourceDeleted: true,
				},
			},
		},
		{
			name:       "Pod deletion scenario (Failed phase)",
			apiVersion: "core/v1",
			pluralKind: "pods",
			kindsToWaitExactDeletionToDetermineDeletion: map[string]struct{}{
				"core/v1#pod": {},
			},
			steps: []step{
				{
					verb: enum.RevisionVerbCreate,
					resourceBodyYAML: `apiVersion: v1
kind: Pod
metadata:
  uid: "test-uid"
status:
  phase: Running`,
					wantResourceCreated: true,
					wantDeletionStarted: false,
					wantResourceDeleted: false,
				},
				{
					verb: enum.RevisionVerbUpdate,
					resourceBodyYAML: `apiVersion: v1
kind: Pod
metadata:
  uid: "test-uid"
status:
  phase: Failed`,
					wantDeletionStarted: false,
					wantResourceDeleted: false,
				},
				{
					verb: enum.RevisionVerbDeleteCollection,
					resourceBodyYAML: `apiVersion: v1
kind: Pod
metadata:
  uid: "test-uid"
status:
  phase: Failed`,
					wantDeletionStarted: false,
					wantResourceDeleted: true,
				},
			},
		},
		{
			name:       "EndpointSlice deletion scenario",
			apiVersion: "discovery.k8s.io/v1",
			pluralKind: "endpointslices",
			kindsToWaitExactDeletionToDetermineDeletion: map[string]struct{}{
				"core/v1#pod": {},
			},
			steps: []step{
				{
					verb: enum.RevisionVerbCreate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"`,
					wantResourceCreated: true,
					wantDeletionStarted: false,
					wantResourceDeleted: false,
				},
				{
					verb: enum.RevisionVerbUpdate,
					resourceBodyYAML: `metadata:
  uid: "test-uid"`,
					wantDeletionStarted: false,
					wantResourceDeleted: false,
				},
				{
					verb: enum.RevisionVerbDeleteCollection,
					resourceBodyYAML: `metadata:
  uid: "test-uid"`,
					wantDeletionStarted: true,
					wantResourceDeleted: true,
				},
				{
					verb: enum.RevisionVerbDeleteCollection,
					resourceBodyYAML: `metadata:
  uid: "test-uid"`,
					wantDeletionStarted: true,
					wantResourceDeleted: true,
				},
			},
		},
		{
			name:       "patch and update later",
			apiVersion: "core/v1",
			pluralKind: "nodes",
			kindsToWaitExactDeletionToDetermineDeletion: map[string]struct{}{},
			steps: []step{
				{
					verb: enum.RevisionVerbPatch,
					resourceBodyYAML: `metadata:
  uid: "test-uid"`,
					wantResourceCreated: true,
					wantDeletionStarted: false,
					wantResourceDeleted: false,
				},
				{
					verb: enum.RevisionVerbPatch,
					resourceBodyYAML: `metadata:
  uid: "test-uid"`,
					wantDeletionStarted: false,
					wantResourceDeleted: false,
				},
				{
					verb: enum.RevisionVerbUpdate,
					resourceBodyYAML: `apiVersion: v1
kind: Node
metadata:
  uid: "test-uid"
  creationTimestamp: "2025-12-01T10:00:00Z"`,
					wantDeletionStarted: false,
					wantResourceDeleted: false,
				},
			},
		},
		{
			name:       "patch deletionGracePeriodSeconds=0 but with finalizer",
			apiVersion: "core/v1",
			pluralKind: "nodes",
			kindsToWaitExactDeletionToDetermineDeletion: map[string]struct{}{},
			steps: []step{
				{
					verb: enum.RevisionVerbPatch,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
  finalizers:
    - foo`,
					wantResourceCreated: true,
					wantDeletionStarted: false,
					wantResourceDeleted: false,
				},
				{
					verb: enum.RevisionVerbDeleteCollection,
					resourceBodyYAML: `metadata:
  uid: "test-uid"
  deletionGracePeriodSeconds: 0
  finalizers:
    - foo`,
					wantDeletionStarted: true,
					wantResourceDeleted: false,
				},
				{
					verb: enum.RevisionVerbPatch,
					resourceBodyYAML: `apiVersion: v1
kind: Node
metadata:
  uid: "test-uid"
  deletionGracePeriodSeconds: 0
  finalizers: []`,
					wantDeletionStarted: false,
					wantResourceDeleted: true,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tracker := &lifeTimeTrackerTaskSetting{
				kindsToWaitExactDeletionToDetermineDeletion: tc.kindsToWaitExactDeletionToDetermineDeletion,
			}
			var state *lifeTimeTrackerGroupState
			for i, s := range tc.steps {
				fs := newTestK8sAuditLogFieldSet(s.verb, tc.apiVersion, tc.pluralKind)
				logObj := log.NewLogWithFieldSetsForTest(fs)
				var reader *structured.NodeReader
				if s.resourceBodyYAML != "" {
					node, err := structured.FromYAML(s.resourceBodyYAML)
					if err != nil {
						t.Fatalf("step %d: failed to parse resource body: %v", i, err)
					}
					reader = structured.NewNodeReader(node)
				}
				resourceLog := &commonlogk8sauditv2_contract.ResourceManifestLog{
					Log:                logObj,
					ResourceBodyReader: reader,
					ResourceBodyYAML:   s.resourceBodyYAML,
				}
				var err error
				state, err = tracker.DetectLifetimeLogEvent(t.Context(), resourceLog, state)
				if err != nil {
					t.Fatalf("step %d: failed to detect lifetime log event: %v", i, err)
				}

				if state.DeletionStarted != s.wantDeletionStarted {
					t.Errorf("step %d: expected DeletionStarted=%v, got %v", i, s.wantDeletionStarted, state.DeletionStarted)
				}
				if resourceLog.ResourceDeleted != s.wantResourceDeleted {
					t.Errorf("step %d: expected ResourceDeleted=%v, got %v", i, s.wantResourceDeleted, resourceLog.ResourceDeleted)
				}
				if resourceLog.ResourceCreated != s.wantResourceCreated {
					t.Errorf("step %d: expected ResourceCreated=%v, got %v", i, s.wantResourceCreated, resourceLog.ResourceCreated)
				}
			}
		})
	}
}
