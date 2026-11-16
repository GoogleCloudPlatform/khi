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
	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/scene"
)

const (
	defaultTimelineRenderTimeout      = 30 * time.Second
	defaultTimelineDeviceScaleFactor  = 2
	fallbackTimelineDeviceScaleFactor = 1
	// maxTimelinePhysicalDimension caps the rendered backing store dimension so SwiftShader
	// WebGL2 and Chromium Canvas2D stay safely within an 8192px texture and renderbuffer limit.
	maxTimelinePhysicalDimension = 8192
)

var _ TimelineImageRenderer = (*ChromeTimelineImageRenderer)(nil)

// TimelineRenderRequest holds the parameters passed to the frontend /render/mcp-timeline page.
type TimelineRenderRequest struct {
	// SceneID is the immutable scene ID associated with this render request, if any.
	SceneID string `json:"sceneId,omitempty"`
	// ParentSceneID is the parent scene ID from which this request was derived, if any.
	ParentSceneID string `json:"parentSceneId,omitempty"`
	// InspectionID is the unique identifier of the target inspection.
	InspectionID string `json:"inspectionId"`
	// TimelineQuery is the Timeline CEL filter expression.
	TimelineQuery string `json:"timelineQuery,omitempty"`
	// TimelineExclusionQuery is the Timeline CEL exclusion filter expression.
	TimelineExclusionQuery string `json:"timelineExclusionQuery,omitempty"`
	// LogQuery is the Log CEL filter expression.
	LogQuery string `json:"logQuery,omitempty"`
	// ExcludeNoLogs controls whether timelines with no matching logs are omitted.
	ExcludeNoLogs bool `json:"excludeNoLogs,omitempty"`
	// FilterStartTimeNs is the inclusive filter start timestamp in Unix nanoseconds as a string.
	FilterStartTimeNs string `json:"filterStartTimeNs,omitempty"`
	// FilterEndTimeNs is the inclusive filter end timestamp in Unix nanoseconds as a string.
	FilterEndTimeNs string `json:"filterEndTimeNs,omitempty"`
	// ViewStartTimeMs is the optional viewport start timestamp in Unix milliseconds.
	ViewStartTimeMs *int64 `json:"viewStartTimeMs,omitempty"`
	// ViewEndTimeMs is the optional viewport end timestamp in Unix milliseconds.
	ViewEndTimeMs *int64 `json:"viewEndTimeMs,omitempty"`
	// TimelineIDs restricts rendering to the specified timelines and their ancestors when non-empty.
	TimelineIDs []uint32 `json:"timelineIds,omitempty"`
	// MaxDepth limits the rendered timeline tree depth (0 means unlimited).
	MaxDepth int `json:"maxDepth,omitempty"`
	// MaxNodes limits the number of timeline rows rendered.
	MaxNodes int `json:"maxNodes,omitempty"`
	// TimezoneShiftHours specifies the UTC offset in hours for the timeline ruler.
	TimezoneShiftHours float64 `json:"timezoneShiftHours,omitempty"`
	// Overlay holds resolved visual annotations and evidence cards to render on the timeline.
	Overlay *scene.Overlay `json:"overlay,omitempty"`
}

// TimelineRenderResult holds the captured PNG image and measurement metadata.
type TimelineRenderResult struct {
	// PNGBytes contains the encoded PNG screenshot bytes.
	PNGBytes []byte
	// RenderedCount is the number of timeline rows included in the screenshot.
	RenderedCount int
	// TotalMatched is the total number of timelines matching the filter before truncation.
	TotalMatched int
	// HasMore reports whether matching timelines were omitted due to MaxNodes.
	HasMore bool
	// ViewStartTimeMs is the rendered viewport start timestamp in Unix milliseconds.
	ViewStartTimeMs int64
	// ViewEndTimeMs is the rendered viewport end timestamp in Unix milliseconds.
	ViewEndTimeMs int64
}

// TimelineImageRenderer renders timeline views into PNG screenshots.
type TimelineImageRenderer interface {
	// RenderTimeline applies req and captures a PNG screenshot of the rendered timeline view.
	RenderTimeline(ctx context.Context, req TimelineRenderRequest) (*TimelineRenderResult, error)
	// Close releases resources held by the renderer.
	Close() error
}

// ChromeTimelineImageRenderer renders timelines using a warm headless Chrome page over CDP.
type ChromeTimelineImageRenderer struct {
	mu        sync.Mutex
	browser   *cdp.Browser
	baseURL   string
	client    *cdp.Client
	pageReady bool
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

// NewChromeTimelineImageRenderer creates a new ChromeTimelineImageRenderer using the provided CDP Browser and server baseURL.
func NewChromeTimelineImageRenderer(browser *cdp.Browser, baseURL string) *ChromeTimelineImageRenderer {
	return &ChromeTimelineImageRenderer{
		browser: browser,
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

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
	renderURL := r.baseURL + "/render/mcp-timeline"
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
			return fmt.Errorf("CDP connection failed while waiting for /render/mcp-timeline bridge initialization: %w", err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for /render/mcp-timeline bridge initialization: %w", ctx.Err())
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

	scaleFactor := timelineDeviceScaleFactor(measure.Width, measure.Height)
	if err := r.client.SetViewportSize(ctx, measure.Width, measure.Height, float64(scaleFactor)); err != nil {
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

// timelineDeviceScaleFactor returns Retina 2x scale when the physical dimensions fit within
// WebGL backing store limits, and falls back to 1x for oversized layouts.
func timelineDeviceScaleFactor(width, height int) int {
	if max(width, height)*defaultTimelineDeviceScaleFactor > maxTimelinePhysicalDimension {
		return fallbackTimelineDeviceScaleFactor
	}
	return defaultTimelineDeviceScaleFactor
}
