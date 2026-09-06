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
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var sinkChangeSet *TimelineChangeSet

// BenchmarkNewTimelineChangeSet measures allocation churn and throughput for staging timeline
// mutations across the common usage patterns observed in GKE audit log mappers.
// Evaluates heap allocations from 3 Go maps created unconditionally in NewTimelineChangeSet.
func BenchmarkNewTimelineChangeSet(b *testing.B) {
	idGen := id.NewGenerator()
	pool := NewTestInternPool(idGen)
	pathPool := NewTimelinePathPool(idGen, pool)

	timelineTypeID := uint32(1)
	timelineType := &pb.TimelineType{Id: &timelineTypeID}
	targetPath := pathPool.Get(nil, PathSegment{Name: "pod-1", Type: timelineType})
	aliasPath := pathPool.Get(nil, PathSegment{Name: "pod-alias", Type: timelineType})

	emptyNode := structured.NewStandardMap(nil, nil)
	mockLog := log.NewLog(idGen, structured.NewNodeReader(emptyNode))
	mockLog.Timestamp = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	stagingRev := &StagingRevision{
		ChangedTime: mockLog.Timestamp,
		Principal:   "system:serviceaccount:kube-system:generic-garbage-collector",
	}

	b.Run("AllocOnly", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cs := NewTimelineChangeSet(mockLog)
			sinkChangeSet = cs
			cs.Release()
		}
	})

	b.Run("AddSingleEvent", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cs := NewTimelineChangeSet(mockLog)
			cs.AddEvent(targetPath)
			sinkChangeSet = cs
			cs.Release()
		}
	})

	b.Run("AddSingleRevision", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cs := NewTimelineChangeSet(mockLog)
			cs.AddRevision(targetPath, stagingRev)
			sinkChangeSet = cs
			cs.Release()
		}
	})

	b.Run("AddEventAndRevision", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cs := NewTimelineChangeSet(mockLog)
			cs.AddEvent(targetPath)
			cs.AddRevision(targetPath, stagingRev)
			sinkChangeSet = cs
			cs.Release()
		}
	})

	b.Run("AddEventRevisionAndAlias", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cs := NewTimelineChangeSet(mockLog)
			cs.AddEvent(targetPath)
			cs.AddRevision(targetPath, stagingRev)
			cs.AddAlias(aliasPath, targetPath)
			sinkChangeSet = cs
			cs.Release()
		}
	})
}

// TimelineChangeSet_LegacyMap represents the legacy map-based implementation for comparative benchmarking.
type TimelineChangeSet_LegacyMap struct {
	Log       *log.Log
	Events    map[*TimelinePath]bool
	Revisions map[*TimelinePath][]*StagingRevision
	Aliases   map[*TimelinePath]*TimelinePath
}

// NewTimelineChangeSet_LegacyMap creates a legacy map-backed changeset allocating 3 Go maps unconditionally.
func NewTimelineChangeSet_LegacyMap(l *log.Log) *TimelineChangeSet_LegacyMap {
	return &TimelineChangeSet_LegacyMap{
		Log:       l,
		Events:    make(map[*TimelinePath]bool),
		Revisions: make(map[*TimelinePath][]*StagingRevision),
		Aliases:   make(map[*TimelinePath]*TimelinePath),
	}
}

// AddEvent stages a timeline event in the map.
func (cs *TimelineChangeSet_LegacyMap) AddEvent(path *TimelinePath) {
	cs.Events[path] = true
}

// AddRevision stages a resource revision in the map.
func (cs *TimelineChangeSet_LegacyMap) AddRevision(path *TimelinePath, revision *StagingRevision) {
	cs.Revisions[path] = append(cs.Revisions[path], revision)
}

// AddAlias stages an alias mapping in the map.
func (cs *TimelineChangeSet_LegacyMap) AddAlias(aliasPath, targetPath *TimelinePath) {
	cs.Aliases[aliasPath] = targetPath
}

var sinkLegacyChangeSet *TimelineChangeSet_LegacyMap

// BenchmarkTimelineChangeSet_LegacyMap measures allocation churn and throughput
// of the previous map-based implementation for comparative baseline analysis.
func BenchmarkTimelineChangeSet_LegacyMap(b *testing.B) {
	idGen := id.NewGenerator()
	pool := NewTestInternPool(idGen)
	pathPool := NewTimelinePathPool(idGen, pool)

	timelineTypeID := uint32(1)
	timelineType := &pb.TimelineType{Id: &timelineTypeID}
	targetPath := pathPool.Get(nil, PathSegment{Name: "pod-1", Type: timelineType})
	aliasPath := pathPool.Get(nil, PathSegment{Name: "pod-alias", Type: timelineType})

	emptyNode := structured.NewStandardMap(nil, nil)
	mockLog := log.NewLog(idGen, structured.NewNodeReader(emptyNode))
	mockLog.Timestamp = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	stagingRev := &StagingRevision{
		ChangedTime: mockLog.Timestamp,
		Principal:   "system:serviceaccount:kube-system:generic-garbage-collector",
	}

	b.Run("AllocOnly", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cs := NewTimelineChangeSet_LegacyMap(mockLog)
			sinkLegacyChangeSet = cs
		}
	})

	b.Run("AddSingleEvent", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cs := NewTimelineChangeSet_LegacyMap(mockLog)
			cs.AddEvent(targetPath)
			sinkLegacyChangeSet = cs
		}
	})

	b.Run("AddSingleRevision", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cs := NewTimelineChangeSet_LegacyMap(mockLog)
			cs.AddRevision(targetPath, stagingRev)
			sinkLegacyChangeSet = cs
		}
	})

	b.Run("AddEventAndRevision", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cs := NewTimelineChangeSet_LegacyMap(mockLog)
			cs.AddEvent(targetPath)
			cs.AddRevision(targetPath, stagingRev)
			sinkLegacyChangeSet = cs
		}
	})

	b.Run("AddEventRevisionAndAlias", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cs := NewTimelineChangeSet_LegacyMap(mockLog)
			cs.AddEvent(targetPath)
			cs.AddRevision(targetPath, stagingRev)
			cs.AddAlias(aliasPath, targetPath)
			sinkLegacyChangeSet = cs
		}
	})
}

func extractChangeSetMaps(cs *TimelineChangeSet) (map[*TimelinePath]bool, map[*TimelinePath][]*StagingRevision, map[*TimelinePath]*TimelinePath) {
	events := make(map[*TimelinePath]bool)
	cs.ForEachEvent(func(p *TimelinePath) {
		events[p] = true
	})
	revisions := make(map[*TimelinePath][]*StagingRevision)
	cs.ForEachRevision(func(p *TimelinePath, revs []*StagingRevision) {
		revisions[p] = append(revisions[p], revs...)
	})
	aliases := make(map[*TimelinePath]*TimelinePath)
	cs.ForEachAlias(func(a, t *TimelinePath) {
		aliases[a] = t
	})
	return events, revisions, aliases
}

// TestTimelineChangeSet_Equivalence asserts that TimelineChangeSet
// correctly records mutations and matches TimelineChangeSet_LegacyMap under all conditions.
func TestTimelineChangeSet_Equivalence(t *testing.T) {
	idGen := id.NewGenerator()
	pool := NewTestInternPool(idGen)
	pathPool := NewTimelinePathPool(idGen, pool)

	timelineTypeID := uint32(1)
	timelineType := &pb.TimelineType{Id: &timelineTypeID}
	p1 := pathPool.Get(nil, PathSegment{Name: "pod-1", Type: timelineType})
	p2 := pathPool.Get(nil, PathSegment{Name: "pod-2", Type: timelineType})
	p3 := pathPool.Get(nil, PathSegment{Name: "pod-3", Type: timelineType})
	a1 := pathPool.Get(nil, PathSegment{Name: "alias-1", Type: timelineType})
	a2 := pathPool.Get(nil, PathSegment{Name: "alias-2", Type: timelineType})
	a3 := pathPool.Get(nil, PathSegment{Name: "alias-3", Type: timelineType})

	emptyNode := structured.NewStandardMap(nil, nil)
	mockLog := log.NewLog(idGen, structured.NewNodeReader(emptyNode))
	mockLog.Timestamp = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	rev1 := &StagingRevision{ChangedTime: mockLog.Timestamp, Principal: "sa-1"}
	rev2 := &StagingRevision{ChangedTime: mockLog.Timestamp, Principal: "sa-2"}
	rev3 := &StagingRevision{ChangedTime: mockLog.Timestamp, Principal: "sa-3"}

	testCases := []struct {
		name     string
		populate func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet)
	}{
		{
			name: "SingleEvent",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				legacy.AddEvent(p1)
				got.AddEvent(p1)
			},
		},
		{
			name: "DeduplicatedEvents",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				legacy.AddEvent(p1)
				legacy.AddEvent(p1)
				got.AddEvent(p1)
				got.AddEvent(p1)
			},
		},
		{
			name: "SingleRevision",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				legacy.AddRevision(p1, rev1)
				got.AddRevision(p1, rev1)
			},
		},
		{
			name: "EventAndRevision",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				legacy.AddEvent(p1)
				legacy.AddRevision(p1, rev1)
				got.AddEvent(p1)
				got.AddRevision(p1, rev1)
			},
		},
		{
			name: "OverflowCapacity3Items",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				// 3 events (exceeds capacity 2)
				legacy.AddEvent(p1)
				legacy.AddEvent(p2)
				legacy.AddEvent(p3)
				got.AddEvent(p1)
				got.AddEvent(p2)
				got.AddEvent(p3)

				// 3 revisions (exceeds capacity 2)
				legacy.AddRevision(p1, rev1)
				legacy.AddRevision(p2, rev2)
				legacy.AddRevision(p3, rev3)
				got.AddRevision(p1, rev1)
				got.AddRevision(p2, rev2)
				got.AddRevision(p3, rev3)

				// 3 aliases (exceeds capacity 2)
				legacy.AddAlias(a1, p1)
				legacy.AddAlias(a2, p2)
				legacy.AddAlias(a3, p3)
				got.AddAlias(a1, p1)
				got.AddAlias(a2, p2)
				got.AddAlias(a3, p3)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			legacy := NewTimelineChangeSet_LegacyMap(mockLog)
			got := NewTimelineChangeSet(mockLog)

			tc.populate(legacy, got)

			gotEvents, gotRevisions, gotAliases := extractChangeSetMaps(got)

			opts := cmpopts.EquateComparable((*TimelinePath)(nil))

			// Compare Events
			if diff := cmp.Diff(legacy.Events, gotEvents, opts); diff != "" {
				t.Errorf("events mismatch (-legacy +got):\n%s", diff)
			}

			// Compare Revisions
			if diff := cmp.Diff(legacy.Revisions, gotRevisions, opts); diff != "" {
				t.Errorf("revisions mismatch (-legacy +got):\n%s", diff)
			}

			// Compare Aliases
			if diff := cmp.Diff(legacy.Aliases, gotAliases, opts); diff != "" {
				t.Errorf("aliases mismatch (-legacy +got):\n%s", diff)
			}
		})
	}
}

// TestTimelineChangeSet_AdversarialStress stress-tests TimelineChangeSet
// across capacity transitions (0, exactly 2, 3, 100), duplicate additions, multi-revisions per path,
// and alias chains/updates, asserting full parity with baseline TimelineChangeSet_LegacyMap.
func TestTimelineChangeSet_AdversarialStress(t *testing.T) {
	idGen := id.NewGenerator()
	pool := NewTestInternPool(idGen)
	pathPool := NewTimelinePathPool(idGen, pool)

	timelineTypeID := uint32(1)
	timelineType := &pb.TimelineType{Id: &timelineTypeID}

	makePath := func(name string) *TimelinePath {
		return pathPool.Get(nil, PathSegment{Name: name, Type: timelineType})
	}

	emptyNode := structured.NewStandardMap(nil, nil)
	mockLog := log.NewLog(idGen, structured.NewNodeReader(emptyNode))
	mockLog.Timestamp = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	p1 := makePath("pod-1")
	p2 := makePath("pod-2")
	p3 := makePath("pod-3")
	a1 := makePath("alias-1")
	a2 := makePath("alias-2")
	a3 := makePath("alias-3")

	rev := func(tag string) *StagingRevision {
		return &StagingRevision{ChangedTime: mockLog.Timestamp, Principal: tag}
	}

	testCases := []struct {
		name     string
		populate func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet)
	}{
		{
			name: "Capacity0_EmptyChangeSet",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				// No mutations added.
			},
		},
		{
			name: "CapacityExactly2_InlineLimits",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				legacy.AddEvent(p1)
				legacy.AddEvent(p2)
				got.AddEvent(p1)
				got.AddEvent(p2)

				legacy.AddRevision(p1, rev("rev1"))
				legacy.AddRevision(p2, rev("rev2"))
				got.AddRevision(p1, rev("rev1"))
				got.AddRevision(p2, rev("rev2"))

				legacy.AddAlias(a1, p1)
				legacy.AddAlias(a2, p2)
				got.AddAlias(a1, p1)
				got.AddAlias(a2, p2)
			},
		},
		{
			name: "CapacityExactly2_MultipleRevisionsSamePath",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				// 2 revisions on the exact same path within inline array capacity
				legacy.AddRevision(p1, rev("rev1"))
				legacy.AddRevision(p1, rev("rev2"))
				got.AddRevision(p1, rev("rev1"))
				got.AddRevision(p1, rev("rev2"))
			},
		},
		{
			name: "Capacity3_OverflowBoundary",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				// Adding exactly 3 items triggers inline-to-overflow transition
				legacy.AddEvent(p1)
				legacy.AddEvent(p2)
				legacy.AddEvent(p3)
				got.AddEvent(p1)
				got.AddEvent(p2)
				got.AddEvent(p3)

				legacy.AddRevision(p1, rev("r1"))
				legacy.AddRevision(p2, rev("r2"))
				legacy.AddRevision(p3, rev("r3"))
				got.AddRevision(p1, rev("r1"))
				got.AddRevision(p2, rev("r2"))
				got.AddRevision(p3, rev("r3"))

				legacy.AddAlias(a1, p1)
				legacy.AddAlias(a2, p2)
				legacy.AddAlias(a3, p3)
				got.AddAlias(a1, p1)
				got.AddAlias(a2, p2)
				got.AddAlias(a3, p3)
			},
		},
		{
			name: "DuplicateEventAdditions_BeforeAndAfterOverflow",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				// p1 added before overflow, p2 added, p1 re-added, p3 added (overflow), p1 re-added
				legacy.AddEvent(p1)
				legacy.AddEvent(p2)
				legacy.AddEvent(p1)
				legacy.AddEvent(p3)
				legacy.AddEvent(p1)
				legacy.AddEvent(p2)

				got.AddEvent(p1)
				got.AddEvent(p2)
				got.AddEvent(p1)
				got.AddEvent(p3)
				got.AddEvent(p1)
				got.AddEvent(p2)
			},
		},
		{
			name: "AliasOverwrites_WithinCapacityAndOverflow",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				// Overwrite alias target for a1 within inline array
				legacy.AddAlias(a1, p1)
				legacy.AddAlias(a1, p2)
				got.AddAlias(a1, p1)
				got.AddAlias(a1, p2)

				// Now cause overflow with a2 and a3, then overwrite a2
				legacy.AddAlias(a2, p2)
				legacy.AddAlias(a3, p3)
				legacy.AddAlias(a2, p1)
				got.AddAlias(a2, p2)
				got.AddAlias(a3, p3)
				got.AddAlias(a2, p1)
			},
		},
		{
			name: "AliasChain",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				// Chain: a1 -> p1, a2 -> a1, a3 -> a2
				legacy.AddAlias(a1, p1)
				legacy.AddAlias(a2, a1)
				legacy.AddAlias(a3, a2)
				got.AddAlias(a1, p1)
				got.AddAlias(a2, a1)
				got.AddAlias(a3, a2)
			},
		},
		{
			name: "Capacity100_LargeScaleStress",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				for i := 0; i < 100; i++ {
					p := makePath("path-" + string(rune('A'+i%26)) + "-" + string(rune('0'+i/10)))
					a := makePath("alias-" + string(rune('A'+i%26)) + "-" + string(rune('0'+i/10)))
					r := rev("rev-" + string(rune('0'+i%10)))

					legacy.AddEvent(p)
					got.AddEvent(p)

					legacy.AddRevision(p, r)
					got.AddRevision(p, r)

					legacy.AddAlias(a, p)
					got.AddAlias(a, p)
				}
			},
		},
		{
			name: "Capacity100_RevisionsOnSinglePath",
			populate: func(legacy *TimelineChangeSet_LegacyMap, got *TimelineChangeSet) {
				for i := 0; i < 100; i++ {
					r := rev("rev-" + string(rune('0'+i%10)))
					legacy.AddRevision(p1, r)
					got.AddRevision(p1, r)
				}
			},
		},
	}

	opts := cmpopts.EquateComparable((*TimelinePath)(nil))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			legacy := NewTimelineChangeSet_LegacyMap(mockLog)
			got := NewTimelineChangeSet(mockLog)

			tc.populate(legacy, got)

			gotEvents, gotRevisions, gotAliases := extractChangeSetMaps(got)

			if diff := cmp.Diff(legacy.Events, gotEvents, opts); diff != "" {
				t.Fatalf("events mismatch (-legacy +got):\n%s", diff)
			}
			if diff := cmp.Diff(legacy.Revisions, gotRevisions, opts); diff != "" {
				t.Fatalf("revisions mismatch (-legacy +got):\n%s", diff)
			}
			if diff := cmp.Diff(legacy.Aliases, gotAliases, opts); diff != "" {
				t.Fatalf("aliases mismatch (-legacy +got):\n%s", diff)
			}
		})
	}
}
