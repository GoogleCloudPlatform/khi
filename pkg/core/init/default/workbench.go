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
