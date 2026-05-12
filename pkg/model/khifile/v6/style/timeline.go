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
	_ "embed"
	"encoding/json"
	"slices"
	"sync"

	"google.golang.org/protobuf/proto"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
)

//go:embed assets/zzz-icon-codepoints.json
var iconCodepointsBytes []byte

//go:embed assets/zzz-material-icons-msdf.json
var materialIconsMSDFJSONBytes []byte

//go:embed assets/zzz-material-icons-msdf.png
var materialIconsMSDFPNGBytes []byte

var (
	iconAtlasOnce   sync.Once
	cachedIconAtlas *pb.IconAtlas
)

// GetIconAtlas returns the embedded IconAtlas instance cached globally.
func GetIconAtlas() *pb.IconAtlas {
	iconAtlasOnce.Do(func() {
		var codepoints map[string]string
		if err := json.Unmarshal(iconCodepointsBytes, &codepoints); err != nil {
			panic("failed to unmarshal embedded zzz-icon-codepoints.json: " + err.Error())
		}

		cachedIconAtlas = &pb.IconAtlas{
			MsdfIconImage:    [][]byte{materialIconsMSDFPNGBytes},
			BmfontJson:       materialIconsMSDFJSONBytes,
			NameToCodepoints: codepoints,
		}
	})
	return cachedIconAtlas
}

var (
	mu             sync.RWMutex
	severities     []*pb.Severity
	verbs          []*pb.Verb
	logTypes       []*pb.LogType
	revisionStates []*pb.RevisionState
	timelineTypes  []*pb.TimelineType
)

// reset clears the registry. This is primarily intended for testing to avoid test pollution
// across different packages testing plugin loading.
func reset() {
	mu.Lock()
	defer mu.Unlock()
	severities = nil
	verbs = nil
	logTypes = nil
	revisionStates = nil
	timelineTypes = nil
}

// Color represents a color with high dynamic range (HDR) capabilities.
type Color struct {
	R, G, B, A float32
}

func (c Color) toProto() *pb.HDRColor4 {
	return &pb.HDRColor4{
		R: proto.Float32(c.R),
		G: proto.Float32(c.G),
		B: proto.Float32(c.B),
		A: proto.Float32(c.A),
	}
}

// RegisterTimelineType registers a TimelineType, assigns a unique ID to it,
// and returns the generated pointer. This allows for global inline initialization in plugins.
func RegisterTimelineType(label string, description string, backgroundColor Color, foregroundColor Color, visible bool, sortPriority int32) *pb.TimelineType {
	mu.Lock()
	defer mu.Unlock()

	id := uint32(len(timelineTypes) + 1)
	t := &pb.TimelineType{
		Id:              proto.Uint32(id),
		Label:           proto.String(label),
		Description:     proto.String(description),
		BackgroundColor: backgroundColor.toProto(),
		ForegroundColor: foregroundColor.toProto(),
		Visible:         proto.Bool(visible),
		SortPriority:    proto.Int32(sortPriority),
	}
	timelineTypes = append(timelineTypes, t)
	return t
}

// RegisterSeverity registers a Severity, assigns a unique ID to it,
// and returns the generated pointer.
func RegisterSeverity(label string, shortLabel string, backgroundColor Color, foregroundColor Color, order int32) *pb.Severity {
	mu.Lock()
	defer mu.Unlock()

	id := uint32(len(severities) + 1)
	s := &pb.Severity{
		Id:              proto.Uint32(id),
		Label:           proto.String(label),
		ShortLabel:      proto.String(shortLabel),
		BackgroundColor: backgroundColor.toProto(),
		ForegroundColor: foregroundColor.toProto(),
		Order:           proto.Int32(order),
	}
	severities = append(severities, s)
	return s
}

// RegisterVerb registers a Verb, assigns a unique ID to it,
// and returns the generated pointer.
func RegisterVerb(label string, backgroundColor Color, foregroundColor Color, visible bool) *pb.Verb {
	mu.Lock()
	defer mu.Unlock()

	id := uint32(len(verbs) + 1)
	v := &pb.Verb{
		Id:              proto.Uint32(id),
		Label:           proto.String(label),
		BackgroundColor: backgroundColor.toProto(),
		ForegroundColor: foregroundColor.toProto(),
		Visible:         proto.Bool(visible),
	}
	verbs = append(verbs, v)
	return v
}

// RegisterLogType registers a LogType, assigns a unique ID to it,
// and returns the generated pointer.
func RegisterLogType(label string, description string, backgroundColor Color, foregroundColor Color) *pb.LogType {
	mu.Lock()
	defer mu.Unlock()

	id := uint32(len(logTypes) + 1)
	l := &pb.LogType{
		Id:              proto.Uint32(id),
		Label:           proto.String(label),
		Description:     proto.String(description),
		BackgroundColor: backgroundColor.toProto(),
		ForegroundColor: foregroundColor.toProto(),
	}
	logTypes = append(logTypes, l)
	return l
}

// RegisterRevisionState registers a RevisionState, assigns a unique ID to it,
// and returns the generated pointer.
func RegisterRevisionState(label string, icon string, description string, backgroundColor Color, style pb.RevisionStateStyle) *pb.RevisionState {
	mu.Lock()
	defer mu.Unlock()

	id := uint32(len(revisionStates) + 1)
	rs := &pb.RevisionState{
		Id:              proto.Uint32(id),
		Label:           proto.String(label),
		Icon:            proto.String(icon),
		Description:     proto.String(description),
		BackgroundColor: backgroundColor.toProto(),
		Style:           &style,
	}
	revisionStates = append(revisionStates, rs)
	return rs
}

// GenerateChunk generates the final TimelineStyleChunk containing all registered styles.
func GenerateChunk() *pb.TimelineStyleChunk {
	mu.RLock()
	defer mu.RUnlock()

	// Create shallow copies of the slices to avoid concurrent modification issues
	// if someone decides to append while another is iterating over the chunk.
	return &pb.TimelineStyleChunk{
		Severities:     slices.Clone(severities),
		Verbs:          slices.Clone(verbs),
		LogTypes:       slices.Clone(logTypes),
		RevisionStates: slices.Clone(revisionStates),
		TimelineTypes:  slices.Clone(timelineTypes),
		IconAtlas:      GetIconAtlas(),
	}
}
