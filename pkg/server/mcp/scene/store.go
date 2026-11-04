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

package scene

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultTTL defines how long an issued timeline scene remains accessible in memory.
const DefaultTTL = 30 * time.Minute

var (
	// ErrUnknownScene indicates that a scene ID was never issued or has already been evicted.
	ErrUnknownScene = errors.New("unknown scene ID")
	// ErrExpiredScene indicates that a scene ID was accessed after its TTL elapsed.
	ErrExpiredScene = errors.New("scene has expired")
)

// Stored represents an immutable timeline scene stored in Store.
type Stored struct {
	SceneID       string
	ParentSceneID string
	ExpiresAt     time.Time
	Spec          Spec
	Document      Document
}

func (st Stored) clone() Stored {
	st.Spec = st.Spec.Clone()
	st.Document = cloneDocument(st.Document)
	return st
}

type sceneEntry struct {
	stored   Stored
	jsonData []byte
}

type errorResponse struct {
	Error string `json:"error"`
}

// Store stores immutable timeline scene definitions in memory with a TTL and serves them over HTTP.
type Store struct {
	ttl time.Duration
	now func() time.Time

	mu          sync.Mutex
	entriesByID map[string]sceneEntry
}

// NewStore creates an in-memory Store that retains issued scenes for ttl.
func NewStore(ttl time.Duration) *Store {
	return &Store{
		ttl:         ttl,
		now:         time.Now,
		entriesByID: make(map[string]sceneEntry),
	}
}

// Issue stores a new immutable scene derived from optional parentSceneID and returns its Stored record.
func (s *Store) Issue(parentSceneID string, spec Spec, doc Document) (Stored, error) {
	now := s.now().UTC()
	expiresAt := now.Add(s.ttl)
	sceneID := "scn_" + strings.ToLower(rand.Text()[:16])

	doc.SceneID = sceneID
	doc.ParentSceneID = parentSceneID
	doc.ExpiresAt = expiresAt

	jsonData, err := json.Marshal(doc)
	if err != nil {
		return Stored{}, fmt.Errorf("failed to marshal timeline scene document: %w", err)
	}

	stored := Stored{
		SceneID:       sceneID,
		ParentSceneID: parentSceneID,
		ExpiresAt:     expiresAt,
		Spec:          spec,
		Document:      doc,
	}.clone()
	ret := stored.clone()

	s.mu.Lock()
	// Sweeping expired entries on issue bounds memory growth without a background goroutine.
	for id, entry := range s.entriesByID {
		if !now.Before(entry.stored.ExpiresAt) {
			delete(s.entriesByID, id)
		}
	}
	s.entriesByID[sceneID] = sceneEntry{
		stored:   stored,
		jsonData: jsonData,
	}
	s.mu.Unlock()

	return ret, nil
}

// Get retrieves an unexpired Stored scene by sceneID without mutating the stored scene.
func (s *Store) Get(sceneID string) (Stored, error) {
	entry, err := s.lookup(sceneID)
	if err != nil {
		return Stored{}, err
	}
	return entry.stored.clone(), nil
}

// MergePatch resolves parentSceneID when set and merges patch overrides and annotations into a new immutable Spec.
func (s *Store) MergePatch(parentSceneID string, patch Spec) (Spec, *mcpsdk.CallToolResult) {
	if parentSceneID == "" {
		if patch.Annotations != nil {
			if errRes := applyClearLayers(&Annotations{}, patch.Annotations.ClearLayers); errRes != nil {
				return Spec{}, errRes
			}
		}
		merged := patch.Clone()
		if merged.Annotations != nil {
			merged.Annotations.ClearLayers = nil
			merged.Annotations.RemoveIDs = nil
			if isAnnotationsEmpty(merged.Annotations) {
				merged.Annotations = nil
			}
		}
		return merged, nil
	}

	parent, err := s.Get(parentSceneID)
	if err != nil {
		if errors.Is(err, ErrExpiredScene) {
			res, _, _ := mdtemplate.ErrorResult(
				"SCENE_EXPIRED",
				fmt.Sprintf("Parent scene `%s` has expired.", parentSceneID),
				"Create a new scene by specifying `inspectionId` and `filter`.",
			)
			return Spec{}, res
		}
		res, _, _ := mdtemplate.ErrorResult(
			"SCENE_NOT_FOUND",
			fmt.Sprintf("No scene with ID `%s` exists.", parentSceneID),
			"Create a new scene by specifying `inspectionId` and `filter`.",
		)
		return Spec{}, res
	}

	if patch.InspectionID != "" && patch.InspectionID != parent.Spec.InspectionID {
		res, _, _ := mdtemplate.ErrorResult(
			"INVALID_ARGUMENT",
			fmt.Sprintf("inspectionId `%s` does not match parentSceneId `%s` (`%s`).", patch.InspectionID, parentSceneID, parent.Spec.InspectionID),
			"Omit `inspectionId` when patching a parent scene, or omit `parentSceneId` to create a new scene.",
		)
		return Spec{}, res
	}

	merged := parent.Spec
	merged.Filter = mergeFilters(merged.Filter, patch.Filter)
	if len(patch.TimelineIDs) > 0 {
		merged.TimelineIDs = slices.Clone(patch.TimelineIDs)
	}
	if patch.MaxDepth > 0 {
		merged.MaxDepth = patch.MaxDepth
	}
	if patch.MaxNodes > 0 {
		merged.MaxNodes = patch.MaxNodes
	}
	if patch.Width > 0 {
		merged.Width = patch.Width
	}
	if patch.TimezoneShiftHours != 0 {
		merged.TimezoneShiftHours = patch.TimezoneShiftHours
	}

	if patch.Annotations != nil {
		var errRes *mcpsdk.CallToolResult
		merged.Annotations, errRes = mergeAnnotations(merged.Annotations, patch.Annotations)
		if errRes != nil {
			return Spec{}, errRes
		}
	}

	return merged, nil
}

// ServeScene serves the JSON scene document for sceneID over HTTP GET or HEAD.
func (s *Store) ServeScene(w http.ResponseWriter, r *http.Request, sceneID string) {
	entry, err := s.lookup(sceneID)
	if err != nil {
		status := http.StatusNotFound
		if errors.Is(err, ErrExpiredScene) {
			status = http.StatusGone
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(status)
		if r.Method == http.MethodHead {
			return
		}
		if encodeErr := json.NewEncoder(w).Encode(errorResponse{Error: err.Error()}); encodeErr != nil {
			slog.WarnContext(r.Context(), "failed to write MCP scene error response", "error", encodeErr)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(entry.jsonData)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := w.Write(entry.jsonData); err != nil {
		slog.WarnContext(r.Context(), "failed to write MCP scene response", "error", err)
	}
}

func (s *Store) lookup(sceneID string) (sceneEntry, error) {
	if sceneID == "" {
		return sceneEntry{}, ErrUnknownScene
	}
	now := s.now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()
	entry, found := s.entriesByID[sceneID]
	if !found {
		return sceneEntry{}, ErrUnknownScene
	}
	if !now.Before(entry.stored.ExpiresAt) {
		delete(s.entriesByID, sceneID)
		return sceneEntry{}, ErrExpiredScene
	}
	return entry, nil
}

func applyClearLayers(out *Annotations, layers []string) *mcpsdk.CallToolResult {
	for _, layer := range layers {
		switch layer {
		case "all":
			*out = Annotations{}
		case "palette":
			out.Palette = nil
		case "markers":
			out.Markers = nil
		case "spans":
			out.Spans = nil
		case "rowHighlights":
			out.RowHighlights = nil
		case "diffExcerpts":
			out.DiffExcerpts = nil
		case "logExcerpts":
			out.LogExcerpts = nil
		case "connectors":
			out.Connectors = nil
		case "callouts":
			out.Callouts = nil
		default:
			res, _, _ := mdtemplate.ErrorResult(
				"INVALID_ARGUMENT",
				fmt.Sprintf("Unknown annotation layer `%s` in `annotations.clearLayers`.", layer),
				"Supported layers: `palette`, `markers`, `spans`, `rowHighlights`, `diffExcerpts`, `logExcerpts`, `connectors`, `callouts`, `all`.",
			)
			return res
		}
	}
	return nil
}

func mergeFilters(base, override workbench.Filter) workbench.Filter {
	out := cloneFilter(base)
	if override.TimelineQuery != "" {
		out.TimelineQuery = override.TimelineQuery
	}
	if override.TimelineExclusionQuery != "" {
		out.TimelineExclusionQuery = override.TimelineExclusionQuery
	}
	if override.LogQuery != "" {
		out.LogQuery = override.LogQuery
	}
	if override.StartTime != nil {
		t := override.StartTime.UTC()
		out.StartTime = &t
	}
	if override.EndTime != nil {
		t := override.EndTime.UTC()
		out.EndTime = &t
	}
	if override.ExcludeTimelinesWithoutLogs != nil {
		v := *override.ExcludeTimelinesWithoutLogs
		out.ExcludeTimelinesWithoutLogs = &v
	}
	return out
}

func deleteNilIfEmpty[T any](s []T, del func(T) bool) []T {
	s = slices.DeleteFunc(s, del)
	if len(s) == 0 {
		return nil
	}
	return s
}

func mergeAnnotations(base, patch *Annotations) (*Annotations, *mcpsdk.CallToolResult) {
	var out Annotations
	if base != nil {
		out = *base
	}
	if errRes := applyClearLayers(&out, patch.ClearLayers); errRes != nil {
		return nil, errRes
	}

	if len(patch.RemoveIDs) > 0 {
		removeSet := make(map[string]bool, len(patch.RemoveIDs))
		for _, id := range patch.RemoveIDs {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				removeSet[trimmed] = true
			}
		}
		out.Palette = deleteNilIfEmpty(out.Palette, func(pe PaletteEntry) bool {
			return removeSet[strings.TrimSpace(pe.ID)]
		})
		out.Markers = deleteNilIfEmpty(out.Markers, func(m MarkerAnnotation) bool {
			return removeSet[m.Badge] || removeSet[m.Label]
		})
		out.Spans = deleteNilIfEmpty(out.Spans, func(sp SpanAnnotation) bool {
			return removeSet[sp.Label]
		})
		out.RowHighlights = deleteNilIfEmpty(out.RowHighlights, func(rh RowHighlight) bool {
			return removeSet[rh.RoleLabel]
		})
		out.DiffExcerpts = deleteNilIfEmpty(out.DiffExcerpts, func(de DiffExcerpt) bool {
			return removeSet[de.Badge]
		})
		out.LogExcerpts = deleteNilIfEmpty(out.LogExcerpts, func(le LogExcerpt) bool {
			return removeSet[le.Badge]
		})
		out.Connectors = deleteNilIfEmpty(out.Connectors, func(c ConnectorAnnotation) bool {
			return removeSet[c.Badge] || removeSet[c.Label]
		})
		out.Callouts = deleteNilIfEmpty(out.Callouts, func(ca CalloutAnnotation) bool {
			return removeSet[ca.Label]
		})
	}

	if patch.DimUnlistedRows != nil {
		v := *patch.DimUnlistedRows
		out.DimUnlistedRows = &v
	}
	patchCopy := cloneAnnotations(patch)
	for _, pe := range patchCopy.Palette {
		pe.ID = strings.TrimSpace(pe.ID)
		idx := slices.IndexFunc(out.Palette, func(existing PaletteEntry) bool {
			return strings.TrimSpace(existing.ID) == pe.ID
		})
		if idx >= 0 {
			out.Palette[idx] = pe
		} else {
			out.Palette = append(out.Palette, pe)
		}
	}
	out.Markers = append(out.Markers, patchCopy.Markers...)
	out.Spans = append(out.Spans, patchCopy.Spans...)
	out.RowHighlights = append(out.RowHighlights, patchCopy.RowHighlights...)
	out.DiffExcerpts = append(out.DiffExcerpts, patchCopy.DiffExcerpts...)
	out.LogExcerpts = append(out.LogExcerpts, patchCopy.LogExcerpts...)
	out.Connectors = append(out.Connectors, patchCopy.Connectors...)
	out.Callouts = append(out.Callouts, patchCopy.Callouts...)
	out.ClearLayers = nil
	out.RemoveIDs = nil

	if isAnnotationsEmpty(&out) {
		return nil, nil
	}
	return &out, nil
}
