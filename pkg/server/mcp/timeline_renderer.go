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
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

var devToolsWSRegex = regexp.MustCompile(`DevTools listening on (ws://[^\s]+)`)

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

type cdpRequest struct {
	ID     int            `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

type cdpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cdpResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *cdpError       `json:"error"`
}

type cdpEvaluateResult struct {
	Result struct {
		Value json.RawMessage `json:"value"`
	} `json:"result"`
	ExceptionDetails *struct {
		Text      string `json:"text"`
		Exception *struct {
			Description string `json:"description"`
		} `json:"exception"`
	} `json:"exceptionDetails"`
}

type cdpTargetInfo struct {
	Type                 string `json:"type"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// cdpClient manages a single Chrome DevTools Protocol WebSocket connection.
type cdpClient struct {
	ws     *websocket.Conn
	nextID int
}

func dialCDPClient(ctx context.Context, wsURL string) (*cdpClient, error) {
	cfg, err := websocket.NewConfig(wsURL, "http://127.0.0.1")
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket config: %w", err)
	}
	var d net.Dialer
	rawConn, err := d.DialContext(ctx, "tcp", cfg.Location.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to dial CDP tcp endpoint: %w", err)
	}
	conn, err := websocket.NewClient(cfg, rawConn)
	if err != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("failed to establish CDP websocket client: %w", err)
	}
	return &cdpClient{ws: conn}, nil
}

func (c *cdpClient) call(ctx context.Context, method string, params map[string]any, out any) error {
	c.nextID++
	reqID := c.nextID

	if deadline, ok := ctx.Deadline(); ok {
		_ = c.ws.SetDeadline(deadline)
	}
	stop := context.AfterFunc(ctx, func() {
		_ = c.ws.SetDeadline(time.Now())
	})
	defer func() {
		stop()
		_ = c.ws.SetDeadline(time.Time{})
	}()

	if err := websocket.JSON.Send(c.ws, cdpRequest{
		ID:     reqID,
		Method: method,
		Params: params,
	}); err != nil {
		return fmt.Errorf("failed to send CDP command %s: %w", method, err)
	}

	for {
		var raw string
		if err := websocket.Message.Receive(c.ws, &raw); err != nil {
			return fmt.Errorf("failed to receive CDP response for %s: %w", method, err)
		}
		var resp cdpResponse
		if err := json.Unmarshal([]byte(raw), &resp); err != nil {
			continue
		}
		if resp.ID != reqID {
			continue
		}
		if resp.Error != nil {
			return fmt.Errorf("CDP command %s failed (%d): %s", method, resp.Error.Code, resp.Error.Message)
		}
		if out != nil && len(resp.Result) > 0 {
			if err := json.Unmarshal(resp.Result, out); err != nil {
				return fmt.Errorf("failed to unmarshal CDP result for %s: %w", method, err)
			}
		}
		return nil
	}
}

func (c *cdpClient) evaluate(ctx context.Context, expression string, awaitPromise bool, out any) error {
	var evalRes cdpEvaluateResult
	err := c.call(ctx, "Runtime.evaluate", map[string]any{
		"expression":    expression,
		"awaitPromise":  awaitPromise,
		"returnByValue": true,
	}, &evalRes)
	if err != nil {
		return err
	}
	if evalRes.ExceptionDetails != nil {
		msg := evalRes.ExceptionDetails.Text
		if evalRes.ExceptionDetails.Exception != nil && evalRes.ExceptionDetails.Exception.Description != "" {
			msg = evalRes.ExceptionDetails.Exception.Description
		}
		return fmt.Errorf("javascript evaluation error: %s", msg)
	}
	if out != nil && len(evalRes.Result.Value) > 0 {
		if err := json.Unmarshal(evalRes.Result.Value, out); err != nil {
			return fmt.Errorf("failed to decode evaluation result: %w", err)
		}
	}
	return nil
}

func (c *cdpClient) close() error {
	return c.ws.Close()
}

// ChromeTimelineImageRenderer renders timelines using a warm headless Chrome page over CDP.
type ChromeTimelineImageRenderer struct {
	mu            sync.Mutex
	baseURL       string
	pageWSURL     string
	cmd           *exec.Cmd
	browserCancel context.CancelFunc
	userDataDir   string
	client        *cdpClient
	pageReady     bool
}

var _ TimelineImageRenderer = (*ChromeTimelineImageRenderer)(nil)

// NewChromeTimelineImageRenderer creates a new ChromeTimelineImageRenderer targeting baseURL.
func NewChromeTimelineImageRenderer(baseURL string) *ChromeTimelineImageRenderer {
	return &ChromeTimelineImageRenderer{
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
		// Reset the browser session on failure so subsequent calls can recover cleanly.
		_ = r.closeLocked()
		return nil, err
	}
	return res, nil
}

// Close terminates the headless browser process and removes its temporary profile directory.
func (r *ChromeTimelineImageRenderer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closeLocked()
}

func (r *ChromeTimelineImageRenderer) ensurePageReadyLocked(ctx context.Context) error {
	if r.client != nil && r.pageReady {
		return nil
	}

	wsURL := r.pageWSURL
	if wsURL == "" {
		launchedWSURL, err := r.launchBrowserLocked(ctx)
		if err != nil {
			return err
		}
		wsURL = launchedWSURL
	}

	client, err := dialCDPClient(ctx, wsURL)
	if err != nil {
		_ = r.closeLocked()
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

func (r *ChromeTimelineImageRenderer) launchBrowserLocked(ctx context.Context) (string, error) {
	chromeBin, err := findChromeBinary()
	if err != nil {
		return "", err
	}

	userDataDir, err := os.MkdirTemp("", "khi-timeline-renderer-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp chrome profile dir: %w", err)
	}
	r.userDataDir = userDataDir

	browserCtx, browserCancel := context.WithCancel(context.Background())
	r.browserCancel = browserCancel

	cmd := exec.CommandContext(
		browserCtx,
		chromeBin,
		"--headless",
		"--no-sandbox",
		"--use-gl=angle",
		"--use-angle=swiftshader",
		"--ignore-gpu-blocklist",
		"--disable-dev-shm-usage",
		"--disable-extensions",
		"--password-store=basic",
		"--no-proxy-server",
		"--disable-background-networking",
		"--enable-webgl",
		"--enable-unsafe-swiftshader",
		"--hide-scrollbars",
		"--window-size=1600,900",
		"--remote-debugging-port=0",
		"--remote-allow-origins=http://127.0.0.1",
		"--user-data-dir="+userDataDir,
		"about:blank",
	)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		_ = r.closeLocked()
		return "", fmt.Errorf("failed to attach chrome stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = r.closeLocked()
		return "", fmt.Errorf("failed to start headless chrome: %w", err)
	}
	r.cmd = cmd

	browserWSURL, err := waitForDevToolsURL(ctx, stderrPipe)
	if err != nil {
		_ = r.closeLocked()
		return "", err
	}

	pageWSURL, err := resolvePageWebSocketURL(ctx, browserWSURL)
	if err != nil {
		_ = r.closeLocked()
		return "", err
	}
	return pageWSURL, nil
}

func (r *ChromeTimelineImageRenderer) navigateToRenderPageLocked(ctx context.Context) error {
	if err := r.client.call(ctx, "Page.enable", nil, nil); err != nil {
		return err
	}
	if err := r.client.call(ctx, "Runtime.enable", nil, nil); err != nil {
		return err
	}

	renderURL := r.baseURL + "/render/timeline"
	if err := r.client.call(ctx, "Page.navigate", map[string]any{"url": renderURL}, nil); err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", renderURL, err)
	}

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		var ready bool
		if err := r.client.evaluate(ctx, "typeof window.__khiTimelineRenderer !== 'undefined'", false, &ready); err == nil && ready {
			return nil
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
	applyExpr := fmt.Sprintf("window.__khiTimelineRenderer.applyAndMeasure(%s)", string(reqJSON))
	if err := r.client.evaluate(ctx, applyExpr, true, &measure); err != nil {
		return nil, fmt.Errorf("failed to apply timeline render parameters: %w", err)
	}

	if err := r.client.call(ctx, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width":             measure.Width,
		"height":            measure.Height,
		"deviceScaleFactor": 1,
		"mobile":            false,
	}, nil); err != nil {
		return nil, fmt.Errorf("failed to set viewport dimensions: %w", err)
	}

	if err := r.client.evaluate(ctx, "window.__khiTimelineRenderer.waitForStableFrame()", true, nil); err != nil {
		return nil, fmt.Errorf("failed waiting for stable timeline frame: %w", err)
	}

	var screenshotRes struct {
		Data string `json:"data"`
	}
	if err := r.client.call(ctx, "Page.captureScreenshot", map[string]any{
		"format": "png",
	}, &screenshotRes); err != nil {
		return nil, fmt.Errorf("failed to capture timeline screenshot: %w", err)
	}

	pngBytes, err := base64.StdEncoding.DecodeString(screenshotRes.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode screenshot PNG data: %w", err)
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
	var errs []error
	if r.client != nil {
		if err := r.client.close(); err != nil {
			errs = append(errs, err)
		}
		r.client = nil
	}
	if r.browserCancel != nil {
		r.browserCancel()
		r.browserCancel = nil
	}
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
		_ = r.cmd.Wait()
		r.cmd = nil
	}
	if r.userDataDir != "" {
		if err := os.RemoveAll(r.userDataDir); err != nil {
			errs = append(errs, err)
		}
		r.userDataDir = ""
	}
	return errors.Join(errs...)
}

func findChromeBinary() (string, error) {
	if envBin := os.Getenv("CHROME_BIN"); envBin != "" {
		if path, err := exec.LookPath(envBin); err == nil {
			return path, nil
		}
	}
	candidates := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
	}
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", errors.New("no Chrome or Chromium binary found in PATH or CHROME_BIN")
}

func waitForDevToolsURL(ctx context.Context, stderr io.Reader) (string, error) {
	urlCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if matches := devToolsWSRegex.FindStringSubmatch(line); len(matches) == 2 {
				urlCh <- matches[1]
				_, _ = io.Copy(io.Discard, stderr)
				return
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		} else {
			errCh <- errors.New("chrome exited before printing DevTools listening URL")
		}
	}()

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("timed out waiting for Chrome DevTools endpoint: %w", ctx.Err())
	case err := <-errCh:
		return "", err
	case wsURL := <-urlCh:
		return wsURL, nil
	}
}

func resolvePageWebSocketURL(ctx context.Context, browserWSURL string) (string, error) {
	// Extract host:port from ws://127.0.0.1:<port>/devtools/browser/<id>
	trimmed := strings.TrimPrefix(browserWSURL, "ws://")
	slashIdx := strings.IndexByte(trimmed, '/')
	if slashIdx <= 0 {
		return "", fmt.Errorf("invalid browser DevTools URL: %s", browserWSURL)
	}
	hostPort := trimmed[:slashIdx]
	listURL := fmt.Sprintf("http://%s/json/list", hostPort)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query %s: %w", listURL, err)
	}
	defer resp.Body.Close()

	var targets []cdpTargetInfo
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return "", fmt.Errorf("failed to decode CDP target list: %w", err)
	}
	for _, target := range targets {
		if target.Type == "page" && target.WebSocketDebuggerURL != "" {
			return target.WebSocketDebuggerURL, nil
		}
	}
	return "", errors.New("no page target found in Chrome DevTools target list")
}
