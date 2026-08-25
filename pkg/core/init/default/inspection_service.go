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

package defaultinit

import (
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	serverapiv1 "github.com/GoogleCloudPlatform/khi/pkg/server/api/v1"
)

// InitializerIDInspectionService mounts Connect-RPC InspectionService onto Gin engine.
const InitializerIDInspectionService coreinit.InitializerID = "khi.default/inspection-service"

// InspectionServiceInitializer initializes and registers the InspectionService handlers.
var InspectionServiceInitializer = &coreinit.Initializer{
	ID: InitializerIDInspectionService,
	Dependencies: []coreinit.InitializerID{
		InitializerIDGinServer,
		InitializerIDInspectionTaskServer,
	},
	Before: []coreinit.InitializerID{
		InitializerIDServerRunner,
	},
	Init: func(ctx *coreinit.InitContext) error {
		jobParams := coreinit.MustGet(ctx, JobParametersKey)
		if *jobParams.JobMode {
			return nil
		}

		inspectionServer := coreinit.MustGet(ctx, InspectionTaskServerKey)
		router := coreinit.MustGet(ctx, GinRouterKey)
		basePath := coreinit.MustGet(ctx, BasePathKey)

		inspectionServiceServer := serverapiv1.NewInspectionServiceServer(inspectionServer)
		inspectionServicePath, inspectionServiceHandler := apiv1connect.NewInspectionServiceHandler(inspectionServiceServer)
		coreinit.RegisterConnectServiceHandler(router, basePath, inspectionServicePath, inspectionServiceHandler)

		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(InspectionServiceInitializer)
}
