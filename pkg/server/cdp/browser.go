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

package cdp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// DefaultStartTimeout is the default timeout for launching the headless Chrome process.
const DefaultStartTimeout = 30 * time.Second

const (
	defaultWindowWidth  = 1600
	defaultWindowHeight = 900
)

var devToolsWSRegex = regexp.MustCompile(`DevTools listening on (ws://[^\s]+)`)

// BrowserOptions configures headless Chrome launch parameters.
type BrowserOptions struct {
	// WindowWidth sets the initial browser window width in pixels (defaults to 1600).
	WindowWidth int
	// WindowHeight sets the initial browser window height in pixels (defaults to 900).
	WindowHeight int
	// ExtraArgs appends additional Chromium command-line flags.
	ExtraArgs []string
}

// BrowserVersion holds version metadata reported by the Chrome DevTools /json/version endpoint.
type BrowserVersion struct {
	// Browser is the Chrome product and version string (e.g., "HeadlessChrome/136.0.7103.92").
	Browser string `json:"Browser"`
	// ProtocolVersion is the Chrome DevTools Protocol version string.
	ProtocolVersion string `json:"Protocol-Version"`
	// UserAgent is the browser User-Agent string.
	UserAgent string `json:"User-Agent"`
	// V8Version is the embedded V8 engine version string.
	V8Version string `json:"V8-Version"`
}

// Browser manages a headless Chrome process and its Chrome DevTools Protocol page endpoint.
type Browser struct {
	mu               sync.Mutex
	pageWSURL        string
	devToolsHostPort string
	cmd              *exec.Cmd
	browserCancel    context.CancelFunc
	userDataDir      string
}

type targetInfo struct {
	Type                 string `json:"type"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// FindChromeBinary locates a Chrome or Chromium executable from CHROME_BIN or standard PATH names.
func FindChromeBinary() (string, error) {
	if envBin := os.Getenv("CHROME_BIN"); envBin != "" {
		if path, err := exec.LookPath(envBin); err == nil {
			return path, nil
		}
		return "", fmt.Errorf("chrome binary specified in CHROME_BIN (%q) was not found: %w", envBin, exec.ErrNotFound)
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

// StartBrowser launches a headless Chrome process with opts and resolves its initial page WebSocket URL.
func StartBrowser(ctx context.Context, opts BrowserOptions) (*Browser, error) {
	b := &Browser{}
	if err := b.start(ctx, opts); err != nil {
		return nil, err
	}
	return b, nil
}

// NewBrowserFromPageWSURL creates a Browser targeting an existing CDP page WebSocket endpoint without managing a process.
func NewBrowserFromPageWSURL(pageWSURL string) *Browser {
	hostPort, _ := extractWebSocketHostPort(pageWSURL)
	return &Browser{
		pageWSURL:        pageWSURL,
		devToolsHostPort: hostPort,
	}
}

// DialPage connects a new CDP Client to the browser's page target.
func (b *Browser) DialPage(ctx context.Context) (*Client, error) {
	b.mu.Lock()
	wsURL := b.pageWSURL
	b.mu.Unlock()

	if wsURL == "" {
		return nil, errors.New("headless Chrome browser has no active page WebSocket URL")
	}
	return Dial(ctx, wsURL)
}

// Version queries the DevTools /json/version endpoint for browser version metadata.
func (b *Browser) Version(ctx context.Context) (BrowserVersion, error) {
	b.mu.Lock()
	hostPort := b.devToolsHostPort
	b.mu.Unlock()

	if hostPort == "" {
		return BrowserVersion{}, errors.New("headless Chrome browser has no active DevTools host")
	}
	return fetchBrowserVersion(ctx, hostPort)
}

// Close terminates the headless Chrome process and removes its temporary profile directory.
func (b *Browser) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closeLocked()
}

func (b *Browser) start(ctx context.Context, opts BrowserOptions) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	chromeBin, err := FindChromeBinary()
	if err != nil {
		return err
	}

	userDataDir, err := os.MkdirTemp("", "khi-headless-chrome-*")
	if err != nil {
		return fmt.Errorf("failed to create temp chrome profile dir: %w", err)
	}
	b.userDataDir = userDataDir

	browserCtx, browserCancel := context.WithCancel(context.Background())
	b.browserCancel = browserCancel

	args := buildChromeArgs(userDataDir, opts)
	cmd := exec.CommandContext(browserCtx, chromeBin, args...)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		_ = b.closeLocked()
		return fmt.Errorf("failed to attach chrome stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = b.closeLocked()
		return fmt.Errorf("failed to start headless chrome: %w", err)
	}
	b.cmd = cmd

	browserWSURL, err := waitForDevToolsURL(ctx, stderrPipe)
	if err != nil {
		_ = b.closeLocked()
		return err
	}

	hostPort, err := extractWebSocketHostPort(browserWSURL)
	if err != nil {
		_ = b.closeLocked()
		return err
	}
	pageWSURL, err := resolvePageWebSocketURL(ctx, browserWSURL)
	if err != nil {
		_ = b.closeLocked()
		return err
	}
	b.devToolsHostPort = hostPort
	b.pageWSURL = pageWSURL
	return nil
}

func (b *Browser) closeLocked() error {
	b.pageWSURL = ""
	b.devToolsHostPort = ""
	var errs []error
	if b.browserCancel != nil {
		b.browserCancel()
		b.browserCancel = nil
	}
	if b.cmd != nil && b.cmd.Process != nil {
		_ = b.cmd.Process.Kill()
		_ = b.cmd.Wait()
		b.cmd = nil
	}
	if b.userDataDir != "" {
		if err := os.RemoveAll(b.userDataDir); err != nil {
			errs = append(errs, err)
		}
		b.userDataDir = ""
	}
	return errors.Join(errs...)
}

func buildChromeArgs(userDataDir string, opts BrowserOptions) []string {
	width := opts.WindowWidth
	if width <= 0 {
		width = defaultWindowWidth
	}
	height := opts.WindowHeight
	if height <= 0 {
		height = defaultWindowHeight
	}
	args := []string{
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
		fmt.Sprintf("--window-size=%d,%d", width, height),
		"--remote-debugging-port=0",
		"--remote-allow-origins=http://127.0.0.1",
		"--user-data-dir=" + userDataDir,
	}
	args = append(args, opts.ExtraArgs...)
	args = append(args, "about:blank")
	return args
}

func extractWebSocketHostPort(wsURL string) (string, error) {
	trimmed, ok := strings.CutPrefix(wsURL, "ws://")
	if !ok {
		return "", fmt.Errorf("invalid browser DevTools URL: %s", wsURL)
	}
	hostPort, _, _ := strings.Cut(trimmed, "/")
	if hostPort == "" {
		return "", fmt.Errorf("invalid browser DevTools URL: %s", wsURL)
	}
	return hostPort, nil
}

func fetchBrowserVersion(ctx context.Context, hostPort string) (BrowserVersion, error) {
	versionURL := fmt.Sprintf("http://%s/json/version", hostPort)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionURL, nil)
	if err != nil {
		return BrowserVersion{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return BrowserVersion{}, fmt.Errorf("failed to query %s: %w", versionURL, err)
	}
	defer resp.Body.Close()

	var version BrowserVersion
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return BrowserVersion{}, fmt.Errorf("failed to decode CDP version metadata: %w", err)
	}
	return version, nil
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

	var targets []targetInfo
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
