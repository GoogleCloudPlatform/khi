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

package mcp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/mdtemplate"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

type linkedTimelineTemplateData struct {
	TimelineID string
	Segments   []workbench.TimelineSegment
}

type otherTimelineTemplateData struct {
	linkedTimelineTemplateData
	MatchedLogCount int
}

type timelineLogTemplateData struct {
	LogID      string
	Time       time.Time
	Severity   string
	LogType    string
	Summary    string
	AlsoOnCell string
}

type timelineLogsTemplateData struct {
	TimelineID              string
	Segments                []workbench.TimelineSegment
	Applied                 workbench.AppliedFilter
	Start                   int
	End                     int
	Total                   int
	TotalOtherTimelineCount int
	OtherTimelines          []otherTimelineTemplateData
	Logs                    []timelineLogTemplateData
	NextPageToken           string
}

type logTemplateData struct {
	LogID           string
	Time            time.Time
	Severity        string
	LogType         string
	Summary         string
	LinkedTimelines []linkedTimelineTemplateData
	BodySection     string
}

type revisionTemplateData struct {
	Index     int
	Time      time.Time
	Verb      string
	State     string
	Principal string
	LogID     string
	Changes   string
}

type resourceRevisionsTemplateData struct {
	TimelineID   string
	Segments     []workbench.TimelineSegment
	MatchedCount int
	// Start and End are the 1-based positions of the page within the matched revisions.
	Start         int
	End           int
	FirstIndex    int
	LastIndex     int
	Revisions     []revisionTemplateData
	NextPageToken string
}

type resourceManifestTemplateData struct {
	TimelineID  string
	Segments    []workbench.TimelineSegment
	Revision    revisionTemplateData
	BodySection string
}

type resourceDiffTemplateData struct {
	TimelineID   string
	Segments     []workbench.TimelineSegment
	RevisionLine string
	PreviousLine string
	Changes      string
	DiffSection  string
}

func buildTimelineLogsTemplateData(res *workbench.TimelineLogsResult, page mdtemplate.PageResult[workbench.TimelineLogEntry]) timelineLogsTemplateData {
	otherTimelines := make([]otherTimelineTemplateData, len(res.OtherTimelines))
	for i, o := range res.OtherTimelines {
		otherTimelines[i] = otherTimelineTemplateData{
			linkedTimelineTemplateData: toLinkedTimelineTemplateData(o.LinkedTimeline),
			MatchedLogCount:            o.MatchedLogCount,
		}
	}
	logs := make([]timelineLogTemplateData, len(page.Items))
	for i, l := range page.Items {
		logs[i] = timelineLogTemplateData{
			LogID:      formatID(l.LogID),
			Time:       l.Time,
			Severity:   l.Severity.GetLabel(),
			LogType:    l.LogType,
			Summary:    l.Summary,
			AlsoOnCell: formatTimelineIDsCell(l.OtherTimelineIDs),
		}
	}
	return timelineLogsTemplateData{
		TimelineID:              formatID(res.TimelineID),
		Segments:                res.Segments,
		Applied:                 res.Applied,
		Start:                   page.Start,
		End:                     page.End,
		Total:                   page.Total,
		TotalOtherTimelineCount: res.TotalOtherTimelineCount,
		OtherTimelines:          otherTimelines,
		Logs:                    logs,
		NextPageToken:           page.NextPageToken,
	}
}

func buildLogTemplateData(detail *workbench.LogDetail) logTemplateData {
	linked := make([]linkedTimelineTemplateData, len(detail.LinkedTimelines))
	for i, lt := range detail.LinkedTimelines {
		linked[i] = toLinkedTimelineTemplateData(lt)
	}
	return logTemplateData{
		LogID:           formatID(detail.LogID),
		Time:            detail.Time,
		Severity:        detail.Severity.GetLabel(),
		LogType:         detail.LogType,
		Summary:         detail.Summary,
		LinkedTimelines: linked,
		BodySection:     formatBodySection("Body", "yaml", detail.Body),
	}
}

func buildResourceRevisionsTemplateData(res *workbench.ResourceRevisionsResult, offset int) resourceRevisionsTemplateData {
	revisions := make([]revisionTemplateData, len(res.Revisions))
	for i, rev := range res.Revisions {
		revisions[i] = toRevisionTemplateData(rev.ResourceRevision)
		revisions[i].Changes = formatLineChanges(rev.Changes)
	}
	data := resourceRevisionsTemplateData{
		TimelineID:   formatID(res.TimelineID),
		Segments:     res.Segments,
		MatchedCount: res.MatchedCount,
		Revisions:    revisions,
	}
	if len(res.Revisions) == 0 {
		return data
	}
	nextOffset := offset + len(res.Revisions)
	data.Start = offset + 1
	data.End = nextOffset
	data.FirstIndex = res.Revisions[0].Index
	data.LastIndex = res.Revisions[len(res.Revisions)-1].Index
	if nextOffset < res.MatchedCount {
		data.NextPageToken = mdtemplate.EncodePageToken(nextOffset)
	}
	return data
}

func buildResourceManifestTemplateData(manifest *workbench.ResourceManifest) resourceManifestTemplateData {
	return resourceManifestTemplateData{
		TimelineID:  formatID(manifest.TimelineID),
		Segments:    manifest.Segments,
		Revision:    toRevisionTemplateData(manifest.Revision),
		BodySection: formatBodySection("Body", "yaml", manifest.Body),
	}
}

func buildResourceDiffTemplateData(diff *workbench.ResourceDiff) resourceDiffTemplateData {
	previousLine := "none"
	if diff.Previous != nil {
		previousLine = formatRevisionLine(*diff.Previous)
	}
	diffSection := "## Diff (0 bytes)\n\nThe manifest is identical to the previous revision."
	if diff.Diff.TotalBytes > 0 {
		diffSection = formatBodySection("Diff", "diff", diff.Diff)
	}
	return resourceDiffTemplateData{
		TimelineID:   formatID(diff.TimelineID),
		Segments:     diff.Segments,
		RevisionLine: formatRevisionLine(diff.Revision),
		PreviousLine: previousLine,
		Changes:      formatLineChanges(diff.Changes),
		DiffSection:  diffSection,
	}
}

func toLinkedTimelineTemplateData(lt workbench.LinkedTimeline) linkedTimelineTemplateData {
	return linkedTimelineTemplateData{
		TimelineID: formatID(lt.TimelineID),
		Segments:   lt.Segments,
	}
}

func toRevisionTemplateData(rev workbench.ResourceRevision) revisionTemplateData {
	return revisionTemplateData{
		Index:     rev.Index,
		Time:      rev.Time,
		Verb:      rev.Verb,
		State:     rev.State,
		Principal: rev.Principal,
		LogID:     formatID(rev.LogID),
	}
}

func formatID(id uint32) string {
	return strconv.FormatUint(uint64(id), 10)
}

func formatLineChanges(c workbench.LineChangeCount) string {
	return fmt.Sprintf("+%d -%d", c.Added, c.Deleted)
}

// formatRevisionLine formats a revision on one line, such as "1, 2026-09-24T01:20:05Z, UPDATE by user, state Running, log `12`".
func formatRevisionLine(rev workbench.ResourceRevision) string {
	return fmt.Sprintf("%d, %s, %s by %s, state %s, log %s",
		rev.Index, mdtemplate.FormatTime(rev.Time), rev.Verb, rev.Principal, rev.State, mdtemplate.Code(formatID(rev.LogID)))
}

// formatBodySection renders a "## <heading> (N bytes)" section with the body in a fenced code block.
// When the body is truncated, it ends with a "byteOffset: N" line, so the section must be the last part of a response.
func formatBodySection(heading, lang string, body workbench.BodyChunk) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s (%d bytes)\n\n", heading, body.TotalBytes)
	sb.WriteString(mdtemplate.Fence(lang, body.Content))
	if body.Truncated() {
		fmt.Fprintf(&sb, "\n\nbyteOffset: %d", body.NextByteOffset)
	}
	return sb.String()
}
