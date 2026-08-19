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

		srv := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", *serverParams.Host, *serverParams.Port),
			Handler: engine,
		}

		ctx.OnRun(func(runCtx context.Context) error {
			slog.Info("Starting Kubernetes History Inspector server...")
			go func() {
				if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					slog.Error(fmt.Sprintf("Failed to start server: %v", err))
				}
			}()
			displayStartMessage(*serverParams.Host, *serverParams.Port, debugParams.NoColor != nil && *debugParams.NoColor)
			return nil
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
