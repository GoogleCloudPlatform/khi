package defaultinit

import (
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/oauth"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/options"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

// InitializerIDOAuth initializes OAuth authentication handlers if enabled.
const InitializerIDOAuth coreinit.InitializerID = "khi.default/oauth"

// OAuthInitializer sets up OAuth server endpoints and attaches credentials to the inspection task server.
var OAuthInitializer = &coreinit.Initializer{
	ID: InitializerIDOAuth,
	Dependencies: []coreinit.InitializerID{
		InitializerIDGinServer,
		InitializerIDInspectionTaskServer,
		InitializerIDParameterParse,
	},
	Before: []coreinit.InitializerID{
		InitializerIDServerRunner,
	},
	Init: func(ctx *coreinit.InitContext) error {
		authParams := coreinit.MustGet(ctx, AuthParametersKey)
		if !authParams.OAuthEnabled() {
			return nil
		}
		engine := coreinit.MustGet(ctx, GinEngineKey)
		taskServer := coreinit.MustGet(ctx, InspectionTaskServerKey)

		oauthServer := oauth.NewOAuthServer(engine, authParams.GetOAuthConfig(), *authParams.OAuthRedirectTargetServingPath, *authParams.OAuthStateSuffix)
		taskServer.AddRunContextOption(
			coreinspection.RunContextOptionArrayElementFromValue(googlecloudcommon_contract.APIClientFactoryOptionsContextKey, options.OAuth(oauthServer)),
		)
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(OAuthInitializer)
}
