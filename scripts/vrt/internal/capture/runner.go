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

package capture

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/cdp"
	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/storybook"
)

const (
	// DefaultViewportWidth is the default logical capture viewport width in pixels.
	DefaultViewportWidth = 1280
	// DefaultViewportHeight is the default logical capture viewport height in pixels.
	DefaultViewportHeight = 720
	// DefaultStoryTimeout is the per-story render and screenshot timeout.
	DefaultStoryTimeout = 30 * time.Second
	// DefaultRetries is the number of automatic retries after a capture failure.
	DefaultRetries = 1

	maxDefaultWorkers = 8
)

var vrtChromeExtraArgs = []string{
	"--font-render-hinting=none",
	"--force-color-profile=srgb",
	"--disable-lcd-text",
	"--host-resolver-rules=MAP * ~NOTFOUND, EXCLUDE 127.0.0.1, EXCLUDE localhost",
}

// Config configures Storybook screenshot capture execution.
type Config struct {
	// StorybookDir is the built storybook-static directory path.
	StorybookDir string
	// OutputDir is the destination directory where manifest.json and images/ are written.
	OutputDir string
	// FilterPattern is an optional case-insensitive regex restricting captured stories.
	FilterPattern string
	// Workers controls headless Chromium worker concurrency (0 uses an automatic CPU-based default).
	Workers int
	// ViewportWidth overrides DefaultViewportWidth when positive.
	ViewportWidth int
	// ViewportHeight overrides DefaultViewportHeight when positive.
	ViewportHeight int
	// StoryTimeout overrides DefaultStoryTimeout when positive.
	StoryTimeout time.Duration
	// Retries sets how many times to retry a failed story capture (negative disables retries).
	Retries int
	// CommitSHA records the Git commit SHA in the written Manifest.
	CommitSHA string
	// LogWriter receives progress messages (defaults to io.Discard when nil).
	LogWriter io.Writer
}

type browserFactory func(ctx context.Context, opts cdp.BrowserOptions) (*cdp.Browser, error)

type capturePool struct {
	cfg        Config
	baseURL    string
	totalCount int
	factory    browserFactory
	jobs       chan storybook.StoryEntry

	mu       sync.Mutex
	entries  map[string]SnapshotEntry
	doneCnt  int
	errCnt   int
	firstErr error
}

// Fingerprint launches a temporary headless Chromium instance to compute the deterministic builder cache key.
func Fingerprint(ctx context.Context) (string, error) {
	return computeFingerprint(ctx, cdp.StartBrowser, "/etc/os-release")
}

// Run serves storybook-static, captures all matching stories in parallel headless Chromium workers, and writes manifest.json.
func Run(ctx context.Context, cfg Config) (Manifest, error) {
	return runWithFactory(ctx, cfg, cdp.StartBrowser, "/etc/os-release")
}

func (p *capturePool) runWorker(ctx context.Context, browserOpts cdp.BrowserOptions) {
	session := &workerSession{
		factory:        p.factory,
		browserOpts:    browserOpts,
		viewportWidth:  p.cfg.ViewportWidth,
		viewportHeight: p.cfg.ViewportHeight,
	}
	defer session.close()

	for story := range p.jobs {
		if p.recordContextCancellation(ctx) {
			return
		}
		entry := session.captureWithRetries(ctx, p.cfg, p.baseURL, story)
		p.recordEntry(entry)
	}
}

func (p *capturePool) recordContextCancellation(ctx context.Context) bool {
	if ctx.Err() == nil {
		return false
	}
	p.mu.Lock()
	if p.firstErr == nil {
		p.firstErr = ctx.Err()
	}
	p.mu.Unlock()
	return true
}

func (p *capturePool) recordEntry(entry SnapshotEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries[entry.ID] = entry
	p.doneCnt++
	if entry.Error != "" {
		p.errCnt++
	}
	if p.doneCnt%25 == 0 || p.doneCnt == p.totalCount {
		fmt.Fprintf(p.cfg.LogWriter, "Captured %d/%d stories (%d errors)\n", p.doneCnt, p.totalCount, p.errCnt)
	}
}

func runWithFactory(ctx context.Context, cfg Config, factory browserFactory, osReleasePath string) (Manifest, error) {
	resolved := resolveConfig(cfg)
	stories, err := storybook.LoadStories(resolved.StorybookDir, resolved.FilterPattern)
	if err != nil {
		return Manifest{}, err
	}
	if len(stories) == 0 {
		return Manifest{}, fmt.Errorf("no stories matched in %s (filter=%q)", resolved.StorybookDir, resolved.FilterPattern)
	}

	fingerprint, err := computeFingerprint(ctx, factory, osReleasePath)
	if err != nil {
		return Manifest{}, err
	}

	srv, err := storybook.StartStaticServer(resolved.StorybookDir)
	if err != nil {
		return Manifest{}, err
	}
	defer srv.Close()

	if err := os.MkdirAll(filepath.Join(resolved.OutputDir, ImagesDirName, "story"), 0o755); err != nil {
		return Manifest{}, fmt.Errorf("failed to create output image directory: %w", err)
	}

	entries, err := captureStoriesInPool(ctx, resolved, srv.BaseURL(), stories, factory)
	if err != nil {
		return Manifest{}, err
	}

	manifest := Manifest{
		Version:            ManifestVersion,
		CommitSHA:          resolved.CommitSHA,
		BuilderFingerprint: fingerprint,
		CapturedAt:         time.Now().UTC().Format(time.RFC3339),
		ViewportWidth:      resolved.ViewportWidth,
		ViewportHeight:     resolved.ViewportHeight,
		Entries:            entries,
	}
	if err := WriteManifest(filepath.Join(resolved.OutputDir, ManifestFileName), manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func resolveConfig(cfg Config) Config {
	if cfg.ViewportWidth <= 0 {
		cfg.ViewportWidth = DefaultViewportWidth
	}
	if cfg.ViewportHeight <= 0 {
		cfg.ViewportHeight = DefaultViewportHeight
	}
	if cfg.StoryTimeout <= 0 {
		cfg.StoryTimeout = DefaultStoryTimeout
	}
	if cfg.Workers <= 0 {
		cfg.Workers = defaultWorkerCount(runtime.NumCPU())
	}
	if cfg.Retries < 0 {
		cfg.Retries = 0
	} else if cfg.Retries == 0 {
		cfg.Retries = DefaultRetries
	}
	if cfg.LogWriter == nil {
		cfg.LogWriter = io.Discard
	}
	return cfg
}

func defaultWorkerCount(numCPU int) int {
	return min(max(numCPU/2, 1), maxDefaultWorkers)
}

func computeFingerprint(ctx context.Context, factory browserFactory, osReleasePath string) (string, error) {
	startCtx, cancel := context.WithTimeout(ctx, cdp.DefaultStartTimeout)
	defer cancel()

	browser, err := factory(startCtx, cdp.BrowserOptions{ExtraArgs: vrtChromeExtraArgs})
	if err != nil {
		return "", fmt.Errorf("failed to start browser for fingerprint: %w", err)
	}
	defer browser.Close()

	ver, err := browser.Version(startCtx)
	if err != nil {
		return "", fmt.Errorf("failed to query browser version: %w", err)
	}
	return SanitizeFingerprint(ver.Browser, readOSCodename(osReleasePath)), nil
}

func readOSCodename(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if val, ok := strings.CutPrefix(strings.TrimSpace(line), "VERSION_CODENAME="); ok {
			return strings.Trim(val, `"'`)
		}
	}
	return ""
}

func captureStoriesInPool(
	ctx context.Context,
	cfg Config,
	baseURL string,
	stories []storybook.StoryEntry,
	factory browserFactory,
) (map[string]SnapshotEntry, error) {
	jobs := make(chan storybook.StoryEntry, len(stories))
	for _, story := range stories {
		jobs <- story
	}
	close(jobs)

	pool := &capturePool{
		cfg:        cfg,
		baseURL:    baseURL,
		totalCount: len(stories),
		factory:    factory,
		jobs:       jobs,
		entries:    make(map[string]SnapshotEntry, len(stories)),
	}
	browserOpts := cdp.BrowserOptions{
		WindowWidth:  cfg.ViewportWidth,
		WindowHeight: cfg.ViewportHeight,
		ExtraArgs:    vrtChromeExtraArgs,
	}

	var wg sync.WaitGroup
	workerCount := min(cfg.Workers, len(stories))
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.runWorker(ctx, browserOpts)
		}()
	}
	wg.Wait()
	if pool.firstErr != nil {
		return nil, pool.firstErr
	}
	return pool.entries, nil
}
