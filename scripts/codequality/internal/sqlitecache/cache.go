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

package sqlitecache

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const sqlitePRAGMAQuery = "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"

// DefaultPath resolves a persistent SQLite cache file path from envVar (via getenv) or the OS user cache directory.
func DefaultPath(getenv func(string) string, envVar string, subDir string, fileName string) (string, error) {
	if getenv != nil {
		if custom := strings.TrimSpace(getenv(envVar)); custom != "" {
			return filepath.Join(custom, fileName), nil
		}
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed resolving user cache directory: %w", err)
	}
	return filepath.Join(base, "khi", subDir, fileName), nil
}

// Open opens or initializes a WAL-mode SQLite database at path (or in-memory when path is empty) and runs schemaSQL.
func Open(ctx context.Context, path string, schemaSQL string) (*sql.DB, error) {
	dsn, err := prepareDSN(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed opening sqlite cache: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed initializing sqlite cache schema: %w", err)
	}
	return db, nil
}

// OpenWith opens or initializes a SQLite database at path with schemaSQL and wraps it using wrap.
func OpenWith[T any](ctx context.Context, path string, schemaSQL string, wrap func(*sql.DB) *T) (*T, error) {
	db, err := Open(ctx, path, schemaSQL)
	if err != nil {
		return nil, err
	}
	return wrap(db), nil
}

// CloseWith closes the *sql.DB returned by getDB(holder) when both holder and the database handle are non-nil.
func CloseWith[T any](holder *T, getDB func(*T) *sql.DB) error {
	if holder == nil {
		return nil
	}
	if db := getDB(holder); db != nil {
		return db.Close()
	}
	return nil
}

func prepareDSN(path string) (string, error) {
	if path == "" {
		return ":memory:", nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("failed creating sqlite cache directory: %w", err)
	}
	return path + sqlitePRAGMAQuery, nil
}
