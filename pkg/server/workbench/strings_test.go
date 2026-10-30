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

package workbench

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile"
	pbv6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
)

func buildTestStringPoolKHIBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := khifilev6model.NewWriter(&buf)
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}

	meta := &pbv6.MetadataChunk{
		Metadata: []*pbv6.MetadataItem{
			{
				Payload: &pbv6.MetadataItem_Header{
					Header: &pbv6.HeaderMetadata{
						InspectionType:    proto.String("gcp-gke"),
						InspectionName:    proto.String("Prod GKE Cluster"),
						SuggestedFilename: proto.String("prod-gke.khi"),
					},
				},
			},
			{
				Payload: &pbv6.MetadataItem_Query{
					Query: &pbv6.QueryMetadata{
						Queries: []*pbv6.QueryItem{
							{
								Id:    proto.String("q1"),
								Name:  proto.String("Audit Logs"),
								Query: proto.String(`resource.labels.project_id="my-secret-proj"`),
							},
						},
					},
				},
			},
		},
	}
	if err := w.WriteChunk(khifilev6model.ChunkTypeMetadata, meta); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	clientPool := &pbv6.InterningPoolChunk{
		Strings: []*pbv6.InternString{
			{Id: proto.Uint32(1), Value: proto.String("core/v1")},
			{Id: proto.Uint32(2), Value: proto.String("pod")},
			{Id: proto.Uint32(3), Value: proto.String("cluster-scope")},
			{Id: proto.Uint32(4), Value: proto.String("payments-ns")},
			{Id: proto.Uint32(5), Value: proto.String("checkout-pod-abc")},
			{Id: proto.Uint32(6), Value: proto.String("binding")},
			{Id: proto.Uint32(7), Value: proto.String("user:alice@acme.corp")},
			{Id: proto.Uint32(8), Value: proto.String("log summary with token-xyz")},
		},
	}
	if err := w.WriteChunk(khifilev6model.ChunkTypeInternPool, clientPool); err != nil {
		t.Fatalf("failed to write client pool: %v", err)
	}

	serverPool := &pbv6.InterningPoolChunk{
		Strings: []*pbv6.InternString{
			{Id: proto.Uint32(id.ServerStringIDBase + 1), Value: proto.String("spec\x00nodeName")},
			{Id: proto.Uint32(id.ServerStringIDBase + 2), Value: proto.String("gke-secret-pool-node-1")},
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
	if err := w.WriteChunk(khifilev6model.ChunkTypeServerInternPool, serverPool); err != nil {
		t.Fatalf("failed to write server pool: %v", err)
	}

	timelineChunk := &pbv6.TimelineChunk{
		Timelines: []*pbv6.Timeline{
			{Id: proto.Uint32(1), ParentTimelineId: proto.Uint32(0), NameStringId: proto.Uint32(1)},                                     // Layer 0: core/v1
			{Id: proto.Uint32(2), ParentTimelineId: proto.Uint32(1), NameStringId: proto.Uint32(2)},                                     // Layer 1: pod
			{Id: proto.Uint32(3), ParentTimelineId: proto.Uint32(2), NameStringId: proto.Uint32(3)},                                     // Fixed structural: cluster-scope
			{Id: proto.Uint32(4), ParentTimelineId: proto.Uint32(2), NameStringId: proto.Uint32(4)},                                     // Layer 2 namespace: payments-ns
			{Id: proto.Uint32(5), ParentTimelineId: proto.Uint32(4), NameStringId: proto.Uint32(5), TimelineItemsId: proto.Uint32(100)}, // Layer 3 resource: checkout-pod-abc
			{Id: proto.Uint32(6), ParentTimelineId: proto.Uint32(5), NameStringId: proto.Uint32(6)},                                     // Fixed subresource: binding
		},
		TimelineItems: []*pbv6.TimelineItems{
			{
				Id: proto.Uint32(100),
				Revisions: []*pbv6.Revision{
					{
						LogId:             proto.Uint32(1),
						PrincipalStringId: proto.Uint32(7),
					},
				},
			},
		},
	}
	if err := w.WriteChunk(khifilev6model.ChunkTypeTimeline, timelineChunk); err != nil {
		t.Fatalf("failed to write timeline chunk: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	return buf.Bytes()
}

func TestWorkbench_LazyStringPoolIndex(t *testing.T) {
	data := buildTestStringPoolKHIBytes(t)
	wb, err := NewFromReader(context.Background(), "wb-1", "insp-1", bytes.NewReader(data), int64(len(data)), nil)
	if err != nil {
		t.Fatalf("NewFromReader() unexpected error: %v", err)
	}
	defer wb.Close()

	if wb.HasInitializedStringPoolIndex() {
		t.Fatalf("HasInitializedStringPoolIndex() = true immediately after NewFromReader, want false (lazy on-demand initialization)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, unsubscribe, err := wb.SubscribeStringPoolIndexProgress(ctx)
	if err != nil {
		t.Fatalf("SubscribeStringPoolIndexProgress() unexpected error: %v", err)
	}
	defer unsubscribe()

	if !wb.HasInitializedStringPoolIndex() {
		t.Fatalf("HasInitializedStringPoolIndex() = false after SubscribeStringPoolIndexProgress, want true")
	}

	reachedReady := false
	for ev := range ch {
		if ev.State == IndexStateFailed {
			t.Fatalf("StringPoolIndex failed during construction: %v", ev.Err)
		}
		if ev.State == IndexStateReady {
			reachedReady = true
			break
		}
	}
	if !reachedReady {
		state, _, _, statusErr := wb.StringPoolIndexStatus()
		if state != IndexStateReady {
			t.Fatalf("StringPoolIndex final state = %v (err=%v), want IndexStateReady", state, statusErr)
		}
	}

	// Verify Tier 1 ordering collects timeline names, revision principals, and header/query metadata ahead of general log strings.
	gotTop10, _, err := wb.ListStrings("", 0, 10)
	if err != nil {
		t.Fatalf("ListStrings() unexpected error: %v", err)
	}
	wantTop10 := []string{
		"core/v1",
		"pod",
		"cluster-scope",
		"payments-ns",
		"checkout-pod-abc",
		"binding",
		"user:alice@acme.corp",
		"Prod GKE Cluster",
		"prod-gke.khi",
		"Audit Logs",
	}
	if diff := cmp.Diff(wantTop10, gotTop10); diff != "" {
		t.Errorf("ListStrings(top 10 Tier 1 priority) mismatch (-want +got):\n%s", diff)
	}
}

func TestStringPoolIndex_SearchLinearFallbackAndTrigramReady(t *testing.T) {
	values := []string{
		"payments-ns",
		"checkout-pod-abc",
		"user:alice@acme.corp",
		"my-secret-proj",
		"gke-secret-pool-node-1",
		"10.240.0.15",
		"ProdCluster-East",
		"prodcluster-west",
	}

	type searchOutcome struct {
		Values []string
		Total  int
	}

	testCases := []struct {
		name     string
		buildTri bool
		query    string
		offset   int
		limit    int
		wantErr  error
		want     searchOutcome
	}{
		{
			name:     "linear fallback before trigram build completes matches >=3 rune pattern",
			buildTri: false,
			query:    "SECRET",
			offset:   0,
			limit:    10,
			want: searchOutcome{
				Values: []string{"my-secret-proj", "gke-secret-pool-node-1"},
				Total:  2,
			},
		},
		{
			name:     "trigram index ready matches >=3 rune pattern with pagination",
			buildTri: true,
			query:    "secret",
			offset:   1,
			limit:    1,
			want: searchOutcome{
				Values: []string{"gke-secret-pool-node-1"},
				Total:  2,
			},
		},
		{
			name:     "escaped regex metacharacters match literal IPv4 prefix",
			buildTri: true,
			query:    `10\.240\.`,
			offset:   0,
			limit:    10,
			want: searchOutcome{
				Values: []string{"10.240.0.15"},
				Total:  1,
			},
		},
		{
			name:     "trigram index ready falls back cleanly for 2-rune query",
			buildTri: true,
			query:    "ns",
			offset:   0,
			limit:    10,
			want: searchOutcome{
				Values: []string{"payments-ns"},
				Total:  1,
			},
		},
		{
			name:     "regex search uses trigram index for literal-constrained RE2 pattern",
			buildTri: true,
			query:    `^(payments|checkout)-[a-z-]+$`,
			offset:   0,
			limit:    10,
			want: searchOutcome{
				Values: []string{"payments-ns", "checkout-pod-abc"},
				Total:  2,
			},
		},
		{
			name:     "regex search falls back to linear scan for character-class-only IPv4 pattern when trigram index is ready",
			buildTri: true,
			query:    `^\d+\.\d+\.\d+\.\d+$`,
			offset:   0,
			limit:    10,
			want: searchOutcome{
				Values: []string{"10.240.0.15"},
				Total:  1,
			},
		},
		{
			name:     "regex search works during linear scan before trigram index is built",
			buildTri: false,
			query:    `^user:[a-z]+@acme\.corp$`,
			offset:   0,
			limit:    10,
			want: searchOutcome{
				Values: []string{"user:alice@acme.corp"},
				Total:  1,
			},
		},
		{
			name:     "regex search is case-insensitive by default and honors inline (?-i) flag",
			buildTri: true,
			query:    `(?-i)^ProdCluster`,
			offset:   0,
			limit:    10,
			want: searchOutcome{
				Values: []string{"ProdCluster-East"},
				Total:  1,
			},
		},
		{
			name:     "invalid regular expression returns ErrInvalidRegexQuery",
			buildTri: true,
			query:    `[unclosed`,
			offset:   0,
			limit:    10,
			wantErr:  ErrInvalidRegexQuery,
		},
		{
			name:     "unbalanced closing parenthesis in regex returns ErrInvalidRegexQuery",
			buildTri: false,
			query:    `secret)proj`,
			offset:   0,
			limit:    10,
			wantErr:  ErrInvalidRegexQuery,
		},
		{
			name:     "empty query paginates full priority-ordered list",
			buildTri: true,
			query:    "",
			offset:   0,
			limit:    2,
			want: searchOutcome{
				Values: []string{"payments-ns", "checkout-pod-abc"},
				Total:  8,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idx := NewStringPoolIndex(values)
			defer idx.Close()

			if tc.buildTri {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				idx.StartAsyncTrigramBuild(ctx)
				if err := idx.AwaitReady(ctx); err != nil {
					t.Fatalf("AwaitReady() unexpected error: %v", err)
				}
			}

			gotVals, gotTotal, err := idx.Search(tc.query, tc.offset, tc.limit)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Search(%q) error = %v, want %v", tc.query, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Search(%q) unexpected error: %v", tc.query, err)
			}
			got := searchOutcome{
				Values: gotVals,
				Total:  gotTotal,
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Search(%q, %d, %d) mismatch (-want +got):\n%s", tc.query, tc.offset, tc.limit, diff)
			}
		})
	}
}
