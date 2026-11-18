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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testCacheHolder struct {
	db *sql.DB
}

func TestDefaultPath(t *testing.T) {
	testCases := []struct {
		name       string
		getenv     func(string) string
		wantExact  string
		wantSuffix string
	}{
		// Uses the trimmed environment override directory when envVar resolves to a non-blank path.
		{
			name: "explicit env override uses custom dir",
			getenv: func(key string) string {
				if key == "CODEQUALITY_TEST_CACHE_DIR" {
					return "  /tmp/custom-sqlitecache  "
				}
				return ""
			},
			wantExact: filepath.Join("/tmp/custom-sqlitecache", "cache_v1.db"),
		},
		// Falls back to the user cache directory when getenv is non-nil but envVar contains only whitespace.
		{
			name: "blank env override falls back to user cache dir",
			getenv: func(string) string {
				return "   "
			},
			wantSuffix: filepath.Join("khi", "testsub", "cache_v1.db"),
		},
		// Falls back to the user cache directory when getenv is nil.
		{
			name:       "nil getenv falls back to user cache dir",
			getenv:     nil,
			wantSuffix: filepath.Join("khi", "testsub", "cache_v1.db"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, err := DefaultPath(tc.getenv, "CODEQUALITY_TEST_CACHE_DIR", "testsub", "cache_v1.db")

			if err != nil {
				t.Fatalf("DefaultPath() unexpected error: %v", err)
			}
			if tc.wantExact != "" && gotPath != tc.wantExact {
				t.Errorf("DefaultPath() = %q, want %q", gotPath, tc.wantExact)
			}
			if tc.wantSuffix != "" && !strings.HasSuffix(gotPath, tc.wantSuffix) {
				t.Errorf("DefaultPath() = %q, want suffix %q", gotPath, tc.wantSuffix)
			}
		})
	}
}

// TestDefaultPathErrorWhenUserCacheDirUnavailable verifies that DefaultPath returns an error when os.UserCacheDir cannot resolve a directory.
func TestDefaultPathErrorWhenUserCacheDirUnavailable(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")

	_, err := DefaultPath(nil, "CODEQUALITY_TEST_CACHE_DIR", "testsub", "cache_v1.db")

	if err == nil || !strings.Contains(err.Error(), "user cache directory") {
		t.Errorf("DefaultPath() error = %v, want user cache directory error", err)
	}
}

func TestOpen(t *testing.T) {
	t.Chdir(t.TempDir())
	ctx := context.Background()
	nestedPath := filepath.Join(t.TempDir(), "nested", "items.db")

	testCases := []struct {
		name        string
		path        string
		wantJournal string
	}{
		// Opens an in-memory SQLite database and initializes schemaSQL when path is empty.
		{
			name:        "empty path opens in-memory database",
			path:        "",
			wantJournal: "memory",
		},
		// Creates nested parent directories, enables WAL journal mode, and initializes schemaSQL for a file path.
		{
			name:        "file path creates parent directory and enables WAL",
			path:        nestedPath,
			wantJournal: "wal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := Open(ctx, tc.path, "CREATE TABLE items (id INTEGER PRIMARY KEY);")

			if err != nil {
				t.Fatalf("Open(%q) unexpected error: %v", tc.path, err)
			}
			defer db.Close()
			if gotJournal := queryJournalMode(t, ctx, db); gotJournal != tc.wantJournal {
				t.Errorf("journal_mode = %q, want %q", gotJournal, tc.wantJournal)
			}
			if _, execErr := db.ExecContext(ctx, "INSERT INTO items (id) VALUES (1)"); execErr != nil {
				t.Errorf("ExecContext(INSERT) unexpected error: %v", execErr)
			}
		})
	}
}

func TestOpenErrors(t *testing.T) {
	t.Chdir(t.TempDir())
	ctx := context.Background()
	blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile(blockingFile) unexpected error: %v", err)
	}

	testCases := []struct {
		name       string
		path       string
		schemaSQL  string
		wantErrSub string
	}{
		// Returns a directory-creation error before opening SQLite when the parent path collides with a regular file.
		{
			name:       "parent path collision returns mkdir error",
			path:       filepath.Join(blockingFile, "items.db"),
			schemaSQL:  "CREATE TABLE items (id INTEGER PRIMARY KEY);",
			wantErrSub: "failed creating sqlite cache directory",
		},
		// Closes the opened SQLite handle and returns a schema error when schemaSQL is invalid.
		{
			name:       "invalid schema closes db and returns error",
			path:       "",
			schemaSQL:  "CREATE TABLE (",
			wantErrSub: "failed initializing sqlite cache schema",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := Open(ctx, tc.path, tc.schemaSQL)

			if err == nil || !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Errorf("Open(%q) error = %v, want substring %q", tc.path, err, tc.wantErrSub)
			}
			if db != nil {
				t.Errorf("Open(%q) db = %v, want nil on error", tc.path, db)
			}
		})
	}
}

// TestOpenWithWrapsDatabase verifies that OpenWith invokes wrap with the initialized *sql.DB and returns the wrapped holder.
func TestOpenWithWrapsDatabase(t *testing.T) {
	t.Chdir(t.TempDir())
	ctx := context.Background()
	var wrapCalled bool
	wrap := func(db *sql.DB) *testCacheHolder {
		wrapCalled = true
		return &testCacheHolder{db: db}
	}

	gotHolder, err := OpenWith(ctx, "", "CREATE TABLE entries (id INTEGER PRIMARY KEY);", wrap)

	if err != nil {
		t.Fatalf("OpenWith() unexpected error: %v", err)
	}
	defer gotHolder.db.Close()
	if !wrapCalled {
		t.Errorf("wrapCalled = false, want true")
	}
	if gotHolder.db == nil {
		t.Errorf("OpenWith() returned holder with nil db")
	}
}

// TestOpenWithPropagatesOpenError verifies that OpenWith returns schema initialization errors without invoking wrap.
func TestOpenWithPropagatesOpenError(t *testing.T) {
	t.Chdir(t.TempDir())
	ctx := context.Background()
	var wrapCalled bool
	wrap := func(db *sql.DB) *testCacheHolder {
		wrapCalled = true
		return &testCacheHolder{db: db}
	}

	gotHolder, err := OpenWith(ctx, "", "INVALID SQL", wrap)

	if err == nil || !strings.Contains(err.Error(), "failed initializing sqlite cache schema") {
		t.Errorf("OpenWith() error = %v, want schema error", err)
	}
	if wrapCalled {
		t.Errorf("wrapCalled = true, want false on error")
	}
	if gotHolder != nil {
		t.Errorf("OpenWith() holder = %v, want nil on error", gotHolder)
	}
}

func TestCloseWith(t *testing.T) {
	ctx := context.Background()
	openedDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() unexpected error: %v", err)
	}

	testCases := []struct {
		name            string
		holder          *testCacheHolder
		wantGetDBCalled bool
		wantClosed      bool
	}{
		// Returns nil without calling getDB when the holder pointer is nil.
		{
			name:            "nil holder is a no-op",
			holder:          nil,
			wantGetDBCalled: false,
		},
		// Returns nil safely when holder is non-nil and getDB returns a nil database handle.
		{
			name:            "non-nil holder with nil db is a no-op",
			holder:          &testCacheHolder{db: nil},
			wantGetDBCalled: true,
		},
		// Closes the non-nil *sql.DB returned by getDB so subsequent database pings fail.
		{
			name:            "non-nil holder closes underlying db",
			holder:          &testCacheHolder{db: openedDB},
			wantGetDBCalled: true,
			wantClosed:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var getDBCalled bool
			getDB := func(holder *testCacheHolder) *sql.DB {
				getDBCalled = true
				return holder.db
			}

			closeErr := CloseWith(tc.holder, getDB)

			if closeErr != nil {
				t.Fatalf("CloseWith() unexpected error: %v", closeErr)
			}
			if getDBCalled != tc.wantGetDBCalled {
				t.Errorf("getDBCalled = %v, want %v", getDBCalled, tc.wantGetDBCalled)
			}
			if tc.wantClosed {
				if pingErr := tc.holder.db.PingContext(ctx); pingErr == nil {
					t.Errorf("PingContext() after CloseWith() = nil, want closed-database error")
				}
			}
		})
	}
}

func queryJournalMode(t *testing.T, ctx context.Context, db *sql.DB) string {
	t.Helper()
	var mode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode unexpected error: %v", err)
	}
	return mode
}
