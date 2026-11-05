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
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/md"
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/RoaringBitmap/roaring/v2"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultHeaderSlack = time.Second

type resolvedAnchorPoint struct {
	time          time.Time
	timelineID    uint32
	logID         uint32
	revisionIndex *int
}

func resolveAnchor(
	wb *workbench.Workbench,
	inspectionID string,
	applied workbench.AppliedFilter,
	rawFilter workbench.Filter,
	fieldPath string,
	anchor Anchor,
) (resolvedAnchorPoint, *mcpsdk.CallToolResult, error) {
	hasLog := anchor.LogID != 0
	hasRev := anchor.RevisionIndex != nil
	hasTime := anchor.Time != nil && !anchor.Time.IsZero()
	count := 0
	if hasLog {
		count++
	}
	if hasRev {
		count++
	}
	if hasTime {
		count++
	}
	if count != 1 {
		res, _, _ := mdtemplate.ErrorResult(
			"INVALID_ANCHOR",
			fmt.Sprintf("Specify exactly one of `logId`, `revisionIndex`, or `time` in `%s`.", fieldPath),
		)
		return resolvedAnchorPoint{}, res, nil
	}

	idx := wb.SearchIndex()
	var pt resolvedAnchorPoint

	switch {
	case hasLog:
		logEntry := idx.GetLog(anchor.LogID)
		if logEntry == nil || logEntry.ID == 0 {
			res, _, _ := mdtemplate.ErrorResult(
				"LOG_NOT_FOUND",
				fmt.Sprintf("No log with ID %d exists in inspection %s.", anchor.LogID, md.Code(inspectionID)),
				"Use a log ID from `search_logs` or `get_timeline_logs`.",
			)
			return resolvedAnchorPoint{}, res, nil
		}
		pt.logID = anchor.LogID
		pt.time = time.Unix(0, logEntry.Timestamp).UTC()
		if anchor.TimelineID != 0 {
			if idx == nil || idx.TimelineMap[anchor.TimelineID] == nil {
				res, _, _ := timelineNotFoundResult(inspectionID, anchor.TimelineID)
				return resolvedAnchorPoint{}, res, nil
			}
			pt.timelineID = anchor.TimelineID
		} else {
			pt.timelineID = selectPrimaryTimelineForLog(idx, anchor.LogID)
		}

	case hasRev:
		if anchor.TimelineID == 0 {
			res, _, _ := mdtemplate.ErrorResult(
				"INVALID_ANCHOR",
				fmt.Sprintf("`%s.timelineId` is required when specifying `revisionIndex`.", fieldPath),
			)
			return resolvedAnchorPoint{}, res, nil
		}
		edge := anchor.RevisionEdge
		if edge == "" {
			edge = RevisionEdgeStart
		}
		if edge != RevisionEdgeStart && edge != RevisionEdgeEnd {
			res, _, _ := mdtemplate.ErrorResult(
				"INVALID_ANCHOR",
				fmt.Sprintf("Unsupported `%s.revisionEdge` `%s` (use `start` or `end`).", fieldPath, edge),
			)
			return resolvedAnchorPoint{}, res, nil
		}
		bounds, err := wb.ResolveRevisionBounds(anchor.TimelineID, *anchor.RevisionIndex)
		if err != nil {
			if errRes, handled := revisionLookupErrorResult(err, inspectionID, anchor.TimelineID); handled {
				return resolvedAnchorPoint{}, errRes, nil
			}
			return resolvedAnchorPoint{}, nil, err
		}
		revIdx := *anchor.RevisionIndex
		pt.timelineID = anchor.TimelineID
		pt.logID = bounds.Revision.LogID
		pt.revisionIndex = &revIdx
		pt.time = bounds.Revision.Time
		if edge == RevisionEdgeEnd {
			if bounds.NextRevisionTime != nil {
				pt.time = *bounds.NextRevisionTime
			} else if !applied.EndTime.IsZero() && applied.EndTime.After(bounds.Revision.Time) {
				pt.time = applied.EndTime
			}
		}

	case hasTime:
		pt.time = anchor.Time.UTC()
		if anchor.TimelineID != 0 {
			if idx == nil || idx.TimelineMap[anchor.TimelineID] == nil {
				res, _, _ := timelineNotFoundResult(inspectionID, anchor.TimelineID)
				return resolvedAnchorPoint{}, res, nil
			}
			pt.timelineID = anchor.TimelineID
		}
	}

	if errRes := checkViewportRange(pt.time, applied, rawFilter, fieldPath); errRes != nil {
		return resolvedAnchorPoint{}, errRes, nil
	}
	return pt, nil, nil
}

func checkViewportRange(anchorTime time.Time, applied workbench.AppliedFilter, rawFilter workbench.Filter, fieldPath string) *mcpsdk.CallToolResult {
	if anchorTime.IsZero() {
		return nil
	}
	minBound := applied.StartTime
	if !minBound.IsZero() && rawFilter.StartTime == nil {
		minBound = minBound.Add(-defaultHeaderSlack)
	}
	maxBound := applied.EndTime
	if !maxBound.IsZero() && rawFilter.EndTime == nil {
		maxBound = maxBound.Add(defaultHeaderSlack)
	}

	if (!minBound.IsZero() && anchorTime.Before(minBound)) || (!maxBound.IsZero() && anchorTime.After(maxBound)) {
		res, _, _ := mdtemplate.ErrorResult(
			"ANCHOR_OUT_OF_VIEWPORT",
			fmt.Sprintf("Resolved anchor `%s` at `%s` falls outside the rendered viewport (`%s` to `%s`).",
				fieldPath,
				mdtemplate.FormatTime(anchorTime),
				mdtemplate.FormatTime(applied.StartTime),
				mdtemplate.FormatTime(applied.EndTime),
			),
			"Widen `filter.startTime` / `filter.endTime` (or omit them to render the full inspection duration).",
		)
		return res
	}
	return nil
}

func resolveScopedTimelineIDs(
	ctx context.Context,
	wb *workbench.Workbench,
	inspectionID string,
	fieldPath string,
	query string,
	timelineIDs []uint32,
	includeSubtree bool,
	autoInclude *roaring.Bitmap,
) ([]uint32, *mcpsdk.CallToolResult, error) {
	idx := wb.SearchIndex()
	if idx == nil {
		return nil, nil, fmt.Errorf("search index is not ready")
	}

	matched := roaring.NewBitmap()
	if query != "" {
		if _, err := workbench.ValidateFilter(workbench.Filter{TimelineQuery: query}); err != nil {
			res, _, celErr := mdtemplate.CELErrorResult(fieldPath+".timelineQuery", query, err)
			return nil, res, celErr
		}
		stages := []workbench.TimelineFilter{workbench.NewTimelineCELFilter(query)}
		if includeSubtree {
			stages = append(stages, workbench.NewIncludeDescendantsFilter())
		}
		fctx, err := workbench.NewPipeline(stages...).ExecuteToFilterContext(ctx, idx, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to evaluate scoped timelineQuery: %w", err)
		}
		matched.Or(fctx.TimelineIDs)
		autoInclude.Or(matched)
	}

	for _, tid := range timelineIDs {
		if tid == 0 {
			res, _, _ := mdtemplate.ErrorResult("INVALID_ARGUMENT", fmt.Sprintf("`%s` contains invalid ID 0.", fieldPath))
			return nil, res, nil
		}
		if idx.TimelineMap[tid] == nil {
			res, _, _ := timelineNotFoundResult(inspectionID, tid)
			return nil, res, nil
		}
		autoInclude.Add(tid)
		matched.Add(tid)
		if includeSubtree {
			addDescendants(idx, tid, matched)
		}
	}

	return matched.ToArray(), nil, nil
}

func addDescendants(idx *workbench.SearchIndex, timelineID uint32, dest *roaring.Bitmap) {
	tl := idx.TimelineMap[timelineID]
	if tl == nil {
		return
	}
	for _, childID := range tl.ChildrenIDs {
		if dest.Contains(childID) {
			continue
		}
		dest.Add(childID)
		addDescendants(idx, childID, dest)
	}
}

func selectPrimaryTimelineForLog(idx *workbench.SearchIndex, logID uint32) uint32 {
	if idx == nil {
		return 0
	}
	linked := idx.GetTimelineIDsForLog(logID)
	if len(linked) == 0 {
		return 0
	}
	bestID := linked[0]
	bestDepth := timelineDepth(idx, bestID)
	for _, candidateID := range linked[1:] {
		d := timelineDepth(idx, candidateID)
		if d > bestDepth || (d == bestDepth && candidateID < bestID) {
			bestID = candidateID
			bestDepth = d
		}
	}
	return bestID
}

func timelineDepth(idx *workbench.SearchIndex, timelineID uint32) int {
	segments, ok := idx.TimelineSegments(timelineID)
	if !ok {
		return 0
	}
	return len(segments)
}

func revisionLookupErrorResult(err error, inspectionID string, timelineID uint32) (*mcpsdk.CallToolResult, bool) {
	switch {
	case errors.Is(err, workbench.ErrTimelineNotFound):
		res, _, _ := timelineNotFoundResult(inspectionID, timelineID)
		return res, true
	case errors.Is(err, workbench.ErrRevisionNotFound):
		res, _, _ := mdtemplate.ErrorResult("REVISION_NOT_FOUND",
			fmt.Sprintf("Timeline %d has no matching revision: %s.", timelineID, err.Error()),
			"Use a revision index from `get_resource_revisions`.",
		)
		return res, true
	default:
		return nil, false
	}
}

func timelineNotFoundResult(inspectionID string, timelineID uint32) (*mcpsdk.CallToolResult, any, error) {
	return mdtemplate.ErrorResult("TIMELINE_NOT_FOUND",
		fmt.Sprintf("No timeline with ID %d exists in inspection %s.", timelineID, md.Code(inspectionID)),
		"Use a timeline ID from `search_timelines` or `search_logs`.",
	)
}

func formatShiftedClock(t time.Time, shiftHours float64) string {
	if t.IsZero() {
		return ""
	}
	offsetSec := int(math.Round(shiftHours * 3600))
	shifted := t.In(time.FixedZone("shifted", offsetSec))
	if offsetSec == 0 {
		return shifted.Format("15:04:05Z")
	}
	sign := "+"
	absSec := offsetSec
	if absSec < 0 {
		sign = "-"
		absSec = -absSec
	}
	h := absSec / 3600
	m := (absSec % 3600) / 60
	if m == 0 {
		return fmt.Sprintf("%s (UTC%s%d)", shifted.Format("15:04:05"), sign, h)
	}
	return fmt.Sprintf("%s (UTC%s%d:%02d)", shifted.Format("15:04:05"), sign, h, m)
}

func formatElapsedDelta(d time.Duration, signed bool) string {
	neg := d < 0
	if neg {
		d = -d
	}
	prefix := ""
	if signed {
		if neg {
			prefix = "-"
		} else {
			prefix = "+"
		}
	}
	if d == 0 {
		return prefix + "0s"
	}
	if d < time.Second {
		return fmt.Sprintf("%s%dms", prefix, d.Milliseconds())
	}
	totalSec := int64(math.Round(d.Seconds()))
	if totalSec < 60 {
		return fmt.Sprintf("%s%ds", prefix, totalSec)
	}
	mins := totalSec / 60
	secs := totalSec % 60
	if mins < 60 {
		if secs == 0 {
			return fmt.Sprintf("%s%dm", prefix, mins)
		}
		return fmt.Sprintf("%s%dm %ds", prefix, mins, secs)
	}
	hours := mins / 60
	remMins := mins % 60
	if remMins == 0 {
		return fmt.Sprintf("%s%dh", prefix, hours)
	}
	return fmt.Sprintf("%s%dh %dm", prefix, hours, remMins)
}
