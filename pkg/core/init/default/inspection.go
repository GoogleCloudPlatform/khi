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
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/legacy"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/options"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/tracing"
	"github.com/GoogleCloudPlatform/khi/pkg/generated"
	"github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6/style"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"go.opentelemetry.io/otel"
)

var (
	// IOConfigKey stores the IOConfig instance.
	IOConfigKey = typedmap.NewTypedKey[*inspectioncore_contract.IOConfig]("khi.google.com/init/io-config")

	// InspectionTaskServerKey stores the InspectionTaskServer instance.
	InspectionTaskServerKey = typedmap.NewTypedKey[*coreinspection.InspectionTaskServer]("khi.google.com/init/inspection-task-server")
)

// InitializerIDInspectionTaskServer creates InspectionTaskServer and registers tasks.
const InitializerIDInspectionTaskServer coreinit.InitializerID = "khi.default/inspection-task-server"

// InspectionTaskServerInitializer constructs the inspection task server and registers all tasks.
var InspectionTaskServerInitializer = &coreinit.Initializer{
	ID: InitializerIDInspectionTaskServer,
	Dependencies: []coreinit.InitializerID{
		InitializerIDParameterParse,
		InitializerIDDebugFeatures,
		InitializerIDK8sMergeConfig,
	},
	Init: func(ctx *coreinit.InitContext) error {
		commonParams := coreinit.MustGet(ctx, CommonParametersKey)
		authParams := coreinit.MustGet(ctx, AuthParametersKey)
		debugParams := coreinit.MustGet(ctx, DebugParametersKey)

		ioconfig, err := inspectioncore_contract.NewIOConfigFromParameter(commonParams)
		if err != nil {
			return fmt.Errorf("failed to construct IOConfig: %w", err)
		}
		inspectionServer, err := coreinspection.NewServer(ioconfig)
		if err != nil {
			return fmt.Errorf("failed to construct inspection server: %w", err)
		}

		if err := generated.RegisterAllInspectionTasks(inspectionServer); err != nil {
			return err
		}
		style.LockRegistry()
		inspectionServer.AddRunContextOption(coreinspection.RunContextOptionArrayElementFromValue(
			googlecloudcommon_contract.APIClientFactoryOptionsContextKey,
			options.GRPCConnPool(*authParams.GRPCConnPool),
		))
		if *authParams.QuotaProjectID != "" {
			inspectionServer.AddRunContextOption(coreinspection.RunContextOptionArrayElementFromValue(
				googlecloudcommon_contract.APIClientFactoryOptionsContextKey,
				options.QuotaProject(*authParams.QuotaProjectID),
			))
		}
		if *authParams.AccessToken != "" {
			inspectionServer.AddRunContextOption(coreinspection.RunContextOptionArrayElementFromValue(
				googlecloudcommon_contract.APIClientFactoryOptionsContextKey,
				options.TokenSource(legacy.NewRawTokenTokenSource(*authParams.AccessToken)),
			))
		}
		if *debugParams.CloudTrace {
			inspectionServer.AddInspectionInterceptor(tracing.NewInspectionTraceInterceptor(otel.Tracer("khi")))
		}

		coreinit.Set(ctx, IOConfigKey, ioconfig)
		coreinit.Set(ctx, InspectionTaskServerKey, inspectionServer)
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(InspectionTaskServerInitializer)
}
