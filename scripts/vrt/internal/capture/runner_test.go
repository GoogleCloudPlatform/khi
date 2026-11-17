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
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/websocket"

	"github.com/GoogleCloudPlatform/khi/pkg/server/cdp"
	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/storybook"
)

func makeTestPNG(t *testing.T, w int, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	img.SetNRGBA(0, 0, color.NRGBA{R: 12, G: 34, B: 56, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}
	return buf.Bytes()
}

func TestRunWithFactory(t *testing.T) {
	pngBytes := makeTestPNG(t, 1280, 720)
	b64PNG := base64.StdEncoding.EncodeToString(pngBytes)

	var mu sync.Mutex
	var visitedIDs []string
	var customViewportSeen bool
	var maskAppliedFor []string

	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Browser":"HeadlessChrome/136.0.0.1","Protocol-Version":"1.3"}`))
	})
	mux.Handle("/devtools/page/1", websocket.Handler(func(ws *websocket.Conn) {
		currentStoryID := ""
		for {
			var raw string
			if err := websocket.Message.Receive(ws, &raw); err != nil {
				return
			}
			var req struct {
				ID     int            `json:"id"`
				Method string         `json:"method"`
				Params map[string]any `json:"params"`
			}
			if err := json.Unmarshal([]byte(raw), &req); err != nil {
				return
			}

			var res any = map[string]any{}
			switch req.Method {
			case "Page.navigate":
				rawURL, _ := req.Params["url"].(string)
				parsed, _ := url.Parse(rawURL)
				currentStoryID = parsed.Query().Get("id")
				mu.Lock()
				visitedIDs = append(visitedIDs, currentStoryID)
				mu.Unlock()
			case "Emulation.setDeviceMetricsOverride":
				if wVal, _ := req.Params["width"].(float64); int(wVal) == 800 {
					mu.Lock()
					customViewportSeen = true
					mu.Unlock()
				}
			case "Runtime.evaluate":
				expr, _ := req.Params["expression"].(string)
				switch {
				case strings.Contains(expr, "hasBridge"):
					res = map[string]any{"result": map[string]any{"value": map[string]any{"hasBridge": true, "legacyReady": true}}}
				case strings.Contains(expr, "waitForReady"):
					state := vrtBridgeState{StoryID: currentStoryID, Status: "ready"}
					switch currentStoryID {
					case "skipped--story":
						state.Parameters.Skip = true
					case "errored--story":
						state.Status = "error"
						state.ErrorMessage = "boom in story"
					case "custom--story":
						state.Parameters.Viewport = &vrtViewport{Width: 800, Height: 600}
						state.Parameters.FullPage = true
						state.Parameters.DiffPixelRatio = 0.005
						state.Parameters.MaskSelectors = []string{".dynamic-badge"}
					}
					res = map[string]any{"result": map[string]any{"value": state}}
				case strings.Contains(expr, "applyKhiVrtMasks"):
					mu.Lock()
					maskAppliedFor = append(maskAppliedFor, currentStoryID)
					mu.Unlock()
				case strings.Contains(expr, "__khiVRTPageErrors"):
					errs := []string{}
					if currentStoryID == "pageerr--story" {
						errs = []string{"uncaught error: bad template"}
					}
					res = map[string]any{"result": map[string]any{"value": errs}}
				}
			case "Page.captureScreenshot":
				res = map[string]any{"data": b64PNG}
			}

			payload, _ := json.Marshal(map[string]any{"id": req.ID, "result": res})
			_ = websocket.Message.Send(ws, string(payload))
		}
	}))

	cdpSrv := httptest.NewServer(mux)
	defer cdpSrv.Close()
	pageWSURL := "ws" + strings.TrimPrefix(cdpSrv.URL, "http") + "/devtools/page/1"

	storybookDir := t.TempDir()
	indexJSON := `{
  "v": 5,
  "entries": {
    "ok--story": {"type":"story","id":"ok--story","title":"Ok","name":"Default","importPath":"./ok.stories.ts"},
    "custom--story": {"type":"story","id":"custom--story","title":"Custom","name":"Default","importPath":"./custom.stories.ts"},
    "skipped--story": {"type":"story","id":"skipped--story","title":"Skipped","name":"Default","importPath":"./skip.stories.ts"},
    "errored--story": {"type":"story","id":"errored--story","title":"Errored","name":"Default","importPath":"./err.stories.ts"},
    "pageerr--story": {"type":"story","id":"pageerr--story","title":"PageErr","name":"Default","importPath":"./perr.stories.ts"}
  }
}`
	if err := os.WriteFile(filepath.Join(storybookDir, "index.json"), []byte(indexJSON), 0o600); err != nil {
		t.Fatalf("failed to write index.json: %v", err)
	}

	osReleasePath := filepath.Join(t.TempDir(), "os-release")
	if err := os.WriteFile(osReleasePath, []byte("ID=debian\nVERSION_CODENAME=trixie\n"), 0o600); err != nil {
		t.Fatalf("failed to write os-release: %v", err)
	}

	outDir := t.TempDir()
	var logBuf bytes.Buffer
	factory := func(_ context.Context, _ cdp.BrowserOptions) (*cdp.Browser, error) {
		return cdp.NewBrowserFromPageWSURL(pageWSURL), nil
	}

	manifest, err := runWithFactory(context.Background(), Config{
		StorybookDir: storybookDir,
		OutputDir:    outDir,
		Workers:      2,
		Retries:      -1,
		StoryTimeout: 5 * time.Second,
		CommitSHA:    "deadbeef1234",
		LogWriter:    &logBuf,
	}, factory, osReleasePath)
	if err != nil {
		t.Fatalf("runWithFactory() unexpected error = %v", err)
	}

	wantLog := "Captured 5/5 stories (2 errors)\n"
	if logBuf.String() != wantLog {
		t.Errorf("progress log = %q, want %q", logBuf.String(), wantLog)
	}

	if manifest.BuilderFingerprint != "headlesschrome-136.0.0.1-trixie" {
		t.Errorf("BuilderFingerprint = %q, want %q", manifest.BuilderFingerprint, "headlesschrome-136.0.0.1-trixie")
	}
	if !customViewportSeen {
		t.Error("expected custom 800x600 viewport override to be sent via CDP")
	}
	if diff := cmp.Diff([]string{"custom--story"}, maskAppliedFor); diff != "" {
		t.Errorf("maskAppliedFor mismatch (-want +got):\n%s", diff)
	}

	type entryOutcome struct {
		HasImage       bool
		Skipped        bool
		DiffPixelRatio float64
		Error          string
	}
	gotOutcomes := map[string]entryOutcome{}
	for id, e := range manifest.Entries {
		gotOutcomes[id] = entryOutcome{
			HasImage:       e.ImageRelPath != "" && e.SHA256 != "" && e.Width == 1280 && e.Height == 720,
			Skipped:        e.Skipped,
			DiffPixelRatio: e.DiffPixelRatio,
			Error:          e.Error,
		}
	}
	wantOutcomes := map[string]entryOutcome{
		"story/ok--story":      {HasImage: true},
		"story/custom--story":  {HasImage: true, DiffPixelRatio: 0.005},
		"story/skipped--story": {Skipped: true},
		"story/errored--story": {Error: "boom in story"},
		"story/pageerr--story": {Error: "uncaught error: bad template"},
	}
	if diff := cmp.Diff(wantOutcomes, gotOutcomes); diff != "" {
		t.Errorf("captured entry outcomes mismatch (-want +got):\n%s", diff)
	}

	if _, err := os.Stat(filepath.Join(outDir, "images", "story", "ok--story.png")); err != nil {
		t.Errorf("expected captured PNG file on disk, got error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, ManifestFileName)); err != nil {
		t.Errorf("expected manifest.json on disk, got error: %v", err)
	}
}

func TestCaptureEdgeCases(t *testing.T) {
	t.Run("resolveConfig applies all defaults", func(t *testing.T) {
		got := resolveConfig(Config{})
		if got.ViewportWidth != DefaultViewportWidth || got.ViewportHeight != DefaultViewportHeight ||
			got.StoryTimeout != DefaultStoryTimeout || got.Workers <= 0 || got.Retries != DefaultRetries || got.LogWriter == nil {
			t.Errorf("resolveConfig(empty) = %+v, expected populated defaults", got)
		}
		if w := defaultWorkerCount(4); w != 2 {
			t.Errorf("defaultWorkerCount(4) = %d, want 2", w)
		}
	})

	t.Run("readOSCodename returns empty for missing or codename-free file", func(t *testing.T) {
		if got := readOSCodename(filepath.Join(t.TempDir(), "missing")); got != "" {
			t.Errorf("readOSCodename(missing) = %q, want empty", got)
		}
		noCodename := filepath.Join(t.TempDir(), "os-release")
		_ = os.WriteFile(noCodename, []byte("ID=linux\n"), 0o600)
		if got := readOSCodename(noCodename); got != "" {
			t.Errorf("readOSCodename(no codename) = %q, want empty", got)
		}
	})

	t.Run("runWithFactory returns error when filter matches zero stories", func(t *testing.T) {
		sbDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(sbDir, "index.json"), []byte(`{"v":5,"entries":{}}`), 0o600)
		_, err := runWithFactory(context.Background(), Config{StorybookDir: sbDir, OutputDir: t.TempDir()}, nil, "")
		if err == nil {
			t.Fatal("expected error for empty story list, got nil")
		}
	})

	t.Run("Fingerprint and Run fail fast when CHROME_BIN is missing", func(t *testing.T) {
		t.Setenv("CHROME_BIN", "/nonexistent/chrome")
		if _, err := Fingerprint(context.Background()); err == nil {
			t.Error("Fingerprint() expected error, got nil")
		}
		sbDir := t.TempDir()
		_ = os.WriteFile(filepath.Join(sbDir, "index.json"), []byte(`{"v":5,"entries":{"a":{"type":"story","id":"a"}}}`), 0o600)
		if _, err := Run(context.Background(), Config{StorybookDir: sbDir, OutputDir: t.TempDir()}); err == nil {
			t.Error("Run() expected error, got nil")
		}
	})

	t.Run("pool stops cleanly on canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		factoryCalls := 0
		stories := []storybook.StoryEntry{{ID: "a--default"}}
		_, err := captureStoriesInPool(ctx, resolveConfig(Config{Workers: 1}), "http://127.0.0.1", stories, func(_ context.Context, _ cdp.BrowserOptions) (*cdp.Browser, error) {
			factoryCalls++
			return nil, context.Canceled
		})
		if err == nil {
			t.Fatal("captureStoriesInPool(canceled) expected context error, got nil")
		}
		if factoryCalls != 0 {
			t.Errorf("factoryCalls = %d, want 0 when context is already canceled", factoryCalls)
		}
	})
}

func TestWorkerSession_RetriesLegacyFallbackAndTimeout(t *testing.T) {
	pngBytes := makeTestPNG(t, 1280, 720)
	b64PNG := base64.StdEncoding.EncodeToString(pngBytes)

	var mu sync.Mutex
	attempts := 0
	pollCount := 0

	mux := http.NewServeMux()
	mux.Handle("/devtools/page/1", websocket.Handler(func(ws *websocket.Conn) {
		currentStoryID := ""
		for {
			var raw string
			if err := websocket.Message.Receive(ws, &raw); err != nil {
				return
			}
			var req struct {
				ID     int            `json:"id"`
				Method string         `json:"method"`
				Params map[string]any `json:"params"`
			}
			if err := json.Unmarshal([]byte(raw), &req); err != nil {
				return
			}
			var res any = map[string]any{}
			switch req.Method {
			case "Page.navigate":
				rawURL, _ := req.Params["url"].(string)
				parsed, _ := url.Parse(rawURL)
				currentStoryID = parsed.Query().Get("id")
				mu.Lock()
				if currentStoryID == "flaky--story" {
					attempts++
				}
				mu.Unlock()
			case "Runtime.evaluate":
				expr, _ := req.Params["expression"].(string)
				switch {
				case strings.Contains(expr, "hasBridge"):
					mu.Lock()
					pollCount++
					c := pollCount
					mu.Unlock()
					switch currentStoryID {
					case "legacy--story":
						if c == 1 {
							res = map[string]any{"result": map[string]any{"value": map[string]any{"hasBridge": false, "legacyReady": false}}}
						} else {
							res = map[string]any{"result": map[string]any{"value": map[string]any{"hasBridge": false, "legacyReady": true}}}
						}
					case "timeout--story":
						res = map[string]any{"result": map[string]any{"value": map[string]any{"hasBridge": false, "legacyReady": false}}}
					default:
						res = map[string]any{"result": map[string]any{"value": map[string]any{"hasBridge": true, "legacyReady": true}}}
					}
				case strings.Contains(expr, "sb-show-errordisplay"):
					res = map[string]any{"result": map[string]any{"value": vrtBridgeState{Status: "ready"}}}
				case strings.Contains(expr, "waitForReady"):
					mu.Lock()
					a := attempts
					mu.Unlock()
					if currentStoryID == "flaky--story" && a == 1 {
						res = map[string]any{"result": map[string]any{"value": vrtBridgeState{Status: "error", ErrorMessage: "transient"}}}
					} else {
						res = map[string]any{"result": map[string]any{"value": vrtBridgeState{
							Status: "ready",
							Parameters: vrtParameters{
								Viewport: &vrtViewport{Width: DefaultViewportWidth, Height: DefaultViewportHeight},
							},
						}}}
					}
				case strings.Contains(expr, "__khiVRTPageErrors"):
					res = map[string]any{"result": map[string]any{"value": []string{}}}
				}
			case "Page.captureScreenshot":
				res = map[string]any{"data": b64PNG}
			}
			payload, _ := json.Marshal(map[string]any{"id": req.ID, "result": res})
			_ = websocket.Message.Send(ws, string(payload))
		}
	}))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/devtools/page/1"
	outDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(outDir, ImagesDirName, "story"), 0o755)
	cfg := resolveConfig(Config{
		OutputDir:    outDir,
		Retries:      1,
		StoryTimeout: 200 * time.Millisecond,
	})
	factoryCalls := 0
	session := &workerSession{
		factory: func(_ context.Context, _ cdp.BrowserOptions) (*cdp.Browser, error) {
			factoryCalls++
			return cdp.NewBrowserFromPageWSURL(wsURL), nil
		},
		viewportWidth:  cfg.ViewportWidth,
		viewportHeight: cfg.ViewportHeight,
	}

	// 1. Flaky story fails attempt 1, succeeds on retry attempt 2 (and exercises same-size viewport no-op).
	flaky := session.captureWithRetries(context.Background(), cfg, "http://127.0.0.1", storybook.StoryEntry{ID: "flaky--story"})
	if flaky.Error != "" || attempts != 2 || factoryCalls != 2 {
		t.Errorf("flaky capture = (err=%q, attempts=%d, factoryCalls=%d), want (\"\", 2, 2)", flaky.Error, attempts, factoryCalls)
	}
	if session.browser == nil || session.client == nil {
		t.Fatal("expected active session.browser and session.client after successful capture")
	}

	// 2. Legacy story without window.__khiVRT polls once then succeeds via legacy fallback, reusing existing browser session.
	legacy := session.captureWithRetries(context.Background(), cfg, "http://127.0.0.1", storybook.StoryEntry{ID: "legacy--story"})
	if legacy.Error != "" || factoryCalls != 2 {
		t.Errorf("legacy capture = (err=%q, factoryCalls=%d), want (\"\", 2)", legacy.Error, factoryCalls)
	}

	// 3. Timeout story times out cleanly in waitForVRTBridge and closes the worker session.
	timeoutCfg := cfg
	timeoutCfg.Retries = 0
	timeoutCfg.StoryTimeout = 80 * time.Millisecond
	timedOut := session.captureWithRetries(context.Background(), timeoutCfg, "http://127.0.0.1", storybook.StoryEntry{ID: "timeout--story"})
	if !strings.Contains(timedOut.Error, "timed out") {
		t.Errorf("timeout capture error = %q, want timeout message", timedOut.Error)
	}
	session.close()
	if session.browser != nil || session.client != nil {
		t.Errorf("session after close() = (browser=%v, client=%v), want (nil, nil)", session.browser, session.client)
	}
}

func startViewportRecordingCDP(t *testing.T) (*cdp.Client, func() [][2]int) {
	t.Helper()
	var mu sync.Mutex
	var applied [][2]int
	mux := http.NewServeMux()
	mux.Handle("/devtools/page/1", websocket.Handler(func(ws *websocket.Conn) {
		for {
			var raw string
			if err := websocket.Message.Receive(ws, &raw); err != nil {
				return
			}
			var req struct {
				ID     int            `json:"id"`
				Method string         `json:"method"`
				Params map[string]any `json:"params"`
			}
			if err := json.Unmarshal([]byte(raw), &req); err != nil {
				return
			}
			if req.Method == "Emulation.setDeviceMetricsOverride" {
				w, _ := req.Params["width"].(float64)
				h, _ := req.Params["height"].(float64)
				mu.Lock()
				applied = append(applied, [2]int{int(w), int(h)})
				mu.Unlock()
			}
			payload, _ := json.Marshal(map[string]any{"id": req.ID, "result": map[string]any{}})
			_ = websocket.Message.Send(ws, string(payload))
		}
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client, err := cdp.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/devtools/page/1")
	if err != nil {
		t.Fatalf("cdp.Dial() unexpected error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	return client, func() [][2]int {
		mu.Lock()
		defer mu.Unlock()
		return append([][2]int(nil), applied...)
	}
}

func TestSessionEntryTimingAndCustomViewport(t *testing.T) {
	t.Run("entry builders record positive elapsed duration", func(t *testing.T) {
		past := time.Now().Add(-50 * time.Millisecond)
		outDir := t.TempDir()
		_ = os.MkdirAll(filepath.Join(outDir, ImagesDirName, "story"), 0o755)
		base := SnapshotEntry{ID: StorySnapshotID("timing--story")}

		skipped := skippedEntry(base, past)
		if !skipped.Skipped || skipped.DurationMs <= 0 {
			t.Errorf("skippedEntry() = %+v, want Skipped=true and DurationMs>0", skipped)
		}
		failed := failedEntry(base, "boom", past)
		if failed.Error != "boom" || failed.DurationMs <= 0 {
			t.Errorf("failedEntry() = %+v, want Error=boom and DurationMs>0", failed)
		}
		finalized := finalizeEntry(outDir, base, makeTestPNG(t, 16, 16), 0.01, past)
		if finalized.Error != "" || finalized.DurationMs <= 0 {
			t.Errorf("finalizeEntry() = %+v, want empty Error and DurationMs>0", finalized)
		}
	})

	t.Run("applyCustomViewport respects zero dimensions and partial overrides", func(t *testing.T) {
		client, recorded := startViewportRecordingCDP(t)
		cases := []*vrtViewport{
			nil,
			{Width: 0, Height: 600},
			{Width: 800, Height: 0},
			{Width: 1280, Height: 720},
			{Width: 1280, Height: 600},
			{Width: 900, Height: 720},
		}
		for _, vp := range cases {
			if err := applyCustomViewport(context.Background(), client, vp, 1280, 720); err != nil {
				t.Fatalf("applyCustomViewport(%+v) unexpected error = %v", vp, err)
			}
		}
		want := [][2]int{{1280, 600}, {900, 720}}
		if diff := cmp.Diff(want, recorded()); diff != "" {
			t.Errorf("applied viewports mismatch (-want +got):\n%s", diff)
		}
	})
}
