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
	_, err := StartBrowser(context.Background())
	if err == nil {
		t.Fatal("StartBrowser() expected error when CHROME_BIN does not exist, got nil")
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
			case "Page.enable", "Runtime.enable", "Emulation.setDeviceMetricsOverride":
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
				res = map[string]any{
					"data": base64.StdEncoding.EncodeToString(wantPNG),
				}
			}

			respBytes, _ := json.Marshal(map[string]any{
				"id":     req.ID,
				"result": res,
			})
			_ = websocket.Message.Send(ws, string(respBytes))
			if req.Method == "Page.captureScreenshot" {
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

	if err := client.SetViewportSize(context.Background(), 1600, 480); err != nil {
		t.Fatalf("client.SetViewportSize() unexpected error = %v", err)
	}

	var got int
	if err := client.Evaluate(context.Background(), "21 * 2", false, &got); err != nil {
		t.Fatalf("client.Evaluate() unexpected error = %v", err)
	}
	if got != 42 {
		t.Errorf("client.Evaluate() = %d, want 42", got)
	}

	gotPNG, err := client.CapturePNGScreenshot(context.Background())
	if err != nil {
		t.Fatalf("client.CapturePNGScreenshot() unexpected error = %v", err)
	}
	if !bytes.Equal(gotPNG, wantPNG) {
		t.Errorf("client.CapturePNGScreenshot() = %v, want %v", gotPNG, wantPNG)
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
				"deviceScaleFactor": float64(1),
				"mobile":            false,
			},
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
