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

package architecturegraph

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/sparsebitset"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func internJSON(t *testing.T, pool *khifilev6model.InternPool, jsonStr string) uint32 {
	t.Helper()
	node := structured.NewLazyJSONNodeFromBytes([]byte(jsonStr))
	ref, err := khifilev6model.ToInternedStruct(node, pool)
	if err != nil {
		t.Fatalf("failed to intern JSON: %v", err)
	}
	return ref.ID()
}

func TestBuilder_Build(t *testing.T) {
	testCases := []struct {
		name    string
		setup   func(t *testing.T) (*Builder, *apiv1.GetArchitectureGraphRequest)
		want    *apiv1.GetArchitectureGraphResponse
		wantErr bool
	}{
		{
			name: "complete topology with node, pod, service, replicaset, deployment and edges",
			setup: func(t *testing.T) (*Builder, *apiv1.GetArchitectureGraphRequest) {
				pool := khifilev6model.NewInternPool(khifilev6model.NewIDGenerator())

				nodeStructID := internJSON(t, pool, `{
					"metadata": {"uid": "node-uid-1", "labels": {"kubernetes.io/hostname": "node-1"}},
					"spec": {
						"podCIDR": "10.244.0.0/24",
						"taints": [{"key": "node.kubernetes.io/unreachable", "effect": "NoExecute"}]
					},
					"status": {
						"addresses": [
							{"type": "InternalIP", "address": "192.168.1.10"},
							{"type": "ExternalIP", "address": "35.200.1.10"}
						],
						"conditions": [
							{"type": "Ready", "status": "True", "message": "kubelet is posting ready status"}
						]
					}
				}`)

				podStructID := internJSON(t, pool, `{
					"metadata": {
						"uid": "pod-uid-1",
						"labels": {"app": "nginx", "tier": "frontend"},
						"ownerReferences": [{"uid": "rs-uid-1"}]
					},
					"spec": {
						"nodeName": "node-1",
						"initContainers": [{"name": "init-sysctl"}],
						"containers": [{"name": "nginx"}]
					},
					"status": {
						"phase": "Running",
						"podIP": "10.244.0.5",
						"initContainerStatuses": [{"name": "init-sysctl", "ready": true, "state": {"terminated": {"reason": "Completed", "exitCode": 0}}}],
						"containerStatuses": [{"name": "nginx", "ready": true, "state": {"running": {}}}],
						"conditions": [{"type": "Ready", "status": "True", "message": ""}]
					}
				}`)

				svcStructID := internJSON(t, pool, `{
					"metadata": {"uid": "svc-uid-1", "labels": {"app": "nginx"}},
					"spec": {
						"type": "ClusterIP",
						"clusterIP": "10.96.0.1",
						"selector": {"app": "nginx"}
					}
				}`)

				rsStructID := internJSON(t, pool, `{
					"metadata": {
						"uid": "rs-uid-1",
						"labels": {"app": "nginx"},
						"ownerReferences": [{"uid": "deploy-uid-1"}]
					}
				}`)

				deployStructID := internJSON(t, pool, `{
					"metadata": {
						"uid": "deploy-uid-1",
						"labels": {"app": "nginx"}
					}
				}`)

				timelines := []*cel.TimelineData{
					{ID: 1, Name: "cluster-1"},
					{ID: 2, ParentID: 1, Name: "v1"},
					{ID: 3, ParentID: 2, Name: "node"},
					{ID: 4, ParentID: 3, Name: "cluster-scope"},
					{
						ID:       5,
						ParentID: 4,
						Name:     "node-1",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 1000, Verb: "Create", ResourceBodyStructID: nodeStructID},
						},
					},

					{ID: 6, ParentID: 2, Name: "pod"},
					{ID: 7, ParentID: 6, Name: "default"},
					{
						ID:       8,
						ParentID: 7,
						Name:     "nginx-pod",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 1100, Verb: "Create", ResourceBodyStructID: podStructID},
						},
					},

					{ID: 9, ParentID: 2, Name: "service"},
					{ID: 10, ParentID: 9, Name: "default"},
					{
						ID:       11,
						ParentID: 10,
						Name:     "nginx-svc",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 1050, Verb: "Create", ResourceBodyStructID: svcStructID},
						},
					},

					{ID: 12, ParentID: 1, Name: "apps/v1"},
					{ID: 13, ParentID: 12, Name: "replicaset"},
					{ID: 14, ParentID: 13, Name: "default"},
					{
						ID:       15,
						ParentID: 14,
						Name:     "nginx-rs",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 1020, Verb: "Create", ResourceBodyStructID: rsStructID},
						},
					},

					{ID: 16, ParentID: 12, Name: "deployment"},
					{ID: 17, ParentID: 16, Name: "default"},
					{
						ID:       18,
						ParentID: 17,
						Name:     "nginx-deploy",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 1010, Verb: "Create", ResourceBodyStructID: deployStructID},
						},
					},
				}

				tlMap := make(map[uint32]*cel.TimelineData)
				for _, tl := range timelines {
					tlMap[tl.ID] = tl
				}

				builder := NewBuilder(timelines, tlMap, pool)
				req := &apiv1.GetArchitectureGraphRequest{
					TimestampNs: proto.Int64(2000),
				}
				return builder, req
			},
			want: &apiv1.GetArchitectureGraphResponse{
				TimestampNs: proto.Int64(2000),
				Nodes: []*apiv1.GraphNode{
					{
						Id:         proto.String("node/cluster-scope/node-1"),
						TimelineId: proto.Uint32(5),
						Uid:        proto.String("node-uid-1"),
						Name:       proto.String("node-1"),
						PodCidr:    proto.String("10.244.0.0/24"),
						InternalIp: proto.String("192.168.1.10"),
						ExternalIp: proto.String("35.200.1.10"),
						Taints:     []string{"node.kubernetes.io/unreachable(NoExecute)"},
						Conditions: []*apiv1.GraphCondition{
							{Type: proto.String("Ready"), Status: proto.String("True"), Message: proto.String("kubelet is posting ready status"), IsPositive: proto.Bool(true)},
						},
						UpdatedAtNs: proto.Int64(1000),
						DeletedAtNs: proto.Int64(0),
						Labels:      map[string]string{"kubernetes.io/hostname": "node-1"},
					},
				},
				Pods: []*apiv1.GraphPod{
					{
						Id:             proto.String("pod/default/nginx-pod"),
						TimelineId:     proto.Uint32(8),
						Uid:            proto.String("pod-uid-1"),
						Name:           proto.String("nginx-pod"),
						Namespace:      proto.String("default"),
						NodeName:       proto.String("node-1"),
						PodIp:          proto.String("10.244.0.5"),
						Phase:          proto.String("Running"),
						IsPhaseHealthy: proto.Bool(true),
						OwnerUids:      []string{"rs-uid-1"},
						UpdatedAtNs:    proto.Int64(1100),
						DeletedAtNs:    proto.Int64(0),
						Labels:         map[string]string{"app": "nginx", "tier": "frontend"},
						Containers: []*apiv1.GraphContainer{
							{
								Name:                   proto.String("init-sysctl"),
								IsInitContainer:        proto.Bool(true),
								IsStatusHealthy:        proto.Bool(true),
								Ready:                  proto.Bool(true),
								Status:                 proto.String("Completed"),
								Reason:                 proto.String("Completed"),
								Code:                   proto.Int32(0),
								StatusReadFromManifest: proto.Bool(true),
							},
							{
								Name:                   proto.String("nginx"),
								IsInitContainer:        proto.Bool(false),
								IsStatusHealthy:        proto.Bool(true),
								Ready:                  proto.Bool(true),
								Status:                 proto.String("Running"),
								Reason:                 proto.String("Unknown"),
								StatusReadFromManifest: proto.Bool(true),
							},
						},
						Conditions: []*apiv1.GraphCondition{
							{Type: proto.String("Ready"), Status: proto.String("True"), Message: proto.String(""), IsPositive: proto.Bool(true)},
						},
					},
				},
				Services: []*apiv1.GraphService{
					{
						Id:          proto.String("service/default/nginx-svc"),
						TimelineId:  proto.Uint32(11),
						Uid:         proto.String("svc-uid-1"),
						Name:        proto.String("nginx-svc"),
						Namespace:   proto.String("default"),
						Type:        proto.String("ClusterIP"),
						ClusterIp:   proto.String("10.96.0.1"),
						UpdatedAtNs: proto.Int64(1050),
						DeletedAtNs: proto.Int64(0),
						Labels:      map[string]string{"app": "nginx"},
					},
				},
				PodOwners: []*apiv1.GraphPodOwner{
					{
						Id:          proto.String("replicaset/default/nginx-rs"),
						TimelineId:  proto.Uint32(15),
						Uid:         proto.String("rs-uid-1"),
						Name:        proto.String("nginx-rs"),
						Namespace:   proto.String("default"),
						Kind:        proto.String("replicaset"),
						OwnerUids:   []string{"deploy-uid-1"},
						UpdatedAtNs: proto.Int64(1020),
						DeletedAtNs: proto.Int64(0),
						Labels:      map[string]string{"app": "nginx"},
					},
				},
				PodOwnerOwners: []*apiv1.GraphPodOwnerOwner{
					{
						Id:          proto.String("deployment/default/nginx-deploy"),
						TimelineId:  proto.Uint32(18),
						Uid:         proto.String("deploy-uid-1"),
						Name:        proto.String("nginx-deploy"),
						Namespace:   proto.String("default"),
						Kind:        proto.String("deployment"),
						UpdatedAtNs: proto.Int64(1010),
						DeletedAtNs: proto.Int64(0),
						Labels:      map[string]string{"app": "nginx"},
					},
				},
				Edges: []*apiv1.GraphEdge{
					{
						Type:     apiv1.GraphEdge_EDGE_TYPE_SERVICE_TO_POD.Enum(),
						SourceId: proto.String("service/default/nginx-svc"),
						TargetId: proto.String("pod/default/nginx-pod"),
					},
					{
						Type:     apiv1.GraphEdge_EDGE_TYPE_POD_OWNER_TO_POD.Enum(),
						SourceId: proto.String("replicaset/default/nginx-rs"),
						TargetId: proto.String("pod/default/nginx-pod"),
					},
					{
						Type:     apiv1.GraphEdge_EDGE_TYPE_POD_OWNER_OWNER_TO_POD_OWNER.Enum(),
						SourceId: proto.String("deployment/default/nginx-deploy"),
						TargetId: proto.String("replicaset/default/nginx-rs"),
					},
				},
			},
		},
		{
			name: "virtual node synthesized when pod references unobserved node",
			setup: func(t *testing.T) (*Builder, *apiv1.GetArchitectureGraphRequest) {
				pool := khifilev6model.NewInternPool(khifilev6model.NewIDGenerator())

				podStructID := internJSON(t, pool, `{
					"metadata": {"uid": "pod-uid-2"},
					"spec": {"nodeName": "node-unobserved"},
					"status": {"phase": "Pending"}
				}`)

				timelines := []*cel.TimelineData{
					{ID: 1, Name: "cluster-1"},
					{ID: 2, ParentID: 1, Name: "v1"},
					{ID: 3, ParentID: 2, Name: "pod"},
					{ID: 4, ParentID: 3, Name: "default"},
					{
						ID:       5,
						ParentID: 4,
						Name:     "pending-pod",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 500, Verb: "Create", ResourceBodyStructID: podStructID},
						},
					},
				}
				tlMap := make(map[uint32]*cel.TimelineData)
				for _, tl := range timelines {
					tlMap[tl.ID] = tl
				}

				builder := NewBuilder(timelines, tlMap, pool)
				req := &apiv1.GetArchitectureGraphRequest{
					TimestampNs: proto.Int64(600),
				}
				return builder, req
			},
			want: &apiv1.GetArchitectureGraphResponse{
				TimestampNs: proto.Int64(600),
				Nodes: []*apiv1.GraphNode{
					{
						Id:         proto.String("node/cluster-scope/node-unobserved"),
						TimelineId: proto.Uint32(0),
						Name:       proto.String("node-unobserved"),
						PodCidr:    proto.String("-"),
						InternalIp: proto.String("-"),
						ExternalIp: proto.String("-"),
						Labels:     map[string]string{},
					},
				},
				Pods: []*apiv1.GraphPod{
					{
						Id:             proto.String("pod/default/pending-pod"),
						TimelineId:     proto.Uint32(5),
						Uid:            proto.String("pod-uid-2"),
						Name:           proto.String("pending-pod"),
						Namespace:      proto.String("default"),
						NodeName:       proto.String("node-unobserved"),
						PodIp:          proto.String("-"),
						Phase:          proto.String("Pending"),
						IsPhaseHealthy: proto.Bool(false),
						UpdatedAtNs:    proto.Int64(500),
						DeletedAtNs:    proto.Int64(0),
						Labels:         map[string]string{},
					},
				},
				Services:       []*apiv1.GraphService{},
				PodOwners:      []*apiv1.GraphPodOwner{},
				PodOwnerOwners: []*apiv1.GraphPodOwnerOwner{},
				Edges:          []*apiv1.GraphEdge{},
			},
		},
		{
			name: "node name resolved from binding child timeline",
			setup: func(t *testing.T) (*Builder, *apiv1.GetArchitectureGraphRequest) {
				pool := khifilev6model.NewInternPool(khifilev6model.NewIDGenerator())

				podStructID := internJSON(t, pool, `{
					"metadata": {"uid": "pod-uid-3"},
					"spec": {},
					"status": {"phase": "Running"}
				}`)

				bindingStructID := internJSON(t, pool, `{
					"target": {"name": "bound-node"}
				}`)

				timelines := []*cel.TimelineData{
					{ID: 1, Name: "cluster-1"},
					{ID: 2, ParentID: 1, Name: "v1"},
					{ID: 3, ParentID: 2, Name: "pod"},
					{ID: 4, ParentID: 3, Name: "default"},
					{
						ID:          5,
						ParentID:    4,
						ChildrenIDs: []uint32{6},
						Name:        "bound-pod",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 100, Verb: "Create", ResourceBodyStructID: podStructID},
						},
					},
					{
						ID:       6,
						ParentID: 5,
						Name:     "binding",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 110, Verb: "Create", ResourceBodyStructID: bindingStructID},
						},
					},
				}
				tlMap := make(map[uint32]*cel.TimelineData)
				for _, tl := range timelines {
					tlMap[tl.ID] = tl
				}

				builder := NewBuilder(timelines, tlMap, pool)
				req := &apiv1.GetArchitectureGraphRequest{
					TimestampNs: proto.Int64(200),
				}
				return builder, req
			},
			want: &apiv1.GetArchitectureGraphResponse{
				TimestampNs: proto.Int64(200),
				Nodes: []*apiv1.GraphNode{
					{
						Id:         proto.String("node/cluster-scope/bound-node"),
						TimelineId: proto.Uint32(0),
						Name:       proto.String("bound-node"),
						PodCidr:    proto.String("-"),
						InternalIp: proto.String("-"),
						ExternalIp: proto.String("-"),
						Labels:     map[string]string{},
					},
				},
				Pods: []*apiv1.GraphPod{
					{
						Id:             proto.String("pod/default/bound-pod"),
						TimelineId:     proto.Uint32(5),
						Uid:            proto.String("pod-uid-3"),
						Name:           proto.String("bound-pod"),
						Namespace:      proto.String("default"),
						NodeName:       proto.String("bound-node"),
						PodIp:          proto.String("-"),
						Phase:          proto.String("Running"),
						IsPhaseHealthy: proto.Bool(true),
						UpdatedAtNs:    proto.Int64(100),
						DeletedAtNs:    proto.Int64(0),
						Labels:         map[string]string{},
					},
				},
				Services:       []*apiv1.GraphService{},
				PodOwners:      []*apiv1.GraphPodOwner{},
				PodOwnerOwners: []*apiv1.GraphPodOwnerOwner{},
				Edges:          []*apiv1.GraphEdge{},
			},
		},
		{
			name: "deletion threshold filters expired resources and keeps recent deletions",
			setup: func(t *testing.T) (*Builder, *apiv1.GetArchitectureGraphRequest) {
				pool := khifilev6model.NewInternPool(khifilev6model.NewIDGenerator())

				pod1StructID := internJSON(t, pool, `{"metadata": {"uid": "pod-1"}}`)
				pod2StructID := internJSON(t, pool, `{"metadata": {"uid": "pod-2"}}`)

				timelines := []*cel.TimelineData{
					{ID: 1, Name: "cluster-1"},
					{ID: 2, ParentID: 1, Name: "v1"},
					{ID: 3, ParentID: 2, Name: "pod"},
					{ID: 4, ParentID: 3, Name: "default"},
					{
						ID:       5,
						ParentID: 4,
						Name:     "deleted-recently-pod",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 100 * 1e9, Verb: "Create", ResourceBodyStructID: pod1StructID},
							{ChangedTime: 150 * 1e9, Verb: "Delete"},
						},
					},
					{
						ID:       6,
						ParentID: 4,
						Name:     "deleted-long-ago-pod",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 10 * 1e9, Verb: "Create", ResourceBodyStructID: pod2StructID},
							{ChangedTime: 20 * 1e9, Verb: "Delete"},
						},
					},
				}
				tlMap := make(map[uint32]*cel.TimelineData)
				for _, tl := range timelines {
					tlMap[tl.ID] = tl
				}

				builder := NewBuilder(timelines, tlMap, pool)
				req := &apiv1.GetArchitectureGraphRequest{
					TimestampNs:              proto.Int64(200 * 1e9),
					DeletionThresholdSeconds: proto.Float64(100.0),
				}
				return builder, req
			},
			want: &apiv1.GetArchitectureGraphResponse{
				TimestampNs: proto.Int64(200 * 1e9),
				Nodes:       []*apiv1.GraphNode{},
				Pods: []*apiv1.GraphPod{
					{
						Id:             proto.String("pod/default/deleted-recently-pod"),
						TimelineId:     proto.Uint32(5),
						Uid:            proto.String("pod-1"),
						Name:           proto.String("deleted-recently-pod"),
						Namespace:      proto.String("default"),
						PodIp:          proto.String("-"),
						Phase:          proto.String("Unknown"),
						IsPhaseHealthy: proto.Bool(false),
						UpdatedAtNs:    proto.Int64(0),
						DeletedAtNs:    proto.Int64(150 * 1e9),
						Labels:         map[string]string{},
					},
				},
				Services:       []*apiv1.GraphService{},
				PodOwners:      []*apiv1.GraphPodOwner{},
				PodOwnerOwners: []*apiv1.GraphPodOwnerOwner{},
				Edges:          []*apiv1.GraphEdge{},
			},
		},
		{
			name: "timeline bitset filtering restricts resources",
			setup: func(t *testing.T) (*Builder, *apiv1.GetArchitectureGraphRequest) {
				pool := khifilev6model.NewInternPool(khifilev6model.NewIDGenerator())

				pod1StructID := internJSON(t, pool, `{"metadata": {"uid": "pod-1"}}`)
				pod2StructID := internJSON(t, pool, `{"metadata": {"uid": "pod-2"}}`)

				timelines := []*cel.TimelineData{
					{ID: 1, Name: "cluster-1"},
					{ID: 2, ParentID: 1, Name: "v1"},
					{ID: 3, ParentID: 2, Name: "pod"},
					{ID: 4, ParentID: 3, Name: "default"},
					{
						ID:       5,
						ParentID: 4,
						Name:     "pod-included",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 100, Verb: "Create", ResourceBodyStructID: pod1StructID},
						},
					},
					{
						ID:       6,
						ParentID: 4,
						Name:     "pod-excluded",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 100, Verb: "Create", ResourceBodyStructID: pod2StructID},
						},
					},
				}
				tlMap := make(map[uint32]*cel.TimelineData)
				for _, tl := range timelines {
					tlMap[tl.ID] = tl
				}

				builder := NewBuilder(timelines, tlMap, pool)
				bitset := sparsebitset.Encode([]uint32{5})
				req := &apiv1.GetArchitectureGraphRequest{
					TimestampNs:    proto.Int64(200),
					TimelineBitset: bitset,
				}
				return builder, req
			},
			want: &apiv1.GetArchitectureGraphResponse{
				TimestampNs: proto.Int64(200),
				Nodes:       []*apiv1.GraphNode{},
				Pods: []*apiv1.GraphPod{
					{
						Id:             proto.String("pod/default/pod-included"),
						TimelineId:     proto.Uint32(5),
						Uid:            proto.String("pod-1"),
						Name:           proto.String("pod-included"),
						Namespace:      proto.String("default"),
						PodIp:          proto.String("-"),
						Phase:          proto.String("Unknown"),
						IsPhaseHealthy: proto.Bool(false),
						UpdatedAtNs:    proto.Int64(100),
						DeletedAtNs:    proto.Int64(0),
						Labels:         map[string]string{},
					},
				},
				Services:       []*apiv1.GraphService{},
				PodOwners:      []*apiv1.GraphPodOwner{},
				PodOwnerOwners: []*apiv1.GraphPodOwnerOwner{},
				Edges:          []*apiv1.GraphEdge{},
			},
		},
		{
			name: "empty timeline bitset filters all resources",
			setup: func(t *testing.T) (*Builder, *apiv1.GetArchitectureGraphRequest) {
				pool := khifilev6model.NewInternPool(khifilev6model.NewIDGenerator())
				pod1StructID := internJSON(t, pool, `{"metadata": {"uid": "pod-1"}}`)

				timelines := []*cel.TimelineData{
					{ID: 1, Name: "cluster-1"},
					{ID: 2, ParentID: 1, Name: "v1"},
					{ID: 3, ParentID: 2, Name: "pod"},
					{ID: 4, ParentID: 3, Name: "default"},
					{
						ID:       5,
						ParentID: 4,
						Name:     "pod-included",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 100, Verb: "Create", ResourceBodyStructID: pod1StructID},
						},
					},
				}
				tlMap := make(map[uint32]*cel.TimelineData)
				for _, tl := range timelines {
					tlMap[tl.ID] = tl
				}

				builder := NewBuilder(timelines, tlMap, pool)
				bitset := sparsebitset.Encode([]uint32{})
				req := &apiv1.GetArchitectureGraphRequest{
					TimestampNs:    proto.Int64(200),
					TimelineBitset: bitset,
				}
				return builder, req
			},
			want: &apiv1.GetArchitectureGraphResponse{
				TimestampNs:    proto.Int64(200),
				Nodes:          []*apiv1.GraphNode{},
				Pods:           []*apiv1.GraphPod{},
				Services:       []*apiv1.GraphService{},
				PodOwners:      []*apiv1.GraphPodOwner{},
				PodOwnerOwners: []*apiv1.GraphPodOwnerOwner{},
				Edges:          []*apiv1.GraphEdge{},
			},
		},
		{
			name: "node condition positivity and container waiting states",
			setup: func(t *testing.T) (*Builder, *apiv1.GetArchitectureGraphRequest) {
				pool := khifilev6model.NewInternPool(khifilev6model.NewIDGenerator())

				nodeStructID := internJSON(t, pool, `{
					"metadata": {"uid": "node-uid-2"},
					"status": {
						"conditions": [
							{"type": "Ready", "status": "False", "message": "kubelet stopped posting ready status"},
							{"type": "MemoryPressure", "status": "True", "message": "node has memory pressure"},
							{"type": "DiskPressure", "status": "False", "message": "node has no disk pressure"}
						]
					}
				}`)

				podStructID := internJSON(t, pool, `{
					"metadata": {"uid": "pod-uid-crash"},
					"spec": {
						"nodeName": "node-2",
						"containers": [{"name": "failing-app"}]
					},
					"status": {
						"phase": "Running",
						"containerStatuses": [
							{
								"name": "failing-app",
								"ready": false,
								"state": {
									"waiting": {
										"reason": "CrashLoopBackOff"
									}
								}
							}
						]
					}
				}`)

				timelines := []*cel.TimelineData{
					{ID: 1, Name: "cluster-1"},
					{ID: 2, ParentID: 1, Name: "v1"},
					{ID: 3, ParentID: 2, Name: "node"},
					{ID: 4, ParentID: 3, Name: "cluster-scope"},
					{
						ID:       5,
						ParentID: 4,
						Name:     "node-2",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 100, Verb: "Create", ResourceBodyStructID: nodeStructID},
						},
					},
					{ID: 6, ParentID: 2, Name: "pod"},
					{ID: 7, ParentID: 6, Name: "default"},
					{
						ID:       8,
						ParentID: 7,
						Name:     "failing-pod",
						Revisions: []cel.RevisionInfo{
							{ChangedTime: 120, Verb: "Create", ResourceBodyStructID: podStructID},
						},
					},
				}
				tlMap := make(map[uint32]*cel.TimelineData)
				for _, tl := range timelines {
					tlMap[tl.ID] = tl
				}

				builder := NewBuilder(timelines, tlMap, pool)
				req := &apiv1.GetArchitectureGraphRequest{
					TimestampNs: proto.Int64(200),
				}
				return builder, req
			},
			want: &apiv1.GetArchitectureGraphResponse{
				TimestampNs: proto.Int64(200),
				Nodes: []*apiv1.GraphNode{
					{
						Id:          proto.String("node/cluster-scope/node-2"),
						TimelineId:  proto.Uint32(5),
						Uid:         proto.String("node-uid-2"),
						Name:        proto.String("node-2"),
						PodCidr:     proto.String("-"),
						InternalIp:  proto.String("-"),
						ExternalIp:  proto.String("-"),
						UpdatedAtNs: proto.Int64(100),
						DeletedAtNs: proto.Int64(0),
						Labels:      map[string]string{},
						Conditions: []*apiv1.GraphCondition{
							{Type: proto.String("Ready"), Status: proto.String("False"), Message: proto.String("kubelet stopped posting ready status"), IsPositive: proto.Bool(false)},
							{Type: proto.String("MemoryPressure"), Status: proto.String("True"), Message: proto.String("node has memory pressure"), IsPositive: proto.Bool(false)},
							{Type: proto.String("DiskPressure"), Status: proto.String("False"), Message: proto.String("node has no disk pressure"), IsPositive: proto.Bool(true)},
						},
					},
				},
				Pods: []*apiv1.GraphPod{
					{
						Id:             proto.String("pod/default/failing-pod"),
						TimelineId:     proto.Uint32(8),
						Uid:            proto.String("pod-uid-crash"),
						Name:           proto.String("failing-pod"),
						Namespace:      proto.String("default"),
						NodeName:       proto.String("node-2"),
						PodIp:          proto.String("-"),
						Phase:          proto.String("Running"),
						IsPhaseHealthy: proto.Bool(true),
						UpdatedAtNs:    proto.Int64(120),
						DeletedAtNs:    proto.Int64(0),
						Labels:         map[string]string{},
						Containers: []*apiv1.GraphContainer{
							{
								Name:                   proto.String("failing-app"),
								IsInitContainer:        proto.Bool(false),
								IsStatusHealthy:        proto.Bool(false),
								Ready:                  proto.Bool(false),
								Status:                 proto.String("CrashLoopBackOff"),
								Reason:                 proto.String("CrashLoopBackOff"),
								StatusReadFromManifest: proto.Bool(true),
							},
						},
					},
				},
				Services:       []*apiv1.GraphService{},
				PodOwners:      []*apiv1.GraphPodOwner{},
				PodOwnerOwners: []*apiv1.GraphPodOwnerOwner{},
				Edges:          []*apiv1.GraphEdge{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder, req := tc.setup(t)
			got, err := builder.Build(context.Background(), req)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Build() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("Build() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
