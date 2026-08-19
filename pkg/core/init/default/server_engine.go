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
	"github.com/gin-gonic/gin"
)

var (
	// GinEngineKey stores the gin.Engine instance.
	GinEngineKey = typedmap.NewTypedKey[*gin.Engine]("khi.google.com/init/gin-engine")
)

// InitializerIDGinEngine creates the Gin engine and attaches global middlewares.
const InitializerIDGinEngine coreinit.InitializerID = "khi.default/gin-engine"

// GinEngineInitializer creates the gin.Engine instance and sets up base global middlewares.
var GinEngineInitializer = &coreinit.Initializer{
	ID: InitializerIDGinEngine,
	Dependencies: []coreinit.InitializerID{
		InitializerIDParameterParse,
	},
	Before: []coreinit.InitializerID{
		InitializerIDGinServer,
	},
	Init: func(ctx *coreinit.InitContext) error {
		jobParams := coreinit.MustGet(ctx, JobParametersKey)
		if *jobParams.JobMode {
			return nil
		}
		debugParams := coreinit.MustGet(ctx, DebugParametersKey)

		serverMode := gin.ReleaseMode
		if *debugParams.Verbose {
			serverMode = gin.DebugMode
		}
		gin.SetMode(serverMode)

		engine := gin.New()
		engine.Use(gin.Recovery())
		if *debugParams.Verbose {
			engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
				SkipPaths: []string{"/api/v3/inspection", "/api/v3/popup"},
			}))
		}

		coreinit.Set(ctx, GinEngineKey, engine)
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(GinEngineInitializer)
}
