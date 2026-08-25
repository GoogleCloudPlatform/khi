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
	"github.com/GoogleCloudPlatform/khi/pkg/server/importinspection"
)

// InitializerIDImportInspection mounts Connect-RPC ImportInspectionService onto Gin engine.
const InitializerIDImportInspection coreinit.InitializerID = "khi.default/import-inspection"

// ImportInspectionInitializer initializes and registers the ImportInspectionService handlers.
var ImportInspectionInitializer = &coreinit.Initializer{
	ID: InitializerIDImportInspection,
	Dependencies: []coreinit.InitializerID{
		InitializerIDGinServer,
		InitializerIDInspectionTaskServer,
		InitializerIDInspectionIndexManager,
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
		indexManager := coreinit.MustGet(ctx, InspectionIndexManagerKey)
		router := coreinit.MustGet(ctx, GinRouterKey)
		basePath := coreinit.MustGet(ctx, BasePathKey)

		importSessionManager := importinspection.NewImportSessionManager(inspectionServer, inspectionServer.IOConfig())
		importInspectionServer := serverapiv1.NewImportInspectionServiceServer(importSessionManager, indexManager)
		importInspectionPath, importInspectionHandler := apiv1connect.NewImportInspectionServiceHandler(importInspectionServer)
		coreinit.RegisterConnectServiceHandler(router, basePath, importInspectionPath, importInspectionHandler)

		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(ImportInspectionInitializer)
}
