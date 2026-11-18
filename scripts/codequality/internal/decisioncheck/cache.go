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
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/sqlitecache"
)

const (
	createDecisionCacheSchemaSQL = `CREATE TABLE IF NOT EXISTS decision_responses (
		key_hash      BLOB PRIMARY KEY,
		response_json BLOB NOT NULL,
		updated_at    INTEGER NOT NULL
	) WITHOUT ROWID;`
	upsertDecisionCacheSQL = `INSERT INTO decision_responses (key_hash, response_json, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key_hash) DO UPDATE SET
			response_json = excluded.response_json,
			updated_at = excluded.updated_at;`
)

// CacheKey is a 32-byte SHA-256 digest over model ID, shared input, and canonical question JSON.
type CacheKey [32]byte

// DecisionCache persists Gemini Decisions API responses in a WAL-mode SQLite database.
type DecisionCache struct {
	db *sql.DB
}

// OpenDecisionCache opens or initializes the SQLite decision cache at path (or in-memory when path is empty).
func OpenDecisionCache(ctx context.Context, path string) (*DecisionCache, error) {
	return sqlitecache.OpenWith(ctx, path, createDecisionCacheSchemaSQL, func(db *sql.DB) *DecisionCache {
		return &DecisionCache{db: db}
	})
}

// Close releases the underlying SQLite database connection (no-op when c is nil).
func (c *DecisionCache) Close() error {
	return sqlitecache.CloseWith(c, func(cache *DecisionCache) *sql.DB { return cache.db })
}

// Lookup fetches a cached DecisionResponse for key when present, returning a miss when c is nil or cached JSON is invalid.
func (c *DecisionCache) Lookup(ctx context.Context, key CacheKey) (DecisionResponse, bool, error) {
	if c == nil {
		return DecisionResponse{}, false, nil
	}
	var raw []byte
	err := c.db.QueryRowContext(ctx, "SELECT response_json FROM decision_responses WHERE key_hash = ?", key[:]).Scan(&raw)
	if err == sql.ErrNoRows {
		return DecisionResponse{}, false, nil
	}
	if err != nil {
		return DecisionResponse{}, false, fmt.Errorf("failed querying decision cache: %w", err)
	}
	var resp DecisionResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return DecisionResponse{}, false, nil
	}
	return resp, true, nil
}

// Store persists resp under key in the SQLite cache (no-op when c is nil).
func (c *DecisionCache) Store(ctx context.Context, key CacheKey, resp DecisionResponse) error {
	if c == nil {
		return nil
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed marshaling decision response for cache: %w", err)
	}
	if _, err := c.db.ExecContext(ctx, upsertDecisionCacheSQL, key[:], raw, time.Now().Unix()); err != nil {
		return fmt.Errorf("failed storing decision cache entry: %w", err)
	}
	return nil
}

// DefaultCachePath resolves the persistent SQLite cache path from CODEQUALITY_DECISIONS_CACHE_DIR or OS cache dir.
func DefaultCachePath(getenv func(string) string) (string, error) {
	return sqlitecache.DefaultPath(getenv, "CODEQUALITY_DECISIONS_CACHE_DIR", "decisions", "decisions_v1.db")
}

// ComputeBatchCacheKey computes the SHA-256 content address of req.
func ComputeBatchCacheKey(req DecisionRequest) (CacheKey, error) {
	qBytes, err := CanonicalQuestionsJSON(req.Questions)
	if err != nil {
		return CacheKey{}, err
	}
	h := sha256.New()
	_, _ = h.Write([]byte(req.Model))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(req.Input))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(qBytes)
	var out CacheKey
	copy(out[:], h.Sum(nil))
	return out, nil
}
