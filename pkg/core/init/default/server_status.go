package defaultinit

import (
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	apiv1impl "github.com/GoogleCloudPlatform/khi/pkg/server/api/v1"
)

// InitializerIDServerStatusService identifies the Initializer that mounts the ServerStatusService.
const InitializerIDServerStatusService coreinit.InitializerID = "khi.default/server-status-service"

// ServerStatusServiceInitializer mounts the ServerStatusService Connect-RPC handler onto the Gin router.
var ServerStatusServiceInitializer = &coreinit.Initializer{
	ID: InitializerIDServerStatusService,
	Dependencies: []coreinit.InitializerID{
		InitializerIDGinServer,
	},
	Before: []coreinit.InitializerID{
		InitializerIDServerRunner,
	},
	Init: func(ctx *coreinit.InitContext) error {
		jobParams := coreinit.MustGet(ctx, JobParametersKey)
		if *jobParams.JobMode {
			return nil
		}
		router := coreinit.MustGet(ctx, GinRouterKey)
		basePath := coreinit.MustGet(ctx, BasePathKey)
		serverConfig := coreinit.MustGet(ctx, ServerConfigKey)

		serverStatusPath, serverStatusHandler := apiv1connect.NewServerStatusServiceHandler(
			apiv1impl.NewServerStatusServiceServer(serverConfig.ResourceMonitor),
		)
		coreinit.RegisterConnectServiceHandler(router, basePath, serverStatusPath, serverStatusHandler)
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(ServerStatusServiceInitializer)
}
