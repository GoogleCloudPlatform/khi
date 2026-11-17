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
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/RoaringBitmap/roaring/v2"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

var (
	validSpanPatterns        = []SpanPattern{SpanPatternSolid, SpanPatternHatched, SpanPatternOutline}
	validSpanLabelPlacements = []SpanLabelPlacement{SpanLabelPlacementAuto, SpanLabelPlacementAbove, SpanLabelPlacementInside}
)

type spanResolveContext struct {
	wb           *workbench.Workbench
	inspectionID string
	spec         *Spec
	applied      workbench.AppliedFilter
	paletteReg   *paletteRegistry
	autoInclude  *roaring.Bitmap
	overlay      *Overlay
}

type validatedSpanStyle struct {
	pattern        SpanPattern
	labelPlacement SpanLabelPlacement
	paletteID      string
}

type spanBounds struct {
	start resolvedAnchorPoint
	end   resolvedAnchorPoint
}

// resolveSpans validates and resolves all span annotations into env.overlay.Spans.
func resolveSpans(ctx context.Context, env spanResolveContext) (*mcpsdk.CallToolResult, error) {
	for i := range env.spec.Annotations.Spans {
		s := &env.spec.Annotations.Spans[i]
		resolved, errRes, err := resolveSingleSpan(ctx, env, i, s)
		if err != nil || errRes != nil {
			return errRes, err
		}
		env.overlay.Spans = append(env.overlay.Spans, resolved)
	}
	return nil, nil
}

// resolveSingleSpan validates and resolves one SpanAnnotation at index into a ResolvedSpan.
func resolveSingleSpan(
	ctx context.Context,
	env spanResolveContext,
	index int,
	s *SpanAnnotation,
) (ResolvedSpan, *mcpsdk.CallToolResult, error) {
	fieldPath := fmt.Sprintf("annotations.spans[%d]", index)
	style, errRes := validateSpanStyle(env.paletteReg, fieldPath, s)
	if errRes != nil {
		return ResolvedSpan{}, errRes, nil
	}

	bounds, errRes, err := resolveSpanBounds(env, fieldPath, s)
	if err != nil || errRes != nil {
		return ResolvedSpan{}, errRes, err
	}

	var scopedIDs []uint32
	if s.TimelineQuery != "" || len(s.TimelineIDs) > 0 {
		includeSubtree := s.IncludeSubtree == nil || *s.IncludeSubtree
		scopedIDs, errRes, err = resolveScopedTimelineIDs(ctx, env.wb, env.inspectionID, fieldPath, s.TimelineQuery, s.TimelineIDs, includeSubtree, env.autoInclude)
		if err != nil || errRes != nil {
			return ResolvedSpan{}, errRes, err
		}
	}

	durationLabel := ""
	if !s.HideDuration {
		durationLabel = formatElapsedDelta(bounds.end.time.Sub(bounds.start.time), false)
	}
	return ResolvedSpan{
		ID:             fmt.Sprintf("span-%d", index+1),
		Label:          s.Label,
		StartTimeMs:    bounds.start.time.UnixMilli(),
		EndTimeMs:      bounds.end.time.UnixMilli(),
		DurationLabel:  durationLabel,
		TimelineIDs:    scopedIDs,
		Pattern:        style.pattern,
		LabelPlacement: style.labelPlacement,
		HideDuration:   s.HideDuration,
		PaletteID:      style.paletteID,
	}, nil, nil
}

// validateSpanStyle validates a span's label, pattern, label placement, and explicit palette ID.
func validateSpanStyle(reg *paletteRegistry, fieldPath string, s *SpanAnnotation) (validatedSpanStyle, *mcpsdk.CallToolResult) {
	if strings.TrimSpace(s.Label) == "" {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.label` is required.", fieldPath))
		return validatedSpanStyle{}, res
	}
	if utf8.RuneCountInString(s.Label) > maxAnnotationRunes {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s.label` exceeds maximum length of %d characters.", fieldPath, maxAnnotationRunes))
		return validatedSpanStyle{}, res
	}
	pattern := s.Pattern
	if pattern == "" {
		pattern = SpanPatternSolid
	}
	if !slices.Contains(validSpanPatterns, pattern) {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Unsupported `%s.pattern` `%s` (use `solid`, `hatched`, or `outline`).", fieldPath, pattern))
		return validatedSpanStyle{}, res
	}
	labelPlacement := s.LabelPlacement
	if labelPlacement == "" {
		labelPlacement = SpanLabelPlacementAuto
	}
	if !slices.Contains(validSpanLabelPlacements, labelPlacement) {
		res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("Unsupported `%s.labelPlacement` `%s` (use `auto`, `above`, or `inside`).", fieldPath, labelPlacement))
		return validatedSpanStyle{}, res
	}
	paletteID, errRes := validateExplicitPaletteID(reg, fieldPath, s.PaletteID)
	if errRes != nil {
		return validatedSpanStyle{}, errRes
	}
	return validatedSpanStyle{
		pattern:        pattern,
		labelPlacement: labelPlacement,
		paletteID:      paletteID,
	}, nil
}

// resolveSpanBounds resolves start and end anchors, enforces chronological order, and registers auto-included timelines.
func resolveSpanBounds(
	env spanResolveContext,
	fieldPath string,
	s *SpanAnnotation,
) (spanBounds, *mcpsdk.CallToolResult, error) {
	startPt, errRes, err := resolveAnchor(env.wb, env.inspectionID, env.applied, env.spec.Filter, fieldPath+".start", s.Start)
	if err != nil || errRes != nil {
		return spanBounds{}, errRes, err
	}
	endPt, errRes, err := resolveAnchor(env.wb, env.inspectionID, env.applied, env.spec.Filter, fieldPath+".end", s.End)
	if err != nil || errRes != nil {
		return spanBounds{}, errRes, err
	}
	if endPt.time.Before(startPt.time) {
		res, _, _ := mdtemplate.ErrorResult(
			"INVALID_ANCHOR",
			fmt.Sprintf("`%s.end` (%s) must not be before `%s.start` (%s).", fieldPath, mdtemplate.FormatTime(endPt.time), fieldPath, mdtemplate.FormatTime(startPt.time)),
		)
		return spanBounds{}, res, nil
	}
	if startPt.timelineID != 0 {
		env.autoInclude.Add(startPt.timelineID)
	}
	if endPt.timelineID != 0 {
		env.autoInclude.Add(endPt.timelineID)
	}
	return spanBounds{start: startPt, end: endPt}, nil, nil
}
