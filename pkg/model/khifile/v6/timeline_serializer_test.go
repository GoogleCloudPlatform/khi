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
	"cmp"
	"slices"
	"testing"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	googlecmp "github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestExtractTimelinesAndItemsChunkSource(t *testing.T) {
	id1 := uint32(1)
	id2 := uint32(2)
	typeA := &pb.TimelineType{Id: &id1}
	typeB := &pb.TimelineType{Id: &id2}

	type timelineDef struct {
		id        string
		parent    string
		name      string
		ttype     *pb.TimelineType
		aliasOf   string
		events    []*pb.Event
		revisions []*pb.Revision
	}

	testCases := []struct {
		name      string
		timelines []timelineDef
		want      func(paths map[string]*TimelinePath, reg *TimelineRegistry) ([]*pb.Timeline, []*pb.TimelineItems)
	}{
		{
			name: "should extract timelines and timeline items correctly with root, child, alias, and empty paths",
			timelines: []timelineDef{
				{
					id:     "root",
					name:   "root",
					ttype:  typeA,
					events: []*pb.Event{{LogId: &id1}},
				},
				{
					id:        "child",
					parent:    "root",
					name:      "child",
					ttype:     typeB,
					revisions: []*pb.Revision{{LogId: &id2}},
				},
				{
					id:      "alias",
					name:    "alias",
					ttype:   typeA,
					aliasOf: "root",
				},
				{
					id:    "empty",
					name:  "empty",
					ttype: typeA,
				},
			},
			want: func(paths map[string]*TimelinePath, reg *TimelineRegistry) ([]*pb.Timeline, []*pb.TimelineItems) {
				rootID := paths["root"].ID
				childID := paths["child"].ID
				aliasID := paths["alias"].ID
				emptyID := paths["empty"].ID

				rootNameID := paths["root"].Name.id
				childNameID := paths["child"].Name.id
				aliasNameID := paths["alias"].Name.id
				emptyNameID := paths["empty"].Name.id

				rootItemsID := reg.GetBuilder(paths["root"]).TimelineItemsID
				childItemsID := reg.GetBuilder(paths["child"]).TimelineItemsID

				return []*pb.Timeline{
						{Id: &rootID, TimelineType: typeA.Id, NameStringId: &rootNameID, TimelineItemsId: &rootItemsID},
						{Id: &childID, ParentTimelineId: &rootID, TimelineType: typeB.Id, NameStringId: &childNameID, TimelineItemsId: &childItemsID},
						{Id: &aliasID, TimelineType: typeA.Id, NameStringId: &aliasNameID, TimelineItemsId: &rootItemsID},
						{Id: &emptyID, TimelineType: typeA.Id, NameStringId: &emptyNameID},
					}, []*pb.TimelineItems{
						{Id: &rootItemsID, Events: []*pb.Event{{LogId: &id1}}},
						{Id: &childItemsID, Revisions: []*pb.Revision{{LogId: &id2}}},
					}
			},
		},
		{
			name:      "should handle empty pool and registry",
			timelines: []timelineDef{},
			want: func(paths map[string]*TimelinePath, reg *TimelineRegistry) ([]*pb.Timeline, []*pb.TimelineItems) {
				return nil, nil
			},
		},
		{
			name: "should extract nested timelines correctly when only the deepest child has items",
			timelines: []timelineDef{
				{
					id:    "timelineA",
					name:  "A",
					ttype: typeA,
				},
				{
					id:     "timelineB",
					parent: "timelineA",
					name:   "B",
					ttype:  typeA,
				},
				{
					id:        "timelineC",
					parent:    "timelineB",
					name:      "C",
					ttype:     typeA,
					revisions: []*pb.Revision{{LogId: &id1}},
				},
			},
			want: func(paths map[string]*TimelinePath, reg *TimelineRegistry) ([]*pb.Timeline, []*pb.TimelineItems) {
				aID := paths["timelineA"].ID
				bID := paths["timelineB"].ID
				cID := paths["timelineC"].ID

				aNameID := paths["timelineA"].Name.id
				bNameID := paths["timelineB"].Name.id
				cNameID := paths["timelineC"].Name.id

				cItemsID := reg.GetBuilder(paths["timelineC"]).TimelineItemsID

				return []*pb.Timeline{ // Timelines should be serialized for any path segments.
						{Id: &aID, TimelineType: typeA.Id, NameStringId: &aNameID},
						{Id: &bID, ParentTimelineId: &aID, TimelineType: typeA.Id, NameStringId: &bNameID},
						{Id: &cID, ParentTimelineId: &bID, TimelineType: typeA.Id, NameStringId: &cNameID, TimelineItemsId: &cItemsID},
					}, []*pb.TimelineItems{ // TimelineItems should be serialized only for path segments with items.
						{Id: &cItemsID, Revisions: []*pb.Revision{{LogId: &id1}}},
					}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Isolated Setup for deterministic IDs
			idGen := &IDGenerator{}
			internPool := NewInternPool(idGen)
			pathPool := NewTimelinePathPool(idGen, internPool)
			registry := NewTimelineRegistry(idGen, internPool)

			pathMap := make(map[string]*TimelinePath)

			// 1. Setup the state
			for _, def := range tc.timelines {
				var parentPath *TimelinePath
				if def.parent != "" {
					parentPath = pathMap[def.parent]
				}

				path := pathPool.Get(parentPath, PathSegment{Name: def.name, Type: def.ttype})
				pathMap[def.id] = path

				if def.aliasOf != "" {
					registry.SetAlias(path, pathMap[def.aliasOf])
					continue
				}

				builder := registry.GetBuilder(path)
				for _, e := range def.events {
					builder.AddEvent(e)
				}
				for _, r := range def.revisions {
					builder.AddRevision(r)
				}
			}

			wantTimelines, wantItems := tc.want(pathMap, registry)

			slices.SortFunc(wantTimelines, func(a, b *pb.Timeline) int {
				return cmp.Compare(a.GetId(), b.GetId())
			})
			slices.SortFunc(wantItems, func(a, b *pb.TimelineItems) int {
				return cmp.Compare(a.GetId(), b.GetId())
			})

			gotTimelines, gotItems := ExtractTimelinesAndItemsChunkSource(pathPool, registry)

			if diff := googlecmp.Diff(wantTimelines, gotTimelines, protocmp.Transform()); diff != "" {
				t.Errorf("Timelines mismatch (-want +got):\n%s", diff)
			}

			if diff := googlecmp.Diff(wantItems, gotItems, protocmp.Transform()); diff != "" {
				t.Errorf("TimelineItems mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
