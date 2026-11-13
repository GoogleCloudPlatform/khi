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

package simcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/genai"
)

type wireGenaiEmbedRequest struct {
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
	EmbedContentConfig struct {
		OutputDimensionality int32 `json:"outputDimensionality"`
	} `json:"embedContentConfig"`
}

func TestVertexClientGenaiSDKCachingAndQuotaHeader(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if r.Header.Get("X-Goog-User-Project") != "test-project" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"message":"missing X-Goog-User-Project"}}`))
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/publishers/google/models/gemini-embedding-2:embedContent") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"message":"transient 503"}}`))
			return
		}
		var req wireGenaiEmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Content.Parts) != 1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.EmbedContentConfig.OutputDimensionality != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		partText := req.Content.Parts[0].Text
		if !utf8.ValidString(partText) || utf8.RuneCountInString(partText) > maxInstanceChars {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if partText == "alpha" {
			_, _ = w.Write([]byte(`{"embedding":{"values":[3,4]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"embedding":{"values":[0,5]}}`))
	}))
	defer srv.Close()

	attempts := int32(3)
	delay := 0.001
	cachePath := filepath.Join(t.TempDir(), "cache.db")
	client, err := NewVertexClient(context.Background(), VertexConfig{
		ProjectID:   "test-project",
		Dimensions:  2,
		Concurrency: 1,
		CachePath:   cachePath,
		HTTPClient:  srv.Client(),
		HTTPOptions: genai.HTTPOptions{
			BaseURL: srv.URL,
			RetryOptions: &genai.HTTPRetryOptions{
				Attempts:     &attempts,
				InitialDelay: &delay,
				MaxDelay:     &delay,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewVertexClient() failed: %v", err)
	}

	emptyOut, err := client.EmbedBatch(context.Background(), nil)
	if err != nil || emptyOut != nil {
		t.Fatalf("EmbedBatch(nil) = (%v, %v), want (nil, nil)", emptyOut, err)
	}

	longMultibyte := strings.Repeat("あ", maxInstanceChars+50)
	got, err := client.EmbedBatch(context.Background(), []string{"alpha", longMultibyte, "alpha"})
	if err != nil {
		t.Fatalf("EmbedBatch() failed: %v", err)
	}
	want := [][]float32{
		{0.6, 0.8},
		{0, 1},
		{0.6, 0.8},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("EmbedBatch() mismatch (-want +got):\n%s", diff)
	}
	if calls.Load() != 3 {
		t.Errorf("HTTP calls = %d, want 3 (1x 503 retry + 2 unique texts)", calls.Load())
	}

	if _, err := client.EmbedBatch(context.Background(), []string{"alpha", longMultibyte}); err != nil {
		t.Fatalf("cached EmbedBatch() failed: %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("HTTP calls increased on cache hit: got %d, want 3", calls.Load())
	}

	if err := client.Close(); err != nil {
		t.Fatalf("client.Close() failed: %v", err)
	}
	if _, err := client.EmbedBatch(context.Background(), []string{"alpha"}); err == nil {
		t.Errorf("EmbedBatch() after Close() succeeded, want error")
	}
}

func TestVertexClientDefaultsHeadersAndErrorCases(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := NewVertexClient(context.Background(), VertexConfig{}); err == nil {
		t.Errorf("NewVertexClient(empty ProjectID) succeeded, want error")
	}

	var nilClient *VertexClient
	if err := nilClient.Close(); err != nil {
		t.Errorf("nil VertexClient.Close() = %v, want nil", err)
	}

	gotDef := applyVertexDefaults(VertexConfig{ProjectID: "p"})
	wantDef := VertexConfig{
		ProjectID:   "p",
		Location:    defaultVertexLocation,
		Model:       defaultEmbeddingModel,
		Dimensions:  defaultDimensions,
		Concurrency: defaultConcurrency,
	}
	if diff := cmp.Diff(wantDef, gotDef); diff != "" {
		t.Errorf("applyVertexDefaults(zero) mismatch (-want +got):\n%s", diff)
	}

	customCfg := VertexConfig{
		ProjectID:   "p",
		Location:    "us-central1",
		Model:       "custom-model",
		Dimensions:  256,
		Concurrency: 4,
	}
	if diff := cmp.Diff(customCfg, applyVertexDefaults(customCfg)); diff != "" {
		t.Errorf("applyVertexDefaults(custom) mismatch (-want +got):\n%s", diff)
	}

	preserved := withQuotaProjectHeader(genai.HTTPOptions{
		Headers: http.Header{"X-Goog-User-Project": []string{"explicit-quota"}},
	}, "fallback-proj")
	if gotHdr := preserved.Headers.Get("X-Goog-User-Project"); gotHdr != "explicit-quota" {
		t.Errorf("withQuotaProjectHeader(explicit) = %q, want explicit-quota", gotHdr)
	}

	testCases := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "returns error on HTTP 400 failure",
			status: http.StatusBadRequest,
			body:   `{"error":{"message":"invalid request"}}`,
		},
		{
			name:   "returns error on empty embedding values",
			status: http.StatusOK,
			body:   `{"embedding":{"values":[]}}`,
		},
	}

	noRetry := int32(1)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			client, err := NewVertexClient(context.Background(), VertexConfig{
				ProjectID:  "p",
				HTTPClient: srv.Client(),
				HTTPOptions: genai.HTTPOptions{
					BaseURL:      srv.URL,
					RetryOptions: &genai.HTTPRetryOptions{Attempts: &noRetry},
				},
			})
			if err != nil {
				t.Fatalf("NewVertexClient() failed: %v", err)
			}
			defer client.Close()

			if _, err := client.EmbedBatch(context.Background(), []string{"probe"}); err == nil {
				t.Errorf("EmbedBatch() succeeded, want error")
			}
		})
	}
}

func TestTryNewEmbedderProjectEnvAndMetadataResolution(t *testing.T) {
	if isCloudBuildEnv(nil) || isCloudBuildEnv(func(string) string { return "" }) {
		t.Errorf("isCloudBuildEnv(empty) = true, want false")
	}
	if !isCloudBuildEnv(func(k string) string {
		if k == "BUILD_ID" {
			return "build-123"
		}
		return ""
	}) {
		t.Errorf("isCloudBuildEnv(BUILD_ID) = false, want true")
	}
	if !isCloudBuildEnv(func(k string) string {
		if k == "BUILDER_OUTPUT" {
			return "/builder/outputs"
		}
		return ""
	}) {
		t.Errorf("isCloudBuildEnv(BUILDER_OUTPUT) = false, want true")
	}

	if got := resolveEnvProjectID(nil); got != "" {
		t.Errorf("resolveEnvProjectID(nil) = %q, want empty", got)
	}
	if got := resolveEnvProjectID(func(string) string { return "" }); got != "" {
		t.Errorf("resolveEnvProjectID(empty env) = %q, want empty", got)
	}
	if got := resolveEnvProjectID(func(k string) string {
		if k == "PROJECT_ID" {
			return " cb-proj "
		}
		return ""
	}); got != "cb-proj" {
		t.Errorf("resolveEnvProjectID(PROJECT_ID) = %q, want cb-proj", got)
	}

	cacheDir := t.TempDir()
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(cacheDir, "nonexistent_adc.json"))

	defEmb, defReason, err := TryNewDefaultEmbedder(context.Background(), EmbedderOptions{
		ProjectID: " khi-ci-project ",
		Getenv: func(k string) string {
			if k == "CODEQUALITY_SIMILARITY_CACHE_DIR" {
				return cacheDir
			}
			return ""
		},
	})
	if err != nil || defEmb != nil || !strings.Contains(defReason, "Vertex AI client unavailable") {
		t.Errorf("TryNewDefaultEmbedder(missing ADC) = (%v, %q, %v)", defEmb, defReason, err)
	}

	noProjEmb, noProjReason, err := tryNewEmbedderWithDeps(context.Background(), EmbedderOptions{}, buildMockGenaiDeps(filepath.Join(cacheDir, "a.db"), "", nil))
	if err != nil || noProjEmb != nil || !strings.Contains(noProjReason, "no GCP project") {
		t.Errorf("missing project = (%v, %q, %v)", noProjEmb, noProjReason, err)
	}

	explicitEmb, explicitReason, err := tryNewEmbedderWithDeps(context.Background(), EmbedderOptions{ProjectID: " explicit-proj "}, buildMockGenaiDeps(filepath.Join(cacheDir, "b.db"), "", &http.Client{}))
	if err != nil || explicitEmb == nil || explicitReason != "" {
		t.Fatalf("explicit project client = (%v, %q, %v)", explicitEmb, explicitReason, err)
	}
	if gotProj := explicitEmb.(*VertexClient).cfg.ProjectID; gotProj != "explicit-proj" {
		t.Errorf("explicit ProjectID = %q, want explicit-proj", gotProj)
	}
	_ = explicitEmb.(*VertexClient).Close()

	okEmb, okReason, err := tryNewEmbedderWithDeps(context.Background(), EmbedderOptions{}, buildMockGenaiDeps(filepath.Join(cacheDir, "c.db"), "proj-a", &http.Client{}))
	if err != nil || okEmb == nil || okReason != "" {
		t.Fatalf("valid client = (%v, %q, %v)", okEmb, okReason, err)
	}
	_ = okEmb.(*VertexClient).Close()
}

func TestQueryMetadataProjectID(t *testing.T) {
	testCases := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{
			name:   "resolves trimmed Cloud Build project ID with Metadata-Flavor Google",
			status: http.StatusOK,
			body:   "  example-ci-project \n",
			want:   "example-ci-project",
		},
		{
			name:   "returns empty on HTTP 404",
			status: http.StatusNotFound,
			body:   "not found",
			want:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Metadata-Flavor") != "Google" || r.URL.Path != "/computeMetadata/v1/project/project-id" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			got := queryMetadataProjectID(context.Background(), srv.Client(), srv.URL)
			if got != tc.want {
				t.Errorf("queryMetadataProjectID() = %q, want %q", got, tc.want)
			}
		})
	}

	if got := queryMetadataProjectID(context.Background(), &http.Client{}, "://bad-url"); got != "" {
		t.Errorf("queryMetadataProjectID(invalid URL) = %q, want empty", got)
	}
	if got := queryMetadataProjectID(context.Background(), &http.Client{}, "http://127.0.0.1:1"); got != "" {
		t.Errorf("queryMetadataProjectID(unreachable) = %q, want empty", got)
	}
}

func buildMockGenaiDeps(dbPath string, proj string, httpClient *http.Client) defaultEmbedderDeps {
	return defaultEmbedderDeps{
		resolveProject: func(pCtx context.Context, _ func(string) string) string {
			if dl, ok := pCtx.Deadline(); !ok || time.Until(dl) < time.Second {
				return ""
			}
			return proj
		},
		resolveCache: func(func(string) string) (string, error) {
			return dbPath, nil
		},
		httpClient:  httpClient,
		httpOptions: genai.HTTPOptions{BaseURL: "https://example.invalid"},
	}
}
