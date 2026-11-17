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

// ManifestSliceOptions configures JSONPath slicing of a resource YAML manifest body.
type ManifestSliceOptions struct {
	// Paths specifies non-empty JSONPath selectors evaluated against the resource manifest.
	Paths []string
	// MaxLines caps the total number of manifest lines returned across all selected paths.
	MaxLines int
}

// ManifestSliceResult holds the sliced lines and total line count for a resource manifest.
type ManifestSliceResult struct {
	// TotalLines is the total number of lines in the managedFields-stripped YAML manifest.
	TotalLines int
	// Lines contains the sliced manifest lines including omitted-gap markers.
	Lines []ExcerptLine
}

// ResourceManifestSlice combines revision and active interval metadata with sliced manifest lines.
type ResourceManifestSlice struct {
	// TimelineID is the resource timeline ID containing the revision.
	TimelineID uint32
	// Segments holds the timeline path segments identifying the resource.
	Segments []workbench.TimelineSegment
	// Revision holds metadata for the target resource revision.
	Revision workbench.ResourceRevision
	// NextRevisionTime is the timestamp of the following revision, or nil if this is the latest revision.
	NextRevisionTime *time.Time
	// Slice contains the sliced manifest lines and total line count.
	Slice ManifestSliceResult
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

// SliceManifestBody slices a managedFields-stripped YAML manifest using explicit JSONPath selectors and context folding.
func SliceManifestBody(yamlBody string, opts ManifestSliceOptions) (*ManifestSliceResult, error) {
	maxLines := normalizeMaxLines(opts.MaxLines)
	compiledPaths, err := compileJSONPaths(opts.Paths)
	if err != nil {
		return nil, err
	}

	doc, err := parseIndexedYAMLDocument(yamlBody)
	if err != nil {
		return nil, err
	}
	totalLines := len(doc.lines)
	if totalLines == 0 {
		return nil, &SliceQueryNotFoundError{
			WindowIndex: compiledPaths[0].index,
			Path:        compiledPaths[0].raw,
			Preview:     "(empty resource manifest)",
		}
	}

	matchSpansByLine := make(map[int][]byteSpan)
	var rawIntervals []lineInterval

	for _, p := range compiledPaths {
		selections := evalCompiledJSONPath(doc, p)
		if len(selections) == 0 {
			matchedPrefix, availableKeys := diagnoseUnmatchedJSONPath(p, doc)
			return nil, &SliceQueryNotFoundError{
				WindowIndex:   p.index,
				Path:          p.raw,
				MatchedPrefix: matchedPrefix,
				AvailableKeys: availableKeys,
				TopLevelKeys:  extractTopLevelYAMLKeys(yamlBody),
				Preview:       buildManifestPreview(doc.lines),
			}
		}
		for _, sel := range selections {
			rawIntervals = append(rawIntervals, lineInterval{
				start: sel.startLine - 1,
				end:   sel.endLine,
			})
			if sel.highlightLine != 0 {
				lineIdx := sel.highlightLine - 1
				matchSpansByLine[lineIdx] = append(matchSpansByLine[lineIdx], sel.highlightSpan)
			}
		}
	}

	merged := capIntervals(mergeIntervals(rawIntervals), maxLines)
	return &ManifestSliceResult{
		TotalLines: totalLines,
		Lines:      renderLogIntervals(doc.lines, merged, matchSpansByLine, ""),
	}, nil
}

func buildManifestPreview(lines []string) string {
	return strings.Join(lines[:min(len(lines), maxSlicePreviewLines)], "\n")
}
