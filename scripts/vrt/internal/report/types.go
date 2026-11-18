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

package report

const (
	// ReportJSONFileName is the filename of the machine-readable comparison report JSON.
	ReportJSONFileName = "report.json"
	// ReportHTMLFileName is the filename of the interactive HTML visual diff viewer.
	ReportHTMLFileName = "index.html"

	// StatusUnchanged indicates the story rendered identically within the diff threshold.
	StatusUnchanged Status = "unchanged"
	// StatusChanged indicates a visual diff exceeding the threshold was detected.
	StatusChanged Status = "changed"
	// StatusAdded indicates a new story exists only in the candidate snapshot.
	StatusAdded Status = "added"
	// StatusRemoved indicates a baseline story was removed in the candidate snapshot.
	StatusRemoved Status = "removed"
	// StatusErrored indicates story capture or runtime verification failed.
	StatusErrored Status = "errored"
	// StatusSkipped indicates the story opted out of VRT.
	StatusSkipped Status = "skipped"
)

// Status classifies the visual comparison result of a single snapshot.
type Status string

// StoryComparison holds the visual comparison outcome and relative artifact paths for one snapshot.
type StoryComparison struct {
	// ID is the namespaced snapshot identifier (e.g., "story/header--default").
	ID string `json:"id"`
	// StoryID is the raw Storybook story identifier.
	StoryID string `json:"storyId"`
	// Title is the Storybook hierarchy title.
	Title string `json:"title"`
	// Name is the story variant name.
	Name string `json:"name"`
	// ImportPath is the source path of the story file.
	ImportPath string `json:"importPath"`
	// Status is the visual comparison classification.
	Status Status `json:"status"`
	// DiffPixels is the count of pixels exceeding the perceptual threshold.
	DiffPixels int `json:"diffPixels,omitempty"`
	// TotalPixels is the total pixels in the comparison canvas.
	TotalPixels int `json:"totalPixels,omitempty"`
	// DiffRatio is DiffPixels divided by TotalPixels.
	DiffRatio float64 `json:"diffRatio,omitempty"`
	// SizeMismatch reports whether baseline and candidate dimensions differed.
	SizeMismatch bool `json:"sizeMismatch,omitempty"`
	// BeforeWidth is the baseline image width in pixels.
	BeforeWidth int `json:"beforeWidth,omitempty"`
	// BeforeHeight is the baseline image height in pixels.
	BeforeHeight int `json:"beforeHeight,omitempty"`
	// AfterWidth is the candidate image width in pixels.
	AfterWidth int `json:"afterWidth,omitempty"`
	// AfterHeight is the candidate image height in pixels.
	AfterHeight int `json:"afterHeight,omitempty"`
	// BeforeRelPath is the report-relative PNG path for the baseline image.
	BeforeRelPath string `json:"beforeRelPath,omitempty"`
	// AfterRelPath is the report-relative PNG path for the candidate image.
	AfterRelPath string `json:"afterRelPath,omitempty"`
	// DiffRelPath is the report-relative PNG path for the highlighted pixel diff image.
	DiffRelPath string `json:"diffRelPath,omitempty"`
	// CompositeRelPath is the report-relative PNG path for the 3-panel Before | Diff | After strip.
	CompositeRelPath string `json:"compositeRelPath,omitempty"`
	// Error holds the failure description when Status is StatusErrored.
	Error string `json:"error,omitempty"`
}

// Summary aggregates snapshot counts across all statuses.
type Summary struct {
	// Total is the total number of evaluated snapshots.
	Total int `json:"total"`
	// Changed is the number of stories with visual diffs exceeding the threshold.
	Changed int `json:"changed"`
	// Added is the number of newly added stories.
	Added int `json:"added"`
	// Removed is the number of removed stories.
	Removed int `json:"removed"`
	// Errored is the number of stories that failed to render or had runtime errors.
	Errored int `json:"errored"`
	// Skipped is the number of stories that opted out of VRT.
	Skipped int `json:"skipped"`
	// Unchanged is the number of stories matching the baseline.
	Unchanged int `json:"unchanged"`
}

// ComparisonReport is the top-level structured visual regression report written to report.json.
type ComparisonReport struct {
	// BaseSHA is the baseline Git commit SHA (empty for standalone capture runs).
	BaseSHA string `json:"baseSha,omitempty"`
	// HeadSHA is the candidate Git commit SHA.
	HeadSHA string `json:"headSha,omitempty"`
	// BuilderFingerprint identifies the Chromium and OS rendering environment.
	BuilderFingerprint string `json:"builderFingerprint,omitempty"`
	// GeneratedAt is the UTC RFC3339 timestamp when comparison finished.
	GeneratedAt string `json:"generatedAt"`
	// BaselineCacheHit reports whether the baseline snapshot was fetched from GCS cache rather than rebuilt.
	BaselineCacheHit bool `json:"baselineCacheHit,omitempty"`
	// AssetBaseURL is an optional URL prefix prepended to relative image paths in index.html (e.g., GCS report URL).
	AssetBaseURL string `json:"assetBaseUrl,omitempty"`
	// Summary aggregates counts by status.
	Summary Summary `json:"summary"`
	// Items contains per-story comparison results ordered by severity and diff ratio.
	Items []StoryComparison `json:"items"`
}

// HasVisualDiffs reports whether any story was changed, added, or removed.
func (r ComparisonReport) HasVisualDiffs() bool {
	return r.Summary.Changed > 0 || r.Summary.Added > 0 || r.Summary.Removed > 0
}

// HasErrors reports whether any story failed to render or threw a runtime error.
func (r ComparisonReport) HasErrors() bool {
	return r.Summary.Errored > 0
}
