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
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
)

var (
	// CommonParametersKey stores parsed CommonParameters.
	CommonParametersKey = typedmap.NewTypedKey[*parameters.CommonParameters]("khi.google.com/init/params/common")

	// ServerParametersKey stores parsed ServerParameters.
	ServerParametersKey = typedmap.NewTypedKey[*parameters.ServerParameters]("khi.google.com/init/params/server")

	// JobParametersKey stores parsed JobParameters.
	JobParametersKey = typedmap.NewTypedKey[*parameters.JobParameters]("khi.google.com/init/params/job")

	// AuthParametersKey stores parsed AuthParameters.
	AuthParametersKey = typedmap.NewTypedKey[*parameters.AuthParameters]("khi.google.com/init/params/auth")

	// DebugParametersKey stores parsed DebugParameters.
	DebugParametersKey = typedmap.NewTypedKey[*parameters.DebugParameters]("khi.google.com/init/params/debug")

	// RateLimitParametersKey stores parsed RateLimitParameters.
	RateLimitParametersKey = typedmap.NewTypedKey[*parameters.RateLimitParameters]("khi.google.com/init/params/ratelimit")
)

const (
	// InitializerIDParameterStores registers parameter stores.
	InitializerIDParameterStores coreinit.InitializerID = "khi.default/parameter-stores"

	// InitializerIDParameterParse parses parameters and injects them into InitContext.
	InitializerIDParameterParse coreinit.InitializerID = "khi.default/parameter-parse"
)

// ParameterStoresInitializer registers the standard KHI parameter stores.
var ParameterStoresInitializer = &coreinit.Initializer{
	ID:     InitializerIDParameterStores,
	Before: []coreinit.InitializerID{InitializerIDParameterParse},
	Init: func(ctx *coreinit.InitContext) error {
		parameters.AddStore(parameters.Help)
		parameters.AddStore(parameters.Common)
		parameters.AddStore(parameters.Server)
		parameters.AddStore(parameters.Job)
		parameters.AddStore(parameters.Auth)
		parameters.AddStore(parameters.Debug)
		parameters.AddStore(parameters.RateLimit)
		return nil
	},
}

// ParameterParseInitializer parses CLI flags and injects parameter stores into InitContext.
var ParameterParseInitializer = &coreinit.Initializer{
	ID: InitializerIDParameterParse,
	Dependencies: []coreinit.InitializerID{
		InitializerIDLogger,
	},
	Init: func(ctx *coreinit.InitContext) error {
		if err := parameters.Parse(); err != nil {
			return err
		}
		coreinit.Set(ctx, CommonParametersKey, parameters.Common)
		coreinit.Set(ctx, ServerParametersKey, parameters.Server)
		coreinit.Set(ctx, JobParametersKey, parameters.Job)
		coreinit.Set(ctx, AuthParametersKey, parameters.Auth)
		coreinit.Set(ctx, DebugParametersKey, parameters.Debug)
		coreinit.Set(ctx, RateLimitParametersKey, parameters.RateLimit)
		return nil
	},
}

// RegisterParameterStore registers a parameter store that will be added before parameters.Parse is executed.
func RegisterParameterStore(id coreinit.InitializerID, store parameters.ParameterStore) {
	coreinit.RegisterInitializer(&coreinit.Initializer{
		ID:     id,
		Before: []coreinit.InitializerID{InitializerIDParameterParse},
		Init: func(ctx *coreinit.InitContext) error {
			parameters.AddStore(store)
			return nil
		},
	})
}

func init() {
	coreinit.RegisterInitializer(ParameterStoresInitializer)
	coreinit.RegisterInitializer(ParameterParseInitializer)
}
