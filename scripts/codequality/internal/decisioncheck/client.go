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
	"bufio"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/credentials"
)

const (
	// DefaultDecisionsEndpoint is the Gemini API v1beta endpoint for structured Decisions evaluation.
	DefaultDecisionsEndpoint = "https://generativelanguage.googleapis.com/v1beta/decisions"
	// DefaultDecisionsModel is the default Gemini Decision model identifier.
	DefaultDecisionsModel = "gemini-decision-1"

	defaultHTTPTimeout    = 60 * time.Second
	maxErrorBodyBytes     = 2048
	generativeLangScope   = "https://www.googleapis.com/auth/generative-language.retriever"
	cloudPlatformScopeURL = "https://www.googleapis.com/auth/cloud-platform"
)

var _ Decider = (*HTTPClient)(nil)

// Decider evaluates structured closed-set questions against a shared input context.
type Decider interface {
	// Decide submits req to the Gemini Decisions backend and returns calibrated answers.
	Decide(ctx context.Context, req DecisionRequest) (DecisionResponse, error)
}

// ClientConfig configures HTTPClient requests to the Gemini Decisions API.
type ClientConfig struct {
	// Endpoint overrides DefaultDecisionsEndpoint when non-empty.
	Endpoint string
	// APIKey authenticates via the x-goog-api-key header when non-empty.
	APIKey string
	// TokenProvider authenticates via Authorization: Bearer when APIKey is empty.
	TokenProvider auth.TokenProvider
	// QuotaProject sets the x-goog-user-project header when using TokenProvider.
	QuotaProject string
	// HTTPClient optionally overrides the underlying *http.Client.
	HTTPClient *http.Client
}

// DeciderOptions configures credential discovery for TryNewDefaultDecider.
type DeciderOptions struct {
	// ProjectID specifies the optional GCP billing project for scoped OAuth credentials.
	ProjectID string
	// Getenv reads environment variables for API key and project lookup.
	Getenv func(string) string
	// ConfigMkPath optionally overrides ~/.config/khi/codequality.mk for local developer API key lookup.
	ConfigMkPath string
}

// HTTPClient implements Decider over REST JSON against POST /v1beta/decisions.
type HTTPClient struct {
	endpoint      string
	apiKey        string
	tokenProvider auth.TokenProvider
	quotaProject  string
	httpClient    *http.Client
}

// NewHTTPClient validates cfg and constructs an HTTPClient for the Gemini Decisions API.
func NewHTTPClient(cfg ClientConfig) (*HTTPClient, error) {
	if strings.TrimSpace(cfg.APIKey) == "" && cfg.TokenProvider == nil {
		return nil, fmt.Errorf("either APIKey or TokenProvider is required")
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &HTTPClient{
		endpoint:      cmp.Or(strings.TrimSpace(cfg.Endpoint), DefaultDecisionsEndpoint),
		apiKey:        strings.TrimSpace(cfg.APIKey),
		tokenProvider: cfg.TokenProvider,
		quotaProject:  strings.TrimSpace(cfg.QuotaProject),
		httpClient:    hc,
	}, nil
}

// Decide sends req to the Gemini Decisions endpoint and decodes the calibrated DecisionResponse.
func (c *HTTPClient) Decide(ctx context.Context, req DecisionRequest) (DecisionResponse, error) {
	req.Model = cmp.Or(req.Model, DefaultDecisionsModel)
	payload, err := json.Marshal(req)
	if err != nil {
		return DecisionResponse{}, fmt.Errorf("failed marshaling decision request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return DecisionResponse{}, fmt.Errorf("failed creating decision HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if err := c.applyAuthHeaders(ctx, httpReq); err != nil {
		return DecisionResponse{}, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return DecisionResponse{}, fmt.Errorf("decisions API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		return DecisionResponse{}, fmt.Errorf("decisions API HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}
	var out DecisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return DecisionResponse{}, fmt.Errorf("failed decoding decisions API response: %w", err)
	}
	return out, nil
}

// TryNewDefaultDecider resolves an API key or scoped GCP OAuth credential, returning (nil, reason, nil) when unconfigured.
func TryNewDefaultDecider(ctx context.Context, opts DeciderOptions) (Decider, string, error) {
	return tryNewDefaultDecider(ctx, opts, tryNewScopedADCClient)
}

func (c *HTTPClient) applyAuthHeaders(ctx context.Context, req *http.Request) error {
	if c.apiKey != "" {
		req.Header.Set("x-goog-api-key", c.apiKey)
		return nil
	}
	tok, err := c.tokenProvider.Token(ctx)
	if err != nil {
		return fmt.Errorf("failed acquiring OAuth token for decisions API: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok.Value)
	if c.quotaProject != "" {
		req.Header.Set("x-goog-user-project", c.quotaProject)
	}
	return nil
}

func tryNewDefaultDecider(ctx context.Context, opts DeciderOptions, tryADC func(context.Context, string) (*HTTPClient, bool)) (Decider, string, error) {
	if key := resolveAPIKey(opts); key != "" {
		client, err := NewHTTPClient(ClientConfig{APIKey: key})
		if err != nil {
			return nil, "", err
		}
		return client, "", nil
	}
	if proj := resolveDecisionsProject(opts); proj != "" {
		if client, ok := tryADC(ctx, proj); ok {
			return client, "", nil
		}
	}
	return nil, "no CODEQUALITY_DECISIONS_API_KEY or GEMINI_API_KEY configured", nil
}

// CODEQUALITY_ALLOW(func_uncovered_lines, external): Ambient GCP service-account ADC detection depends on host environment.
// CODEQUALITY_ALLOW(func_surviving_mutants, external): Ambient GCP service-account ADC detection depends on host environment.
func tryNewScopedADCClient(ctx context.Context, projectID string) (*HTTPClient, bool) {
	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
		Scopes: []string{generativeLangScope, cloudPlatformScopeURL},
	})
	if err != nil {
		return nil, false
	}
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if _, err := creds.Token(probeCtx); err != nil {
		return nil, false
	}
	client, err := NewHTTPClient(ClientConfig{
		TokenProvider: creds,
		QuotaProject:  projectID,
	})
	return client, err == nil
}

func resolveAPIKey(opts DeciderOptions) string {
	envKeys := []string{"CODEQUALITY_DECISIONS_API_KEY", "GEMINI_API_KEY", "GOOGLE_API_KEY"}
	if opts.Getenv != nil {
		for _, k := range envKeys {
			if val := strings.TrimSpace(opts.Getenv(k)); val != "" {
				return val
			}
		}
		if isCIEnvironment(opts.Getenv) {
			return ""
		}
	}
	mkVals := parseLocalConfigMk(resolveConfigMkPath(opts.ConfigMkPath))
	for _, k := range envKeys {
		if val := mkVals[k]; val != "" {
			return val
		}
	}
	return ""
}

func resolveDecisionsProject(opts DeciderOptions) string {
	if strings.TrimSpace(opts.ProjectID) != "" {
		return strings.TrimSpace(opts.ProjectID)
	}
	if opts.Getenv == nil {
		return ""
	}
	return strings.TrimSpace(opts.Getenv("CODEQUALITY_DECISIONS_PROJECT"))
}

func isCIEnvironment(getenv func(string) string) bool {
	return strings.TrimSpace(getenv("GITHUB_ACTIONS")) == "true" ||
		strings.TrimSpace(getenv("BUILD_ID")) != "" ||
		strings.TrimSpace(getenv("BUILDER_OUTPUT")) != ""
}

func resolveConfigMkPath(override string) string {
	if override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "khi", "codequality.mk")
}

func parseLocalConfigMk(path string) map[string]string {
	out := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		for _, sep := range []string{":=", "?=", "="} {
			if k, v, ok := strings.Cut(line, sep); ok {
				out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
				break
			}
		}
	}
	return out
}
