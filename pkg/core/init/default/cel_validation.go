package defaultinit

import (
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	"github.com/GoogleCloudPlatform/khi/pkg/server"
	apiv1impl "github.com/GoogleCloudPlatform/khi/pkg/server/api/v1"
)

// InitializerIDCELValidationService identifies the Initializer that mounts the CELValidationService.
const InitializerIDCELValidationService coreinit.InitializerID = "khi.default/cel-validation-service"

// CELValidationServiceInitializer mounts the CELValidationService Connect-RPC handler onto the Gin router.
var CELValidationServiceInitializer = &coreinit.Initializer{
	ID: InitializerIDCELValidationService,
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

		celValidationPath, celValidationHandler := apiv1connect.NewCELValidationServiceHandler(apiv1impl.NewCELValidationServer())
		server.RegisterConnectServiceHandler(router, basePath, celValidationPath, celValidationHandler)
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(CELValidationServiceInitializer)
}
