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
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/genai"
)

const (
	defaultVertexLocation  = "global"
	defaultEmbeddingModel  = "gemini-embedding-2"
	cacheNamespaceMode     = "ast_norm"
	defaultDimensions      = 768
	defaultConcurrency     = 16
	persistChunkSize       = 128
	maxInstanceChars       = 32000
	defaultMetadataBaseURL = "http://metadata.google.internal"
	metadataProbeTimeout   = 500 * time.Millisecond
)

var _ Embedder = (*VertexClient)(nil)

// VertexConfig configures Vertex AI gemini-embedding-2 requests via google.golang.org/genai and local SQLite caching.
type VertexConfig struct {
	// ProjectID is the Google Cloud project ID billed for Vertex AI calls.
	ProjectID string
	// Location is the Vertex AI location (defaults to "global").
	Location string
	// Model is the Vertex AI publisher model ID (defaults to "gemini-embedding-2").
	Model string
	// Dimensions is the output vector dimensionality (defaults to 768).
	Dimensions int32
	// Concurrency caps parallel in-flight EmbedContent requests (defaults to 16).
	Concurrency int
	// CachePath is the SQLite file path used to persist vectors (or empty for in-memory).
	CachePath string
	// HTTPClient optionally overrides the underlying HTTP client (primarily for unit tests).
	HTTPClient *http.Client
	// HTTPOptions configures base URL, custom headers, and SDK retry behavior.
	HTTPOptions genai.HTTPOptions
}

// EmbedderOptions configures GCP project resolution and environment lookup for TryNewDefaultEmbedder.
type EmbedderOptions struct {
	// ProjectID explicitly specifies the GCP project ID billed for Vertex AI calls when non-empty.
	ProjectID string
	// Getenv reads environment variables for fallback project and cache directory lookup.
	Getenv func(string) string
}

// VertexClient implements Embedder using the official google.golang.org/genai SDK backed by VectorCache.
type VertexClient struct {
	cfg         VertexConfig
	genaiClient *genai.Client
	cache       *VectorCache
}

type defaultEmbedderDeps struct {
	resolveProject func(ctx context.Context, getenv func(string) string) string
	resolveCache   func(getenv func(string) string) (string, error)
	httpClient     *http.Client
	httpOptions    genai.HTTPOptions
}

type pendingItem struct {
	key  CacheKey
	text string
}

type embedChunkState struct {
	client   *VertexClient
	items    []pendingItem
	entries  []CacheEntry
	nextIdx  atomic.Int64
	cancel   context.CancelFunc
	errMu    sync.Mutex
	firstErr error
}

// NewVertexClient initializes a genai.Client targeting Vertex AI and opens its local SQLite vector cache.
func NewVertexClient(ctx context.Context, cfg VertexConfig) (*VertexClient, error) {
	cfg = applyVertexDefaults(cfg)
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("vertex project ID is required")
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend:     genai.BackendVertexAI,
		Project:     cfg.ProjectID,
		Location:    cfg.Location,
		HTTPClient:  cfg.HTTPClient,
		HTTPOptions: withQuotaProjectHeader(cfg.HTTPOptions, cfg.ProjectID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed creating Vertex AI genai client: %w", err)
	}
	vc, err := OpenVectorCache(ctx, cfg.CachePath)
	if err != nil {
		return nil, err
	}
	return &VertexClient{
		cfg:         cfg,
		genaiClient: client,
		cache:       vc,
	}, nil
}

// Close releases the underlying SQLite cache connection.
func (c *VertexClient) Close() error {
	if c == nil {
		return nil
	}
	return c.cache.Close()
}

// EmbedBatch returns L2-normalized embedding vectors aligned 1-to-1 with texts.
func (c *VertexClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	dims := int(c.cfg.Dimensions)
	keys := make([]CacheKey, len(texts))
	for i, text := range texts {
		keys[i] = ComputeCacheKey(c.cfg.Model, cacheNamespaceMode, dims, text)
	}
	cached, err := c.cache.LookupBatch(ctx, keys)
	if err != nil {
		return nil, err
	}

	missing := collectMissingItems(texts, keys, cached)
	if err := c.fetchAndStoreMissing(ctx, missing, cached); err != nil {
		return nil, err
	}
	return assembleBatchResults(keys, cached), nil
}

func (c *VertexClient) fetchAndStoreMissing(ctx context.Context, missing []pendingItem, cached map[CacheKey][]float32) error {
	for chunk := range slices.Chunk(missing, persistChunkSize) {
		entries, err := c.embedChunkConcurrently(ctx, chunk)
		if err != nil {
			return err
		}
		if err := c.cache.StoreBatch(ctx, entries); err != nil {
			return err
		}
		for _, entry := range entries {
			cached[entry.Key] = entry.Values
		}
	}
	return nil
}

func (c *VertexClient) embedChunkConcurrently(ctx context.Context, items []pendingItem) ([]CacheEntry, error) {
	chunkCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	state := &embedChunkState{
		client:  c,
		items:   items,
		entries: make([]CacheEntry, len(items)),
		cancel:  cancel,
	}

	workers := min(c.cfg.Concurrency, len(items))
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			state.runWorker(chunkCtx)
		}()
	}
	wg.Wait()
	return state.entries, state.firstErr
}

func (c *VertexClient) embedSingle(ctx context.Context, item pendingItem) (CacheEntry, error) {
	runes := []rune(item.text)
	trimmed := string(runes[:min(len(runes), maxInstanceChars)])
	dims := c.cfg.Dimensions
	resp, err := c.genaiClient.Models.EmbedContent(ctx, c.cfg.Model, genai.Text(trimmed), &genai.EmbedContentConfig{
		OutputDimensionality: &dims,
	})
	if err != nil {
		return CacheEntry{}, fmt.Errorf("vertex AI embedContent failed: %w", err)
	}
	if len(resp.Embeddings) == 0 || len(resp.Embeddings[0].Values) == 0 {
		return CacheEntry{}, fmt.Errorf("empty embedding vector returned by Vertex AI")
	}
	return CacheEntry{
		Key:    item.key,
		Values: NormalizeL2(resp.Embeddings[0].Values),
	}, nil
}

// TryNewDefaultEmbedder initializes a VertexClient via ADC and project configuration, returning (nil, reason, nil) when unavailable.
func TryNewDefaultEmbedder(ctx context.Context, opts EmbedderOptions) (Embedder, string, error) {
	return tryNewEmbedderWithDeps(ctx, opts, defaultEmbedderDeps{
		resolveProject: resolveProjectID,
		resolveCache:   DefaultCachePath,
	})
}

func (s *embedChunkState) runWorker(ctx context.Context) {
	for {
		idx := int(s.nextIdx.Add(1) - 1)
		if idx >= len(s.items) || ctx.Err() != nil {
			return
		}
		entry, err := s.client.embedSingle(ctx, s.items[idx])
		if err != nil {
			s.errMu.Lock()
			if s.firstErr == nil {
				s.firstErr = err
				s.cancel()
			}
			s.errMu.Unlock()
			return
		}
		s.entries[idx] = entry
	}
}

func withQuotaProjectHeader(opts genai.HTTPOptions, projectID string) genai.HTTPOptions {
	headers := make(http.Header)
	if opts.Headers != nil {
		headers = opts.Headers.Clone()
	}
	if headers.Get("X-Goog-User-Project") == "" {
		headers.Set("X-Goog-User-Project", projectID)
	}
	opts.Headers = headers
	return opts
}

func tryNewEmbedderWithDeps(ctx context.Context, opts EmbedderOptions, deps defaultEmbedderDeps) (Embedder, string, error) {
	projectID := strings.TrimSpace(opts.ProjectID)
	if projectID == "" {
		probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		projectID = deps.resolveProject(probeCtx, opts.Getenv)
	}
	if projectID == "" {
		return nil, "no GCP project configured for Vertex AI embeddings", nil
	}
	cachePath, err := deps.resolveCache(opts.Getenv)
	if err != nil {
		return nil, "", err
	}
	client, err := NewVertexClient(ctx, VertexConfig{
		ProjectID:   projectID,
		CachePath:   cachePath,
		HTTPClient:  deps.httpClient,
		HTTPOptions: deps.httpOptions,
	})
	if err != nil {
		return nil, fmt.Sprintf("Vertex AI client unavailable: %v", err), nil
	}
	return client, "", nil
}

func collectMissingItems(texts []string, keys []CacheKey, cached map[CacheKey][]float32) []pendingItem {
	var missing []pendingItem
	seen := make(map[CacheKey]bool)
	for i, key := range keys {
		if _, ok := cached[key]; ok {
			continue
		}
		if !seen[key] {
			seen[key] = true
			missing = append(missing, pendingItem{key: key, text: texts[i]})
		}
	}
	return missing
}

func assembleBatchResults(keys []CacheKey, cached map[CacheKey][]float32) [][]float32 {
	results := make([][]float32, len(keys))
	for i, key := range keys {
		results[i] = cached[key]
	}
	return results
}

func applyVertexDefaults(cfg VertexConfig) VertexConfig {
	if cfg.Location == "" {
		cfg.Location = defaultVertexLocation
	}
	if cfg.Model == "" {
		cfg.Model = defaultEmbeddingModel
	}
	if cfg.Dimensions <= 0 {
		cfg.Dimensions = defaultDimensions
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = defaultConcurrency
	}
	return cfg
}

// CODEQUALITY_ALLOW(func_uncovered_lines, external): GCE metadata server and gcloud CLI fallbacks depend on host environment.
// CODEQUALITY_ALLOW(func_surviving_mutants, external): GCE metadata server and gcloud CLI fallbacks depend on host environment.
func resolveProjectID(ctx context.Context, getenv func(string) string) string {
	if val := resolveEnvProjectID(getenv); val != "" {
		return val
	}
	if isCloudBuildEnv(getenv) {
		if val := queryMetadataProjectID(ctx, http.DefaultClient, defaultMetadataBaseURL); val != "" {
			return val
		}
	}
	out, err := exec.CommandContext(ctx, "gcloud", "config", "get-value", "project").Output()
	if err == nil {
		proj := strings.TrimSpace(string(out))
		if proj != "" && proj != "(unset)" {
			return proj
		}
	}
	return queryMetadataProjectID(ctx, http.DefaultClient, defaultMetadataBaseURL)
}

func isCloudBuildEnv(getenv func(string) string) bool {
	if getenv == nil {
		return false
	}
	return strings.TrimSpace(getenv("BUILD_ID")) != "" || strings.TrimSpace(getenv("BUILDER_OUTPUT")) != ""
}

func resolveEnvProjectID(getenv func(string) string) string {
	if getenv == nil {
		return ""
	}
	keys := []string{
		"CODEQUALITY_SIMILARITY_PROJECT",
		"GOOGLE_CLOUD_PROJECT",
		"GCLOUD_PROJECT",
		"CLOUDSDK_CORE_PROJECT",
		"PROJECT_ID",
	}
	for _, key := range keys {
		if val := strings.TrimSpace(getenv(key)); val != "" {
			return val
		}
	}
	return ""
}

func queryMetadataProjectID(ctx context.Context, client *http.Client, baseURL string) string {
	metaCtx, cancel := context.WithTimeout(ctx, metadataProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(metaCtx, http.MethodGet, baseURL+"/computeMetadata/v1/project/project-id", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}
