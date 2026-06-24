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

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCompareAlphabetical(t *testing.T) {
	idGen := &IDGenerator{}
	internPool := NewInternPool(idGen)
	pool := NewTimelinePathPool(idGen, internPool)
	timelineType := &pb.TimelineType{Id: proto.Uint32(1)}

	testCases := []struct {
		name             string
		aName            string
		bName            string
		prioritizedNames []string
		want             int // < 0 means a < b, > 0 means a > b, 0 means a == b
	}{
		{
			name:  "standard alphabetical comparison",
			aName: "app",
			bName: "web",
			want:  -1,
		},
		{
			name:             "respects prioritized names",
			aName:            "web",
			bName:            "app",
			prioritizedNames: []string{"web", "db"},
			want:             -1,
		},
		{
			name:             "respects priority order",
			aName:            "db",
			bName:            "web",
			prioritizedNames: []string{"web", "db"},
			want:             1,
		},
		{
			name:             "prioritized comes before unprioritized",
			aName:            "db",
			bName:            "app",
			prioritizedNames: []string{"web", "db"},
			want:             -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := pool.Get(nil, PathSegment{Name: tc.aName, Type: timelineType})
			b := pool.Get(nil, PathSegment{Name: tc.bName, Type: timelineType})

			m := make(map[string]int)
			for i, name := range tc.prioritizedNames {
				m[name] = i
			}

			got := CompareAlphabetical(a, b, m)
			if (got < 0 && tc.want >= 0) || (got > 0 && tc.want <= 0) || (got == 0 && tc.want != 0) {
				t.Errorf("CompareAlphabetical(%q, %q) = %d; want sign of %d", tc.aName, tc.bName, got, tc.want)
			}
		})
	}
}

func TestCompareChronological(t *testing.T) {
	idGen := &IDGenerator{}
	internPool := NewInternPool(idGen)
	pool := NewTimelinePathPool(idGen, internPool)
	timelineType := &pb.TimelineType{Id: proto.Uint32(1)}

	// Setup timelines
	pathA := pool.Get(nil, PathSegment{Name: "A", Type: timelineType})
	pathB := pool.Get(nil, PathSegment{Name: "B", Type: timelineType})
	pathAChild := pool.Get(pathA, PathSegment{Name: "ChildA", Type: timelineType})
	pathAGrandChild := pool.Get(pathAChild, PathSegment{Name: "GrandChildA", Type: timelineType})

	parentToChildren := map[*TimelinePath][]*TimelinePath{
		pathA:      {pathAChild},
		pathAChild: {pathAGrandChild},
	}

	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	testCases := []struct {
		name      string
		setupFunc func(registry *TimelineRegistry)
		maxDepth  int32
		want      int
	}{
		{
			name: "both have direct revisions, A is older",
			setupFunc: func(registry *TimelineRegistry) {
				bA := registry.GetBuilder(pathA)
				bA.AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t1)})

				bB := registry.GetBuilder(pathB)
				bB.AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t2)})
			},
			maxDepth: 0,
			want:     -1,
		},
		{
			name: "A's grandchild is older, A comes first when depth allows search",
			setupFunc: func(registry *TimelineRegistry) {
				bGrandChild := registry.GetBuilder(pathAGrandChild)
				bGrandChild.AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t1)})

				bB := registry.GetBuilder(pathB)
				bB.AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t2)})
			},
			maxDepth: 2, // depth 2 allows reaching grandchild
			want:     -1,
		},
		{
			name: "A's grandchild is older, but A loses when search depth is 1",
			setupFunc: func(registry *TimelineRegistry) {
				bGrandChild := registry.GetBuilder(pathAGrandChild)
				bGrandChild.AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t1)})

				bB := registry.GetBuilder(pathB)
				bB.AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t2)})
			},
			maxDepth: 1, // depth 1 only reaches ChildA, not GrandChildA
			want:     1, // B comes first since B has t2, A has no times within depth 1
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logAcc := NewLogAccumulator(internPool, idGen)
			registry := NewTimelineRegistry(idGen, internPool, logAcc)

			tc.setupFunc(registry)

			got := CompareChronological(pathA, pathB, registry, parentToChildren, tc.maxDepth)
			if (got < 0 && tc.want >= 0) || (got > 0 && tc.want <= 0) || (got == 0 && tc.want != 0) {
				t.Errorf("CompareChronological() = %d; want sign of %d", got, tc.want)
			}
		})
	}
}

func TestCompareGroupedChronological(t *testing.T) {
	idGen := &IDGenerator{}
	internPool := NewInternPool(idGen)
	pool := NewTimelinePathPool(idGen, internPool)
	timelineType := &pb.TimelineType{Id: proto.Uint32(1)}

	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 9, 30, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 1, 9, 45, 0, 0, time.UTC)

	path1 := pool.Get(nil, PathSegment{Name: "aaaa-bbbb-cccc-1111", Type: timelineType})
	path2 := pool.Get(nil, PathSegment{Name: "aaaa-bbbb-cccc-2222", Type: timelineType})
	path3 := pool.Get(nil, PathSegment{Name: "aaaa-bbbb-dddd-3333", Type: timelineType})
	pathParent := pool.Get(nil, PathSegment{Name: "aaaa-bbbb", Type: timelineType})
	pathChild := pool.Get(nil, PathSegment{Name: "aaaa-bbbb-cccc-dddd", Type: timelineType})
	pathGroupA := pool.Get(nil, PathSegment{Name: "aaaa-bb-01", Type: timelineType})
	pathGroupB := pool.Get(nil, PathSegment{Name: "aaaa-bbbb-01", Type: timelineType})
	pathGroupB2 := pool.Get(nil, PathSegment{Name: "aaaa-bbbb-02", Type: timelineType})

	t0800 := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	t0900 := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	t1000 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	parentToChildren := map[*TimelinePath][]*TimelinePath{
		nil: {path1, path2, path3, pathParent, pathChild, pathGroupA, pathGroupB, pathGroupB2},
	}

	testCases := []struct {
		name      string
		a         *TimelinePath
		b         *TimelinePath
		setupFunc func(registry *TimelineRegistry)
		want      int
	}{
		{
			name: "same group timelines sorted chronologically",
			a:    path1,
			b:    path2,
			setupFunc: func(registry *TimelineRegistry) {
				registry.GetBuilder(path1).AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t1)}) // 10:00
				registry.GetBuilder(path2).AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t2)}) // 09:30
			},
			want: 1, // path1 (10:00) > path2 (09:30), so path2 comes first
		},
		{
			name: "group representative time determines group order",
			a:    path1, // cccc group (min time t2 = 09:30)
			b:    path3, // dddd group (min time t3 = 09:45)
			setupFunc: func(registry *TimelineRegistry) {
				registry.GetBuilder(path1).AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t1)}) // 10:00
				registry.GetBuilder(path2).AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t2)}) // 09:30
				registry.GetBuilder(path3).AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t3)}) // 09:45
			},
			want: -1, // cccc group (09:30) < dddd group (09:45)
		},
		{
			name: "exact parent prefix comes before child subresource",
			a:    pathParent,
			b:    pathChild,
			setupFunc: func(registry *TimelineRegistry) {
				registry.GetBuilder(pathParent).AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t1)})
				registry.GetBuilder(pathChild).AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t2)})
			},
			want: -1, // pathParent ("aaaa-bbbb") comes before pathChild ("aaaa-bbbb-cccc-dddd") regardless of times
		},
		{
			name: "prevents false positive substring matches across token boundaries",
			a:    pathGroupA, // "aaaa-bb-01" (time = 09:00)
			b:    pathGroupB, // "aaaa-bbbb-01" (time = 10:00, but sibling "aaaa-bbbb-02" has 08:00)
			setupFunc: func(registry *TimelineRegistry) {
				registry.GetBuilder(pathGroupA).AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t0900)})  // 09:00
				registry.GetBuilder(pathGroupB).AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t1000)})  // 10:00
				registry.GetBuilder(pathGroupB2).AddRevision(&pb.Revision{ChangedTime: timestamppb.New(t0800)}) // 08:00
			},
			want: 1, // pathGroupB's group ("aaaa-bbbb") has representative time 08:00 < 09:00 ("aaaa-bb"). If "aaaa-bbbb" leaked into "aaaa-bb", it would tie at 08:00 and return -1.
		},
		{
			name:      "fallback to alphabetical when both have no timestamps",
			a:         path1,
			b:         path2,
			setupFunc: func(registry *TimelineRegistry) {},
			want:      -1, // "1111" < "2222"
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logAcc := NewLogAccumulator(internPool, idGen)
			registry := NewTimelineRegistry(idGen, internPool, logAcc)
			tc.setupFunc(registry)

			got := CompareGroupedChronological(tc.a, tc.b, registry, parentToChildren, "-")
			if (got < 0 && tc.want >= 0) || (got > 0 && tc.want <= 0) || (got == 0 && tc.want != 0) {
				t.Errorf("CompareGroupedChronological() = %d; want sign of %d", got, tc.want)
			}
		})
	}
}
