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

package workbench

import (
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifile "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/google/go-cmp/cmp"
)

func TestWorkbench_Lifecycle(t *testing.T) {
	wb := NewWorkbench("user-session-1", "inspection-100")

	if got := wb.ID(); got != "user-session-1" {
		t.Errorf("ID() = %q, want %q", got, "user-session-1")
	}

	if got := wb.InspectionID(); got != "inspection-100" {
		t.Errorf("InspectionID() = %q, want %q", got, "inspection-100")
	}

	if wb.IsClosed() {
		t.Errorf("expected new workbench not to be closed")
	}

	// Close marks as closed
	wb.Close()
	if !wb.IsClosed() {
		t.Errorf("expected workbench to be closed after Close()")
	}
}

func TestWorkbench_ReadStructYAMLs(t *testing.T) {
	str1ID := uint32(1)
	str1Val := "message"
	str2ID := uint32(2)
	str2Val := "hello world"
	str3ID := uint32(3)
	str3Val := "second message"

	fs1ID := uint32(1)
	struct1ID := uint32(1)
	struct2ID := uint32(2)

	chunk := &khifilev6.InterningPoolChunk{
		Strings: []*khifilev6.InternString{
			{Id: &str1ID, Value: &str1Val},
			{Id: &str2ID, Value: &str2Val},
			{Id: &str3ID, Value: &str3Val},
		},
		FieldPathSets: []*khifilev6.InternFieldPathSet{
			{Id: &fs1ID, FieldPathStringIds: []uint32{str1ID}},
		},
		Structs: []*khifile.InternedStruct{
			{
				Id:             &struct1ID,
				FieldPathSetId: &fs1ID,
				Values: []*khifile.InternedValue{
					{Kind: &khifile.InternedValue_StringValue{StringValue: str2ID}},
				},
			},
			{
				Id:             &struct2ID,
				FieldPathSetId: &fs1ID,
				Values: []*khifile.InternedValue{
					{Kind: &khifile.InternedValue_StringValue{StringValue: str3ID}},
				},
			},
		},
	}

	testCases := []struct {
		name      string
		setupWb   func() *Workbench
		structIDs []uint32
		wantErrIs error
		wantYAMLs map[uint32]string
	}{
		{
			name: "successfully decodes multiple structs to YAML",
			setupWb: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				wb.internPool.IngestChunk(chunk)
				return wb
			},
			structIDs: []uint32{struct1ID, struct2ID},
			wantYAMLs: map[uint32]string{
				struct1ID: "message: hello world\n",
				struct2ID: "message: second message\n",
			},
		},
		{
			name: "successfully decodes struct with multi-chunk ingestion",
			setupWb: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				c1 := &khifilev6.InterningPoolChunk{Strings: chunk.Strings}
				c2 := &khifilev6.InterningPoolChunk{FieldPathSets: chunk.FieldPathSets}
				c3 := &khifilev6.InterningPoolChunk{Structs: chunk.Structs}
				wb.internPool.IngestChunk(c1)
				wb.internPool.IngestChunk(c2)
				wb.internPool.IngestChunk(c3)
				return wb
			},
			structIDs: []uint32{struct1ID},
			wantYAMLs: map[uint32]string{
				struct1ID: "message: hello world\n",
			},
		},
		{
			name: "skips non-existent struct ID gracefully",
			setupWb: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				wb.internPool.IngestChunk(chunk)
				return wb
			},
			structIDs: []uint32{struct1ID, 999, 0},
			wantYAMLs: map[uint32]string{
				struct1ID: "message: hello world\n",
			},
		},
		{
			name: "returns empty map when intern pool is empty",
			setupWb: func() *Workbench {
				return NewWorkbench("wb-1", "insp-1")
			},
			structIDs: []uint32{struct1ID},
			wantYAMLs: map[uint32]string{},
		},
		{
			name: "returns ErrWorkbenchClosed when workbench is closed",
			setupWb: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				wb.internPool.IngestChunk(chunk)
				wb.Close()
				return wb
			},
			structIDs: []uint32{struct1ID},
			wantErrIs: ErrWorkbenchClosed,
		},
		{
			name: "successfully decodes structs on-demand from intern pool when searchIndex is initialized",
			setupWb: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				wb.internPool.IngestChunk(chunk)
				wb.searchIndex = &SearchIndex{
					InternPool: wb.internPool,
				}
				return wb
			},
			structIDs: []uint32{struct1ID},
			wantYAMLs: map[uint32]string{
				struct1ID: "message: hello world\n",
			},
		},
		{
			name: "handles empty structIDs slice",
			setupWb: func() *Workbench {
				return NewWorkbench("wb-1", "insp-1")
			},
			structIDs: []uint32{},
			wantYAMLs: map[uint32]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := tc.setupWb()
			gotYAMLs, err := wb.ReadStructYAMLs(tc.structIDs)
			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Fatalf("ReadStructYAMLs(%v) error = %v, want %v", tc.structIDs, err, tc.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadStructYAMLs(%v) unexpected error = %v", tc.structIDs, err)
			}
			if diff := cmp.Diff(tc.wantYAMLs, gotYAMLs); diff != "" {
				t.Errorf("ReadStructYAMLs(%v) YAML mismatch (-want +got):\n%s", tc.structIDs, diff)
			}
		})
	}
}

func createSamplePodNode(i int) structured.Node {
	name := fmt.Sprintf("frontend-deployment-%d-xyz", i)
	ns := fmt.Sprintf("production-ns-%d", i%10)
	uid := fmt.Sprintf("uid-pod-%08d-abcd-ef01-2345", i)
	rev := fmt.Sprintf("%d", 100000+i*3)
	nodeName := fmt.Sprintf("gke-cluster-node-pool-1-%04d", i%50)
	podIP := fmt.Sprintf("10.244.%d.%d", (i/250)%256, i%250)

	val := map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"name":              name,
			"namespace":         ns,
			"uid":               uid,
			"resourceVersion":   rev,
			"creationTimestamp": "2026-04-20T03:00:00Z",
			"labels": map[string]any{
				"app":     "khi-frontend",
				"tier":    "web",
				"env":     "production",
				"version": "v1.25.0",
			},
			"annotations": map[string]any{
				"kubectl.kubernetes.io/last-applied-configuration": fmt.Sprintf(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"%s","namespace":"%s"}}`, name, ns),
				"prometheus.io/scrape":                             "true",
				"prometheus.io/port":                               "8080",
			},
		},
		"spec": map[string]any{
			"nodeName":           nodeName,
			"restartPolicy":      "Always",
			"serviceAccountName": "default",
			"containers": []any{
				map[string]any{
					"name":  "web-server",
					"image": "gcr.io/khi-demo/web:v1.25.0@sha256:abcdef0123456789",
					"ports": []any{
						map[string]any{
							"containerPort": 8080,
							"protocol":      "TCP",
						},
					},
					"resources": map[string]any{
						"requests": map[string]any{
							"cpu":    "250m",
							"memory": "512Mi",
						},
						"limits": map[string]any{
							"cpu":    "1000m",
							"memory": "1024Mi",
						},
					},
				},
			},
		},
		"status": map[string]any{
			"phase":     "Running",
			"hostIP":    "10.128.0.15",
			"podIP":     podIP,
			"startTime": "2026-04-20T03:00:05Z",
			"conditions": []any{
				map[string]any{
					"type":               "PodScheduled",
					"status":             "True",
					"lastTransitionTime": "2026-04-20T03:00:02Z",
				},
				map[string]any{
					"type":               "Ready",
					"status":             "True",
					"lastTransitionTime": "2026-04-20T03:00:10Z",
				},
			},
		},
	}

	node, _ := structured.FromGoValue(val, &structured.AlphabeticalGoMapKeyOrderProvider{})
	return node
}

func toReadonlyPool(pool *khifilev6model.InternPool) *khifilev6model.ReadonlyInternPool {
	readonlyPool := khifilev6model.NewReadonlyInternPool()
	var strList []*khifilev6.InternString
	for sRef := range pool.SortedStringRefs() {
		strList = append(strList, sRef.ToProto())
	}
	var fsList []*khifilev6.InternFieldPathSet
	for fsRef := range pool.FieldSetRefs() {
		fsList = append(fsList, fsRef.ToProto())
	}
	var structList []*khifile.InternedStruct
	for sRef := range pool.StructRefs() {
		structList = append(structList, sRef.ToProto())
	}
	readonlyPool.IngestChunk(&khifilev6.InterningPoolChunk{
		Strings:       strList,
		FieldPathSets: fsList,
		Structs:       structList,
	})
	return readonlyPool
}

func BenchmarkMemory_InternPoolVsStructYAML(b *testing.B) {
	const numStructs = 5000

	// 0. Base memory before anything
	runtime.GC()
	var mBase runtime.MemStats
	runtime.ReadMemStats(&mBase)

	// 1. Build InternPool with 5,000 realistic unique Pod structs
	idGen := khifilev6model.NewIDGenerator()
	pool := khifilev6model.NewInternPool(idGen)
	structIDs := make([]uint32, numStructs)
	for i := 0; i < numStructs; i++ {
		node := createSamplePodNode(i)
		ref, err := khifilev6model.ToInternedStruct(node, pool)
		if err != nil {
			b.Fatalf("ToInternedStruct error = %v", err)
		}
		structIDs[i] = ref.ID()
	}
	readonlyPool := toReadonlyPool(pool)

	// 2. Measure InternPool memory
	runtime.GC()
	var mIntern runtime.MemStats
	runtime.ReadMemStats(&mIntern)

	// 3. Serialize all structs to YAML strings map (StructYAMLs)
	serializer := khifilev6model.NewDirectYAMLSerializer()
	structYAMLs := make(map[uint32]string, numStructs)
	for _, id := range structIDs {
		yamlStr, err := serializer.SerializeFlatStruct(id, readonlyPool)
		if err != nil {
			b.Fatalf("SerializeFlatStruct error = %v", err)
		}
		structYAMLs[id] = yamlStr
	}

	runtime.GC()
	var mYAML runtime.MemStats
	runtime.ReadMemStats(&mYAML)

	totalYAMLBytes := 0
	for _, s := range structYAMLs {
		totalYAMLBytes += len(s)
	}

	internPoolDeltaMB := float64(mIntern.HeapAlloc-mBase.HeapAlloc) / (1024 * 1024)
	structYAMLDeltaMB := float64(mYAML.HeapAlloc-mIntern.HeapAlloc) / (1024 * 1024)

	b.Logf("=== 5,000 Realistic Pod Manifests Memory Measurement ===")
	b.Logf("Average YAML size per struct: %d bytes (Total YAML text raw: %.2f MB)",
		totalYAMLBytes/numStructs, float64(totalYAMLBytes)/(1024*1024))
	b.Logf("InternPool Memory in Heap: %.2f MB", internPoolDeltaMB)
	b.Logf("StructYAMLs Memory in Heap: %.2f MB (%.2fx larger than InternPool)",
		structYAMLDeltaMB, structYAMLDeltaMB/internPoolDeltaMB)
	b.Logf("Total Heap with both: %.2f MB", float64(mYAML.HeapAlloc)/(1024*1024))

	b.Run("Access_PreSerializedMap", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := structIDs[i%numStructs]
			_ = structYAMLs[id]
		}
	})

	b.Run("Access_OnDemandSerialization", func(b *testing.B) {
		s := khifilev6model.NewDirectYAMLSerializer()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := structIDs[i%numStructs]
			_, _ = s.SerializeFlatStruct(id, readonlyPool)
		}
	})
}
