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
