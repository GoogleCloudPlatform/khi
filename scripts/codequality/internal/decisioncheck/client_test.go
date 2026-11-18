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

package decisioncheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/auth"
	"github.com/google/go-cmp/cmp"
)

type staticTokenProvider struct {
	token string
}

type capturedHTTPRequest struct {
	APIKey       string
	AuthHeader   string
	QuotaProject string
	Model        string
}

func (s staticTokenProvider) Token(context.Context) (*auth.Token, error) {
	return &auth.Token{Value: s.token}, nil
}

func TestHTTPClientDecide(t *testing.T) {
	pTrue := 0.89
	stubResp := DecisionResponse{
		Answers: map[string]WireAnswer{
			"q1": {Noul: &pTrue, Confidence: 0.91},
		},
	}

	testCases := []struct {
		name         string
		cfg          ClientConfig
		req          DecisionRequest
		wantCaptured capturedHTTPRequest
		wantCallErr  string
		wantResponse DecisionResponse
	}{
		// Sends x-goog-api-key header, defaults model to gemini-decision-1, and decodes calibrated answers.
		{
			name: "API key authentication and default model",
			cfg:  ClientConfig{APIKey: "test-api-key"},
			req:  DecisionRequest{Input: "ctx", Questions: map[string]WireQuestion{"q1": {Type: QuestionTypeNoul, Instructions: "ok?"}}},
			wantCaptured: capturedHTTPRequest{
				APIKey: "test-api-key",
				Model:  DefaultDecisionsModel,
			},
			wantResponse: stubResp,
		},
		// Sends Authorization Bearer token and x-goog-user-project headers when configured with OAuth TokenProvider.
		{
			name: "OAuth token provider with quota project header",
			cfg:  ClientConfig{TokenProvider: staticTokenProvider{token: "tok-xyz"}, QuotaProject: "proj-123"},
			req:  DecisionRequest{Input: "ctx"},
			wantCaptured: capturedHTTPRequest{
				AuthHeader:   "Bearer tok-xyz",
				QuotaProject: "proj-123",
				Model:        DefaultDecisionsModel,
			},
			wantResponse: stubResp,
		},
		// Returns a descriptive error containing the HTTP status code when the server responds with 500.
		{
			name: "non-200 HTTP status propagates descriptive error",
			cfg:  ClientConfig{APIKey: "test-api-key"},
			req:  DecisionRequest{Input: "trigger-500"},
			wantCaptured: capturedHTTPRequest{
				APIKey: "test-api-key",
				Model:  DefaultDecisionsModel,
			},
			wantCallErr: "HTTP 500",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv, captured := newStubDecisionServer(t, stubResp)
			defer srv.Close()
			cfg := tc.cfg
			cfg.Endpoint = srv.URL
			cfg.HTTPClient = srv.Client()
			client, err := NewHTTPClient(cfg)
			if err != nil {
				t.Fatalf("NewHTTPClient() unexpected error: %v", err)
			}

			gotResp, callErr := client.Decide(context.Background(), tc.req)

			assertDecideOutcome(t, callErr, tc.wantCallErr, tc.wantResponse, gotResp)
			if diff := cmp.Diff(tc.wantCaptured, *captured); diff != "" {
				t.Errorf("captured HTTP request mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestNewHTTPClientRejectsMissingCredentials verifies that constructing an HTTPClient with neither APIKey nor TokenProvider fails.
func TestNewHTTPClientRejectsMissingCredentials(t *testing.T) {
	_, err := NewHTTPClient(ClientConfig{})

	if err == nil {
		t.Errorf("NewHTTPClient(empty) expected error, got nil")
	}
}

func TestNewHTTPClientConfig(t *testing.T) {
	customHTTP := &http.Client{Timeout: 5 * time.Second}
	type clientConfigSnapshot struct {
		Endpoint       string
		APIKey         string
		QuotaProject   string
		Timeout        time.Duration
		UsesCustomHTTP bool
	}
	testCases := []struct {
		name string
		cfg  ClientConfig
		want clientConfigSnapshot
	}{
		// Applies DefaultDecisionsEndpoint and defaultHTTPTimeout when Endpoint and HTTPClient are omitted.
		{
			name: "default endpoint and HTTP client",
			cfg:  ClientConfig{APIKey: "  test-key  "},
			want: clientConfigSnapshot{
				Endpoint:       DefaultDecisionsEndpoint,
				APIKey:         "test-key",
				Timeout:        defaultHTTPTimeout,
				UsesCustomHTTP: false,
			},
		},
		// Preserves custom HTTPClient, endpoint, and trimmed quota project when explicitly configured.
		{
			name: "explicit custom endpoint and HTTP client",
			cfg: ClientConfig{
				Endpoint:      "  https://example.test/v1beta/decisions  ",
				TokenProvider: staticTokenProvider{token: "tok"},
				QuotaProject:  "  proj-custom  ",
				HTTPClient:    customHTTP,
			},
			want: clientConfigSnapshot{
				Endpoint:       "https://example.test/v1beta/decisions",
				QuotaProject:   "proj-custom",
				Timeout:        5 * time.Second,
				UsesCustomHTTP: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewHTTPClient(tc.cfg)

			if err != nil {
				t.Fatalf("NewHTTPClient() unexpected error: %v", err)
			}
			if client.httpClient == nil {
				t.Fatalf("client.httpClient = nil, want non-nil")
			}
			got := clientConfigSnapshot{
				Endpoint:       client.endpoint,
				APIKey:         client.apiKey,
				QuotaProject:   client.quotaProject,
				Timeout:        client.httpClient.Timeout,
				UsesCustomHTTP: client.httpClient == customHTTP,
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("client configuration mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTryNewDefaultDecider(t *testing.T) {
	tmpMk := filepath.Join(t.TempDir(), "codequality.mk")
	mkContent := "# comment\nexport CODEQUALITY_DECISIONS_API_KEY := \"mk-key-123\"\n"
	if err := os.WriteFile(tmpMk, []byte(mkContent), 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	testCases := []struct {
		name        string
		getenv      func(string) string
		wantDecider bool
		wantSkipped bool
	}{
		// Reads CODEQUALITY_DECISIONS_API_KEY from local codequality.mk when running outside CI.
		{
			name:        "local run resolves key from codequality.mk",
			getenv:      func(string) string { return "" },
			wantDecider: true,
		},
		// Ignores local codequality.mk in CI and returns a graceful skip reason when env keys are unset.
		{
			name: "CI run ignores local mk file and skips cleanly without env key",
			getenv: func(key string) string {
				if key == "GITHUB_ACTIONS" {
					return "true"
				}
				return ""
			},
			wantSkipped: true,
		},
		// Resolves GEMINI_API_KEY from environment variables in both local and CI runs.
		{
			name: "environment variable key takes precedence",
			getenv: func(key string) string {
				if key == "GEMINI_API_KEY" {
					return "env-key-456"
				}
				return ""
			},
			wantDecider: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dec, skipReason, err := TryNewDefaultDecider(context.Background(), DeciderOptions{
				Getenv:       tc.getenv,
				ConfigMkPath: tmpMk,
			})

			if err != nil {
				t.Fatalf("TryNewDefaultDecider() unexpected error: %v", err)
			}
			if (dec != nil) != tc.wantDecider {
				t.Errorf("dec != nil is %v, want %v", dec != nil, tc.wantDecider)
			}
			if (skipReason != "") != tc.wantSkipped {
				t.Errorf("skipReason = %q, wantSkipped = %v", skipReason, tc.wantSkipped)
			}
		})
	}
}

func TestTryNewDefaultDeciderProjectADC(t *testing.T) {
	missingMk := filepath.Join(t.TempDir(), "missing.mk")
	stubADCClient := &HTTPClient{endpoint: DefaultDecisionsEndpoint, quotaProject: "proj-adc"}
	testCases := []struct {
		name         string
		projectID    string
		adcOK        bool
		wantDecider  Decider
		wantADCCalls int
		wantADCProj  string
		wantReason   string
	}{
		// Uses the scoped ADC client when a project ID is configured and ADC probing succeeds.
		{
			name:         "scoped ADC succeeds when project configured",
			projectID:    "proj-adc",
			adcOK:        true,
			wantDecider:  stubADCClient,
			wantADCCalls: 1,
			wantADCProj:  "proj-adc",
		},
		// Returns the skip reason when a project is configured but scoped ADC probing fails.
		{
			name:         "scoped ADC failure returns skip reason",
			projectID:    "proj-missing-adc",
			adcOK:        false,
			wantDecider:  nil,
			wantADCCalls: 1,
			wantADCProj:  "proj-missing-adc",
			wantReason:   "no CODEQUALITY_DECISIONS_API_KEY or GEMINI_API_KEY configured",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var adcCalls int
			var gotADCProj string
			stubTryADC := func(_ context.Context, proj string) (*HTTPClient, bool) {
				adcCalls++
				gotADCProj = proj
				if !tc.adcOK {
					return nil, false
				}
				return stubADCClient, true
			}

			dec, skipReason, err := tryNewDefaultDecider(context.Background(), DeciderOptions{
				ProjectID:    tc.projectID,
				Getenv:       func(string) string { return "" },
				ConfigMkPath: missingMk,
			}, stubTryADC)

			if err != nil {
				t.Fatalf("tryNewDefaultDecider() unexpected error: %v", err)
			}
			if adcCalls != tc.wantADCCalls || gotADCProj != tc.wantADCProj {
				t.Errorf("tryADC calls = (%d, %q), want (%d, %q)", adcCalls, gotADCProj, tc.wantADCCalls, tc.wantADCProj)
			}
			if dec != tc.wantDecider {
				t.Errorf("decider = %v, want %v", dec, tc.wantDecider)
			}
			if skipReason != tc.wantReason {
				t.Errorf("skipReason = %q, want %q", skipReason, tc.wantReason)
			}
		})
	}
}

// TestResolveAPIKeyReturnsEmptyWhenUnconfigured verifies that resolveAPIKey returns empty string when neither env vars nor codequality.mk define a key.
func TestResolveAPIKeyReturnsEmptyWhenUnconfigured(t *testing.T) {
	emptyMk := filepath.Join(t.TempDir(), "empty.mk")
	if err := os.WriteFile(emptyMk, []byte("# no keys configured\nOTHER_VAR=value\n"), 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	got := resolveAPIKey(DeciderOptions{ConfigMkPath: emptyMk})

	if got != "" {
		t.Errorf("resolveAPIKey() = %q, want empty string", got)
	}
}

func TestResolveDecisionsProject(t *testing.T) {
	testCases := []struct {
		name string
		opts DeciderOptions
		want string
	}{
		// Prefers trimmed explicit ProjectID over CODEQUALITY_DECISIONS_PROJECT from Getenv.
		{
			name: "explicit ProjectID takes precedence",
			opts: DeciderOptions{
				ProjectID: "  explicit-proj  ",
				Getenv: func(key string) string {
					if key == "CODEQUALITY_DECISIONS_PROJECT" {
						return "env-proj"
					}
					return ""
				},
			},
			want: "explicit-proj",
		},
		// Returns empty string safely when ProjectID is unset and Getenv is nil.
		{
			name: "nil Getenv returns empty",
			opts: DeciderOptions{},
			want: "",
		},
		// Falls back to trimmed CODEQUALITY_DECISIONS_PROJECT when ProjectID is unset.
		{
			name: "env CODEQUALITY_DECISIONS_PROJECT fallback",
			opts: DeciderOptions{
				Getenv: func(key string) string {
					if key == "CODEQUALITY_DECISIONS_PROJECT" {
						return "  env-proj  "
					}
					return ""
				},
			},
			want: "env-proj",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveDecisionsProject(tc.opts)

			if got != tc.want {
				t.Errorf("resolveDecisionsProject() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveConfigMkPath(t *testing.T) {
	fakeHome := t.TempDir()
	testCases := []struct {
		name     string
		override string
		homeDir  string
		want     string
	}{
		// Returns the explicit override path unchanged when provided.
		{
			name:     "explicit override path",
			override: "/tmp/custom-codequality.mk",
			homeDir:  fakeHome,
			want:     "/tmp/custom-codequality.mk",
		},
		// Resolves ~/.config/khi/codequality.mk under HOME when override is empty.
		{
			name:     "default path under HOME",
			override: "",
			homeDir:  fakeHome,
			want:     filepath.Join(fakeHome, ".config", "khi", "codequality.mk"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", tc.homeDir)

			got := resolveConfigMkPath(tc.override)

			if got != tc.want {
				t.Errorf("resolveConfigMkPath(%q) = %q, want %q", tc.override, got, tc.want)
			}
		})
	}
}

func TestParseLocalConfigMk(t *testing.T) {
	mkPath := filepath.Join(t.TempDir(), "codequality.mk")
	mkContent := `# GEMINI_API_KEY=ignored-comment

include extra.mk
export CODEQUALITY_DECISIONS_API_KEY := "mk-key-123"
CODEQUALITY_DECISIONS_PROJECT ?= 'proj-1'
GOOGLE_API_KEY = plain-key
`
	if err := os.WriteFile(mkPath, []byte(mkContent), 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	testCases := []struct {
		name string
		path string
		want map[string]string
	}{
		// Parses :=, ?=, and = assignments with quote trimming while ignoring blanks, # comments containing =, and non-assignment lines.
		{
			name: "parses assignments and skips comments and non-assignments",
			path: mkPath,
			want: map[string]string{
				"CODEQUALITY_DECISIONS_API_KEY": "mk-key-123",
				"CODEQUALITY_DECISIONS_PROJECT": "proj-1",
				"GOOGLE_API_KEY":                "plain-key",
			},
		},
		// Returns an empty map when the config file does not exist.
		{
			name: "missing file returns empty map",
			path: filepath.Join(t.TempDir(), "does-not-exist.mk"),
			want: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseLocalConfigMk(tc.path)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseLocalConfigMk(%q) mismatch (-want +got):\n%s", tc.path, diff)
			}
		})
	}
}

func newStubDecisionServer(t *testing.T, stubResp DecisionResponse) (*httptest.Server, *capturedHTTPRequest) {
	t.Helper()
	captured := &capturedHTTPRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.APIKey = r.Header.Get("x-goog-api-key")
		captured.AuthHeader = r.Header.Get("Authorization")
		captured.QuotaProject = r.Header.Get("x-goog-user-project")
		var req DecisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		captured.Model = req.Model
		if req.Input == "trigger-500" {
			http.Error(w, "upstream failure", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(stubResp)
	}))
	return srv, captured
}

func assertDecideOutcome(t *testing.T, callErr error, wantCallErr string, wantResp DecisionResponse, gotResp DecisionResponse) {
	t.Helper()
	if wantCallErr != "" {
		if callErr == nil || !strings.Contains(callErr.Error(), wantCallErr) {
			t.Errorf("Decide() error = %v, want substring %q", callErr, wantCallErr)
		}
		return
	}
	if callErr != nil {
		t.Fatalf("Decide() unexpected error: %v", callErr)
	}
	if diff := cmp.Diff(wantResp, gotResp); diff != "" {
		t.Errorf("Decide() response mismatch (-want +got):\n%s", diff)
	}
}
