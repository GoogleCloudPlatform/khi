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

package cel

import (
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestTrigramIndex(t *testing.T) {
	const (
		structID1 = 1
		structID2 = 2
	)

	yaml1 := `metadata:
  name: pod-a
  namespace: default
spec:
  containers:
  - name: nginx
    image: nginx:latest
`
	yaml2 := `metadata:
  name: pod-b
  namespace: kube-system
spec:
  containers:
  - name: coredns
    image: coredns:v1.9
`
	structYAMLs := map[uint32]string{
		structID1: yaml1,
		structID2: yaml2,
	}

	idx := NewTrigramIndex()
	if err := idx.BuildFromStructYAMLs(t.Context(), structYAMLs, nil); err != nil {
		t.Fatalf("BuildFromStructYAMLs() error = %v", err)
	}

	testCases := []struct {
		name            string
		query           string
		wantCandidates  []uint32
		isUnconstrained bool
	}{
		{
			name:           "match literal regex >= 3 chars",
			query:          "nginx",
			wantCandidates: []uint32{structID1},
		},
		{
			name:           "match case-insensitive regex flag",
			query:          "(?i)COREDNS",
			wantCandidates: []uint32{structID2},
		},
		{
			name:           "match regex with wildcard and anchors",
			query:          "(?s)^.*pod-a.*$",
			wantCandidates: []uint32{structID1},
		},
		{
			name:           "match regex matching common pattern in both structs",
			query:          "namespace:\\s+.*",
			wantCandidates: []uint32{structID1, structID2},
		},
		{
			name:           "match regex with alternative literals",
			query:          "nginx|coredns",
			wantCandidates: []uint32{structID1, structID2},
		},
		{
			name:           "match regex with alternative literals where one does not match",
			query:          "nginx|redis",
			wantCandidates: []uint32{structID1},
		},
		{
			name:           "match regex with alternative literals and suffix concatenation",
			query:          "(?s)containers:.*(nginx|coredns)",
			wantCandidates: []uint32{structID1, structID2},
		},
		{
			name:           "match regex with alternative literals and specific suffix constraint",
			query:          "(nginx|coredns).*latest",
			wantCandidates: []uint32{structID1},
		},
		{
			name:            "match regex with unconstrained alternative branch (fallback to all structs)",
			query:           "nginx|.*",
			isUnconstrained: true,
		},
		{
			name:            "match regex with short literal (<3 chars) in alternative branch (fallback to all structs)",
			query:           "nginx|v1",
			isUnconstrained: true,
		},
		{
			name:           "match regex with alternative literals containing 3-char prefix",
			query:          "nginx|v1\\.[0-9]",
			wantCandidates: []uint32{structID1, structID2},
		},
		{
			name:           "match regex with 3-char literal prefix before character class",
			query:          "v1\\.[0-9]",
			wantCandidates: []uint32{structID2},
		},
		{
			name:            "match regex with <3 chars without trigram (fallback to all scan)",
			query:           "v1",
			isUnconstrained: true,
		},
		{
			name:           "non-matching regex",
			query:          "redis",
			wantCandidates: []uint32{},
		},
		{
			name:            "match all with dot star",
			query:           ".*",
			isUnconstrained: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotBitmap := idx.FindCandidateStructs(tc.query)
			if tc.isUnconstrained {
				if gotBitmap != nil {
					t.Errorf("FindCandidateStructs(%q) expected nil (unconstrained), got %v", tc.query, gotBitmap.ToArray())
				}
				return
			}

			var got []uint32
			if gotBitmap != nil {
				got = gotBitmap.ToArray()
			}
			sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
			want := make([]uint32, len(tc.wantCandidates))
			copy(want, tc.wantCandidates)
			sort.Slice(want, func(i, j int) bool { return want[i] < want[j] })

			if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("FindCandidateStructs(%q) mismatch (-want +got):\n%s", tc.query, diff)
			}
		})
	}
}

func TestTrigramIndex_Unicode(t *testing.T) {
	const (
		structID1 = 1
		structID2 = 2
		structID3 = 3
	)

	yaml1 := `metadata:
  name: pod-ascii
  namespace: default
spec:
  containers:
  - name: web
`
	yaml2 := `metadata:
  name: pod-unicode
  namespace: default
  annotations:
    description: データベース接続エラーが発生しました
spec:
  containers:
  - name: db
`
	yaml3 := `metadata:
  name: pod-mixed
  namespace: default
  annotations:
    note: クラスターメンテナンス中 (cluster maintenance)
`
	structYAMLs := map[uint32]string{
		structID1: yaml1,
		structID2: yaml2,
		structID3: yaml3,
	}

	idx := NewTrigramIndex()
	if err := idx.BuildFromStructYAMLs(t.Context(), structYAMLs, nil); err != nil {
		t.Fatalf("BuildFromStructYAMLs() error = %v", err)
	}

	testCases := []struct {
		name            string
		query           string
		wantCandidates  []uint32
		isUnconstrained bool
	}{
		{
			name:           "match exact unicode multi-rune literal",
			query:          "データベース",
			wantCandidates: []uint32{structID2},
		},
		{
			name:           "match unicode multi-rune substring",
			query:          "接続エラー",
			wantCandidates: []uint32{structID2},
		},
		{
			name:           "match unicode substring in different struct",
			query:          "クラスター",
			wantCandidates: []uint32{structID3},
		},
		{
			name:           "match mixed unicode and ascii in query",
			query:          "メンテナンス中",
			wantCandidates: []uint32{structID3},
		},
		{
			name:           "match alternation of unicode literals",
			query:          "データベース|クラスター",
			wantCandidates: []uint32{structID2, structID3},
		},
		{
			name:           "match alternation of ascii and unicode literals",
			query:          "pod-ascii|接続エラー",
			wantCandidates: []uint32{structID1, structID2},
		},
		{
			name:            "unicode query shorter than 3 runes falls back to unconstrained",
			query:           "バグ",
			isUnconstrained: true,
		},
		{
			name:            "alternation with short unicode branch falls back to unconstrained",
			query:           "データベース|バグ",
			isUnconstrained: true,
		},
		{
			name:           "non-matching unicode term returns empty candidates",
			query:          "認証エラー",
			wantCandidates: []uint32{},
		},
		{
			name:           "ascii query against index containing unicode structs",
			query:          "maintenance",
			wantCandidates: []uint32{structID3},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotBitmap := idx.FindCandidateStructs(tc.query)
			if tc.isUnconstrained {
				if gotBitmap != nil {
					t.Errorf("FindCandidateStructs(%q) expected nil (unconstrained), got %v", tc.query, gotBitmap.ToArray())
				}
				return
			}

			var got []uint32
			if gotBitmap != nil {
				got = gotBitmap.ToArray()
			}
			sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
			want := make([]uint32, len(tc.wantCandidates))
			copy(want, tc.wantCandidates)
			sort.Slice(want, func(i, j int) bool { return want[i] < want[j] })

			if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("FindCandidateStructs(%q) mismatch (-want +got):\n%s", tc.query, diff)
			}
		})
	}
}

func TestTrigramIndex_ProgressCallback(t *testing.T) {
	structYAMLs := map[uint32]string{
		1: "foo: bar",
	}

	var reports []float64
	callback := func(progressPercentage float64, message string) error {
		reports = append(reports, progressPercentage)
		return nil
	}

	idx := NewTrigramIndex()
	if err := idx.BuildFromStructYAMLs(t.Context(), structYAMLs, callback); err != nil {
		t.Fatalf("BuildFromStructYAMLs() error = %v", err)
	}

	if len(reports) == 0 {
		t.Errorf("expected progress reports, got none")
	}
}

func TestTrigramIndex_ConcurrentQuery(t *testing.T) {
	const structID = 42
	structYAMLs := map[uint32]string{
		structID: "foo: bar_baz_qux",
	}

	idx := NewTrigramIndex()
	if err := idx.BuildFromStructYAMLs(t.Context(), structYAMLs, nil); err != nil {
		t.Fatalf("BuildFromStructYAMLs() error = %v", err)
	}

	const numGoroutines = 20
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				bm := idx.FindCandidateStructs("bar_baz")
				if bm == nil || !bm.Contains(structID) {
					t.Errorf("expected candidate to contain struct %d", structID)
				}
			}
		}()
	}

	wg.Wait()
}

func BenchmarkBuildFromStructYAMLs(b *testing.B) {
	structYAMLs := make(map[uint32]string, 1000)
	for i := uint32(1); i <= 1000; i++ {
		structYAMLs[i] = `metadata:
  name: pod-sample-` + string(rune('a'+i%26)) + `
  namespace: default
spec:
  containers:
  - name: nginx-` + string(rune('a'+i%26)) + `
    image: nginx:latest
`
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		idx := NewTrigramIndex()
		_ = idx.BuildFromStructYAMLs(context.Background(), structYAMLs, nil)
	}
}
