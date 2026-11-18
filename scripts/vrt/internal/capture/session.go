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
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/cdp"
	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/storybook"
)

const (
	bridgePollInterval = 50 * time.Millisecond

	bridgeProbeExprFormat = `(function(expectedId) {
  var currentId = new URLSearchParams(window.location.search).get('id') || '';
  if (currentId !== expectedId) {
    return { hasBridge: false, legacyReady: false };
  }
  return {
    hasBridge: typeof window.__khiVRT !== 'undefined',
    legacyReady: document.body !== null && (document.body.classList.contains('sb-show-main') || document.body.classList.contains('sb-show-errordisplay'))
  };
})(%s)`
)

var (
	//go:embed assets/init.js
	vrtDocumentInitScript string

	//go:embed assets/legacy_ready.js
	legacyFallbackReadyExpr string

	//go:embed assets/mask.js
	maskHelperScript string
)

type vrtViewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type vrtParameters struct {
	Skip           bool         `json:"skip"`
	Viewport       *vrtViewport `json:"viewport"`
	FullPage       bool         `json:"fullPage"`
	DiffPixelRatio float64      `json:"diffPixelRatio"`
	MaskSelectors  []string     `json:"maskSelectors"`
}

type vrtBridgeState struct {
	StoryID      string        `json:"storyId"`
	Status       string        `json:"status"`
	ErrorMessage string        `json:"errorMessage"`
	Parameters   vrtParameters `json:"parameters"`
	IsFullscreen bool          `json:"isFullscreen"`
}

type bridgeProbeResult struct {
	HasBridge   bool `json:"hasBridge"`
	LegacyReady bool `json:"legacyReady"`
}

type workerSession struct {
	factory        browserFactory
	browserOpts    cdp.BrowserOptions
	browser        *cdp.Browser
	client         *cdp.Client
	viewportWidth  int
	viewportHeight int
}

func (w *workerSession) captureWithRetries(
	ctx context.Context,
	cfg Config,
	baseURL string,
	story storybook.StoryEntry,
) SnapshotEntry {
	var lastEntry SnapshotEntry
	for attempt := 0; attempt <= cfg.Retries; attempt++ {
		lastEntry = w.captureStoryOnce(ctx, cfg, baseURL, story)
		if lastEntry.Error == "" {
			return lastEntry
		}
		w.close()
	}
	return lastEntry
}

func (w *workerSession) captureStoryOnce(
	ctx context.Context,
	cfg Config,
	baseURL string,
	story storybook.StoryEntry,
) SnapshotEntry {
	start := time.Now()
	entry := SnapshotEntry{
		ID:         StorySnapshotID(story.ID),
		StoryID:    story.ID,
		Title:      story.Title,
		Name:       story.Name,
		ImportPath: story.ImportPath,
	}

	storyCtx, cancel := context.WithTimeout(ctx, cfg.StoryTimeout)
	defer cancel()

	state, err := w.navigateAndWaitReady(storyCtx, baseURL, story.ID)
	if err != nil {
		return failedEntry(entry, err.Error(), start)
	}
	if state.Parameters.Skip {
		return skippedEntry(entry, start)
	}
	if state.Status == "error" {
		return failedEntry(entry, state.ErrorMessage, start)
	}

	pngBytes, err := w.renderAndCapturePNG(storyCtx, state.Parameters)
	if err != nil {
		return failedEntry(entry, err.Error(), start)
	}
	return finalizeEntry(cfg.OutputDir, entry, pngBytes, state.Parameters.DiffPixelRatio, start)
}

func (w *workerSession) navigateAndWaitReady(ctx context.Context, baseURL string, storyID string) (vrtBridgeState, error) {
	if err := w.ensureReady(ctx); err != nil {
		return vrtBridgeState{}, err
	}
	if err := w.client.SetViewportSize(ctx, w.viewportWidth, w.viewportHeight, 1); err != nil {
		return vrtBridgeState{}, fmt.Errorf("failed to reset viewport: %w", err)
	}
	storyURL := fmt.Sprintf("%s/iframe.html?id=%s&viewMode=story&vrt=1", baseURL, url.QueryEscape(storyID))
	if err := w.client.Navigate(ctx, storyURL); err != nil {
		return vrtBridgeState{}, fmt.Errorf("failed to navigate: %w", err)
	}
	return waitForVRTBridge(ctx, w.client, storyID)
}

func (w *workerSession) renderAndCapturePNG(ctx context.Context, params vrtParameters) ([]byte, error) {
	if err := applyCustomViewport(ctx, w.client, params.Viewport, w.viewportWidth, w.viewportHeight); err != nil {
		return nil, err
	}
	if err := applyMaskSelectors(ctx, w.client, params.MaskSelectors); err != nil {
		return nil, err
	}
	if pageErr := collectPageErrors(ctx, w.client); pageErr != "" {
		return nil, errors.New(pageErr)
	}
	pngBytes, err := w.client.CapturePNGScreenshot(ctx, cdp.ScreenshotOptions{
		FullPage: params.FullPage,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}
	return pngBytes, nil
}

func (w *workerSession) ensureReady(ctx context.Context) error {
	if w.client != nil {
		return nil
	}
	w.close()

	browser, err := w.factory(ctx, w.browserOpts)
	if err != nil {
		return fmt.Errorf("failed to start worker browser: %w", err)
	}
	client, err := browser.DialPage(ctx)
	if err != nil {
		_ = browser.Close()
		return fmt.Errorf("failed to connect worker CDP page: %w", err)
	}
	if err := configureDeterministicPage(ctx, client); err != nil {
		_ = client.Close()
		_ = browser.Close()
		return err
	}

	w.browser = browser
	w.client = client
	return nil
}

func (w *workerSession) close() {
	if w.client != nil {
		_ = w.client.Close()
		w.client = nil
	}
	if w.browser != nil {
		_ = w.browser.Close()
		w.browser = nil
	}
}

func configureDeterministicPage(ctx context.Context, client *cdp.Client) error {
	if err := client.SetEmulatedMedia(ctx, "screen", []cdp.EmulatedMediaFeature{
		{Name: "prefers-reduced-motion", Value: "reduce"},
	}); err != nil {
		return fmt.Errorf("failed to set reduced motion media: %w", err)
	}
	if err := client.SetTimezoneOverride(ctx, "UTC"); err != nil {
		return fmt.Errorf("failed to set UTC timezone override: %w", err)
	}
	if err := client.SetLocaleOverride(ctx, "en-US"); err != nil {
		return fmt.Errorf("failed to set locale override: %w", err)
	}
	if err := client.AddScriptToEvaluateOnNewDocument(ctx, vrtDocumentInitScript); err != nil {
		return fmt.Errorf("failed to register VRT init script: %w", err)
	}
	return nil
}

func waitForVRTBridge(ctx context.Context, client *cdp.Client, expectedStoryID string) (vrtBridgeState, error) {
	ticker := time.NewTicker(bridgePollInterval)
	defer ticker.Stop()

	for {
		state, done, err := pollVRTBridgeOnce(ctx, client, expectedStoryID)
		if err != nil || done {
			return state, err
		}
		select {
		case <-ctx.Done():
			return vrtBridgeState{}, fmt.Errorf("timed out waiting for Storybook VRT bridge: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func pollVRTBridgeOnce(ctx context.Context, client *cdp.Client, expectedStoryID string) (vrtBridgeState, bool, error) {
	encodedID, err := json.Marshal(expectedStoryID)
	if err != nil {
		return vrtBridgeState{}, false, fmt.Errorf("failed to marshal story id: %w", err)
	}
	probeExpr := fmt.Sprintf(bridgeProbeExprFormat, string(encodedID))
	var probe bridgeProbeResult
	err = client.Evaluate(ctx, probeExpr, false, &probe)
	if err != nil {
		var transportErr *cdp.TransportError
		if errors.As(err, &transportErr) {
			return vrtBridgeState{}, false, fmt.Errorf("CDP connection lost while loading story: %w", err)
		}
		return vrtBridgeState{}, false, nil
	}
	if probe.HasBridge {
		var state vrtBridgeState
		if err := client.Evaluate(ctx, "window.__khiVRT.waitForReady()", true, &state); err != nil {
			return vrtBridgeState{}, false, fmt.Errorf("failed waiting for story readiness: %w", err)
		}
		return state, true, nil
	}
	if probe.LegacyReady {
		var state vrtBridgeState
		if err := client.Evaluate(ctx, legacyFallbackReadyExpr, true, &state); err != nil {
			return vrtBridgeState{}, false, fmt.Errorf("failed waiting for legacy story readiness: %w", err)
		}
		return state, true, nil
	}
	return vrtBridgeState{}, false, nil
}

func applyCustomViewport(
	ctx context.Context,
	client *cdp.Client,
	vp *vrtViewport,
	defaultWidth int,
	defaultHeight int,
) error {
	if vp == nil || vp.Width <= 0 || vp.Height <= 0 {
		return nil
	}
	if vp.Width == defaultWidth && vp.Height == defaultHeight {
		return nil
	}
	if err := client.SetViewportSize(ctx, vp.Width, vp.Height, 1); err != nil {
		return fmt.Errorf("failed to apply custom story viewport: %w", err)
	}
	const settleExpr = `typeof window.__khiVRTWaitSettled === 'function' ? window.__khiVRTWaitSettled(5000) : Promise.resolve()`
	if err := client.Evaluate(ctx, settleExpr, true, nil); err != nil {
		return fmt.Errorf("failed to settle custom viewport frames: %w", err)
	}
	return nil
}

func applyMaskSelectors(ctx context.Context, client *cdp.Client, selectors []string) error {
	if len(selectors) == 0 {
		return nil
	}
	encoded, err := json.Marshal(selectors)
	if err != nil {
		return fmt.Errorf("failed to marshal mask selectors: %w", err)
	}
	expr := fmt.Sprintf("(function() { %s applyKhiVrtMasks(%s); })()", maskHelperScript, string(encoded))
	if err := client.Evaluate(ctx, expr, false, nil); err != nil {
		return fmt.Errorf("failed to apply mask selectors: %w", err)
	}
	return nil
}

func collectPageErrors(ctx context.Context, client *cdp.Client) string {
	var errs []string
	if err := client.Evaluate(ctx, "window.__khiVRTPageErrors || []", false, &errs); err != nil {
		return ""
	}
	return strings.Join(errs, "; ")
}

func skippedEntry(entry SnapshotEntry, start time.Time) SnapshotEntry {
	entry.Skipped = true
	entry.DurationMs = time.Since(start).Milliseconds()
	return entry
}

func failedEntry(entry SnapshotEntry, msg string, start time.Time) SnapshotEntry {
	entry.Error = msg
	entry.DurationMs = time.Since(start).Milliseconds()
	return entry
}

func finalizeEntry(
	outputDir string,
	entry SnapshotEntry,
	pngBytes []byte,
	diffPixelRatio float64,
	start time.Time,
) SnapshotEntry {
	cfg, err := png.DecodeConfig(bytes.NewReader(pngBytes))
	if err != nil {
		return failedEntry(entry, fmt.Sprintf("captured invalid PNG: %v", err), start)
	}
	relPath := ImageRelPathForID(entry.ID)
	absPath := filepath.Join(outputDir, filepath.FromSlash(relPath))
	if err := os.WriteFile(absPath, pngBytes, 0o600); err != nil {
		return failedEntry(entry, fmt.Sprintf("failed to write screenshot: %v", err), start)
	}

	sum := sha256.Sum256(pngBytes)
	entry.ImageRelPath = relPath
	entry.SHA256 = hex.EncodeToString(sum[:])
	entry.Width = cfg.Width
	entry.Height = cfg.Height
	entry.DiffPixelRatio = diffPixelRatio
	entry.DurationMs = time.Since(start).Milliseconds()
	return entry
}
