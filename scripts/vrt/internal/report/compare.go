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

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/capture"
	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/imagediff"
)

// CompareConfig configures snapshot directory comparison and report generation.
type CompareConfig struct {
	// BaseDir is the local directory containing the baseline manifest.json and images/ (optional if no baseline exists).
	BaseDir string
	// HeadDir is the local directory containing the candidate manifest.json and images/.
	HeadDir string
	// ReportDir is the destination directory where report.json, index.html, and diff PNGs are written.
	ReportDir string
	// DiffOptions controls perceptual pixel comparison thresholds.
	DiffOptions imagediff.Options
	// Workers controls parallel diff goroutine count (0 uses runtime.NumCPU).
	Workers int
	// BaselineCacheHit records whether the baseline came from GCS cache.
	BaselineCacheHit bool
	// AssetBaseURL optionally prefixes image URLs in index.html (empty keeps relative paths for local viewing).
	AssetBaseURL string
}

type comparePair struct {
	id      string
	hasBase bool
	base    capture.SnapshotEntry
	hasHead bool
	head    capture.SnapshotEntry
}

type comparePool struct {
	cfg      CompareConfig
	jobs     chan comparePair
	mu       sync.Mutex
	results  []StoryComparison
	firstErr error
}

// CompareDirectories compares BaseDir and HeadDir snapshots in parallel and writes report artifacts to ReportDir.
func CompareDirectories(cfg CompareConfig) (ComparisonReport, error) {
	headManifest, err := capture.ReadManifest(filepath.Join(cfg.HeadDir, capture.ManifestFileName))
	if err != nil {
		return ComparisonReport{}, fmt.Errorf("failed to read head manifest: %w", err)
	}

	baseManifest := capture.Manifest{Entries: map[string]capture.SnapshotEntry{}}
	if cfg.BaseDir != "" {
		baseManifest, err = capture.ReadManifest(filepath.Join(cfg.BaseDir, capture.ManifestFileName))
		if err != nil {
			return ComparisonReport{}, fmt.Errorf("failed to read base manifest: %w", err)
		}
	}

	if err := os.MkdirAll(cfg.ReportDir, 0o755); err != nil {
		return ComparisonReport{}, fmt.Errorf("failed to create report directory: %w", err)
	}

	pairs := buildComparePairs(baseManifest.Entries, headManifest.Entries)
	items, err := evaluatePairs(cfg, pairs)
	if err != nil {
		return ComparisonReport{}, err
	}

	sortComparisonItems(items)
	rep := ComparisonReport{
		BaseSHA:            baseManifest.CommitSHA,
		HeadSHA:            headManifest.CommitSHA,
		BuilderFingerprint: headManifest.BuilderFingerprint,
		GeneratedAt:        time.Now().UTC().Format(time.RFC3339),
		BaselineCacheHit:   cfg.BaselineCacheHit,
		AssetBaseURL:       strings.TrimRight(cfg.AssetBaseURL, "/"),
		Summary:            summarizeItems(items),
		Items:              items,
	}

	if err := writeReportJSON(filepath.Join(cfg.ReportDir, ReportJSONFileName), rep); err != nil {
		return ComparisonReport{}, err
	}
	if err := WriteHTMLReport(filepath.Join(cfg.ReportDir, ReportHTMLFileName), rep); err != nil {
		return ComparisonReport{}, err
	}
	return rep, nil
}

func buildComparePairs(baseEntries map[string]capture.SnapshotEntry, headEntries map[string]capture.SnapshotEntry) []comparePair {
	seen := make(map[string]struct{}, len(baseEntries)+len(headEntries))
	pairs := make([]comparePair, 0, len(baseEntries)+len(headEntries))
	for id, head := range headEntries {
		seen[id] = struct{}{}
		base, hasBase := baseEntries[id]
		pairs = append(pairs, comparePair{
			id:      id,
			hasBase: hasBase,
			base:    base,
			hasHead: true,
			head:    head,
		})
	}
	for id, base := range baseEntries {
		if _, ok := seen[id]; ok {
			continue
		}
		pairs = append(pairs, comparePair{
			id:      id,
			hasBase: true,
			base:    base,
		})
	}
	return pairs
}

func (p *comparePool) runWorker() {
	for pair := range p.jobs {
		item, err := evaluateOnePair(p.cfg, pair)
		p.recordResult(item, err)
	}
}

func (p *comparePool) recordResult(item StoryComparison, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		if p.firstErr == nil {
			p.firstErr = err
		}
		return
	}
	p.results = append(p.results, item)
}

func evaluatePairs(cfg CompareConfig, pairs []comparePair) ([]StoryComparison, error) {
	workers := resolveCompareWorkers(cfg.Workers, len(pairs))

	jobs := make(chan comparePair, len(pairs))
	for _, p := range pairs {
		jobs <- p
	}
	close(jobs)

	pool := &comparePool{
		cfg:     cfg,
		jobs:    jobs,
		results: make([]StoryComparison, 0, len(pairs)),
	}
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.runWorker()
		}()
	}
	wg.Wait()
	if pool.firstErr != nil {
		return nil, pool.firstErr
	}
	return pool.results, nil
}

func evaluateOnePair(cfg CompareConfig, p comparePair) (StoryComparison, error) {
	meta := p.head
	if !p.hasHead {
		meta = p.base
	}
	item := StoryComparison{
		ID:         p.id,
		StoryID:    meta.StoryID,
		Title:      meta.Title,
		Name:       meta.Name,
		ImportPath: meta.ImportPath,
	}

	if p.hasHead && p.head.Error != "" {
		item.Status = StatusErrored
		item.Error = p.head.Error
		return item, nil
	}
	if p.hasHead && p.head.Skipped {
		item.Status = StatusSkipped
		return item, nil
	}
	if !p.hasHead {
		if !hasUsableSnapshot(p.base) {
			item.Status = StatusSkipped
			return item, nil
		}
		return finalizeRemoved(cfg, item, p.base)
	}
	if !p.hasBase || !hasUsableSnapshot(p.base) {
		return finalizeAdded(cfg, item, p.head)
	}
	return compareExistingPair(cfg, item, p.base, p.head)
}

func resolveCompareWorkers(requested int, pairCount int) int {
	workers := requested
	if workers <= 0 {
		workers = max(runtime.NumCPU(), 1)
	}
	return min(workers, max(pairCount, 1))
}

func hasUsableSnapshot(entry capture.SnapshotEntry) bool {
	return !entry.Skipped && entry.Error == "" && entry.ImageRelPath != ""
}

func finalizeAdded(cfg CompareConfig, item StoryComparison, head capture.SnapshotEntry) (StoryComparison, error) {
	rel := path.Join("after", head.ID+".png")
	if err := copyFile(filepath.Join(cfg.HeadDir, filepath.FromSlash(head.ImageRelPath)), filepath.Join(cfg.ReportDir, filepath.FromSlash(rel))); err != nil {
		return StoryComparison{}, err
	}
	item.Status = StatusAdded
	item.AfterWidth = head.Width
	item.AfterHeight = head.Height
	item.AfterRelPath = rel
	return item, nil
}

func finalizeRemoved(cfg CompareConfig, item StoryComparison, base capture.SnapshotEntry) (StoryComparison, error) {
	rel := path.Join("before", base.ID+".png")
	if err := copyFile(filepath.Join(cfg.BaseDir, filepath.FromSlash(base.ImageRelPath)), filepath.Join(cfg.ReportDir, filepath.FromSlash(rel))); err != nil {
		return StoryComparison{}, err
	}
	item.Status = StatusRemoved
	item.BeforeWidth = base.Width
	item.BeforeHeight = base.Height
	item.BeforeRelPath = rel
	return item, nil
}

func compareExistingPair(
	cfg CompareConfig,
	item StoryComparison,
	base capture.SnapshotEntry,
	head capture.SnapshotEntry,
) (StoryComparison, error) {
	if base.SHA256 != "" && base.SHA256 == head.SHA256 {
		item.BeforeWidth = base.Width
		item.BeforeHeight = base.Height
		item.AfterWidth = head.Width
		item.AfterHeight = head.Height
		item.TotalPixels = head.Width * head.Height
		return finalizeUnchangedFromFile(cfg, item, head)
	}

	beforeBytes, err := os.ReadFile(filepath.Join(cfg.BaseDir, filepath.FromSlash(base.ImageRelPath)))
	if err != nil {
		return StoryComparison{}, fmt.Errorf("failed to read base PNG for %s: %w", item.ID, err)
	}
	afterBytes, err := os.ReadFile(filepath.Join(cfg.HeadDir, filepath.FromSlash(head.ImageRelPath)))
	if err != nil {
		return StoryComparison{}, fmt.Errorf("failed to read head PNG for %s: %w", item.ID, err)
	}

	opts := cfg.DiffOptions
	if head.DiffPixelRatio > 0 {
		opts.MaxDiffPixelRatio = head.DiffPixelRatio
	}
	res, err := imagediff.ComparePNGs(beforeBytes, afterBytes, opts)
	if err != nil {
		return StoryComparison{}, fmt.Errorf("failed to compare PNGs for %s: %w", item.ID, err)
	}

	item.DiffPixels = res.DiffPixels
	item.TotalPixels = res.TotalPixels
	item.DiffRatio = res.DiffRatio
	item.SizeMismatch = res.SizeMismatch
	item.BeforeWidth = res.BeforeWidth
	item.BeforeHeight = res.BeforeHeight
	item.AfterWidth = res.AfterWidth
	item.AfterHeight = res.AfterHeight
	if res.Matched {
		return finalizeUnchangedBytes(cfg.ReportDir, item, afterBytes)
	}

	item.Status = StatusChanged
	return writeChangedArtifacts(cfg.ReportDir, item, beforeBytes, afterBytes, res)
}

func finalizeUnchangedFromFile(cfg CompareConfig, item StoryComparison, head capture.SnapshotEntry) (StoryComparison, error) {
	rel := path.Join("after", item.ID+".png")
	if err := copyFile(filepath.Join(cfg.HeadDir, filepath.FromSlash(head.ImageRelPath)), filepath.Join(cfg.ReportDir, filepath.FromSlash(rel))); err != nil {
		return StoryComparison{}, err
	}
	item.Status = StatusUnchanged
	item.AfterRelPath = rel
	return item, nil
}

func finalizeUnchangedBytes(reportDir string, item StoryComparison, afterBytes []byte) (StoryComparison, error) {
	rel := path.Join("after", item.ID+".png")
	if err := writeReportFile(filepath.Join(reportDir, filepath.FromSlash(rel)), afterBytes); err != nil {
		return StoryComparison{}, err
	}
	item.Status = StatusUnchanged
	item.AfterRelPath = rel
	return item, nil
}

func writeChangedArtifacts(
	reportDir string,
	item StoryComparison,
	beforeBytes []byte,
	afterBytes []byte,
	res imagediff.Result,
) (StoryComparison, error) {
	artifacts := []struct {
		rel  string
		data []byte
		set  func(string)
	}{
		{rel: path.Join("before", item.ID+".png"), data: beforeBytes, set: func(r string) { item.BeforeRelPath = r }},
		{rel: path.Join("after", item.ID+".png"), data: afterBytes, set: func(r string) { item.AfterRelPath = r }},
		{rel: path.Join("diff", item.ID+".png"), data: res.DiffPNG, set: func(r string) { item.DiffRelPath = r }},
		{rel: path.Join("composite", item.ID+".png"), data: res.CompositePNG, set: func(r string) { item.CompositeRelPath = r }},
	}
	for _, a := range artifacts {
		absPath := filepath.Join(reportDir, filepath.FromSlash(a.rel))
		if err := writeReportFile(absPath, a.data); err != nil {
			return StoryComparison{}, err
		}
		a.set(a.rel)
	}
	return item, nil
}

func writeReportFile(dst string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("failed to create dir for %s: %w", dst, err)
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", dst, err)
	}
	return nil
}

func copyFile(src string, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", src, err)
	}
	return writeReportFile(dst, data)
}

func sortComparisonItems(items []StoryComparison) {
	slices.SortFunc(items, func(a, b StoryComparison) int {
		if r := cmp.Compare(statusPriority(a.Status), statusPriority(b.Status)); r != 0 {
			return r
		}
		if a.Status == StatusChanged {
			if r := cmp.Compare(b.DiffRatio, a.DiffRatio); r != 0 {
				return r
			}
		}
		return strings.Compare(a.ID, b.ID)
	})
}

func statusPriority(s Status) int {
	switch s {
	case StatusErrored:
		return 0
	case StatusChanged:
		return 1
	case StatusAdded:
		return 2
	case StatusRemoved:
		return 3
	case StatusUnchanged:
		return 4
	case StatusSkipped:
		return 5
	default:
		return 6
	}
}

func summarizeItems(items []StoryComparison) Summary {
	sum := Summary{Total: len(items)}
	for _, item := range items {
		switch item.Status {
		case StatusChanged:
			sum.Changed++
		case StatusAdded:
			sum.Added++
		case StatusRemoved:
			sum.Removed++
		case StatusErrored:
			sum.Errored++
		case StatusSkipped:
			sum.Skipped++
		case StatusUnchanged:
			sum.Unchanged++
		}
	}
	return sum
}

func writeReportJSON(path string, rep ComparisonReport) error {
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report JSON: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write report JSON %s: %w", path, err)
	}
	return nil
}
