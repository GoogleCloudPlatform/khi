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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	apiv1impl "github.com/GoogleCloudPlatform/khi/pkg/server/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
)

var (
	// WorkbenchManagerKey stores the WorkbenchManager instance in the init context.
	WorkbenchManagerKey = typedmap.NewTypedKey[*workbench.WorkbenchManager]("khi.google.com/init/workbench-manager")
)

// InitializerIDWorkbenchService identifies the Initializer that mounts the WorkbenchService.
const InitializerIDWorkbenchService coreinit.InitializerID = "khi.default/workbench-service"

// WorkbenchServiceInitializer mounts the WorkbenchService Connect-RPC handler onto the Gin router.
var WorkbenchServiceInitializer = &coreinit.Initializer{
	ID: InitializerIDWorkbenchService,
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

		workbenchManager := workbench.NewWorkbenchManager(inspectionServer, 60*time.Second, 15*time.Second)
		coreinit.Set(ctx, WorkbenchManagerKey, workbenchManager)

		workbenchPath, workbenchHandler := apiv1connect.NewWorkbenchServiceHandler(apiv1impl.NewWorkbenchServiceServer(workbenchManager))
		coreinit.RegisterConnectServiceHandler(router, basePath, workbenchPath, workbenchHandler)
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(WorkbenchServiceInitializer)
}
