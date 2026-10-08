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

package uploadurl

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/chunkedupload"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/google/go-cmp/cmp"
)

const testMaxSizeBytes = 8

// uploadState is the part of an upload result that the handler is expected to change.
type uploadState struct {
	Status         upload.UploadStatus
	FileName       string
	SizeBytes      int64
	HasUploadError bool
}

type handlerTestEnv struct {
	handler  *Handler
	issuer   *Issuer
	store    *upload.UploadFileStore
	provider upload.UploadFileStoreProvider
}

func newHandlerTestEnv(t *testing.T, ttl time.Duration) handlerTestEnv {
	t.Helper()
	tempDir := t.TempDir()
	provider := upload.NewLocalUploadFileStoreProvider(filepath.Join(tempDir, "store"))
	store := upload.NewUploadFileStore(provider)
	chunkManager := chunkedupload.NewChunkSessionManager(filepath.Join(tempDir, "upload"))
	t.Cleanup(chunkManager.Close)
	issuer := NewIssuer(testBaseURL, testMaxSizeBytes, ttl)
	return handlerTestEnv{
		handler:  NewHandler(issuer, upload.NewFileParameterUploadManager(store, chunkManager)),
		issuer:   issuer,
		store:    store,
		provider: provider,
	}
}

func (e handlerTestEnv) state(t *testing.T, token upload.UploadToken) uploadState {
	t.Helper()
	result, err := e.store.GetResult(token, nil)
	if err != nil {
		t.Fatalf("GetResult() returned an unexpected error: %v", err)
	}
	return uploadState{
		Status:         result.Status,
		FileName:       result.FileName,
		SizeBytes:      result.SizeBytes,
		HasUploadError: result.UploadError != nil,
	}
}

func TestHandler_ServeUpload(t *testing.T) {
	testCases := []struct {
		name          string
		ttl           time.Duration
		urlToken      func(grant Grant) string
		body          string
		modifyRequest func(r *http.Request)
		wantStatus    int
		wantResponse  string
		wantState     uploadState
		wantContent   string
	}{
		{
			name:         "stores the body and reports it as processing",
			ttl:          time.Hour,
			body:         "hello",
			wantStatus:   http.StatusOK,
			wantResponse: `{"fieldId":"field-1","status":"PROCESSING","sizeBytes":5}`,
			wantState:    uploadState{Status: upload.UploadStatusCompleted, SizeBytes: 5},
			wantContent:  "hello",
		},
		{
			name:         "accepts a body of exactly the maximum size",
			ttl:          time.Hour,
			body:         "12345678",
			wantStatus:   http.StatusOK,
			wantResponse: `{"fieldId":"field-1","status":"PROCESSING","sizeBytes":8}`,
			wantState:    uploadState{Status: upload.UploadStatusCompleted, SizeBytes: 8},
			wantContent:  "12345678",
		},
		{
			name: "rejects a token that was never issued",
			ttl:  time.Hour,
			urlToken: func(grant Grant) string {
				return "not-issued"
			},
			body:         "hello",
			wantStatus:   http.StatusNotFound,
			wantResponse: `{"error":"unknown upload URL. Request a new upload URL."}`,
			wantState:    uploadState{Status: upload.UploadStatusWaiting},
		},
		{
			name:         "rejects an expired URL",
			ttl:          0,
			body:         "hello",
			wantStatus:   http.StatusGone,
			wantResponse: `{"error":"upload URL has expired. Request a new upload URL."}`,
			wantState:    uploadState{Status: upload.UploadStatusWaiting},
		},
		{
			name: "rejects a request without Content-Length",
			ttl:  time.Hour,
			body: "hello",
			modifyRequest: func(r *http.Request) {
				r.ContentLength = -1
			},
			wantStatus:   http.StatusLengthRequired,
			wantResponse: `{"error":"the Content-Length header is required"}`,
			wantState:    uploadState{Status: upload.UploadStatusWaiting},
		},
		{
			name:         "rejects an empty body",
			ttl:          time.Hour,
			body:         "",
			wantStatus:   http.StatusBadRequest,
			wantResponse: `{"error":"the file is empty"}`,
			wantState:    uploadState{Status: upload.UploadStatusWaiting},
		},
		{
			name:         "rejects a body larger than the maximum size",
			ttl:          time.Hour,
			body:         "123456789",
			wantStatus:   http.StatusRequestEntityTooLarge,
			wantResponse: `{"error":"the file is 9 bytes, which exceeds the limit of 8 bytes"}`,
			wantState:    uploadState{Status: upload.UploadStatusWaiting},
		},
		{
			name: "records a failure when the body is shorter than Content-Length",
			ttl:  time.Hour,
			body: "hello",
			modifyRequest: func(r *http.Request) {
				r.ContentLength = 8
			},
			wantStatus:   http.StatusBadRequest,
			wantResponse: `{"error":"failed to store the file: request body ended before Content-Length bytes were received: unexpected EOF"}`,
			wantState:    uploadState{Status: upload.UploadStatusWaiting, HasUploadError: true},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := newHandlerTestEnv(t, tc.ttl)
			uploadToken := env.store.GetUploadToken("upload-token-1", nil, "field-1")
			grant := env.issuer.Issue(uploadToken.GetID(), "field-1")
			urlToken := strings.TrimPrefix(grant.URL, testBaseURL)
			if tc.urlToken != nil {
				urlToken = tc.urlToken(grant)
			}
			req := httptest.NewRequest(http.MethodPut, grant.URL, strings.NewReader(tc.body))
			if tc.modifyRequest != nil {
				tc.modifyRequest(req)
			}
			rec := httptest.NewRecorder()

			env.handler.ServeUpload(rec, req, urlToken)

			if rec.Code != tc.wantStatus {
				t.Errorf("status code = %d, want %d", rec.Code, tc.wantStatus)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tc.wantResponse {
				t.Errorf("response body = %s, want %s", got, tc.wantResponse)
			}
			if diff := cmp.Diff(tc.wantState, env.state(t, uploadToken)); diff != "" {
				t.Errorf("upload state mismatch (-want +got):\n%s", diff)
			}
			if tc.wantContent == "" {
				return
			}
			reader, err := env.provider.Read(uploadToken)
			if err != nil {
				t.Fatalf("Read() returned an unexpected error: %v", err)
			}
			defer reader.Close()
			content, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("ReadAll() returned an unexpected error: %v", err)
			}
			if string(content) != tc.wantContent {
				t.Errorf("stored content = %q, want %q", content, tc.wantContent)
			}
		})
	}
}

func TestHandler_ServeUploadOnlyWritesItsOwnField(t *testing.T) {
	env := newHandlerTestEnv(t, time.Hour)
	tokenA := env.store.GetUploadToken("upload-token-a", nil, "field-a")
	tokenB := env.store.GetUploadToken("upload-token-b", nil, "field-b")
	grant := env.issuer.Issue(tokenA.GetID(), "field-a")
	req := httptest.NewRequest(http.MethodPut, grant.URL, strings.NewReader("hello"))
	rec := httptest.NewRecorder()

	env.handler.ServeUpload(rec, req, strings.TrimPrefix(grant.URL, testBaseURL))

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
	if diff := cmp.Diff(uploadState{Status: upload.UploadStatusCompleted, SizeBytes: 5}, env.state(t, tokenA)); diff != "" {
		t.Errorf("state of the target field mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(uploadState{Status: upload.UploadStatusWaiting}, env.state(t, tokenB)); diff != "" {
		t.Errorf("state of the other field mismatch (-want +got):\n%s", diff)
	}
}

func TestHandler_ServeUploadRecordsSessionStartFailure(t *testing.T) {
	tempDir := t.TempDir()
	store := upload.NewUploadFileStore(upload.NewLocalUploadFileStoreProvider(filepath.Join(tempDir, "store")))
	// A regular file at the upload directory path makes creating the chunk session fail.
	uploadDir := filepath.Join(tempDir, "upload")
	if err := os.WriteFile(uploadDir, nil, 0o644); err != nil {
		t.Fatalf("WriteFile() returned an unexpected error: %v", err)
	}
	chunkManager := chunkedupload.NewChunkSessionManager(uploadDir)
	t.Cleanup(chunkManager.Close)
	issuer := NewIssuer(testBaseURL, testMaxSizeBytes, time.Hour)
	handler := NewHandler(issuer, upload.NewFileParameterUploadManager(store, chunkManager))
	uploadToken := store.GetUploadToken("upload-token-1", nil, "field-1")
	grant := issuer.Issue(uploadToken.GetID(), "field-1")
	req := httptest.NewRequest(http.MethodPut, grant.URL, strings.NewReader("hello"))
	rec := httptest.NewRecorder()

	handler.ServeUpload(rec, req, strings.TrimPrefix(grant.URL, testBaseURL))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	result, err := store.GetResult(uploadToken, nil)
	if err != nil {
		t.Fatalf("GetResult() returned an unexpected error: %v", err)
	}
	if result.Status != upload.UploadStatusWaiting {
		t.Errorf("status = %v, want %v", result.Status, upload.UploadStatusWaiting)
	}
	if result.UploadError == nil {
		t.Errorf("UploadError = nil, want the session start failure")
	}
}
