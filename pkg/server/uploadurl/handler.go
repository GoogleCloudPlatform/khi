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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/GoogleCloudPlatform/khi/pkg/server/chunkedupload"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
)

// StatusProcessing is the upload status reported after a file is stored and while it is verified asynchronously.
const StatusProcessing = "PROCESSING"

// errIncompleteBody marks failures caused by a request body shorter than its Content-Length.
var errIncompleteBody = errors.New("request body ended before Content-Length bytes were received")

type uploadResponse struct {
	FieldID   string `json:"fieldId"`
	Status    string `json:"status"`
	SizeBytes int64  `json:"sizeBytes"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Handler stores files sent to upload URLs. It uses the same persistence and verification path as Web UI uploads.
type Handler struct {
	issuer  *Issuer
	manager *upload.FileParameterUploadManager
}

// NewHandler returns a Handler that accepts uploads for URLs issued by issuer and stores them with manager.
func NewHandler(issuer *Issuer, manager *upload.FileParameterUploadManager) *Handler {
	return &Handler{
		issuer:  issuer,
		manager: manager,
	}
}

// ServeUpload stores the request body as the file of the field that urlToken was issued for.
// The request must declare the body size in Content-Length.
func (h *Handler) ServeUpload(w http.ResponseWriter, r *http.Request, urlToken string) {
	target, err := h.issuer.resolve(urlToken)
	switch {
	case errors.Is(err, ErrUnknownURL):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error() + ". Request a new upload URL."})
		return
	case errors.Is(err, ErrExpiredURL):
		writeJSON(w, http.StatusGone, errorResponse{Error: err.Error() + ". Request a new upload URL."})
		return
	}

	size := r.ContentLength
	switch {
	case size < 0:
		writeJSON(w, http.StatusLengthRequired, errorResponse{Error: "the Content-Length header is required"})
		return
	case size == 0:
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "the file is empty"})
		return
	case size > h.issuer.maxSizeBytes:
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{
			Error: fmt.Sprintf("the file is %d bytes, which exceeds the limit of %d bytes", size, h.issuer.maxSizeBytes),
		})
		return
	}

	// An upload URL accepts a single upload so that concurrent uploads cannot overwrite each other.
	// Requests rejected above leave the URL usable because they did not touch the stored file.
	if !h.issuer.consume(urlToken) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: ErrUnknownURL.Error() + ". Request a new upload URL."})
		return
	}

	// The body of a PUT request carries no file name, so the upload is recorded without one.
	session, err := h.manager.StartUploadSession(target.uploadTokenID, "", size)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: fmt.Sprintf("failed to start the upload: %v", err)})
		return
	}

	storedSize, err := h.storeBody(r.Body, session.Token, size)
	if err != nil {
		h.recordFailure(r.Context(), session.Token, target.uploadTokenID, err)
		status := http.StatusInternalServerError
		if errors.Is(err, errIncompleteBody) {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, errorResponse{Error: fmt.Sprintf("failed to store the file: %v", err)})
		return
	}

	writeJSON(w, http.StatusOK, uploadResponse{
		FieldID:   target.fieldID,
		Status:    StatusProcessing,
		SizeBytes: storedSize,
	})
}

// storeBody writes size bytes of body to the upload session in chunks and completes the session.
func (h *Handler) storeBody(body io.Reader, sessionToken string, size int64) (int64, error) {
	buf := make([]byte, min(size, h.manager.SuggestedChunkSize()))
	for offset := int64(0); offset < size; {
		n, err := io.ReadFull(body, buf[:min(int64(len(buf)), size-offset)])
		if err != nil {
			return 0, fmt.Errorf("%w: %v", errIncompleteBody, err)
		}
		if _, err := h.manager.WriteChunk(sessionToken, offset, buf[:n]); err != nil {
			return 0, err
		}
		offset += int64(n)
	}
	return h.manager.CompleteUploadSession(sessionToken)
}

// recordFailure discards the partial upload and records cause so that the file form shows it.
func (h *Handler) recordFailure(ctx context.Context, sessionToken string, uploadTokenID string, cause error) {
	// CompleteUploadSession removes the session even when it fails, so a missing session is expected here.
	if err := h.manager.AbortUploadSession(sessionToken); err != nil && !errors.Is(err, chunkedupload.ErrSessionNotFound) {
		slog.WarnContext(ctx, "failed to abort the upload session", "error", err)
	}
	if err := h.manager.RecordUploadFailure(uploadTokenID, cause); err != nil {
		slog.WarnContext(ctx, "failed to record the upload failure", "error", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Warn("failed to write the upload response", "error", err)
	}
}
