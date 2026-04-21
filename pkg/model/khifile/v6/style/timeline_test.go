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

	"google.golang.org/protobuf/proto"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

func TestRegisterTimelineType(t *testing.T) {
	Reset()

	t1 := &pb.TimelineType{Label: proto.String("Type 1")}
	t2 := &pb.TimelineType{Label: proto.String("Type 2")}

	res1 := RegisterTimelineType(t1)
	RegisterTimelineType(t2)

	// Verify the returned pointer is the same as the input
	if res1 != t1 {
		t.Errorf("Expected returned pointer to be the same, got different")
	}

	// Verify IDs were assigned starting from 1
	if t1.Id == nil || *t1.Id != 1 {
		t.Errorf("Expected ID 1, got %v", t1.Id)
	}
	if t2.Id == nil || *t2.Id != 2 {
		t.Errorf("Expected ID 2, got %v", t2.Id)
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
	Reset()

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			RegisterVerb(&pb.Verb{Label: proto.String("Concurrent Verb")})
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
	Reset()

	RegisterSeverity(&pb.Severity{Label: proto.String("Sev")})
	RegisterVerb(&pb.Verb{Label: proto.String("Verb")})
	RegisterLogType(&pb.LogType{Label: proto.String("Log")})
	RegisterRevisionState(&pb.RevisionState{Label: proto.String("RevState")})
	RegisterTimelineType(&pb.TimelineType{Label: proto.String("Timeline")})

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
