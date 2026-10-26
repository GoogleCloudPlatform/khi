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
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	pbv6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
)

func TestDefaultMaskReplacement(t *testing.T) {
	testCases := []struct {
		name  string
		index int
		want  string
	}{
		{
			name:  "non-positive index clamps to masked_1",
			index: 0,
			want:  "<masked_1>",
		},
		{
			name:  "first sequential placeholder",
			index: 1,
			want:  "<masked_1>",
		},
		{
			name:  "multi-digit sequential placeholder",
			index: 12,
			want:  "<masked_12>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := DefaultMaskReplacement(tc.index)
			if got != tc.want {
				t.Errorf("DefaultMaskReplacement(%d) = %q, want %q", tc.index, got, tc.want)
			}
		})
	}
}

func TestMaskReplacer_Replace(t *testing.T) {
	testCases := []struct {
		name  string
		rules []MaskRule
		input string
		want  string
	}{
		{
			name:  "no rules returns input unchanged",
			rules: nil,
			input: "prod-cluster",
			want:  "prod-cluster",
		},
		{
			name: "longest match takes precedence over shorter prefix in single pass",
			rules: []MaskRule{
				{Target: "prod", Replacement: "ENV"},
				{Target: "prod-cluster-east1", Replacement: "CLUSTER_A"},
			},
			input: "node in prod-cluster-east1 and prod-db",
			want:  "node in CLUSTER_A and ENV-db",
		},
		{
			name: "single pass prevents cascading replacement when replacement overlaps another target",
			rules: []MaskRule{
				{Target: "alice", Replacement: "bob"},
				{Target: "bob", Replacement: "charlie"},
			},
			input: "owner=alice reviewer=bob",
			want:  "owner=bob reviewer=charlie",
		},
		{
			name: "single pass prevents cascading replacement when <masked_N> digits or words overlap another rule",
			rules: []MaskRule{
				{Target: "secret-a", Replacement: "<masked_1>"},
				{Target: "1", Replacement: "<masked_2>"},
				{Target: "masked", Replacement: "<masked_3>"},
			},
			input: "pod=secret-a replica=1 label=masked",
			want:  "pod=<masked_1> replica=<masked_2> label=<masked_3>",
		},
		{
			name: "explicit empty replacement deletes matching substrings",
			rules: []MaskRule{
				{Target: "-secret-suffix", Replacement: ""},
			},
			input: "pod-alpha-secret-suffix",
			want:  "pod-alpha",
		},
		{
			name: "preserves null byte delimiters in interned field path strings",
			rules: []MaskRule{
				{Target: "acme.corp", Replacement: "example.com"},
			},
			input: "metadata\x00annotations\x00acme.corp/owner",
			want:  "metadata\x00annotations\x00example.com/owner",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			replacer := NewMaskReplacer(tc.rules)
			got := replacer.Replace(tc.input)
			if got != tc.want {
				t.Errorf("Replace(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestExportMaskedKHIFile(t *testing.T) {
	type exportedSnapshot struct {
		InspectionName    string
		SuggestedFilename string
		QueryName         string
		QueryText         string
		TimelineNames     map[uint32]string
		LogSummaries      map[uint32]string
		BodyYAML          string
	}

	buildFixtureArchive := func(t *testing.T) []byte {
		t.Helper()
		var buf bytes.Buffer
		w, err := NewWriter(&buf)
		if err != nil {
			t.Fatalf("failed to create fixture writer: %v", err)
		}

		meta := &pbv6.MetadataChunk{
			Metadata: []*pbv6.MetadataItem{
				{
					Payload: &pbv6.MetadataItem_Header{
						Header: &pbv6.HeaderMetadata{
							InspectionType:    proto.String("gcp-gke"),
							InspectionName:    proto.String("Cluster secret-cluster"),
							SuggestedFilename: proto.String("secret-cluster.khi"),
						},
					},
				},
				{
					Payload: &pbv6.MetadataItem_Query{
						Query: &pbv6.QueryMetadata{
							Queries: []*pbv6.QueryItem{
								{
									Id:    proto.String("q1"),
									Name:  proto.String("Audit query for secret-ns"),
									Query: proto.String(`resource.labels.cluster_name="secret-cluster" AND "secret-ns"`),
								},
							},
						},
					},
				},
			},
		}
		if err := w.WriteChunk(ChunkTypeMetadata, meta); err != nil {
			t.Fatalf("failed to write metadata chunk: %v", err)
		}

		logChunk := &pbv6.LogChunk{
			Logs: []*pbv6.Log{
				{
					Id:              proto.Uint32(1),
					SummaryStringId: proto.Uint32(1),
					BodyStructId:    proto.Uint32(1),
				},
				{
					Id:              proto.Uint32(2),
					SummaryStringId: proto.Uint32(2),
					BodyStructId:    proto.Uint32(1),
				},
			},
		}
		if err := w.WriteChunk(ChunkTypeLog, logChunk); err != nil {
			t.Fatalf("failed to write log chunk: %v", err)
		}

		// Intentionally write ChunkTypeTimeline BEFORE ChunkTypeInternPool to match Builder.Build chunk ordering.
		timelineChunk := &pbv6.TimelineChunk{
			Timelines: []*pbv6.Timeline{
				{
					Id:               proto.Uint32(10),
					ParentTimelineId: proto.Uint32(0),
					NameStringId:     proto.Uint32(1), // "secret-ns" (shared with Log #1 summary!)
				},
				{
					Id:               proto.Uint32(11),
					ParentTimelineId: proto.Uint32(10),
					NameStringId:     proto.Uint32(2), // "app-secret-cluster"
				},
			},
		}
		if err := w.WriteChunk(ChunkTypeTimeline, timelineChunk); err != nil {
			t.Fatalf("failed to write timeline chunk: %v", err)
		}

		clientPool := &pbv6.InterningPoolChunk{
			Strings: []*pbv6.InternString{
				{Id: proto.Uint32(1), Value: proto.String("secret-ns")},
				{Id: proto.Uint32(2), Value: proto.String("app-secret-cluster")},
			},
		}
		if err := w.WriteChunk(ChunkTypeInternPool, clientPool); err != nil {
			t.Fatalf("failed to write client intern pool chunk: %v", err)
		}

		serverPool := &pbv6.InterningPoolChunk{
			Strings: []*pbv6.InternString{
				{Id: proto.Uint32(id.ServerStringIDBase + 1), Value: proto.String("metadata\x00namespace")},
				{Id: proto.Uint32(id.ServerStringIDBase + 2), Value: proto.String("secret-ns")},
			},
			FieldPathSets: []*pbv6.InternFieldPathSet{
				{
					Id:                 proto.Uint32(1),
					FieldPathStringIds: []uint32{id.ServerStringIDBase + 1},
				},
			},
			Structs: []*pb.InternedStruct{
				{
					Id:             proto.Uint32(1),
					FieldPathSetId: proto.Uint32(1),
					Values: []*pb.InternedValue{
						{Kind: &pb.InternedValue_StringValue{StringValue: id.ServerStringIDBase + 2}},
					},
				},
			},
		}
		if err := w.WriteChunk(ChunkTypeServerInternPool, serverPool); err != nil {
			t.Fatalf("failed to write server intern pool chunk: %v", err)
		}

		if err := w.Close(); err != nil {
			t.Fatalf("failed to close fixture writer: %v", err)
		}
		return buf.Bytes()
	}

	readSnapshot := func(t *testing.T, data []byte) exportedSnapshot {
		t.Helper()
		r, err := NewReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("failed to read exported archive: %v", err)
		}

		pool := NewReadonlyInternPool()
		rawTimelineNameIDs := make(map[uint32]uint32)
		rawLogSummaryIDs := make(map[uint32]uint32)
		var snap exportedSnapshot

		for {
			chunk, err := r.NextChunk()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("failed to read chunk: %v", err)
			}

			switch chunk.Type {
			case ChunkTypeMetadata:
				var m pbv6.MetadataChunk
				if err := proto.Unmarshal(chunk.Data, &m); err != nil {
					t.Fatalf("failed to unmarshal metadata: %v", err)
				}
				for _, item := range m.Metadata {
					if h := item.GetHeader(); h != nil {
						snap.InspectionName = h.GetInspectionName()
						snap.SuggestedFilename = h.GetSuggestedFilename()
					}
					if q := item.GetQuery(); q != nil && len(q.Queries) > 0 {
						snap.QueryName = q.Queries[0].GetName()
						snap.QueryText = q.Queries[0].GetQuery()
					}
				}
			case ChunkTypeInternPool, ChunkTypeServerInternPool:
				var p pbv6.InterningPoolChunk
				if err := proto.Unmarshal(chunk.Data, &p); err != nil {
					t.Fatalf("failed to unmarshal pool: %v", err)
				}
				pool.IngestChunk(&p)
			case ChunkTypeLog:
				var l pbv6.LogChunk
				if err := proto.Unmarshal(chunk.Data, &l); err != nil {
					t.Fatalf("failed to unmarshal log chunk: %v", err)
				}
				for _, entry := range l.Logs {
					rawLogSummaryIDs[entry.GetId()] = entry.GetSummaryStringId()
				}
			case ChunkTypeTimeline:
				var tl pbv6.TimelineChunk
				if err := proto.Unmarshal(chunk.Data, &tl); err != nil {
					t.Fatalf("failed to unmarshal timeline chunk: %v", err)
				}
				for _, entry := range tl.Timelines {
					rawTimelineNameIDs[entry.GetId()] = entry.GetNameStringId()
				}
			}
		}

		snap.TimelineNames = make(map[uint32]string, len(rawTimelineNameIDs))
		for tlID, strID := range rawTimelineNameIDs {
			snap.TimelineNames[tlID] = pool.ResolveStringFromID(strID)
		}
		snap.LogSummaries = make(map[uint32]string, len(rawLogSummaryIDs))
		for logID, strID := range rawLogSummaryIDs {
			snap.LogSummaries[logID] = pool.ResolveStringFromID(strID)
		}

		yamlStr, err := NewDirectYAMLSerializer().SerializeFlatStruct(1, pool)
		if err != nil {
			t.Fatalf("failed to serialize masked struct: %v", err)
		}
		snap.BodyYAML = yamlStr
		return snap
	}

	sourceArchive := buildFixtureArchive(t)

	testCases := []struct {
		name  string
		rules []MaskRule
		want  exportedSnapshot
	}{
		{
			name: "masks metadata, client strings, and server struct strings with numbered placeholders and safe filename",
			rules: []MaskRule{
				{Target: "secret-cluster", Replacement: DefaultMaskReplacement(1)},
				{Target: "secret-ns", Replacement: DefaultMaskReplacement(2)},
			},
			want: exportedSnapshot{
				InspectionName:    "Cluster <masked_1> (Masked)",
				SuggestedFilename: "masked_1-masked.khi",
				QueryName:         "Audit query for <masked_2>",
				QueryText:         `resource.labels.cluster_name="<masked_1>" AND "<masked_2>"`,
				TimelineNames: map[uint32]string{
					10: "<masked_2>",
					11: "app-<masked_1>",
				},
				LogSummaries: map[uint32]string{
					1: "<masked_2>",
					2: "app-<masked_1>",
				},
				BodyYAML: "metadata:\n  namespace: \"<masked_2>\"\n",
			},
		},
		{
			name: "explicit empty replacement deletes substrings across metadata, timelines, and logs",
			rules: []MaskRule{
				{Target: "secret-ns", Replacement: ""},
				{Target: "-secret-cluster", Replacement: ""},
			},
			want: exportedSnapshot{
				InspectionName:    "Cluster secret-cluster (Masked)",
				SuggestedFilename: "secret-cluster-masked.khi",
				QueryName:         "Audit query for ",
				QueryText:         `resource.labels.cluster_name="secret-cluster" AND ""`,
				TimelineNames: map[uint32]string{
					10: "",
					11: "app",
				},
				LogSummaries: map[uint32]string{
					1: "",
					2: "app",
				},
				BodyYAML: "metadata:\n  namespace: \"\"\n",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			var recordedProgress []float64
			err := ExportMaskedKHIFile(
				context.Background(),
				bytes.NewReader(sourceArchive),
				int64(len(sourceArchive)),
				&out,
				NewMaskReplacer(tc.rules),
				func(progress float64, message string) error {
					if message == "" {
						t.Errorf("onProgress(%f) received empty status message", progress)
					}
					recordedProgress = append(recordedProgress, progress)
					return nil
				},
			)
			if err != nil {
				t.Fatalf("ExportMaskedKHIFile() unexpected error: %v", err)
			}

			if len(recordedProgress) < 3 {
				t.Fatalf("onProgress called %d times, want at least 3 updates", len(recordedProgress))
			}
			if recordedProgress[0] != 0.0 {
				t.Errorf("initial progress = %f, want 0.0", recordedProgress[0])
			}
			if recordedProgress[len(recordedProgress)-1] != 1.0 {
				t.Errorf("final progress = %f, want 1.0", recordedProgress[len(recordedProgress)-1])
			}
			for i := 1; i < len(recordedProgress); i++ {
				if recordedProgress[i] < recordedProgress[i-1] {
					t.Errorf("progress decreased at step %d: %f < %f", i, recordedProgress[i], recordedProgress[i-1])
				}
			}

			got := readSnapshot(t, out.Bytes())
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ExportMaskedKHIFile() snapshot mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("propagates progress callback cancellation error", func(t *testing.T) {
		wantErr := errors.New("client stream aborted")
		var out bytes.Buffer
		err := ExportMaskedKHIFile(
			context.Background(),
			bytes.NewReader(sourceArchive),
			int64(len(sourceArchive)),
			&out,
			NewMaskReplacer([]MaskRule{{Target: "secret-ns", Replacement: "<masked_1>"}}),
			func(progress float64, _ string) error {
				if progress > 0.0 {
					return wantErr
				}
				return nil
			},
		)
		if !errors.Is(err, wantErr) {
			t.Errorf("ExportMaskedKHIFile() error = %v, want %v", err, wantErr)
		}
	})
}
