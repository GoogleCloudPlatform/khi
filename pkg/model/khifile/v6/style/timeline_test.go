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
	"sync"
	"testing"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

func TestRegisterTimelineType(t *testing.T) {
	reset()

	res1 := RegisterTimelineType("Type 1", "Desc 1", Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, true, 1)
	res2 := RegisterTimelineType("Type 2", "Desc 2", Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, true, 2)

	// Verify IDs were assigned starting from 1
	if res1.Id == nil || *res1.Id != 1 {
		t.Errorf("Expected ID 1, got %v", res1.Id)
	}
	if res2.Id == nil || *res2.Id != 2 {
		t.Errorf("Expected ID 2, got %v", res2.Id)
	}

	chunk := GenerateChunk()
	if len(chunk.TimelineTypes) != 2 {
		t.Fatalf("Expected 2 TimelineTypes in chunk, got %d", len(chunk.TimelineTypes))
	}
	if *chunk.TimelineTypes[0].Id != 1 {
		t.Errorf("Expected first chunk TimelineType ID to be 1")
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
			RegisterVerb("Concurrent Verb", Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, true)
		}(i)
	}

	wg.Wait()

	chunk := GenerateChunk()
	if len(chunk.Verbs) != numGoroutines {
		t.Fatalf("Expected %d Verbs registered, got %d", numGoroutines, len(chunk.Verbs))
	}

	// Verify all IDs from 1 to numGoroutines exist exactly once
	idMap := make(map[uint32]bool)
	for _, v := range chunk.Verbs {
		if v.Id == nil {
			t.Fatalf("Verb ID is nil")
		}
		if idMap[*v.Id] {
			t.Errorf("Duplicate ID found: %d", *v.Id)
		}
		idMap[*v.Id] = true
	}

	if len(idMap) != numGoroutines {
		t.Errorf("Expected %d unique IDs, got %d", numGoroutines, len(idMap))
	}
}

func TestGenerateChunkHasAllSlices(t *testing.T) {
	reset()

	RegisterSeverity("Sev", "S", Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, 1)
	RegisterVerb("Verb", Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, true)
	RegisterLogType("Log", "Desc", Color{1, 1, 1, 1}, Color{0, 0, 0, 1})
	RegisterRevisionState("RevState", "icon", "Desc", Color{1, 1, 1, 1}, pb.RevisionStateStyle_REVISION_STATE_STYLE_NORMAL)
	RegisterTimelineType("Timeline", "Desc", Color{1, 1, 1, 1}, Color{0, 0, 0, 1}, true, 1)

	chunk := GenerateChunk()

	if len(chunk.Severities) != 1 {
		t.Errorf("Expected 1 Severity, got %d", len(chunk.Severities))
	}
	if len(chunk.Verbs) != 1 {
		t.Errorf("Expected 1 Verb, got %d", len(chunk.Verbs))
	}
	if len(chunk.LogTypes) != 1 {
		t.Errorf("Expected 1 LogType, got %d", len(chunk.LogTypes))
	}
	if len(chunk.RevisionStates) != 1 {
		t.Errorf("Expected 1 RevisionState, got %d", len(chunk.RevisionStates))
	}
	if len(chunk.TimelineTypes) != 1 {
		t.Errorf("Expected 1 TimelineType, got %d", len(chunk.TimelineTypes))
	}
}

func TestGenerateChunkHasIconAtlas(t *testing.T) {
	reset()

	chunk := GenerateChunk()
	if chunk.IconAtlas == nil {
		t.Fatal("Expected IconAtlas not to be nil in TimelineStyleChunk")
	}

	if len(chunk.IconAtlas.MsdfIconImage) == 0 || len(chunk.IconAtlas.MsdfIconImage[0]) == 0 {
		t.Error("Expected non-empty embedded PNG bytes in MsdfIconImage")
	}

	if len(chunk.IconAtlas.BmfontJson) == 0 {
		t.Error("Expected non-empty embedded BMFont configuration JSON")
	}

	if len(chunk.IconAtlas.NameToCodepoints) == 0 {
		t.Error("Expected populated mapping in NameToCodepoints")
	}
}
