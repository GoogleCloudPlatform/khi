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
	"log/slog"

	"cloud.google.com/go/profiler"
	"github.com/GoogleCloudPlatform/khi/pkg/common/constants"
	"github.com/GoogleCloudPlatform/khi/pkg/common/flag"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// InitializerIDDebugFeatures configures profiler and Cloud Trace.
const InitializerIDDebugFeatures coreinit.InitializerID = "khi.default/debug-features"

// DebugFeaturesInitializer initializes optional debug tooling (Profiler and Cloud Trace).
var DebugFeaturesInitializer = &coreinit.Initializer{
	ID: InitializerIDDebugFeatures,
	Dependencies: []coreinit.InitializerID{
		InitializerIDParameterParse,
	},
	Init: func(ctx *coreinit.InitContext) error {
		debugParams := coreinit.MustGet(ctx, DebugParametersKey)
		if *debugParams.Verbose {
			flag.DumpAll(ctx)
		}
		if *debugParams.Profiler {
			cfg := profiler.Config{
				Service:        *debugParams.ProfilerService,
				ProjectID:      *debugParams.ProfilerProject,
				MutexProfiling: true,
			}
			if err := profiler.Start(cfg); err != nil {
				return err
			}
			slog.Info("Cloud Profiler is enabled")
		}
		if *debugParams.CloudTrace {
			exporter, err := texporter.New(texporter.WithProjectID(*debugParams.CloudTraceProject))
			if err != nil {
				return err
			}
			tp := sdktrace.NewTracerProvider(
				sdktrace.WithBatcher(exporter),
				sdktrace.WithResource(resource.NewWithAttributes(
					semconv.SchemaURL,
					semconv.ServiceNameKey.String("khi"),
					semconv.ServiceVersionKey.String(constants.VERSION),
				)),
			)
			otel.SetTracerProvider(tp)
			slog.Info("Cloud Trace is enabled")
		}
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(DebugFeaturesInitializer)
}
