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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
)

// InitializerIDServerRunner starts the HTTP server listener and prints the banner.
const InitializerIDServerRunner coreinit.InitializerID = "khi.default/server-runner"

func displayStartMessage(host string, port int, noColor bool) {
	var (
		bold  = "\033[1m"
		green = "\033[32m"
		cyan  = "\033[36m"
		reset = "\033[0m"
	)
	if noColor {
		bold = ""
		green = ""
		cyan = ""
		reset = ""
	}
	hostInHintText := host
	if host == "0.0.0.0" || host == "127.0.0.1" {
		hostInHintText = "localhost"
	}
	fmt.Printf(`%[1]s%[2]s%[3]s Starting KHI server with listening %[4]s:%[5]d%[1]s`, reset, bold, green, host, port)
	if hostInHintText == "localhost" {
		fmt.Printf(`
%[4]s%[2]sFor Cloud Shell users:
	Click this address >> %[3]shttp://%[5]s:%[6]d%[1]s%[2]s%[4]s << Click this address

%[1]s%[4]s(For users of the other environments: Access %[3]shttp://%[5]s:%[6]d%[1]s%[4]s with your browser. Consider SSH port-forwarding when you run KHI over SSH.)
%[1]s`, reset, bold, green, cyan, hostInHintText, port)
	}
}

// ServerRunnerInitializer registers the runtime server listener and graceful shutdown hooks.
var ServerRunnerInitializer = &coreinit.Initializer{
	ID: InitializerIDServerRunner,
	Dependencies: []coreinit.InitializerID{
		InitializerIDGinServer,
	},
	Init: func(ctx *coreinit.InitContext) error {
		jobParams := coreinit.MustGet(ctx, JobParametersKey)
		if *jobParams.JobMode {
			return nil
		}
		serverParams := coreinit.MustGet(ctx, ServerParametersKey)
		debugParams := coreinit.MustGet(ctx, DebugParametersKey)
		engine := coreinit.MustGet(ctx, GinEngineKey)
		protocols := &http.Protocols{}
		protocols.SetHTTP1(true)
		protocols.SetUnencryptedHTTP2(true)

		srv := &http.Server{
			Addr:      fmt.Sprintf("%s:%d", *serverParams.Host, *serverParams.Port),
			Handler:   engine,
			Protocols: protocols,
		}

		ctx.OnRun(func(runCtx context.Context) error {
			slog.Info("Starting Kubernetes History Inspector server...")
			errCh := make(chan error, 1)
			go func() {
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					slog.Error(fmt.Sprintf("Failed to start server: %v", err))
					errCh <- err
				}
			}()
			displayStartMessage(*serverParams.Host, *serverParams.Port, debugParams.NoColor != nil && *debugParams.NoColor)

			select {
			case <-runCtx.Done():
				return nil
			case err := <-errCh:
				return err
			}
		})

		ctx.OnTerminate(func() error {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		})

		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(ServerRunnerInitializer)
}
