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

package rule

import (
	"cmp"
	"context"
	"fmt"
	"slices"
)

const (
	// SpeedFast runs only fast local checks.
	SpeedFast Speed = "fast"
	// SpeedSlow runs only slow checks such as tests, coverage, mutation, and remote APIs.
	SpeedSlow Speed = "slow"
	// SpeedAll runs both fast and slow checks.
	SpeedAll Speed = "all"

	// FilterAll includes both MustFix and Advisory items.
	FilterAll Filter = "all"
	// FilterMust includes only MustFix items.
	FilterMust Filter = "must"
	// FilterAdvisory includes only Advisory items.
	FilterAdvisory Filter = "advisory"
)

// Speed selects which execution tier of QA checks runs.
type Speed string

// Filter selects which severity tier of QA items is reported.
type Filter string

// QAItem represents a single normalized MustFix or Advisory finding in a QAReport.
type QAItem struct {
	// ID is a deterministic identifier for this QA item.
	ID string `json:"id"`
	// Source identifies the analyzer or provider that produced this item.
	Source string `json:"source"`
	// Check identifies the specific rule or check name.
	Check string `json:"check"`
	// File is the slash-separated repository-relative file path.
	File string `json:"file"`
	// Line is the 1-based source line number associated with this item.
	Line int `json:"line"`
	// Symbol is the function, type, or unit name associated with this item.
	Symbol string `json:"symbol"`
	// Summary is a concise one-line description of the issue.
	Summary string `json:"summary"`
	// Detail provides extended diagnostic context.
	Detail string `json:"detail"`
	// Evidence holds structured supporting metrics or metadata.
	Evidence map[string]any `json:"evidence"`
	// ReviewPrompt suggests a focused review or remediation action.
	ReviewPrompt string `json:"review_prompt"`
}

// SkippedSource records an analyzer or advisory provider that was skipped during QA evaluation.
type SkippedSource struct {
	// Source identifies the skipped analyzer or provider.
	Source string `json:"source"`
	// Reason explains why the source was skipped.
	Reason string `json:"reason"`
}

// QAReport aggregates normalized MustFix items, Advisory items, and SkippedSources for a QA run.
type QAReport struct {
	// BaseLabel identifies the comparison base revision or branch.
	BaseLabel string `json:"base_label"`
	// Speed is the execution speed tier used for this run.
	Speed Speed `json:"speed"`
	// Filter is the severity filter applied to this report.
	Filter Filter `json:"filter"`
	// MustFix lists blocking findings that must be resolved.
	MustFix []QAItem `json:"mustfix"`
	// Advisory lists non-blocking recommendations and review hints.
	Advisory []QAItem `json:"advisory"`
	// SkippedSources lists checks or providers that could not run.
	SkippedSources []SkippedSource `json:"skipped_sources"`
}

// AdvisoryRequest provides plain Go inputs for an AdvisoryProvider evaluation.
type AdvisoryRequest struct {
	// RepoRoot is the absolute path to the working-tree repository root.
	RepoRoot string
	// TrustedRoot is the trusted repository root used for loading rules or configs.
	TrustedRoot string
	// ChangedFiles lists repository-relative paths of modified or added files.
	ChangedFiles []string
	// DeletedFiles lists repository-relative paths of deleted files.
	DeletedFiles []string
	// ChangedLines maps repository-relative file paths to modified 1-based line numbers.
	ChangedLines map[string]map[int]struct{}
}

// AdvisoryProvider evaluates a single advisory source against changed files.
type AdvisoryProvider interface {
	// Name returns the unique provider identifier.
	Name() string
	// Speed returns the execution speed tier of this provider.
	Speed() Speed
	// Evaluate runs the provider against req and returns advisory items or a skipped-source notice.
	Evaluate(ctx context.Context, req AdvisoryRequest) ([]QAItem, *SkippedSource, error)
}

// ParseSpeed parses raw into a validated Speed value.
func ParseSpeed(raw string) (Speed, error) {
	switch Speed(raw) {
	case SpeedFast:
		return SpeedFast, nil
	case SpeedSlow:
		return SpeedSlow, nil
	case SpeedAll:
		return SpeedAll, nil
	default:
		return "", fmt.Errorf("invalid speed %q: want fast, slow, or all", raw)
	}
}

// IncludesFast reports whether s includes fast checks.
func (s Speed) IncludesFast() bool {
	return s == SpeedFast || s == SpeedAll
}

// IncludesSlow reports whether s includes slow checks.
func (s Speed) IncludesSlow() bool {
	return s == SpeedSlow || s == SpeedAll
}

// ParseFilter parses raw into a validated Filter value.
func ParseFilter(raw string) (Filter, error) {
	switch Filter(raw) {
	case FilterAll:
		return FilterAll, nil
	case FilterMust:
		return FilterMust, nil
	case FilterAdvisory:
		return FilterAdvisory, nil
	default:
		return "", fmt.Errorf("invalid filter %q: want all, must, or advisory", raw)
	}
}

// IncludesMust reports whether f includes MustFix items.
func (f Filter) IncludesMust() bool {
	return f == FilterMust || f == FilterAll
}

// IncludesAdvisory reports whether f includes Advisory items.
func (f Filter) IncludesAdvisory() bool {
	return f == FilterAdvisory || f == FilterAll
}

// HasMustFix reports whether r contains at least one MustFix item.
func (r *QAReport) HasMustFix() bool {
	return r != nil && len(r.MustFix) > 0
}

// Normalize replaces nil slices with empty non-nil slices and sorts items deterministically.
func (r *QAReport) Normalize() {
	if r == nil {
		return
	}
	if r.MustFix == nil {
		r.MustFix = []QAItem{}
	}
	if r.Advisory == nil {
		r.Advisory = []QAItem{}
	}
	if r.SkippedSources == nil {
		r.SkippedSources = []SkippedSource{}
	}
	sortQAItems(r.MustFix)
	r.MustFix = compactQAItems(r.MustFix)
	sortQAItems(r.Advisory)
	r.Advisory = compactQAItems(r.Advisory)
	slices.SortFunc(r.SkippedSources, func(a, b SkippedSource) int {
		return cmp.Or(
			cmp.Compare(a.Source, b.Source),
			cmp.Compare(a.Reason, b.Reason),
		)
	})
	r.SkippedSources = slices.Compact(r.SkippedSources)
}

func compactQAItems(items []QAItem) []QAItem {
	return slices.CompactFunc(items, func(a, b QAItem) bool {
		return a.ID == b.ID &&
			a.Source == b.Source &&
			a.Check == b.Check &&
			a.File == b.File &&
			a.Line == b.Line &&
			a.Symbol == b.Symbol &&
			a.Summary == b.Summary
	})
}

func sortQAItems(items []QAItem) {
	slices.SortFunc(items, func(a, b QAItem) int {
		return cmp.Or(
			cmp.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(a.Source, b.Source),
			cmp.Compare(a.Check, b.Check),
			cmp.Compare(a.Symbol, b.Symbol),
			cmp.Compare(a.ID, b.ID),
			cmp.Compare(a.Summary, b.Summary),
		)
	})
}
