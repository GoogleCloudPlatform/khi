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
	"bytes"
	"context"
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
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
			gotBitmap := idx.FindCandidateLogs(tc.query)
			if tc.isUnconstrained {
				if gotBitmap != nil {
					t.Errorf("FindCandidateLogs(%q) expected nil (unconstrained), got %v", tc.query, gotBitmap.ToArray())
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
				t.Errorf("FindCandidateLogs(%q) mismatch (-want +got):\n%s", tc.query, diff)
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
			gotBitmap := idx.FindCandidateLogs(tc.query)
			if tc.isUnconstrained {
				if gotBitmap != nil {
					t.Errorf("FindCandidateLogs(%q) expected nil (unconstrained), got %v", tc.query, gotBitmap.ToArray())
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
				t.Errorf("FindCandidateLogs(%q) mismatch (-want +got):\n%s", tc.query, diff)
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
				bm := idx.FindCandidateLogs("bar_baz")
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

func TestTrigramIndex_WriteToAndReadFrom(t *testing.T) {
	testCases := []struct {
		name        string
		structYAMLs map[uint32]string
		testQueries []string
	}{
		{
			name:        "empty index round-trip",
			structYAMLs: map[uint32]string{},
			testQueries: []string{"foo", "bar"},
		},
		{
			name: "single struct index round-trip",
			structYAMLs: map[uint32]string{
				1: "metadata:\n  name: pod-sample\n",
			},
			testQueries: []string{"pod", "sample", "nonexistent"},
		},
		{
			name: "multiple structs with diverse UTF-8 content round-trip",
			structYAMLs: map[uint32]string{
				10: "kind: Deployment\nmetadata:\n  name: web-server\n",
				20: "kind: Service\nmetadata:\n  name: web-service\n",
				30: "kind: ConfigMap\nmetadata:\n  name: 設定マップ\n",
			},
			testQueries: []string{"Deployment", "web-server", "web-service", "設定マップ", "xyz"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			orig := NewTrigramIndex()
			if err := orig.BuildFromStructYAMLs(t.Context(), tc.structYAMLs, nil); err != nil {
				t.Fatalf("failed to build original index: %v", err)
			}

			var buf bytes.Buffer
			writtenBytes, err := orig.WriteTo(&buf)
			if err != nil {
				t.Fatalf("WriteTo() failed: %v", err)
			}
			if writtenBytes != int64(buf.Len()) {
				t.Errorf("WriteTo() returned byte count %d, actual buffer length %d", writtenBytes, buf.Len())
			}

			restored := NewTrigramIndex()
			readBytes, err := restored.ReadFrom(&buf)
			if err != nil {
				t.Fatalf("ReadFrom() failed: %v", err)
			}
			if readBytes != writtenBytes {
				t.Errorf("ReadFrom() read %d bytes, expected %d", readBytes, writtenBytes)
			}

			// Verify each test query returns identical candidate IDs
			for _, query := range tc.testQueries {
				wantBM := orig.FindCandidateLogs(query)
				gotBM := restored.FindCandidateLogs(query)

				var wantSlice, gotSlice []uint32
				if wantBM != nil {
					wantSlice = wantBM.ToArray()
				}
				if gotBM != nil {
					gotSlice = gotBM.ToArray()
				}
				if diff := cmp.Diff(wantSlice, gotSlice, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("FindCandidateLogs(%q) mismatch (-want +got):\n%s", query, diff)
				}
			}
		})
	}

	t.Run("invalid header returns ErrInvalidTrigramHeader", func(t *testing.T) {
		invalidData := []byte("INVALID_HEADER_DATA")
		idx := NewTrigramIndex()
		_, err := idx.ReadFrom(bytes.NewReader(invalidData))
		if !errors.Is(err, ErrInvalidTrigramHeader) {
			t.Errorf("ReadFrom() error = %v, want %v", err, ErrInvalidTrigramHeader)
		}
	})
}

func TestTrigramIndex_BuildFromStructPool(t *testing.T) {
	testCases := []struct {
		name        string
		yamls       map[uint32]string
		testQueries []string
	}{
		{
			name: "streaming build matches YAML build candidates",
			yamls: map[uint32]string{
				1: "kind: Pod\nmetadata:\n  name: pod-nginx\n",
				2: "kind: Pod\nmetadata:\n  name: pod-coredns\n",
				3: "kind: Service\nmetadata:\n  name: svc-nginx\n",
			},
			testQueries: []string{"nginx", "coredns", "Service", "missing"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := khifilev6model.NewInternPool(&khifilev6model.IDGenerator{})
			structIDs := make([]uint32, 0, len(tc.yamls))
			for _, yamlStr := range tc.yamls {
				node, err := structured.FromYAML(yamlStr)
				if err != nil {
					t.Fatalf("failed to parse YAML: %v", err)
				}
				sRef, err := khifilev6model.ToInternedStruct(node, pool)
				if err != nil {
					t.Fatalf("failed to intern struct: %v", err)
				}
				structIDs = append(structIDs, sRef.ID())
			}

			idxFromPool := NewTrigramIndex()
			if err := idxFromPool.BuildFromStructPool(pool, structIDs, nil); err != nil {
				t.Fatalf("BuildFromStructPool() failed: %v", err)
			}

			// Verify query candidates are correctly found
			for _, query := range tc.testQueries {
				gotBM := idxFromPool.FindCandidateLogs(query)
				if query == "missing" {
					if gotBM != nil && !gotBM.IsEmpty() {
						t.Errorf("expected empty candidate for %q, got %v", query, gotBM.ToArray())
					}
				} else {
					if gotBM == nil || gotBM.IsEmpty() {
						t.Errorf("expected non-empty candidate for %q, got %v", query, gotBM)
					}
				}
			}
		})
	}
}

func TestTrigramIndex_BuildFromLogPool(t *testing.T) {
	testCases := []struct {
		name        string
		summaries   map[uint32]string
		yamls       map[uint32]string
		logs        []LogTrigramItem
		testQueries map[string][]uint32 // query -> expected candidate log IDs
	}{
		{
			name: "matches logs via summary and via body struct",
			summaries: map[uint32]string{
				10: "Pod sandbox changed successfully",
				20: "設定マップの更新完了",
			},
			yamls: map[uint32]string{
				100: "apiVersion: v1\nkind: Pod\nmetadata:\n  name: nginx-server",
				200: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app-config\ndata:\n  specialKey: rivqt0dytnq",
			},
			logs: []LogTrigramItem{
				{ID: 1, SummaryStringID: 10, BodyStructID: 100}, // summary: sandbox, body: nginx
				{ID: 2, SummaryStringID: 20, BodyStructID: 200}, // summary: 設定マップ, body: rivqt0dytnq
				{ID: 3, SummaryStringID: 10, BodyStructID: 200}, // summary: sandbox, body: rivqt0dytnq
			},
			testQueries: map[string][]uint32{
				"nginx":       {1},
				"sandbox":     {1, 3},
				"rivqt0dytnq": {2, 3},
				"設定マップ":       {2},
				"nonexistent": {},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := khifilev6model.NewInternPool(&khifilev6model.IDGenerator{})

			// Intern strings
			summaryIDMap := make(map[uint32]uint32)
			for origID, str := range tc.summaries {
				sRef := pool.InternString(str)
				summaryIDMap[origID] = sRef.ToProto().GetId()
			}

			// Intern structs
			structIDMap := make(map[uint32]uint32)
			for origID, y := range tc.yamls {
				node, err := structured.FromYAML(y)
				if err != nil {
					t.Fatalf("failed to parse YAML: %v", err)
				}
				sRef, err := khifilev6model.ToInternedStruct(node, pool)
				if err != nil {
					t.Fatalf("failed to intern struct: %v", err)
				}
				structIDMap[origID] = sRef.ID()
			}

			// Map logs with interned IDs
			logs := make([]LogTrigramItem, len(tc.logs))
			for i, l := range tc.logs {
				logs[i] = LogTrigramItem{
					ID:              l.ID,
					SummaryStringID: summaryIDMap[l.SummaryStringID],
					BodyStructID:    structIDMap[l.BodyStructID],
				}
			}

			idx := NewTrigramIndex()
			if err := idx.BuildFromLogPool(pool, logs, nil); err != nil {
				t.Fatalf("BuildFromLogPool() failed: %v", err)
			}

			for query, wantLogIDs := range tc.testQueries {
				gotBM := idx.FindCandidateLogs(query)
				var gotLogIDs []uint32
				if gotBM != nil {
					gotLogIDs = gotBM.ToArray()
				}
				if diff := cmp.Diff(wantLogIDs, gotLogIDs, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("FindCandidateLogs(%q) mismatch (-want +got):\n%s", query, diff)
				}
			}
		})
	}
}

func TestPathToTrigramQuery(t *testing.T) {
	testCases := []struct {
		name      string
		pathKey   string
		wantType  string // "AllQuery", "TermQuery", "AndQuery"
		wantTerms []string
	}{
		{
			name:     "wildcard path",
			pathKey:  "*",
			wantType: "AllQuery",
		},
		{
			name:     "empty path",
			pathKey:  "",
			wantType: "AllQuery",
		},
		{
			name:      "2-letter field name gets colon to form 3 runes",
			pathKey:   "ip",
			wantType:  "TermQuery",
			wantTerms: []string{"ip:"},
		},
		{
			name:      "single field name",
			pathKey:   "status",
			wantType:  "AndQuery",
			wantTerms: []string{"sta", "tat", "atu", "tus", "us:"},
		},
		{
			name:     "multi segment field path",
			pathKey:  "spec.containers.image",
			wantType: "AndQuery",
			wantTerms: []string{
				// spec:
				"spe", "pec", "ec:",
				// containers:
				"con", "ont", "nta", "tai", "ain", "ine", "ner", "ers", "rs:",
				// image:
				"ima", "mag", "age", "ge:",
			},
		},
		{
			name:     "escaped dot in field name",
			pathKey:  "labels.app\\.kubernetes\\.io/name",
			wantType: "AndQuery",
			wantTerms: []string{
				// labels:
				"lab", "abe", "bel", "els", "ls:",
				// app.kubernetes.io/name:
				"app", "pp.", "p.k", ".ku", "kub", "ube", "ber", "ern", "rne", "net", "ete", "tes", "es.", "s.i", ".io", "io/", "o/n", "/na", "nam", "ame", "me:",
			},
		},
		{
			name:     "quoted special key",
			pathKey:  "@type",
			wantType: "AndQuery",
			wantTerms: []string{
				// "@type":
				"\"@t", "@ty", "typ", "ype", "pe\"", "e\":",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := PathToTrigramQuery(tc.pathKey)
			switch tc.wantType {
			case "AllQuery":
				if _, ok := got.(*AllQuery); !ok {
					t.Fatalf("PathToTrigramQuery(%q) expected *AllQuery, got %T (%v)", tc.pathKey, got, got)
				}
			case "TermQuery":
				term, ok := got.(*TermQuery)
				if !ok {
					t.Fatalf("PathToTrigramQuery(%q) expected *TermQuery, got %T (%v)", tc.pathKey, got, got)
				}
				if diff := cmp.Diff(tc.wantTerms[0], term.Term); diff != "" {
					t.Errorf("PathToTrigramQuery(%q) term mismatch (-want +got):\n%s", tc.pathKey, diff)
				}
			case "AndQuery":
				andQ, ok := got.(*AndQuery)
				if !ok {
					t.Fatalf("PathToTrigramQuery(%q) expected *AndQuery, got %T (%v)", tc.pathKey, got, got)
				}
				var gotTerms []string
				for _, child := range andQ.Children {
					if tQ, ok := child.(*TermQuery); ok {
						gotTerms = append(gotTerms, tQ.Term)
					}
				}
				sort.Strings(gotTerms)
				sort.Strings(tc.wantTerms)
				if diff := cmp.Diff(tc.wantTerms, gotTerms, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("PathToTrigramQuery(%q) terms mismatch (-want +got):\n%s", tc.pathKey, diff)
				}
			}
		})
	}
}

func TestFindCandidateLogsWithField(t *testing.T) {
	const (
		idPodNginx = 1
		idPodCore  = 2
		idEventMsg = 3
	)

	structYAMLs := map[uint32]string{
		idPodNginx: `metadata:
  name: pod-a
spec:
  containers:
  - name: nginx
    image: nginx:latest
`,
		idPodCore: `metadata:
  name: pod-b
spec:
  containers:
  - name: coredns
    image: coredns:v1.9
status:
  phase: Running
`,
		idEventMsg: `metadata:
  name: event-1
message: "0/1 nodes available: nginx container cannot be scheduled"
reason: FailedScheduling
`,
	}

	idx := NewTrigramIndex()
	if err := idx.BuildFromStructYAMLs(t.Context(), structYAMLs, nil); err != nil {
		t.Fatalf("BuildFromStructYAMLs() failed: %v", err)
	}

	testCases := []struct {
		name       string
		pathKey    string
		pattern    string
		wantLogIDs []uint32 // nil means unconstrained
	}{
		{
			name:       "wildcard pathKey with pattern matches all structs containing keyword",
			pathKey:    "*",
			pattern:    "nginx",
			wantLogIDs: []uint32{idPodNginx, idEventMsg},
		},
		{
			name:       "field pathKey prunes struct that only has keyword in unmatching field",
			pathKey:    "spec.containers.image",
			pattern:    "nginx",
			wantLogIDs: []uint32{idPodNginx},
		},
		{
			name:       "unconstrained pattern with field pathKey filters by field segments",
			pathKey:    "spec.containers.image",
			pattern:    ".*",
			wantLogIDs: []uint32{idPodNginx, idPodCore},
		},
		{
			name:       "status.phase with unconstrained pattern matches only pod-b",
			pathKey:    "status.phase",
			pattern:    ".*",
			wantLogIDs: []uint32{idPodCore},
		},
		{
			name:       "status.phase with Running matches pod-b",
			pathKey:    "status.phase",
			pattern:    "Running",
			wantLogIDs: []uint32{idPodCore},
		},
		{
			name:       "status.phase with non-matching pattern returns empty",
			pathKey:    "status.phase",
			pattern:    "Failed",
			wantLogIDs: []uint32{},
		},
		{
			name:       "non-existent field path returns empty bitmap",
			pathKey:    "nonexistent.field",
			pattern:    "nginx",
			wantLogIDs: []uint32{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotBM := idx.FindCandidateLogsWithField(tc.pathKey, tc.pattern)
			var gotLogIDs []uint32
			if gotBM != nil {
				gotLogIDs = gotBM.ToArray()
			}
			if diff := cmp.Diff(tc.wantLogIDs, gotLogIDs, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("FindCandidateLogsWithField(%q, %q) mismatch (-want +got):\n%s", tc.pathKey, tc.pattern, diff)
			}
		})
	}
}
