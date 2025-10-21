// Copyright 2025 Google LLC
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
	"context"
	"log/slog"

	"cloud.google.com/go/profiler"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/legacy"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/oauth"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/options"
	"github.com/GoogleCloudPlatform/khi/pkg/common/flag"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	"github.com/GoogleCloudPlatform/khi/pkg/generated"
	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/server"
	"github.com/GoogleCloudPlatform/khi/pkg/server/option"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const DefaultInitExtensionOrder = 10000

func init() {
	coreinit.RegisterInitExtension(DefaultInitExtensionOrder, &DefaultInitExtension{})
}

type oauthServerOption struct {
	taskServer *coreinspection.InspectionTaskServer
}

// Apply implements option.Option.
func (o *oauthServerOption) Apply(engine *gin.Engine) error {
	oauthServer := oauth.NewOAuthServer(engine, parameters.Auth.GetOAuthConfig(), *parameters.Auth.OAuthRedirectTargetServingPath, *parameters.Auth.OAuthStateSuffix)
	o.taskServer.AddRunContextOption(
		coreinspection.RunContextOptionArrayElementFromValue(googlecloudcommon_contract.APIClientFactoryOptionsContextKey, options.OAuth(oauthServer)),
	)
	return nil
}

// ID implements option.Option.
func (o *oauthServerOption) ID() string {
	return "oauth-server"
}

// Order implements option.Option.
func (o *oauthServerOption) Order() int {
	return 10000
}

func newOAuthServerOption(taskServer *coreinspection.InspectionTaskServer) *oauthServerOption {
	return &oauthServerOption{
		taskServer: taskServer,
	}
}

var _ option.Option = (*oauthServerOption)(nil)

type DefaultInitExtension struct {
	taskServer *coreinspection.InspectionTaskServer
}

// BeforeAll implements coreinit.InitExtension.
func (d *DefaultInitExtension) BeforeAll() error {
	logger.InitGlobalKHILogger()
	slog.Info("Initializing Kubernetes History Inspector...")
	return nil
}

// ConfigureParameterStore implements coreinit.InitExtension.
func (d *DefaultInitExtension) ConfigureParameterStore() error {
	parameters.AddStore(parameters.Help)
	parameters.AddStore(parameters.Common)
	parameters.AddStore(parameters.Server)
	parameters.AddStore(parameters.Job)
	parameters.AddStore(parameters.Auth)
	parameters.AddStore(parameters.Debug)
	return nil
}

// AfterParsingParameters implements coreinit.InitExtension.
func (d *DefaultInitExtension) AfterParsingParameters() error {
	if *parameters.Debug.Verbose {
		flag.DumpAll(context.Background())
	}
	if *parameters.Debug.Profiler {
		cfg := profiler.Config{
			Service:        *parameters.Debug.ProfilerService,
			ProjectID:      *parameters.Debug.ProfilerProject,
			MutexProfiling: true,
		}
		if err := profiler.Start(cfg); err != nil {
			return err
		}
		slog.Info("Cloud Profiler is enabled")
	}
	k8s.GenerateDefaultMergeConfig()
	return nil
}

// ConfigureInspectionTaskServer implements coreinit.InitExtension.
func (d *DefaultInitExtension) ConfigureInspectionTaskServer(taskServer *coreinspection.InspectionTaskServer) error {
	d.taskServer = taskServer
	if !*parameters.Server.ViewerMode {
		err := generated.RegisterAllInspectionTasks(taskServer)
		if err != nil {
			return err
		}
	}
	if *parameters.Auth.QuotaProjectID != "" {
		taskServer.AddRunContextOption(coreinspection.RunContextOptionArrayElementFromValue(googlecloudcommon_contract.APIClientFactoryOptionsContextKey, options.QuotaProject(*parameters.Auth.QuotaProjectID)))
	}
	if *parameters.Auth.AccessToken != "" {
		taskServer.AddRunContextOption(coreinspection.RunContextOptionArrayElementFromValue(googlecloudcommon_contract.APIClientFactoryOptionsContextKey, options.TokenSource(legacy.NewRawTokenTokenSource(*parameters.Auth.AccessToken))))
	}
	return nil
}

// ConfigureKHIWebServerFactory implements coreinit.InitExtension.
func (d *DefaultInitExtension) ConfigureKHIWebServerFactory(serverFactory *server.ServerFactory) error {
	serverFactory.AddOptions(option.Required())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	serverFactory.AddOptions(option.CORS(corsConfig))
	if *parameters.Debug.Verbose {
		serverFactory.AddOptions(
			option.AccessLog("/api/v3/inspection", "/api/v3/popup"), // ignoreing noisy paths
		)
	}

	if parameters.Auth.OAuthEnabled() {
		serverFactory.AddOptions(newOAuthServerOption(d.taskServer))
	}
	return nil
}

// BeforeTerminate implements coreinit.InitExtension.
func (d *DefaultInitExtension) BeforeTerminate() error {
	return nil
}

var _ coreinit.InitExtension = (*DefaultInitExtension)(nil)
