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

	"connectrpc.com/connect"
	v1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"google.golang.org/protobuf/proto"
)

// CELValidationServer implements the CELValidationServiceHandler interface.
type CELValidationServer struct{}

var _ apiv1connect.CELValidationServiceHandler = (*CELValidationServer)(nil)

// NewCELValidationServer creates a new instance of CELValidationServer.
func NewCELValidationServer() *CELValidationServer {
	return &CELValidationServer{}
}

// ValidateTimelineQuery validates a CEL timeline expression syntax and schema types.
func (s *CELValidationServer) ValidateTimelineQuery(
	ctx context.Context,
	req *connect.Request[v1.ValidateTimelineQueryRequest],
) (*connect.Response[v1.ValidateTimelineQueryResponse], error) {
	err := cel.ValidateTimelineQuery(req.Msg.GetQuery())
	if err != nil {
		return connect.NewResponse(&v1.ValidateTimelineQueryResponse{
			Valid:        proto.Bool(false),
			ErrorMessage: proto.String(err.Error()),
		}), nil
	}
	return connect.NewResponse(&v1.ValidateTimelineQueryResponse{
		Valid:        proto.Bool(true),
		ErrorMessage: proto.String(""),
	}), nil
}

// ValidateLogQuery validates a CEL log expression syntax and schema types.
func (s *CELValidationServer) ValidateLogQuery(
	ctx context.Context,
	req *connect.Request[v1.ValidateLogQueryRequest],
) (*connect.Response[v1.ValidateLogQueryResponse], error) {
	err := cel.ValidateLogQuery(req.Msg.GetQuery())
	if err != nil {
		return connect.NewResponse(&v1.ValidateLogQueryResponse{
			Valid:        proto.Bool(false),
			ErrorMessage: proto.String(err.Error()),
		}), nil
	}
	return connect.NewResponse(&v1.ValidateLogQueryResponse{
		Valid:        proto.Bool(true),
		ErrorMessage: proto.String(""),
	}), nil
}
