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

package gcpcommon

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
)

func newTestNodeReader(t *testing.T, data map[string]any) *structured.NodeReader {
	t.Helper()
	node, err := structured.FromGoValue(data, &structured.AlphabeticalGoMapKeyOrderProvider{})
	if err != nil {
		t.Fatalf("failed to create structured node from map: %v", err)
	}
	return structured.NewNodeReader(node)
}

func TestExtractCAIAssetNameAndType(t *testing.T) {
	testCases := []struct {
		name      string
		inputData map[string]any
		wantName  string
		wantType  string
	}{
		{
			name: "extracts name and type when present",
			inputData: map[string]any{
				"asset": map[string]any{
					"name":      "//container.googleapis.com/projects/p/locations/l/clusters/c",
					"assetType": "container.googleapis.com/Cluster",
				},
			},
			wantName: "//container.googleapis.com/projects/p/locations/l/clusters/c",
			wantType: "container.googleapis.com/Cluster",
		},
		{
			name:      "returns empty strings when missing",
			inputData: map[string]any{},
			wantName:  "",
			wantType:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := newTestNodeReader(t, tc.inputData)
			gotName := ExtractCAIAssetName(reader)
			if gotName != tc.wantName {
				t.Errorf("ExtractCAIAssetName() = %q, want %q", gotName, tc.wantName)
			}
			gotType := ExtractCAIAssetType(reader)
			if gotType != tc.wantType {
				t.Errorf("ExtractCAIAssetType() = %q, want %q", gotType, tc.wantType)
			}
		})
	}
}

func TestExtractCAITimeWindow(t *testing.T) {
	testCases := []struct {
		name          string
		inputData     map[string]any
		wantStartTime time.Time
		wantEndTime   time.Time
		wantDeleted   bool
	}{
		{
			name: "extracts valid time window and deleted false",
			inputData: map[string]any{
				"window": map[string]any{
					"startTime": "2026-01-01T10:00:00Z",
					"endTime":   "2026-01-01T11:00:00Z",
				},
				"deleted": false,
			},
			wantStartTime: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			wantEndTime:   time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC),
			wantDeleted:   false,
		},
		{
			name: "extracts tombstone deleted true",
			inputData: map[string]any{
				"deleted": true,
			},
			wantStartTime: time.Time{},
			wantEndTime:   time.Time{},
			wantDeleted:   true,
		},
		{
			name:          "extracts empty map as zero times and not deleted",
			inputData:     map[string]any{},
			wantStartTime: time.Time{},
			wantEndTime:   time.Time{},
			wantDeleted:   false,
		},
		{
			name: "falls back to zero times when timestamp strings are malformed",
			inputData: map[string]any{
				"window": map[string]any{
					"startTime": "invalid-start-time",
					"endTime":   "invalid-end-time",
				},
				"deleted": false,
			},
			wantStartTime: time.Time{},
			wantEndTime:   time.Time{},
			wantDeleted:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := newTestNodeReader(t, tc.inputData)
			gotStart, gotEnd, gotDeleted := ExtractCAITimeWindow(reader)
			if !gotStart.Equal(tc.wantStartTime) {
				t.Errorf("ExtractCAITimeWindow() gotStart = %v, want %v", gotStart, tc.wantStartTime)
			}
			if !gotEnd.Equal(tc.wantEndTime) {
				t.Errorf("ExtractCAITimeWindow() gotEnd = %v, want %v", gotEnd, tc.wantEndTime)
			}
			if gotDeleted != tc.wantDeleted {
				t.Errorf("ExtractCAITimeWindow() gotDeleted = %v, want %v", gotDeleted, tc.wantDeleted)
			}
		})
	}
}

func TestIsCAIAssetActiveAt(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	testCases := []struct {
		name       string
		start      time.Time
		end        time.Time
		at         time.Time
		wantActive bool
	}{
		{
			name:       "active when within window",
			start:      baseTime.Add(-time.Hour),
			end:        baseTime.Add(time.Hour),
			at:         baseTime,
			wantActive: true,
		},
		{
			name:       "active when start equals at",
			start:      baseTime,
			end:        baseTime.Add(time.Hour),
			at:         baseTime,
			wantActive: true,
		},
		{
			name:       "inactive when end equals at",
			start:      baseTime.Add(-time.Hour),
			end:        baseTime,
			at:         baseTime,
			wantActive: false,
		},
		{
			name:       "inactive when start is after at",
			start:      baseTime.Add(time.Minute),
			end:        baseTime.Add(time.Hour),
			at:         baseTime,
			wantActive: false,
		},
		{
			name:       "active when start and end are zero",
			start:      time.Time{},
			end:        time.Time{},
			at:         baseTime,
			wantActive: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsCAIAssetActiveAt(tc.start, tc.end, tc.at)
			if got != tc.wantActive {
				t.Errorf("IsCAIAssetActiveAt() = %v, want %v", got, tc.wantActive)
			}
		})
	}
}

func TestExtractCAIResourceBody(t *testing.T) {
	testCases := []struct {
		name      string
		inputData map[string]any
		keyOrder  []string
		wantNil   bool
		wantKeys  []string
	}{
		{
			name: "returns node with keys reordered when keyOrder is provided",
			inputData: map[string]any{
				"asset": map[string]any{
					"resource": map[string]any{
						"data": map[string]any{
							"kind":       "Pod",
							"apiVersion": "v1",
							"metadata":   map[string]any{"name": "p1"},
						},
					},
				},
			},
			keyOrder: []string{"apiVersion", "kind"},
			wantNil:  false,
			wantKeys: []string{"apiVersion", "kind", "metadata"},
		},
		{
			name: "returns node without key order",
			inputData: map[string]any{
				"asset": map[string]any{
					"resource": map[string]any{
						"data": map[string]any{
							"name": "cluster-1",
						},
					},
				},
			},
			keyOrder: nil,
			wantNil:  false,
			wantKeys: []string{"name"},
		},
		{
			name: "returns nil when resource data is absent",
			inputData: map[string]any{
				"asset": map[string]any{},
			},
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := newTestNodeReader(t, tc.inputData)
			got := ExtractCAIResourceBody(reader, tc.keyOrder...)
			if (got == nil) != tc.wantNil {
				t.Fatalf("ExtractCAIResourceBody() nil mismatch: got %v, wantNil %v", got, tc.wantNil)
			}
			if !tc.wantNil {
				var gotKeys []string
				for k := range got.Children() {
					gotKeys = append(gotKeys, k.Key)
				}
				if diff := cmp.Diff(tc.wantKeys, gotKeys); diff != "" {
					t.Errorf("ExtractCAIResourceBody() keys mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
