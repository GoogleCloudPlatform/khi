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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/websocket"
)

func TestWaitForDevToolsURL(t *testing.T) {
	testCases := []struct {
		name      string
		stderr    string
		wantURL   string
		wantError bool
	}{
		{
			name:    "extracts ws URL from stderr output",
			stderr:  "some warning\nDevTools listening on ws://127.0.0.1:41234/devtools/browser/uuid-123\n",
			wantURL: "ws://127.0.0.1:41234/devtools/browser/uuid-123",
		},
		{
			name:      "returns error when stderr closes without URL",
			stderr:    "fatal: failed to initialize sandbox\n",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotURL, err := waitForDevToolsURL(context.Background(), strings.NewReader(tc.stderr))
			if (err != nil) != tc.wantError {
				t.Fatalf("waitForDevToolsURL() error = %v, wantError %v", err, tc.wantError)
			}
			if gotURL != tc.wantURL {
				t.Errorf("waitForDevToolsURL() = %q, want %q", gotURL, tc.wantURL)
			}
		})
	}
}

func TestResolvePageWebSocketURL(t *testing.T) {
	testCases := []struct {
		name         string
		browserWSURL string
		listBody     string
		wantWSURL    string
		wantError    bool
	}{
		{
			name:      "returns existing page target from /json/list",
			listBody:  `[{"id":"worker-1","type":"service_worker","webSocketDebuggerUrl":"ws://127.0.0.1:9222/devtools/worker/1"},{"id":"page-1","type":"page","webSocketDebuggerUrl":"ws://127.0.0.1:9222/devtools/page/1"}]`,
			wantWSURL: "ws://127.0.0.1:9222/devtools/page/1",
		},
		{
			name:      "returns error when /json/list has no page target",
			listBody:  `[{"id":"worker-1","type":"service_worker","webSocketDebuggerUrl":"ws://127.0.0.1:9222/devtools/worker/1"}]`,
			wantError: true,
		},
		{
			name:         "returns error for malformed browser DevTools URL",
			browserWSURL: "ws://invalid-without-slash",
			wantError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/json/list" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(tc.listBody))
					return
				}
				http.NotFound(w, r)
			}))
			defer srv.Close()

			browserWSURL := tc.browserWSURL
			if browserWSURL == "" {
				browserWSURL = "ws" + strings.TrimPrefix(srv.URL, "http") + "/devtools/browser/123"
			}
			gotURL, err := resolvePageWebSocketURL(context.Background(), browserWSURL)
			if (err != nil) != tc.wantError {
				t.Fatalf("resolvePageWebSocketURL() error = %v, wantError %v", err, tc.wantError)
			}
			if gotURL != tc.wantWSURL {
				t.Errorf("resolvePageWebSocketURL() = %q, want %q", gotURL, tc.wantWSURL)
			}
		})
	}
}

func TestStartBrowser_InvalidChromeBinFailsFast(t *testing.T) {
	t.Setenv("CHROME_BIN", "/nonexistent/chrome-binary-for-test")
	_, err := StartBrowser(context.Background(), BrowserOptions{})
	if err == nil {
		t.Fatal("StartBrowser() expected error when CHROME_BIN does not exist, got nil")
	}
}

func TestBrowserClose_ClearsStateAndRemovesUserDataDir(t *testing.T) {
	profileDir, err := os.MkdirTemp("", "cdp-close-test-*")
	if err != nil {
		t.Fatalf("os.MkdirTemp() error = %v", err)
	}
	b := &Browser{
		pageWSURL:        "ws://127.0.0.1:9222/devtools/page/1",
		devToolsHostPort: "127.0.0.1:9222",
		userDataDir:      profileDir,
	}
	if err := b.Close(); err != nil {
		t.Fatalf("b.Close() unexpected error = %v", err)
	}
	if b.pageWSURL != "" || b.devToolsHostPort != "" || b.userDataDir != "" {
		t.Errorf("b.Close() left non-empty state: pageWSURL=%q host=%q dir=%q", b.pageWSURL, b.devToolsHostPort, b.userDataDir)
	}
	if _, statErr := os.Stat(profileDir); !os.IsNotExist(statErr) {
		t.Errorf("expected profileDir %q to be removed, stat err = %v", profileDir, statErr)
	}
	if _, err := b.DialPage(context.Background()); err == nil {
		t.Error("b.DialPage() after Close() expected error, got nil")
	}
	if _, err := b.Version(context.Background()); err == nil {
		t.Error("b.Version() after Close() expected error, got nil")
	}
}

func TestBuildChromeArgs(t *testing.T) {
	testCases := []struct {
		name     string
		opts     BrowserOptions
		wantSize string
		wantLast []string
	}{
		{
			name:     "defaults to 1600x900 without extra args",
			opts:     BrowserOptions{},
			wantSize: "--window-size=1600,900",
			wantLast: []string{"--user-data-dir=/tmp/profile", "about:blank"},
		},
		{
			name: "applies custom window size and appends extra args before about:blank",
			opts: BrowserOptions{
				WindowWidth:  1280,
				WindowHeight: 720,
				ExtraArgs:    []string{"--font-render-hinting=none", "--force-color-profile=srgb"},
			},
			wantSize: "--window-size=1280,720",
			wantLast: []string{
				"--user-data-dir=/tmp/profile",
				"--font-render-hinting=none",
				"--force-color-profile=srgb",
				"about:blank",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildChromeArgs("/tmp/profile", tc.opts)
			foundSize := false
			for _, arg := range got {
				if arg == tc.wantSize {
					foundSize = true
					break
				}
			}
			if !foundSize {
				t.Errorf("buildChromeArgs() missing %q in %v", tc.wantSize, got)
			}
			gotTail := got[len(got)-len(tc.wantLast):]
			if diff := cmp.Diff(tc.wantLast, gotTail); diff != "" {
				t.Errorf("buildChromeArgs() tail mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractWebSocketHostPort(t *testing.T) {
	testCases := []struct {
		name      string
		wsURL     string
		want      string
		wantError bool
	}{
		{
			name:  "extracts host and port from full DevTools path",
			wsURL: "ws://127.0.0.1:9222/devtools/page/1",
			want:  "127.0.0.1:9222",
		},
		{
			name:      "rejects non-ws scheme",
			wsURL:     "http://127.0.0.1:9222/devtools/page/1",
			wantError: true,
		},
		{
			name:      "rejects empty host",
			wsURL:     "ws:///devtools/page/1",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractWebSocketHostPort(tc.wsURL)
			if (err != nil) != tc.wantError {
				t.Fatalf("extractWebSocketHostPort(%q) error = %v, wantError %v", tc.wsURL, err, tc.wantError)
			}
			if got != tc.want {
				t.Errorf("extractWebSocketHostPort(%q) = %q, want %q", tc.wsURL, got, tc.want)
			}
		})
	}
}

func TestBrowserVersion(t *testing.T) {
	testCases := []struct {
		name      string
		body      string
		wsURL     string
		want      BrowserVersion
		wantError bool
	}{
		{
			name: "decodes version metadata from /json/version",
			body: `{"Browser":"HeadlessChrome/136.0.7103.92","Protocol-Version":"1.3","User-Agent":"Mozilla/5.0","V8-Version":"13.6.0"}`,
			want: BrowserVersion{
				Browser:         "HeadlessChrome/136.0.7103.92",
				ProtocolVersion: "1.3",
				UserAgent:       "Mozilla/5.0",
				V8Version:       "13.6.0",
			},
		},
		{
			name:      "returns error on malformed json",
			body:      `not-json`,
			wantError: true,
		},
		{
			name:      "returns error when browser has no DevTools host",
			wsURL:     "invalid",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/json/version" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(tc.body))
					return
				}
				http.NotFound(w, r)
			}))
			defer srv.Close()

			wsURL := tc.wsURL
			if wsURL == "" {
				wsURL = "ws" + strings.TrimPrefix(srv.URL, "http") + "/devtools/page/1"
			}
			browser := NewBrowserFromPageWSURL(wsURL)
			got, err := browser.Version(context.Background())
			if (err != nil) != tc.wantError {
				t.Fatalf("browser.Version() error = %v, wantError %v", err, tc.wantError)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("browser.Version() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClient_HighLevelMethodsAndTransportError(t *testing.T) {
	wantPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	type recordedRequest struct {
		Method string
		Params map[string]any
	}
	var mu sync.Mutex
	var gotRequests []recordedRequest
	captureCount := 0

	wsServer := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		for {
			var raw string
			if err := websocket.Message.Receive(ws, &raw); err != nil {
				return
			}
			var req request
			if err := json.Unmarshal([]byte(raw), &req); err != nil {
				return
			}
			mu.Lock()
			gotRequests = append(gotRequests, recordedRequest{
				Method: req.Method,
				Params: req.Params,
			})
			mu.Unlock()

			var res any
			switch req.Method {
			case "Page.enable",
				"Runtime.enable",
				"Emulation.setDeviceMetricsOverride",
				"Emulation.setEmulatedMedia",
				"Emulation.setTimezoneOverride",
				"Emulation.setLocaleOverride",
				"Page.addScriptToEvaluateOnNewDocument":
				res = map[string]any{}
			case "Page.navigate":
				res = map[string]any{"frameId": "f1"}
			case "Runtime.evaluate":
				res = map[string]any{
					"result": map[string]any{
						"value": 42,
					},
				}
			case "Page.captureScreenshot":
				captureCount++
				res = map[string]any{
					"data": base64.StdEncoding.EncodeToString(wantPNG),
				}
			}

			respBytes, _ := json.Marshal(map[string]any{
				"id":     req.ID,
				"result": res,
			})
			_ = websocket.Message.Send(ws, string(respBytes))
			if req.Method == "Page.captureScreenshot" && captureCount >= 2 {
				_ = ws.Close()
				return
			}
		}
	}))
	defer wsServer.Close()

	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	browser := NewBrowserFromPageWSURL(wsURL)
	defer browser.Close()

	client, err := browser.DialPage(context.Background())
	if err != nil {
		t.Fatalf("browser.DialPage() unexpected error = %v", err)
	}
	defer client.Close()

	if err := client.Navigate(context.Background(), "http://127.0.0.1:8080/render/timeline"); err != nil {
		t.Fatalf("client.Navigate() unexpected error = %v", err)
	}

	if err := client.SetViewportSize(context.Background(), 1600, 480, 2); err != nil {
		t.Fatalf("client.SetViewportSize() unexpected error = %v", err)
	}

	if err := client.SetEmulatedMedia(context.Background(), "screen", []EmulatedMediaFeature{
		{Name: "prefers-reduced-motion", Value: "reduce"},
	}); err != nil {
		t.Fatalf("client.SetEmulatedMedia(with features) unexpected error = %v", err)
	}

	if err := client.SetEmulatedMedia(context.Background(), "print", nil); err != nil {
		t.Fatalf("client.SetEmulatedMedia(without features) unexpected error = %v", err)
	}

	if err := client.SetTimezoneOverride(context.Background(), "UTC"); err != nil {
		t.Fatalf("client.SetTimezoneOverride() unexpected error = %v", err)
	}

	if err := client.SetLocaleOverride(context.Background(), "en-US"); err != nil {
		t.Fatalf("client.SetLocaleOverride() unexpected error = %v", err)
	}

	if err := client.AddScriptToEvaluateOnNewDocument(context.Background(), "window.__vrtInit = true;"); err != nil {
		t.Fatalf("client.AddScriptToEvaluateOnNewDocument() unexpected error = %v", err)
	}

	var got int
	if err := client.Evaluate(context.Background(), "21 * 2", false, &got); err != nil {
		t.Fatalf("client.Evaluate() unexpected error = %v", err)
	}
	if got != 42 {
		t.Errorf("client.Evaluate() = %d, want 42", got)
	}

	gotPNG, err := client.CapturePNGScreenshot(context.Background(), ScreenshotOptions{})
	if err != nil {
		t.Fatalf("client.CapturePNGScreenshot(viewport) unexpected error = %v", err)
	}
	if !bytes.Equal(gotPNG, wantPNG) {
		t.Errorf("client.CapturePNGScreenshot(viewport) = %v, want %v", gotPNG, wantPNG)
	}

	gotFullPNG, err := client.CapturePNGScreenshot(context.Background(), ScreenshotOptions{FullPage: true})
	if err != nil {
		t.Fatalf("client.CapturePNGScreenshot(fullPage) unexpected error = %v", err)
	}
	if !bytes.Equal(gotFullPNG, wantPNG) {
		t.Errorf("client.CapturePNGScreenshot(fullPage) = %v, want %v", gotFullPNG, wantPNG)
	}

	wantRequests := []recordedRequest{
		{Method: "Page.enable"},
		{Method: "Runtime.enable"},
		{
			Method: "Page.navigate",
			Params: map[string]any{"url": "http://127.0.0.1:8080/render/timeline"},
		},
		{
			Method: "Emulation.setDeviceMetricsOverride",
			Params: map[string]any{
				"width":             float64(1600),
				"height":            float64(480),
				"deviceScaleFactor": float64(2),
				"mobile":            false,
			},
		},
		{
			Method: "Emulation.setEmulatedMedia",
			Params: map[string]any{
				"media": "screen",
				"features": []any{
					map[string]any{
						"name":  "prefers-reduced-motion",
						"value": "reduce",
					},
				},
			},
		},
		{
			Method: "Emulation.setEmulatedMedia",
			Params: map[string]any{
				"media": "print",
			},
		},
		{
			Method: "Emulation.setTimezoneOverride",
			Params: map[string]any{"timezoneId": "UTC"},
		},
		{
			Method: "Emulation.setLocaleOverride",
			Params: map[string]any{"locale": "en-US"},
		},
		{Method: "Page.enable"},
		{
			Method: "Page.addScriptToEvaluateOnNewDocument",
			Params: map[string]any{"source": "window.__vrtInit = true;"},
		},
		{
			Method: "Runtime.evaluate",
			Params: map[string]any{
				"expression":    "21 * 2",
				"returnByValue": true,
				"awaitPromise":  false,
			},
		},
		{
			Method: "Page.captureScreenshot",
			Params: map[string]any{"format": "png"},
		},
		{
			Method: "Page.captureScreenshot",
			Params: map[string]any{
				"format":                "png",
				"captureBeyondViewport": true,
			},
		},
	}
	mu.Lock()
	diff := cmp.Diff(wantRequests, gotRequests)
	mu.Unlock()
	if diff != "" {
		t.Errorf("dispatched requests mismatch (-want +got):\n%s", diff)
	}

	// Subsequent call after server closed the WebSocket must return a TransportError.
	err = client.Evaluate(context.Background(), "1 + 1", false, &got)
	var transportErr *TransportError
	if !errors.As(err, &transportErr) {
		t.Errorf("client.Evaluate() after disconnect error = %v, want *TransportError", err)
	}
}
