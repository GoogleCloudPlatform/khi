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
)

func TestResourceOwnerReferenceModifierTask_Process(t *testing.T) {
	task := &resourceOwnerReferenceModifierTaskSetting{
		nonNamespacedOwnerTypes: map[string]struct{}{
			"core/v1#node": {},
		},
	}
	ctx := context.Background()
	podNamespace := "default"
	podName := "nginx"
	podPath := resourcepath.Pod(podNamespace, podName)

	tests := []struct {
		name      string
		yaml      string
		asserters []testchangeset.ChangeSetAsserter
	}{
		{
			name: "No Owner References",
			yaml: `
metadata:
  name: nginx
  namespace: default
`,
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchRevisionCount{
					ResourcePath: podPath.Path,
					WantCount:    0,
				},
			},
		},
		{
			name: "Single Owner (Namespaced)",
			yaml: `
metadata:
  name: nginx
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    kind: ReplicaSet
    name: nginx-replicaset
    uid: 12345
`,
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasAlias{
					Source:      podPath.Path,
					Destination: resourcepath.OwnerSubresource(resourcepath.NameLayerGeneralItem("apps/v1", "replicaset", podNamespace, "nginx-replicaset"), "nginx", "pod").Path,
				},
			},
		},
		{
			name: "Single Owner (Cluster Scoped)",
			yaml: `
metadata:
  name: nginx
  namespace: default
  ownerReferences:
  - apiVersion: v1
    kind: Node
    name: node-1
    uid: 67890
`,
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasAlias{
					Source:      podPath.Path,
					Destination: resourcepath.OwnerSubresource(resourcepath.Node("node-1"), "nginx", "pod").Path,
				},
			},
		},
		{
			name: "Multiple Owners",
			yaml: `
metadata:
  name: nginx
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    kind: ReplicaSet
    name: nginx-replicaset
  - apiVersion: v1
    kind: Node
    name: node-1
`,
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasAlias{
					Source:      podPath.Path,
					Destination: resourcepath.OwnerSubresource(resourcepath.NameLayerGeneralItem("apps/v1", "replicaset", podNamespace, "nginx-replicaset"), "nginx", "pod").Path,
				},
				&testchangeset.HasAlias{
					Source:      podPath.Path,
					Destination: resourcepath.OwnerSubresource(resourcepath.Node("node-1"), "nginx", "pod").Path,
				},
			},
		},
		{
			name: "Missing Fields",
			yaml: `
metadata:
  name: nginx
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    # Missing kind
    name: nginx-replicaset
  - apiVersion: apps/v1
    kind: ReplicaSet
    # Missing name
`,
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchRevisionCount{
					ResourcePath: podPath.Path,
					WantCount:    0,
				},
			},
		},
		{
			name: "Nil Body",
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchRevisionCount{
					ResourcePath: podPath.Path,
					WantCount:    0,
				},
			},
		},
	}

	for _, tc := range tests {
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
			commonFieldSet.Timestamp = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			k8sFieldSet := log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
			k8sFieldSet.K8sOperation = &model.KubernetesObjectOperation{
				Verb:       enum.RevisionVerbUpdate,
				Name:       podName,
				Namespace:  podNamespace,
				PluralKind: "pods",
			}
			k8sFieldSet.Principal = "user-1"

			event := commonlogk8sauditv2_contract.ResourceChangeEvent{
				Log:                   l,
				EventType:             commonlogk8sauditv2_contract.ChangeEventTypeTargetModification,
				EventTargetBodyReader: reader,
				EventTargetResource: &commonlogk8sauditv2_contract.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  podNamespace,
					Name:       podName,
				},
			}

			cs := history.NewChangeSet(l)
			_, err := task.Process(ctx, 0, event, cs, nil, struct{}{})
			if err != nil {
				t.Fatalf("Process failed: %v", err)
			}

			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}
