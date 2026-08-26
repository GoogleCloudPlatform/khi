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

package apiv1

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/server/importinspection"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"google.golang.org/protobuf/proto"
)

// ImportInspectionServiceServer implements the apiv1connect.ImportInspectionServiceHandler interface.
type ImportInspectionServiceServer struct {
	manager      *importinspection.ImportSessionManager
	indexManager *workbench.InspectionIndexManager
}

var _ apiv1connect.ImportInspectionServiceHandler = (*ImportInspectionServiceServer)(nil)

// NewImportInspectionServiceServer creates a new ImportInspectionServiceServer backed by the given manager.
func NewImportInspectionServiceServer(manager *importinspection.ImportSessionManager, indexManager *workbench.InspectionIndexManager) *ImportInspectionServiceServer {
	return &ImportInspectionServiceServer{
		manager:      manager,
		indexManager: indexManager,
	}
}

// StartImportInspection initializes an import session and returns an upload token and suggested chunk size.
func (s *ImportInspectionServiceServer) StartImportInspection(
	ctx context.Context,
	req *connect.Request[apiv1.StartImportInspectionRequest],
) (*connect.Response[apiv1.StartImportInspectionResponse], error) {
	msg := req.Msg
	if msg.GetFileName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("file_name is required"))
	}
	if msg.GetTotalSizeBytes() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("total_size_bytes must be positive"))
	}

	session, err := s.manager.StartSession(msg.GetFileName(), msg.GetTotalSizeBytes())
	if err != nil {
		if errors.Is(err, importinspection.ErrInvalidTotalSize) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start import session: %w", err))
	}

	res := &apiv1.StartImportInspectionResponse{
		ImportToken:             proto.String(session.Token),
		SuggestedChunkSizeBytes: proto.Int64(s.manager.SuggestedChunkSize()),
	}
	return connect.NewResponse(res), nil
}

// UploadInspectionChunk receives and appends a single chunk of data to the session's temporary file.
func (s *ImportInspectionServiceServer) UploadInspectionChunk(
	ctx context.Context,
	req *connect.Request[apiv1.UploadInspectionChunkRequest],
) (*connect.Response[apiv1.UploadInspectionChunkResponse], error) {
	msg := req.Msg
	if msg.GetImportToken() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("import_token is required"))
	}

	totalReceived, err := s.manager.WriteChunk(msg.GetImportToken(), msg.GetOffsetBytes(), msg.GetData())
	if err != nil {
		if errors.Is(err, importinspection.ErrSessionNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		if errors.Is(err, importinspection.ErrEmptyChunkData) || errors.Is(err, importinspection.ErrInvalidOffset) || errors.Is(err, importinspection.ErrChunkSizeTooLarge) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to write chunk: %w", err))
	}

	res := &apiv1.UploadInspectionChunkResponse{
		TotalReceivedBytes: proto.Int64(totalReceived),
	}
	return connect.NewResponse(res), nil
}

// CompleteImportInspection finalizes the upload, validates the KHI file, and registers the inspection.
func (s *ImportInspectionServiceServer) CompleteImportInspection(
	ctx context.Context,
	req *connect.Request[apiv1.CompleteImportInspectionRequest],
) (*connect.Response[apiv1.CompleteImportInspectionResponse], error) {
	msg := req.Msg
	if msg.GetImportToken() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("import_token is required"))
	}

	finalized, err := s.manager.CompleteSession(msg.GetImportToken())
	if err != nil {
		if errors.Is(err, importinspection.ErrSessionNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to complete import: %w", err))
	}

	if s.indexManager != nil {
		s.indexManager.InvalidateInspectionIndex(finalized.InspectionID)
		s.indexManager.StartAsyncIndexing(context.Background(), finalized.InspectionID)
	}

	res := &apiv1.CompleteImportInspectionResponse{
		InspectionId:   proto.String(finalized.InspectionID),
		InspectionName: proto.String(finalized.InspectionName),
		FileSizeBytes:  proto.Int64(finalized.FileSize),
	}
	return connect.NewResponse(res), nil
}

// AbortImportInspection aborts an in-progress import session and cleans up temporary resources.
func (s *ImportInspectionServiceServer) AbortImportInspection(
	ctx context.Context,
	req *connect.Request[apiv1.AbortImportInspectionRequest],
) (*connect.Response[apiv1.AbortImportInspectionResponse], error) {
	msg := req.Msg
	if msg.GetImportToken() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("import_token is required"))
	}

	if err := s.manager.AbortSession(msg.GetImportToken()); err != nil {
		if errors.Is(err, importinspection.ErrSessionNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to abort import session: %w", err))
	}

	res := &apiv1.AbortImportInspectionResponse{
		Aborted: proto.Bool(true),
	}
	return connect.NewResponse(res), nil
}
