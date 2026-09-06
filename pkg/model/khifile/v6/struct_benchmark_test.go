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

package khifilev6

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

// benchmarkPodYAML provides a representative Kubernetes Pod manifest with labels, annotations,
// container specifications, volume mounts, and status conditions typical of GKE audit logs.
const benchmarkPodYAML = `
apiVersion: v1
kind: Pod
metadata:
  name: frontend-6b47b85f-x9z2p
  namespace: production
  uid: 3f8a4b2c-1d5e-4f8a-9b2c-1d5e4f8a9b2c
  resourceVersion: "1849204"
  labels:
    app: frontend
    tier: web
    environment: production
    release: stable
  annotations:
    k8s.v1.cni.cncf.io/network-status: "installed"
    deployment.kubernetes.io/revision: "12"
spec:
  restartPolicy: Always
  serviceAccountName: frontend-sa
  containers:
  - name: web
    image: gcr.io/gke-prod/frontend:v2.1.0
    ports:
    - containerPort: 8080
      protocol: TCP
    env:
    - name: POD_NAME
      value: frontend-6b47b85f-x9z2p
    - name: ENVIRONMENT
      value: production
    resources:
      limits:
        cpu: "500m"
        memory: "512Mi"
      requests:
        cpu: "100m"
        memory: "128Mi"
  - name: mesh-proxy
    image: gcr.io/gke-prod/sidecar:v1.4.2
    ports:
    - containerPort: 15001
      protocol: TCP
status:
  phase: Running
  hostIP: 10.128.0.45
  podIP: 10.244.2.19
  conditions:
  - type: Initialized
    status: "True"
    lastTransitionTime: "2026-09-06T00:00:00Z"
  - type: Ready
    status: "True"
    lastTransitionTime: "2026-09-06T00:00:15Z"
  - type: ContainersReady
    status: "True"
    lastTransitionTime: "2026-09-06T00:00:15Z"
  - type: PodScheduled
    status: "True"
    lastTransitionTime: "2026-09-06T00:00:01Z"
`

// benchmarkDeploymentYAML provides a representative Kubernetes Deployment manifest with nested
// selector, replica count, rolling update strategy, and embedded pod template specifications.
const benchmarkDeploymentYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment-processor
  namespace: billing
  uid: e7b2c1d5-8f4a-4b2c-9d5e-1f8a4b2c9d5e
  resourceVersion: "948201"
  labels:
    app: payment-processor
    team: payments
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: payment-processor
  template:
    metadata:
      labels:
        app: payment-processor
    spec:
      containers:
      - name: processor
        image: gcr.io/billing-prod/processor:1.8.0
        ports:
        - containerPort: 9000
        env:
        - name: DB_HOST
          value: payment-db.internal
        - name: RETRY_ATTEMPTS
          value: "3"
status:
  replicas: 3
  readyReplicas: 3
  updatedReplicas: 3
  availableReplicas: 3
`

// benchmarkSmallMapYAML represents a lightweight metadata-only Kubernetes manifest fragment.
const benchmarkSmallMapYAML = `
metadata:
  name: simple-service
  namespace: default
  resourceVersion: "1001"
`

// BenchmarkToInternedStruct evaluates the allocation churn and latency of interning structured
// Kubernetes AST nodes into compact deduplicated InternPool structs.
func BenchmarkToInternedStruct(b *testing.B) {
	podNode, err := structured.FromYAML(benchmarkPodYAML)
	if err != nil {
		b.Fatalf("failed to parse pod YAML: %v", err)
	}

	deploymentNode, err := structured.FromYAML(benchmarkDeploymentYAML)
	if err != nil {
		b.Fatalf("failed to parse deployment YAML: %v", err)
	}

	smallNode, err := structured.FromYAML(benchmarkSmallMapYAML)
	if err != nil {
		b.Fatalf("failed to parse small map YAML: %v", err)
	}

	b.Run("Pod_Warm", func(b *testing.B) {
		benchmarkToInternedStructWarm(b, podNode)
	})

	b.Run("Pod_Cold", func(b *testing.B) {
		benchmarkToInternedStructCold(b, podNode)
	})

	b.Run("Deployment_Warm", func(b *testing.B) {
		benchmarkToInternedStructWarm(b, deploymentNode)
	})

	b.Run("Deployment_Cold", func(b *testing.B) {
		benchmarkToInternedStructCold(b, deploymentNode)
	})

	b.Run("SmallMap_Warm", func(b *testing.B) {
		benchmarkToInternedStructWarm(b, smallNode)
	})
}

// BenchmarkTraverseLeafNodes isolates the recursion and leaf iteration generated during AST traversal.
func BenchmarkTraverseLeafNodes(b *testing.B) {
	podNode, err := structured.FromYAML(benchmarkPodYAML)
	if err != nil {
		b.Fatalf("failed to parse pod YAML: %v", err)
	}

	keyBuf := make([]byte, 0, 128)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := traverseLeafNodes(podNode, keyBuf, true, func(key []byte, val structured.Node) error {
			return nil
		}); err != nil {
			b.Fatalf("traverseLeafNodes failed: %v", err)
		}
	}
}

// benchmarkToInternedStructWarm measures repeatedly interning a manifest against an already
// populated InternPool, isolating struct reference pointer allocations and path string churn.
func benchmarkToInternedStructWarm(b *testing.B, node structured.Node) {
	pool := NewTestInternPool(id.NewGenerator())
	if _, err := ToInternedStruct(node, pool); err != nil {
		b.Fatalf("warm up interning failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ref, err := ToInternedStruct(node, pool)
		if err != nil {
			b.Fatalf("ToInternedStruct failed: %v", err)
		}
		if ref.id == 0 {
			b.Fatal("unexpected zero struct ID")
		}
	}
}

// benchmarkToInternedStructCold measures first-time interning of a manifest into an unpopulated
// InternPool, including field set string interning, FlatStructStore storage, and staging.
func benchmarkToInternedStructCold(b *testing.B, node structured.Node) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pool := NewTestInternPool(id.NewGenerator())
		b.StartTimer()

		ref, err := ToInternedStruct(node, pool)
		if err != nil {
			b.Fatalf("ToInternedStruct failed: %v", err)
		}
		if ref.id == 0 {
			b.Fatal("unexpected zero struct ID")
		}
	}
}

// toInternedStruct_Legacy converts a structured.Node to an InternedStruct using the unoptimized baseline
// path (allocating intermediate string slices and returning heap-allocated objects) for comparative analysis.
func toInternedStruct_Legacy(node structured.Node, pool *InternPool) (*pb.InternedStruct, uint32, error) {
	if node.Type() != structured.MapNodeType {
		return nil, 0, fmt.Errorf("expected map node, got %v", node.Type())
	}

	flattenedKeys := make([]string, 0, 16)
	flattenedValues := make([]structured.Node, 0, 16)
	keyBuf := make([]byte, 0, 64)
	err := traverseLeafNodes(node, keyBuf, true, func(key []byte, val structured.Node) error {
		flattenedKeys = append(flattenedKeys, string(key))
		flattenedValues = append(flattenedValues, val)
		return nil
	})
	if err != nil {
		return nil, 0, err
	}

	fieldSetRef := pool.InternFieldSet(flattenedKeys)

	key, err := structKeyFromNodes(fieldSetRef.id, flattenedValues, pool, keyBuf)
	if err != nil {
		return nil, 0, err
	}

	if idVal, ok := pool.structToID.Load(key); ok {
		sID := idVal.(uint32)
		return pool.resolveStructFromID(sID), sID, nil
	}

	newID := pool.idGen.New(id.Struct)
	if err := pool.flatStructs.StoreFromNodes(newID, fieldSetRef.id, flattenedValues, pool); err != nil {
		return nil, 0, err
	}

	actual, loaded := pool.structToID.LoadOrStore(key, newID)
	if loaded {
		sID := actual.(uint32)
		return pool.resolveStructFromID(sID), sID, nil
	}

	pool.stageStruct(newID)
	return pool.resolveStructFromID(newID), newID, nil
}

// flattenNode_Prototype_Optimized flattens a map AST into a contiguous key byte buffer
// and offset slice, demonstrating string-allocation-free flattening in isolation.
func flattenNode_Prototype_Optimized(node structured.Node, keyBuf []byte, isRoot bool, keysBuf *[]byte, offsets *[]uint32, values *[]structured.Node) error {
	if node.Type() != structured.MapNodeType {
		return fmt.Errorf("expected map node in flattenNode, got %v", node.Type())
	}

	for key, child := range node.Children() {
		origLen := len(keyBuf)
		if !isRoot {
			keyBuf = append(keyBuf, fieldPathSeparator...)
		}
		keyBuf = append(keyBuf, key.Key...)

		if child.Type() == structured.MapNodeType {
			if child.Len() == 0 {
				*keysBuf = append(*keysBuf, keyBuf...)
				*offsets = append(*offsets, uint32(len(*keysBuf)))
				*values = append(*values, child)
			} else {
				err := flattenNode_Prototype_Optimized(child, keyBuf, false, keysBuf, offsets, values)
				if err != nil {
					return err
				}
			}
		} else {
			*keysBuf = append(*keysBuf, keyBuf...)
			*offsets = append(*offsets, uint32(len(*keysBuf)))
			*values = append(*values, child)
		}
		keyBuf = keyBuf[:origLen]
	}
	return nil
}

// BenchmarkToInternedStruct_Legacy evaluates the performance of the unoptimized legacy baseline
// for direct comparison against optimized BenchmarkToInternedStruct.
func BenchmarkToInternedStruct_Legacy(b *testing.B) {
	podNode, err := structured.FromYAML(benchmarkPodYAML)
	if err != nil {
		b.Fatalf("failed to parse pod YAML: %v", err)
	}

	deploymentNode, err := structured.FromYAML(benchmarkDeploymentYAML)
	if err != nil {
		b.Fatalf("failed to parse deployment YAML: %v", err)
	}

	smallNode, err := structured.FromYAML(benchmarkSmallMapYAML)
	if err != nil {
		b.Fatalf("failed to parse small map YAML: %v", err)
	}

	b.Run("Pod_Warm", func(b *testing.B) {
		benchmarkToInternedStructWarm_Legacy(b, podNode)
	})

	b.Run("Pod_Cold", func(b *testing.B) {
		benchmarkToInternedStructCold_Legacy(b, podNode)
	})

	b.Run("Deployment_Warm", func(b *testing.B) {
		benchmarkToInternedStructWarm_Legacy(b, deploymentNode)
	})

	b.Run("Deployment_Cold", func(b *testing.B) {
		benchmarkToInternedStructCold_Legacy(b, deploymentNode)
	})

	b.Run("SmallMap_Warm", func(b *testing.B) {
		benchmarkToInternedStructWarm_Legacy(b, smallNode)
	})
}

func benchmarkToInternedStructWarm_Legacy(b *testing.B, node structured.Node) {
	pool := NewTestInternPool(id.NewGenerator())
	if _, _, err := toInternedStruct_Legacy(node, pool); err != nil {
		b.Fatalf("warm up interning failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, idVal, err := toInternedStruct_Legacy(node, pool)
		if err != nil {
			b.Fatalf("toInternedStruct_Legacy failed: %v", err)
		}
		if idVal == 0 {
			b.Fatal("unexpected zero struct ID")
		}
	}
}

func benchmarkToInternedStructCold_Legacy(b *testing.B, node structured.Node) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pool := NewTestInternPool(id.NewGenerator())
		b.StartTimer()

		_, idVal, err := toInternedStruct_Legacy(node, pool)
		if err != nil {
			b.Fatalf("toInternedStruct_Legacy failed: %v", err)
		}
		if idVal == 0 {
			b.Fatal("unexpected zero struct ID")
		}
	}
}

// BenchmarkFlattenNode_Prototype_Optimized isolates AST flattening using contiguous byte buffers,
// comparing directly against baseline BenchmarkFlattenNode.
func BenchmarkFlattenNode_Prototype_Optimized(b *testing.B) {
	podNode, err := structured.FromYAML(benchmarkPodYAML)
	if err != nil {
		b.Fatalf("failed to parse pod YAML: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		keyBuf := make([]byte, 0, 64)
		keysBuf := make([]byte, 0, 1024)
		offsets := make([]uint32, 0, 64)
		values := make([]structured.Node, 0, 64)
		if err := flattenNode_Prototype_Optimized(podNode, keyBuf, true, &keysBuf, &offsets, &values); err != nil {
			b.Fatalf("flattenNode_Prototype_Optimized failed: %v", err)
		}
	}
}

// TestToInternedStruct_Equivalence proves that production ToInternedStruct generates
// bit-for-bit identical protobuf structures and struct IDs as the legacy baseline.
func TestToInternedStruct_Equivalence(t *testing.T) {
	testCases := []struct {
		name string
		yaml string
	}{
		{name: "Pod", yaml: benchmarkPodYAML},
		{name: "Deployment", yaml: benchmarkDeploymentYAML},
		{name: "SmallMap", yaml: benchmarkSmallMapYAML},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := structured.FromYAML(tc.yaml)
			if err != nil {
				t.Fatalf("failed to parse YAML: %v", err)
			}

			// 1. Run legacy baseline
			poolBaseline := NewTestInternPool(id.NewGenerator())
			baselineProto, baselineID, err := toInternedStruct_Legacy(node, poolBaseline)
			if err != nil {
				t.Fatalf("baseline toInternedStruct_Legacy failed: %v", err)
			}

			// 2. Run production ToInternedStruct
			poolGot := NewTestInternPool(id.NewGenerator())
			gotRef, err := ToInternedStruct(node, poolGot)
			if err != nil {
				t.Fatalf("production ToInternedStruct failed: %v", err)
			}
			gotProto := gotRef.Resolve()

			// 3. Assert exact equivalence of IDs and protobuf messages
			if baselineID != gotRef.id {
				t.Errorf("struct ID mismatch: baseline=%d, got=%d", baselineID, gotRef.id)
			}

			if diff := cmp.Diff(baselineProto, gotProto, protocmp.Transform()); diff != "" {
				t.Errorf("resolved proto mismatch (-baseline +got):\n%s", diff)
			}
		})
	}
}

func buildNestedMapYAML(depth int, leafContent string) string {
	var sb strings.Builder
	for i := 0; i < depth; i++ {
		sb.WriteString(strings.Repeat("  ", i))
		fmt.Fprintf(&sb, "level_%d:\n", i)
	}
	sb.WriteString(strings.Repeat("  ", depth))
	sb.WriteString(leafContent)
	sb.WriteString("\n")
	return sb.String()
}

// TestToInternedStruct_Stress adversarially stress-tests production ToInternedStruct
// with deeply nested YAML structures, empty maps, complex scalar combinations, and repeated interning,
// verifying that struct IDs, resolved protobuf structures, and unmarshaled nodes match the baseline.
func TestToInternedStruct_Stress(t *testing.T) {
	testCases := []struct {
		name string
		yaml string
	}{
		{
			name: "EmptyRootMap",
			yaml: "{}",
		},
		{
			name: "EmptyChildMaps",
			yaml: `
metadata:
  labels: {}
  annotations: {}
spec:
  emptyConfig: {}
  nested:
    deeperEmpty: {}
`,
		},
		{
			name: "DeeplyNested_20Levels",
			yaml: buildNestedMapYAML(20, "leaf_scalar: 42"),
		},
		{
			name: "DeeplyNested_50Levels",
			yaml: buildNestedMapYAML(50, `leaf_str: "deep_value"`),
		},
		{
			name: "DeeplyNested_30Levels_EmptyMapLeaf",
			yaml: buildNestedMapYAML(30, "leaf_empty_map: {}"),
		},
		{
			name: "SpecialKeyCharacters",
			yaml: `
"k8s.io/custom-annotation": "value.with.dots"
"app/sub-app/name": "slashes-and-dashes"
"key with spaces and symbols": "123!@#"
"日本語のキー": "マルチバイト文字列"
`,
		},
		{
			name: "MixedComplexTypesAndSequences",
			yaml: `
apiVersion: v1
kind: Service
metadata:
  name: test-service
  namespace: test-ns
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
  - name: https
    port: 443
    protocol: TCP
  clusterIP: 10.0.0.1
  selector:
    app: test
  active: true
  score: 98.75
  emptyBlock: {}
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := structured.FromYAML(tc.yaml)
			if err != nil {
				t.Fatalf("failed to parse YAML for %s: %v", tc.name, err)
			}

			// 1. Run legacy baseline
			poolBaseline := NewTestInternPool(id.NewGenerator())
			baselineProto, baselineID, err := toInternedStruct_Legacy(node, poolBaseline)
			if err != nil {
				t.Fatalf("baseline toInternedStruct_Legacy failed: %v", err)
			}

			// 2. Run production ToInternedStruct
			poolGot := NewTestInternPool(id.NewGenerator())
			gotRef, err := ToInternedStruct(node, poolGot)
			if err != nil {
				t.Fatalf("production ToInternedStruct failed: %v", err)
			}
			gotProto := gotRef.Resolve()

			// 3. Verify struct ID equivalence
			if baselineID != gotRef.id {
				t.Errorf("struct ID mismatch: baseline=%d, got=%d", baselineID, gotRef.id)
			}

			// 4. Verify protobuf equivalence
			if diff := cmp.Diff(baselineProto, gotProto, protocmp.Transform()); diff != "" {
				t.Errorf("resolved proto mismatch (-baseline +got):\n%s", diff)
			}

			// 5. Test repeated interning on same pool (idempotent lookup)
			secondRef, err := ToInternedStruct(node, poolGot)
			if err != nil {
				t.Fatalf("repeated interning failed: %v", err)
			}
			if secondRef.id != gotRef.id {
				t.Errorf("repeated interning ID mismatch: first=%d, second=%d", gotRef.id, secondRef.id)
			}

			// 6. Verify FromInternedStruct round-trips correctly from resolved proto
			reconstructedBaseline, err := FromInternedStruct(baselineProto, poolBaseline)
			if err != nil {
				t.Fatalf("FromInternedStruct for baseline failed: %v", err)
			}
			reconstructedGot, err := FromInternedStruct(gotProto, poolGot)
			if err != nil {
				t.Fatalf("FromInternedStruct for got failed: %v", err)
			}

			// Serialize both reconstructed nodes back to YAML to verify structural equivalence
			serializer := &structured.YAMLNodeSerializer{}
			yamlBaseline, err := serializer.Serialize(reconstructedBaseline)
			if err != nil {
				t.Fatalf("Serialize for baseline failed: %v", err)
			}
			yamlGot, err := serializer.Serialize(reconstructedGot)
			if err != nil {
				t.Fatalf("Serialize for got failed: %v", err)
			}
			if diff := cmp.Diff(string(yamlBaseline), string(yamlGot)); diff != "" {
				t.Errorf("reconstructed YAML mismatch (-baseline +got):\n%s", diff)
			}
		})
	}
}
