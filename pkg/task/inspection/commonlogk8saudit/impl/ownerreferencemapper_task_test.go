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
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

// TestResourceOwnerReferenceTimelineMapperTaskV2_ProcessLog verifies that owner references in the resource body are mapped correctly to timeline aliases.
func TestResourceOwnerReferenceTimelineMapperTaskV2_ProcessLog(t *testing.T) {
	task := &resourceOwnerReferenceTimelineMapperTaskSettingV2{
		nonNamespacedOwnerTypes: map[string]struct{}{
			"core/v1#node": {},
		},
	}

	parseYAML := func(yamlStr string) *structured.NodeReader {
		if yamlStr == "" {
			return nil
		}
		node, err := structured.FromYAML(yamlStr)
		if err != nil {
			t.Fatalf("failed to parse YAML: %v", err)
		}
		return structured.NewNodeReader(node)
	}

	tests := []struct {
		name   string
		yaml   string
		assert func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context)
	}{
		{
			name: "No Owner References",
			yaml: `
metadata:
  name: nginx
  namespace: default
`,
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				ownerPath := MustResolveTimelinePath(ctx, "k8s", &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: "apps/v1",
					Kind:       "replicaset",
					Namespace:  "default",
					Name:       "nginx-replicaset",
				})
				expectedTargetPath := commonlogk8saudit_contract.MustOwnedResourceTimeline(ctx, ownerPath, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasNoAlias(expectedTargetPath)
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
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				aliasPath := MustResolveTimelinePath(ctx, "k8s", &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "nginx",
				})
				ownerPath := MustResolveTimelinePath(ctx, "k8s", &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: "apps/v1",
					Kind:       "replicaset",
					Namespace:  "default",
					Name:       "nginx-replicaset",
				})
				expectedTargetPath := commonlogk8saudit_contract.MustOwnedResourceTimeline(ctx, ownerPath, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasAlias(expectedTargetPath, aliasPath)
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
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				aliasPath := MustResolveTimelinePath(ctx, "k8s", &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "nginx",
				})
				ownerPath := MustResolveTimelinePath(ctx, "k8s", &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "node",
					Namespace:  "cluster-scope",
					Name:       "node-1",
				})
				expectedTargetPath := commonlogk8saudit_contract.MustOwnedResourceTimeline(ctx, ownerPath, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasAlias(expectedTargetPath, aliasPath)
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
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				aliasPath := MustResolveTimelinePath(ctx, "k8s", &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "pod",
					Namespace:  "default",
					Name:       "nginx",
				})
				ownerPath1 := MustResolveTimelinePath(ctx, "k8s", &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: "apps/v1",
					Kind:       "replicaset",
					Namespace:  "default",
					Name:       "nginx-replicaset",
				})
				expectedTargetPath1 := commonlogk8saudit_contract.MustOwnedResourceTimeline(ctx, ownerPath1, "nginx[kind:pod]")

				ownerPath2 := MustResolveTimelinePath(ctx, "k8s", &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: "core/v1",
					Kind:       "node",
					Namespace:  "cluster-scope",
					Name:       "node-1",
				})
				expectedTargetPath2 := commonlogk8saudit_contract.MustOwnedResourceTimeline(ctx, ownerPath2, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasAlias(expectedTargetPath1, aliasPath).
					HasAlias(expectedTargetPath2, aliasPath)
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
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				ownerPath := MustResolveTimelinePath(ctx, "k8s", &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: "apps/v1",
					Kind:       "replicaset",
					Namespace:  "default",
					Name:       "nginx-replicaset",
				})
				expectedTargetPath := commonlogk8saudit_contract.MustOwnedResourceTimeline(ctx, ownerPath, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasNoAlias(expectedTargetPath)
			},
		},
		{
			name: "Nil Body",
			yaml: "",
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, ctx context.Context) {
				ownerPath := MustResolveTimelinePath(ctx, "k8s", &commonlogk8saudit_contract.ResourceIdentity{
					APIVersion: "apps/v1",
					Kind:       "replicaset",
					Namespace:  "default",
					Name:       "nginx-replicaset",
				})
				expectedTargetPath := commonlogk8saudit_contract.MustOwnedResourceTimeline(ctx, ownerPath, "nginx[kind:pod]")

				testchangeset.AssertTimeline(t, cs).
					HasNoAlias(expectedTargetPath)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			builder := khifilev6.NewBuilder()
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			nodeReader := parseYAML(tc.yaml)

			k8sFieldSet := &commonlogk8saudit_contract.K8sAuditLogFieldSet{
				Principal:    "user-1",
				APIVersion:   "core/v1",
				PluralKind:   "pods",
				ResourceName: "nginx",
				Namespace:    "default",
				ClusterName:  "k8s",
				Verb:         commonlogk8saudit_contract.VerbUpdate,
			}
			commonFs := &log.CommonFieldSet{
				Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			}
			logObj := log.NewLogWithFieldSetsForTest(k8sFieldSet, commonFs)

			targetResource := &commonlogk8saudit_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Namespace:  "default",
				Name:       "nginx",
			}

			groupSet := commonlogk8saudit_contract.RelatedGroupSet{
				Roles: map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{
					"target": {
						Resource: targetResource,
						Logs: []*commonlogk8saudit_contract.ResourceManifestLog{
							{Log: logObj, ResourceBodyReader: nodeReader, ResourceBodyYAML: tc.yaml},
						},
					},
				},
			}

			event := commonlogk8saudit_contract.MultiGroupLogEvent{
				Log:              logObj,
				GroupRole:        "target",
				ResourceIdentity: targetResource,
				EventType:        commonlogk8saudit_contract.ChangeEventTypeV2Modification,
				GroupSet:         groupSet,
			}

			cs, _, err := task.ProcessLog(ctx, event, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLog() failed: %v", err)
			}

			tc.assert(t, cs, ctx)
		})
	}
}
