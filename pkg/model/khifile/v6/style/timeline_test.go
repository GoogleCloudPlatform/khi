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

package style

import (
	"fmt"
	"sync"
	"testing"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestRegisterTimelineType(t *testing.T) {
	reset()

	res1 := MustRegisterTimelineType("Type 1", "Desc 1", "icon-1", 1.0, Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, Color{0.5, 0.5, 0.5, 1}, Color{0, 0, 0, 1}, true, 1, nil)
	res2 := MustRegisterTimelineType("Type 2", "Desc 2", "icon-2", 1.0, Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, Color{0.5, 0.5, 0.5, 1}, Color{0, 0, 0, 1}, true, 2, nil)

	// Verify IDs were assigned starting from 1
	if res1.Id == nil || *res1.Id != 1 {
		t.Errorf("expected ID 1, got %v", res1.Id)
	}
	if res2.Id == nil || *res2.Id != 2 {
		t.Errorf("expected ID 2, got %v", res2.Id)
	}

	chunk := GenerateChunk()
	if len(chunk.TimelineTypes) != 2 {
		t.Fatalf("expected 2 TimelineTypes in chunk, got %d", len(chunk.TimelineTypes))
	}
	if *chunk.TimelineTypes[0].Id != 1 {
		t.Errorf("expected first chunk TimelineType ID to be 1")
	}
}

func TestRegisterConcurrent(t *testing.T) {
	reset()

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			MustRegisterVerb("Concurrent Verb", Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, true)
		}(i)
	}

	wg.Wait()

	chunk := GenerateChunk()
	if len(chunk.Verbs) != numGoroutines {
		t.Fatalf("expected %d Verbs Mustregistered, got %d", numGoroutines, len(chunk.Verbs))
	}

	// Verify all IDs from 1 to numGoroutines exist exactly once
	idMap := make(map[uint32]bool)
	for _, v := range chunk.Verbs {
		if v.Id == nil {
			t.Fatalf("verb ID is nil")
		}
		if idMap[*v.Id] {
			t.Errorf("duplicate ID found: %d", *v.Id)
		}
		idMap[*v.Id] = true
	}

	if len(idMap) != numGoroutines {
		t.Errorf("expected %d unique IDs, got %d", numGoroutines, len(idMap))
	}
}

func TestGenerateChunkHasAllSlices(t *testing.T) {
	reset()

	MustRegisterSeverity("Sev", "S", Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, 1)
	MustRegisterVerb("Verb", Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, true)
	MustRegisterLogType("Log", "Desc", Color{1, 1, 1, 1}, Color{0, 0, 0, 1})
	MustRegisterRevisionState("RevState", "icon", "Desc", Color{1, 1, 1, 1}, pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL)
	MustRegisterTimelineType("Timeline", "Desc", "icon", 1.0, Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, Color{0.5, 0.5, 0.5, 1}, Color{0, 0, 0, 1}, true, 1, nil)

	chunk := GenerateChunk()

	if len(chunk.Severities) != 1 {
		t.Errorf("expected 1 Severity, got %d", len(chunk.Severities))
	}
	if len(chunk.Verbs) != 1 {
		t.Errorf("expected 1 Verb, got %d", len(chunk.Verbs))
	}
	if len(chunk.LogTypes) != 1 {
		t.Errorf("expected 1 LogType, got %d", len(chunk.LogTypes))
	}
	if len(chunk.RevisionStates) != 1 {
		t.Errorf("expected 1 RevisionState, got %d", len(chunk.RevisionStates))
	}
	if len(chunk.TimelineTypes) != 1 {
		t.Errorf("expected 1 TimelineType, got %d", len(chunk.TimelineTypes))
	}
}

func TestGenerateChunkHasIconAtlas(t *testing.T) {
	reset()

	chunk := GenerateChunk()
	if chunk.IconAtlas == nil {
		t.Fatal("expected IconAtlas not to be nil in TimelineStyleChunk")
	}

	if len(chunk.IconAtlas.MsdfIconImage) == 0 || len(chunk.IconAtlas.MsdfIconImage[0]) == 0 {
		t.Error("expected non-empty embedded PNG bytes in MsdfIconImage")
	}

	if len(chunk.IconAtlas.BmfontJson) == 0 {
		t.Error("expected non-empty embedded BMFont configuration JSON")
	}

	if len(chunk.IconAtlas.NameToCodepoints) == 0 {
		t.Error("expected populated mapping in NameToCodepoints")
	}
}

func TestColorVerify(t *testing.T) {
	testCases := []struct {
		name    string
		color   Color
		wantErr bool
	}{
		{
			name:    "valid color",
			color:   Color{R: 0.5, G: 0.5, B: 0.5, A: 1.0},
			wantErr: false,
		},
		{
			name:    "invalid R (negative)",
			color:   Color{R: -0.1, G: 0.5, B: 0.5, A: 1.0},
			wantErr: true,
		},
		{
			name:    "invalid R (too high)",
			color:   Color{R: 1.1, G: 0.5, B: 0.5, A: 1.0},
			wantErr: true,
		},
		{
			name:    "invalid G (negative)",
			color:   Color{R: 0.5, G: -0.1, B: 0.5, A: 1.0},
			wantErr: true,
		},
		{
			name:    "invalid G (too high)",
			color:   Color{R: 0.5, G: 1.1, B: 0.5, A: 1.0},
			wantErr: true,
		},
		{
			name:    "invalid B (negative)",
			color:   Color{R: 0.5, G: 0.5, B: -0.1, A: 1.0},
			wantErr: true,
		},
		{
			name:    "invalid B (too high)",
			color:   Color{R: 0.5, G: 0.5, B: 1.1, A: 1.0},
			wantErr: true,
		},
		{
			name:    "invalid A (negative)",
			color:   Color{R: 0.5, G: 0.5, B: 0.5, A: -0.1},
			wantErr: true,
		},
		{
			name:    "invalid A (too high)",
			color:   Color{R: 0.5, G: 0.5, B: 0.5, A: 1.1},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.color.Verify()
			if (err != nil) != tc.wantErr {
				t.Errorf("Color.Verify() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestLockRegistry(t *testing.T) {
	testCases := []struct {
		name       string
		label      string
		styleClass string
		fn         func()
	}{
		{
			name:       "RegisterTimelineType",
			label:      "T",
			styleClass: "timeline type",
			fn: func() {
				MustRegisterTimelineType("T", "D", "icon", 1, Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, Color{0, 0, 0, 1}, Color{0, 0, 0, 1}, true, 10, nil)
			},
		},
		{
			name:       "RegisterSeverity",
			label:      "S",
			styleClass: "severity",
			fn: func() {
				MustRegisterSeverity("S", "S", Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, 1)
			},
		},
		{
			name:       "RegisterVerb",
			label:      "V",
			styleClass: "verb",
			fn: func() {
				MustRegisterVerb("V", Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, true)
			},
		},
		{
			name:       "RegisterLogType",
			label:      "L",
			styleClass: "log type",
			fn: func() {
				MustRegisterLogType("L", "D", Color{1, 1, 1, 1}, Color{0, 0, 0, 1})
			},
		},
		{
			name:       "RegisterRevisionState",
			label:      "R",
			styleClass: "revision state",
			fn: func() {
				MustRegisterRevisionState("R", "I", "D", Color{1, 1, 1, 1}, pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reset()
			LockRegistry()

			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("expected panic, but did not panic")
					return
				}
				msg, ok := r.(string)
				if !ok {
					t.Errorf("expected panic message to be a string, got %T", r)
					return
				}
				wantMsg := fmt.Sprintf("failed to register %s style %q: style-related registrations must be done in task/inspection/**/contract packages during package initialization. Did you call it from outside of the contract or not at package initialization timing?", tc.styleClass, tc.label)
				if msg != wantMsg {
					t.Errorf("unexpected panic message: got %q, want %q", msg, wantMsg)
				}
			}()

			tc.fn()
		})
	}
}

func TestRegisterTimelineTypeWithSortPolicy(t *testing.T) {
	testCases := []struct {
		name     string
		sortOpt  TimelineSortOpt
		wantType *pb.TimelineType
	}{
		{
			name:    "alphabetical sort policy",
			sortOpt: AlphabeticalSortPolicy("a", "b"),
			wantType: &pb.TimelineType{
				SortPolicyConfig: &pb.TimelineType_AlphabeticalPolicy{
					AlphabeticalPolicy: &pb.AlphabeticalSortPolicy{
						PrioritizedNames: []string{"a", "b"},
					},
				},
			},
		},
		{
			name:    "chronological sort policy",
			sortOpt: ChronologicalSortPolicy(5),
			wantType: &pb.TimelineType{
				SortPolicyConfig: &pb.TimelineType_ChronologicalPolicy{
					ChronologicalPolicy: &pb.ChronologicalSortPolicy{
						ChronologicalSearchDepth: proto.Int32(5),
					},
				},
			},
		},
		{
			name:    "grouped chronological sort policy",
			sortOpt: GroupedChronologicalSortPolicy("-"),
			wantType: &pb.TimelineType{
				SortPolicyConfig: &pb.TimelineType_GroupedChronologicalPolicy{
					GroupedChronologicalPolicy: &pb.GroupedChronologicalSortPolicy{
						Delimiter: proto.String("-"),
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reset()
			registered := MustRegisterTimelineType("T", "D", "icon", 1, Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, Color{0, 0, 0, 1}, Color{0, 0, 0, 1}, true, 10, tc.sortOpt)

			// We only compare the SortPolicyConfig portion
			got := registered.SortPolicyConfig
			want := tc.wantType.SortPolicyConfig
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("MustRegisterTimelineType() sort policy config mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
