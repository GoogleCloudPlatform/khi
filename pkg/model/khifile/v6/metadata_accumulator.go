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
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"google.golang.org/protobuf/proto"
)

// MetadataAccumulator facilitates the thread-safe accumulation of inspection metadata.
// It filters metadata based on the IncludeInResultBinary label, maps known types to
// strictly defined Protobuf messages (when available), and falls back to standard
// JSON serialization for unknown or third-party metadata types.
type MetadataAccumulator struct {
	metadata []*pb.MetadataItem
	mu       sync.RWMutex
}

// NewMetadataAccumulator creates a new instance of MetadataAccumulator.
func NewMetadataAccumulator() *MetadataAccumulator {
	return &MetadataAccumulator{
		metadata: make([]*pb.MetadataItem, 0),
	}
}

// AddMetadata processes and adds a metadata item to the accumulator.
// If the metadata does not have the IncludeInResultBinary label set to true,
// it is safely ignored.
func (a *MetadataAccumulator) AddMetadata(m inspectionmetadata.Metadata) error {
	labels := m.Labels()
	include, found := typedmap.Get(labels, inspectionmetadata.LabelKeyIncludedInResultBinaryFlag)
	if !found || !include {
		return nil
	}

	serializable := m.ToSerializable()

	var item *pb.MetadataItem

	switch v := serializable.(type) {
	case *inspectionmetadata.HeaderMetadata:
		item = toHeaderMetadataItem(v)
	case []*inspectionmetadata.QueryItem:
		item = toQueryMetadataItem(v)
	default:
		return fmt.Errorf("unknown metadata type: %T", v)
	}

	a.mu.Lock()
	a.metadata = append(a.metadata, item)
	a.mu.Unlock()

	return nil
}

func toHeaderMetadataItem(v *inspectionmetadata.HeaderMetadata) *pb.MetadataItem {
	return &pb.MetadataItem{
		Payload: &pb.MetadataItem_Header{
			Header: &pb.HeaderMetadata{
				InspectionType:         proto.String(v.InspectionType),
				InspectionName:         proto.String(v.InspectionName),
				InspectionTypeIconPath: proto.String(v.InspectionTypeIconPath),
				StartTimeUnixSeconds:   proto.Int64(v.StartTimeUnixSeconds),
				EndTimeUnixSeconds:     proto.Int64(v.EndTimeUnixSeconds),
				InspectTimeUnixSeconds: proto.Int64(v.InspectTimeUnixSeconds),
				SuggestedFilename:      proto.String(v.SuggestedFileName),
				FileSize:               proto.Int64(int64(v.FileSize)),
			},
		},
	}
}

func toQueryMetadataItem(v []*inspectionmetadata.QueryItem) *pb.MetadataItem {
	queries := make([]*pb.QueryItem, len(v))
	for i, q := range v {
		queries[i] = &pb.QueryItem{
			Id:    proto.String(q.Id),
			Name:  proto.String(q.Name),
			Query: proto.String(q.Query),
		}
	}
	return &pb.MetadataItem{
		Payload: &pb.MetadataItem_Query{
			Query: &pb.QueryMetadata{
				Queries: queries,
			},
		},
	}
}

// Accumulate returns a slice of the accumulated *pb.MetadataItem messages.
func (a *MetadataAccumulator) Accumulate() []*pb.MetadataItem {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]*pb.MetadataItem, len(a.metadata))
	copy(result, a.metadata)
	return result
}
