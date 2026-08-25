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
	"github.com/GoogleCloudPlatform/khi/pkg/server/chunkedupload"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"google.golang.org/protobuf/proto"
)

// FileParameterUploadServiceServer implements the apiv1connect.FileParameterUploadServiceHandler interface.
type FileParameterUploadServiceServer struct {
	manager *upload.FileParameterUploadManager
}

var _ apiv1connect.FileParameterUploadServiceHandler = (*FileParameterUploadServiceServer)(nil)

// NewFileParameterUploadServiceServer creates a new FileParameterUploadServiceServer backed by the given manager.
func NewFileParameterUploadServiceServer(manager *upload.FileParameterUploadManager) *FileParameterUploadServiceServer {
	return &FileParameterUploadServiceServer{
		manager: manager,
	}
}

// StartFileUpload initializes a chunked file parameter upload session.
func (s *FileParameterUploadServiceServer) StartFileUpload(
	ctx context.Context,
	req *connect.Request[apiv1.StartFileUploadRequest],
) (*connect.Response[apiv1.StartFileUploadResponse], error) {
	msg := req.Msg
	if msg.GetUploadTokenId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("upload_token_id is required"))
	}
	if msg.GetFileName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("file_name is required"))
	}
	if msg.GetTotalSizeBytes() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("total_size_bytes must be positive"))
	}

	session, err := s.manager.StartUploadSession(msg.GetUploadTokenId(), msg.GetFileName(), msg.GetTotalSizeBytes())
	if err != nil {
		if errors.Is(err, upload.ErrUploadTokenNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		if errors.Is(err, chunkedupload.ErrInvalidTotalSize) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start upload session: %w", err))
	}

	res := &apiv1.StartFileUploadResponse{
		SessionToken:            proto.String(session.Token),
		SuggestedChunkSizeBytes: proto.Int64(s.manager.SuggestedChunkSize()),
	}
	return connect.NewResponse(res), nil
}

// UploadFileChunk receives and appends a single chunk of data for the active upload session.
func (s *FileParameterUploadServiceServer) UploadFileChunk(
	ctx context.Context,
	req *connect.Request[apiv1.UploadFileChunkRequest],
) (*connect.Response[apiv1.UploadFileChunkResponse], error) {
	msg := req.Msg
	if msg.GetSessionToken() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_token is required"))
	}

	totalReceived, err := s.manager.WriteChunk(msg.GetSessionToken(), msg.GetOffsetBytes(), msg.GetData())
	if err != nil {
		if errors.Is(err, chunkedupload.ErrSessionNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		if errors.Is(err, chunkedupload.ErrEmptyChunkData) || errors.Is(err, chunkedupload.ErrInvalidOffset) || errors.Is(err, chunkedupload.ErrChunkSizeTooLarge) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to write chunk: %w", err))
	}

	res := &apiv1.UploadFileChunkResponse{
		TotalReceivedBytes: proto.Int64(totalReceived),
	}
	return connect.NewResponse(res), nil
}

// CompleteFileUpload finalizes the upload session and prepares the uploaded file for task execution.
func (s *FileParameterUploadServiceServer) CompleteFileUpload(
	ctx context.Context,
	req *connect.Request[apiv1.CompleteFileUploadRequest],
) (*connect.Response[apiv1.CompleteFileUploadResponse], error) {
	msg := req.Msg
	if msg.GetSessionToken() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_token is required"))
	}

	fileSize, err := s.manager.CompleteUploadSession(msg.GetSessionToken())
	if err != nil {
		if errors.Is(err, chunkedupload.ErrSessionNotFound) || errors.Is(err, upload.ErrUploadTokenNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to complete upload: %w", err))
	}

	res := &apiv1.CompleteFileUploadResponse{
		FileSizeBytes: proto.Int64(fileSize),
	}
	return connect.NewResponse(res), nil
}

// AbortFileUpload cancels the upload session and deletes temporary chunks.
func (s *FileParameterUploadServiceServer) AbortFileUpload(
	ctx context.Context,
	req *connect.Request[apiv1.AbortFileUploadRequest],
) (*connect.Response[apiv1.AbortFileUploadResponse], error) {
	msg := req.Msg
	if msg.GetSessionToken() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_token is required"))
	}

	err := s.manager.AbortUploadSession(msg.GetSessionToken())
	if err != nil {
		if errors.Is(err, chunkedupload.ErrSessionNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to abort upload: %w", err))
	}

	res := &apiv1.AbortFileUploadResponse{
		Aborted: proto.Bool(true),
	}
	return connect.NewResponse(res), nil
}
