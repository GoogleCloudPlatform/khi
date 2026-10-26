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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/cdp"
)

// TimelineRenderRequest holds the parameters passed to the frontend /render/timeline page.
type TimelineRenderRequest struct {
	InspectionID           string   `json:"inspectionId"`
	TimelineQuery          string   `json:"timelineQuery,omitempty"`
	TimelineExclusionQuery string   `json:"timelineExclusionQuery,omitempty"`
	LogQuery               string   `json:"logQuery,omitempty"`
	ExcludeNoLogs          bool     `json:"excludeNoLogs,omitempty"`
	FilterStartTimeNs      string   `json:"filterStartTimeNs,omitempty"`
	FilterEndTimeNs        string   `json:"filterEndTimeNs,omitempty"`
	ViewStartTimeMs        *int64   `json:"viewStartTimeMs,omitempty"`
	ViewEndTimeMs          *int64   `json:"viewEndTimeMs,omitempty"`
	TimelineIDs            []uint32 `json:"timelineIds,omitempty"`
	MaxDepth               int      `json:"maxDepth,omitempty"`
	MaxNodes               int      `json:"maxNodes,omitempty"`
	Width                  int      `json:"width,omitempty"`
	TimezoneShiftHours     float64  `json:"timezoneShiftHours,omitempty"`
}

// TimelineRenderResult holds the captured PNG image and measurement metadata.
type TimelineRenderResult struct {
	PNGBytes        []byte
	Width           int
	Height          int
	RenderedCount   int
	TotalMatched    int
	HasMore         bool
	ViewStartTimeMs int64
	ViewEndTimeMs   int64
}

// TimelineImageRenderer renders timeline views into PNG screenshots.
type TimelineImageRenderer interface {
	RenderTimeline(ctx context.Context, req TimelineRenderRequest) (*TimelineRenderResult, error)
	Close() error
}

type timelineMeasurePayload struct {
	Width           int   `json:"width"`
	Height          int   `json:"height"`
	RenderedCount   int   `json:"renderedCount"`
	TotalMatched    int   `json:"totalMatched"`
	HasMore         bool  `json:"hasMore"`
	ViewStartTimeMs int64 `json:"viewStartTimeMs"`
	ViewEndTimeMs   int64 `json:"viewEndTimeMs"`
}

// ChromeTimelineImageRenderer renders timelines using a warm headless Chrome page over CDP.
type ChromeTimelineImageRenderer struct {
	mu        sync.Mutex
	browser   *cdp.Browser
	baseURL   string
	client    *cdp.Client
	pageReady bool
}

var _ TimelineImageRenderer = (*ChromeTimelineImageRenderer)(nil)

// NewChromeTimelineImageRenderer creates a new ChromeTimelineImageRenderer using the provided CDP Browser and server baseURL.
func NewChromeTimelineImageRenderer(browser *cdp.Browser, baseURL string) *ChromeTimelineImageRenderer {
	return &ChromeTimelineImageRenderer{
		browser: browser,
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

const defaultTimelineRenderTimeout = 30 * time.Second

// RenderTimeline applies the requested timeline parameters and captures a PNG screenshot.
func (r *ChromeTimelineImageRenderer) RenderTimeline(ctx context.Context, req TimelineRenderRequest) (*TimelineRenderResult, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimelineRenderTimeout)
		defer cancel()
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.ensurePageReadyLocked(ctx); err != nil {
		return nil, err
	}

	res, err := r.executeRenderLocked(ctx, req)
	if err != nil {
		// Reset the CDP page client on failure so subsequent calls reconnect cleanly.
		_ = r.closeLocked()
		return nil, err
	}
	return res, nil
}

// Close closes the active CDP page client connection.
func (r *ChromeTimelineImageRenderer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closeLocked()
}

func (r *ChromeTimelineImageRenderer) ensurePageReadyLocked(ctx context.Context) error {
	if r.client != nil && r.pageReady {
		return nil
	}
	if r.client != nil {
		_ = r.client.Close()
		r.client = nil
	}

	client, err := r.browser.DialPage(ctx)
	if err != nil {
		return err
	}
	r.client = client

	if err := r.navigateToRenderPageLocked(ctx); err != nil {
		_ = r.closeLocked()
		return err
	}
	r.pageReady = true
	return nil
}

func (r *ChromeTimelineImageRenderer) navigateToRenderPageLocked(ctx context.Context) error {
	renderURL := r.baseURL + "/render/timeline"
	if err := r.client.Navigate(ctx, renderURL); err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", renderURL, err)
	}

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		var ready bool
		err := r.client.Evaluate(ctx, "typeof window.__khiCDPBridge?.timelineRenderer !== 'undefined'", false, &ready)
		if err == nil && ready {
			return nil
		}
		var transportErr *cdp.TransportError
		if errors.As(err, &transportErr) {
			return fmt.Errorf("CDP connection failed while waiting for /render/timeline bridge initialization: %w", err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for /render/timeline bridge initialization: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (r *ChromeTimelineImageRenderer) executeRenderLocked(ctx context.Context, req TimelineRenderRequest) (*TimelineRenderResult, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TimelineRenderRequest: %w", err)
	}

	var measure timelineMeasurePayload
	applyExpr := fmt.Sprintf("window.__khiCDPBridge.timelineRenderer.applyAndMeasure(%s)", string(reqJSON))
	if err := r.client.Evaluate(ctx, applyExpr, true, &measure); err != nil {
		return nil, fmt.Errorf("failed to apply timeline render parameters: %w", err)
	}

	if err := r.client.SetViewportSize(ctx, measure.Width, measure.Height); err != nil {
		return nil, fmt.Errorf("failed to set viewport dimensions: %w", err)
	}

	if err := r.client.Evaluate(ctx, "window.__khiCDPBridge.timelineRenderer.waitForStableFrame()", true, nil); err != nil {
		return nil, fmt.Errorf("failed waiting for stable timeline frame: %w", err)
	}

	pngBytes, err := r.client.CapturePNGScreenshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to capture timeline screenshot: %w", err)
	}

	return &TimelineRenderResult{
		PNGBytes:        pngBytes,
		Width:           measure.Width,
		Height:          measure.Height,
		RenderedCount:   measure.RenderedCount,
		TotalMatched:    measure.TotalMatched,
		HasMore:         measure.HasMore,
		ViewStartTimeMs: measure.ViewStartTimeMs,
		ViewEndTimeMs:   measure.ViewEndTimeMs,
	}, nil
}

func (r *ChromeTimelineImageRenderer) closeLocked() error {
	r.pageReady = false
	if r.client != nil {
		err := r.client.Close()
		r.client = nil
		return err
	}
	return nil
}
