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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func TestClient_EvaluateAndTransportError(t *testing.T) {
	wsServer := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		var raw string
		if err := websocket.Message.Receive(ws, &raw); err != nil {
			return
		}
		var req request
		if err := json.Unmarshal([]byte(raw), &req); err != nil {
			return
		}
		respBytes, _ := json.Marshal(map[string]any{
			"id": req.ID,
			"result": map[string]any{
				"result": map[string]any{
					"value": 42,
				},
			},
		})
		_ = websocket.Message.Send(ws, string(respBytes))
		_ = ws.Close()
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

	var got int
	if err := client.Evaluate(context.Background(), "21 * 2", false, &got); err != nil {
		t.Fatalf("client.Evaluate() unexpected error = %v", err)
	}
	if got != 42 {
		t.Errorf("client.Evaluate() = %d, want 42", got)
	}

	// Second call after server closed the WebSocket must return a TransportError.
	err = client.Evaluate(context.Background(), "1 + 1", false, &got)
	var transportErr *TransportError
	if !errors.As(err, &transportErr) {
		t.Errorf("client.Evaluate() after disconnect error = %v, want *TransportError", err)
	}
}
