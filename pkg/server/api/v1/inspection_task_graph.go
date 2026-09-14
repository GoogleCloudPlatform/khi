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
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
)

// InspectionTaskGraphServer implements the apiv1connect.InspectionTaskGraphServiceHandler interface.
type InspectionTaskGraphServer struct {
	inspectionServer *coreinspection.InspectionTaskServer
}

var _ apiv1connect.InspectionTaskGraphServiceHandler = (*InspectionTaskGraphServer)(nil)

// NewInspectionTaskGraphServer creates a new InspectionTaskGraphServer with the given InspectionTaskServer.
func NewInspectionTaskGraphServer(inspectionServer *coreinspection.InspectionTaskServer) *InspectionTaskGraphServer {
	return &InspectionTaskGraphServer{
		inspectionServer: inspectionServer,
	}
}

// GetInspectionTaskRegistry returns all registered tasks grouped by reference identifier and all inspection types.
func (s *InspectionTaskGraphServer) GetInspectionTaskRegistry(
	ctx context.Context,
	req *connect.Request[apiv1.GetInspectionTaskRegistryRequest],
) (*connect.Response[apiv1.GetInspectionTaskRegistryResponse], error) {
	resp, err := coreinspection.InspectRegistry(s.inspectionServer)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ResolveInspectionTaskGraph resolves task filtering against an inspection type and feature configuration,
// and returns step-by-step filtering results alongside the resolved execution DAG.
func (s *InspectionTaskGraphServer) ResolveInspectionTaskGraph(
	ctx context.Context,
	req *connect.Request[apiv1.ResolveInspectionTaskGraphRequest],
) (*connect.Response[apiv1.ResolveInspectionTaskGraphResponse], error) {
	if req.Msg.GetInspectionTypeId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("inspection_type_id must not be empty"))
	}

	resp, err := coreinspection.InspectResolution(
		s.inspectionServer,
		req.Msg.GetInspectionTypeId(),
		req.Msg.GetFeatureOverrides(),
	)
	if err != nil {
		if errors.Is(err, coreinspection.ErrInspectionTypeNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(resp), nil
}
