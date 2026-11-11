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
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

// ManifestSliceOptions configures multi-window slicing of a resource YAML manifest body.
type ManifestSliceOptions struct {
	Windows  []SliceWindow
	MaxLines int
}

// ManifestSliceResult holds the sliced lines and total line count for a resource manifest.
type ManifestSliceResult struct {
	TotalLines int
	Lines      []ExcerptLine
}

// ResourceManifestSlice combines revision and active interval metadata with sliced manifest lines.
type ResourceManifestSlice struct {
	TimelineID       uint32
	Segments         []workbench.TimelineSegment
	Revision         workbench.ResourceRevision
	NextRevisionTime *time.Time
	Slice            ManifestSliceResult
}

// SliceResourceManifest fetches the untruncated managedFields-stripped manifest from wb and slices it according to opts.
func SliceResourceManifest(
	wb *workbench.Workbench,
	timelineID uint32,
	selector workbench.RevisionSelector,
	opts ManifestSliceOptions,
) (*ResourceManifestSlice, error) {
	raw, err := wb.GetRawResourceManifest(timelineID, selector, false)
	if err != nil {
		return nil, fmt.Errorf("failed to read raw resource manifest: %w", err)
	}
	sliced, err := SliceManifestBody(raw.YAMLBody, opts)
	if err != nil {
		return nil, err
	}
	return &ResourceManifestSlice{
		TimelineID:       raw.TimelineID,
		Segments:         raw.Segments,
		Revision:         raw.Revision,
		NextRevisionTime: raw.NextRevisionTime,
		Slice:            *sliced,
	}, nil
}

// SliceManifestBody slices a managedFields-stripped YAML manifest using multi-window search and context folding.
func SliceManifestBody(yamlBody string, opts ManifestSliceOptions) (*ManifestSliceResult, error) {
	maxLines := normalizeMaxLines(opts.MaxLines)
	windows, err := compileSliceWindows(opts.Windows)
	if err != nil {
		return nil, err
	}

	lines := workbench.SplitLines(yamlBody)
	totalLines := len(lines)

	if len(windows) == 0 {
		if totalLines == 0 {
			return &ManifestSliceResult{}, nil
		}
		keptEnd := min(totalLines, maxLines)
		return &ManifestSliceResult{
			TotalLines: totalLines,
			Lines:      renderLogIntervals(lines, []lineInterval{{start: 0, end: keptEnd}}, nil, ""),
		}, nil
	}

	if totalLines == 0 {
		return nil, &SliceQueryNotFoundError{
			WindowIndex: windows[0].index,
			Query:       windows[0].query,
			Preview:     "(empty resource manifest)",
		}
	}

	matchSpansByLine := make(map[int][]byteSpan)
	var rawIntervals []lineInterval

	for _, w := range windows {
		matchedCount := 0
		for lineIdx, line := range lines {
			spans := findQuerySpans(line, w)
			if len(spans) == 0 {
				continue
			}
			matchSpansByLine[lineIdx] = append(matchSpansByLine[lineIdx], spans...)
			rawIntervals = append(rawIntervals, lineInterval{
				start: max(0, lineIdx-w.before),
				end:   min(totalLines, lineIdx+w.after+1),
			})
			matchedCount++
			if matchedCount >= w.maxMatches {
				break
			}
		}
		if matchedCount == 0 {
			return nil, &SliceQueryNotFoundError{
				WindowIndex:  w.index,
				Query:        w.query,
				TopLevelKeys: extractTopLevelYAMLKeys(yamlBody),
				Preview:      buildManifestPreview(lines),
			}
		}
	}

	merged := capIntervals(mergeIntervals(rawIntervals), maxLines)
	return &ManifestSliceResult{
		TotalLines: totalLines,
		Lines:      renderLogIntervals(lines, merged, matchSpansByLine, ""),
	}, nil
}

func buildManifestPreview(lines []string) string {
	if len(lines) == 0 {
		return "(empty resource manifest)"
	}
	return strings.Join(lines[:min(len(lines), maxSlicePreviewLines)], "\n")
}
