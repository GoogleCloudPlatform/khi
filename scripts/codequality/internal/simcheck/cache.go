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
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	sqliteQueryChunkSize = 400
	createSchemaSQL      = `CREATE TABLE IF NOT EXISTS embeddings (
		key_hash   BLOB PRIMARY KEY,
		vec        BLOB NOT NULL,
		updated_at INTEGER NOT NULL
	) WITHOUT ROWID;`
	upsertEntrySQL = `INSERT INTO embeddings (key_hash, vec, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key_hash) DO UPDATE SET
			vec = excluded.vec,
			updated_at = excluded.updated_at;`
)

// CacheKey is a 32-byte SHA-256 digest identifying an embedding configuration and normalized input text.
type CacheKey [32]byte

// CacheEntry represents a single persisted embedding vector keyed by its 32-byte content hash.
type CacheEntry struct {
	// Key is the 32-byte SHA-256 cache lookup digest.
	Key CacheKey
	// Values holds the embedding vector components.
	Values []float32
}

// VectorCache provides multi-process-safe persistent vector storage backed by WAL-mode SQLite.
type VectorCache struct {
	db *sql.DB
}

// DefaultCachePath resolves the shared cross-workspace SQLite cache path from environment or OS user cache dir.
func DefaultCachePath(getenv func(string) string) (string, error) {
	if getenv != nil {
		if custom := strings.TrimSpace(getenv("CODEQUALITY_SIMILARITY_CACHE_DIR")); custom != "" {
			return filepath.Join(custom, "embeddings_v1.db"), nil
		}
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user cache directory: %w", err)
	}
	return filepath.Join(base, "khi", "codesim", "embeddings_v1.db"), nil
}

// ComputeCacheKey computes a 32-byte SHA-256 digest over embedding configuration and input text.
func ComputeCacheKey(model string, mode string, dims int, text string) CacheKey {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s|%d|%s", model, mode, dims, text)
	var out CacheKey
	copy(out[:], h.Sum(nil))
	return out
}

// OpenVectorCache opens or initializes the SQLite vector cache at path (or in-memory when path is empty).
func OpenVectorCache(ctx context.Context, path string) (*VectorCache, error) {
	dsn, err := prepareSQLiteDSN(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite cache: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, createSchemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize sqlite cache schema: %w", err)
	}
	return &VectorCache{db: db}, nil
}

// Close releases the underlying SQLite connection pool.
func (c *VectorCache) Close() error {
	if c == nil {
		return nil
	}
	return c.db.Close()
}

// LookupBatch fetches only the requested keys from SQLite in chunks and returns decoded L2-normalized vectors.
func (c *VectorCache) LookupBatch(ctx context.Context, keys []CacheKey) (map[CacheKey][]float32, error) {
	unique := deduplicateKeys(keys)
	found := make(map[CacheKey][]float32, len(unique))
	for chunk := range slices.Chunk(unique, sqliteQueryChunkSize) {
		if err := c.queryChunk(ctx, chunk, found); err != nil {
			return nil, err
		}
	}
	return found, nil
}

// StoreBatch upserts normalized vectors into SQLite within a single atomic transaction.
func (c *VectorCache) StoreBatch(ctx context.Context, entries []CacheEntry) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin sqlite transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := execBatchUpsert(ctx, tx, entries, time.Now().Unix()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit sqlite cache transaction: %w", err)
	}
	return nil
}

// PruneOlderThan deletes cached embeddings whose updated_at timestamp precedes cutoff.
func (c *VectorCache) PruneOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := c.db.ExecContext(ctx, "DELETE FROM embeddings WHERE updated_at < ?", cutoff.Unix())
	if err != nil {
		return 0, fmt.Errorf("failed pruning stale embeddings: %w", err)
	}
	return res.RowsAffected()
}

func (c *VectorCache) queryChunk(ctx context.Context, chunk []CacheKey, dst map[CacheKey][]float32) error {
	placeholders := strings.Repeat("?,", len(chunk)-1) + "?"
	query := "SELECT key_hash, vec FROM embeddings WHERE key_hash IN (" + placeholders + ")"
	args := make([]any, len(chunk))
	for i, k := range chunk {
		keyCopy := k
		args[i] = keyCopy[:]
	}

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed querying sqlite embedding cache: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rawKey, rawVec []byte
		if err := rows.Scan(&rawKey, &rawVec); err != nil {
			return fmt.Errorf("failed scanning sqlite embedding row: %w", err)
		}
		vec, ok := decodeFloat32Blob(rawVec)
		if len(rawKey) == 32 && ok {
			var k CacheKey
			copy(k[:], rawKey)
			dst[k] = vec
		}
	}
	return rows.Err()
}

func prepareSQLiteDSN(path string) (string, error) {
	if path == "" {
		return ":memory:", nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}
	return path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)", nil
}

func execBatchUpsert(ctx context.Context, tx *sql.Tx, entries []CacheEntry, nowUnix int64) error {
	stmt, err := tx.PrepareContext(ctx, upsertEntrySQL)
	if err != nil {
		return fmt.Errorf("failed to prepare sqlite upsert: %w", err)
	}
	defer stmt.Close()

	for _, entry := range entries {
		if len(entry.Values) == 0 {
			continue
		}
		keyCopy := entry.Key
		blob := encodeFloat32Blob(NormalizeL2(entry.Values))
		if _, err := stmt.ExecContext(ctx, keyCopy[:], blob, nowUnix); err != nil {
			return fmt.Errorf("failed to upsert embedding cache row: %w", err)
		}
	}
	return nil
}

func deduplicateKeys(keys []CacheKey) []CacheKey {
	seen := make(map[CacheKey]struct{}, len(keys))
	unique := make([]CacheKey, 0, len(keys))
	for _, k := range keys {
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			unique = append(unique, k)
		}
	}
	return unique
}

func encodeFloat32Blob(vals []float32) []byte {
	buf := make([]byte, len(vals)*4)
	for i, v := range vals {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

func decodeFloat32Blob(blob []byte) ([]float32, bool) {
	if len(blob) == 0 || len(blob)%4 != 0 {
		return nil, false
	}
	out := make([]float32, len(blob)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[i*4:]))
	}
	return out, true
}
