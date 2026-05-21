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

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

// mockMetadata represents a simple test implementation of inspectionmetadata.Metadata.
type mockMetadata struct {
	payload         string
	includeInResult bool
}

func (m *mockMetadata) ToSerializable() interface{} {
	return map[string]string{
		"data": m.payload,
	}
}

func (m *mockMetadata) Labels() *typedmap.ReadonlyTypedMap {
	labels := typedmap.NewTypedMap()
	if m.includeInResult {
		typedmap.Set(labels, inspectionmetadata.LabelKeyIncludedInResultBinaryFlag, true)
	}
	return labels.AsReadonly()
}

func TestMetadataAccumulator(t *testing.T) {
	testCases := []struct {
		name       string
		setupInput func() inspectionmetadata.Metadata
		wantResult *pb.MetadataItem // nil if it should be ignored
	}{
		{
			name: "metadata without IncludeInResultBinary label is ignored",
			setupInput: func() inspectionmetadata.Metadata {
				return &mockMetadata{payload: "ignored", includeInResult: false}
			},
			wantResult: nil,
		},
		{
			name: "header metadata is mapped to HeaderMetadata proto",
			setupInput: func() inspectionmetadata.Metadata {
				return &inspectionmetadata.HeaderMetadata{
					InspectionType: "test-type",
					InspectionName: "test-name",
					FileSize:       1024,
				}
			},
			wantResult: &pb.MetadataItem{
				Payload: &pb.MetadataItem_Header{
					Header: &pb.HeaderMetadata{
						InspectionType:         proto.String("test-type"),
						InspectionName:         proto.String("test-name"),
						InspectionTypeIconPath: proto.String(""),
						StartTimeUnixSeconds:   proto.Int64(0),
						EndTimeUnixSeconds:     proto.Int64(0),
						InspectTimeUnixSeconds: proto.Int64(0),
						SuggestedFilename:      proto.String(""),
						FileSize:               proto.Int64(1024),
					},
				},
			},
		},
		{
			name: "query metadata is mapped to QueryMetadata proto",
			setupInput: func() inspectionmetadata.Metadata {
				m := inspectionmetadata.NewQueryMetadata()
				m.SetQuery("q1", "Query 1", "SELECT *")
				return m
			},
			wantResult: &pb.MetadataItem{
				Payload: &pb.MetadataItem_Query{
					Query: &pb.QueryMetadata{
						Queries: []*pb.QueryItem{
							{Id: proto.String("q1"), Name: proto.String("Query 1"), Query: proto.String("SELECT *")},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			acc := NewMetadataAccumulator()
			input := tc.setupInput()

			err := acc.AddMetadata(input)
			if err != nil {
				t.Fatalf("unexpected error from AddMetadata: %v", err)
			}

			result := acc.Accumulate()

			if tc.wantResult == nil {
				if len(result) != 0 {
					t.Fatalf("expected 0 metadata items, got %d", len(result))
				}
			} else {
				if len(result) != 1 {
					t.Fatalf("expected 1 metadata item, got %d", len(result))
				}
				if diff := cmp.Diff(tc.wantResult, result[0], protocmp.Transform()); diff != "" {
					t.Errorf("accumulated MetadataItem mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMetadataAccumulator_MultipleItems(t *testing.T) {
	acc := NewMetadataAccumulator()

	err := acc.AddMetadata(&inspectionmetadata.HeaderMetadata{InspectionName: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := inspectionmetadata.NewQueryMetadata()
	m.SetQuery("q1", "Query 1", "resource.type='k8s_pod'")
	err = acc.AddMetadata(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := acc.Accumulate()
	if len(result) != 2 {
		t.Fatalf("expected 2 metadata items, got %d", len(result))
	}
}
