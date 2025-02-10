// Copyright 2024 Google LLC
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"sync"

	"cloud.google.com/go/profiler"
	"github.com/GoogleCloudPlatform/khi/pkg/common/errorreport"
	"github.com/GoogleCloudPlatform/khi/pkg/common/flag"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/common"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/logger"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/lifecycle"
	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/server"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api/accesstoken"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api/quotaproject"

	"log/slog"
)

var taskSetRegistrer []inspection.PrepareInspectionServerFunc

func init() {
	// Add the different parameter stores.
	parameters.AddStore(parameters.Help)
	parameters.AddStore(parameters.Common)
	parameters.AddStore(parameters.Server)
	parameters.AddStore(parameters.Job)
	parameters.AddStore(parameters.Auth)
	parameters.AddStore(parameters.Debug)

	// Register inspection server preparation functions.
	taskSetRegistrer = []inspection.PrepareInspectionServerFunc{
		common.PrepareInspectionServer,
		gcp.PrepareInspectionServer,
	}
}

// fatal logs the error and exits if err is not nil.
func fatal(err error, msg string) {
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %v", msg, err))
		os.Exit(1)
	}
}

// displayStartMessage prints the server start message with optional ANSI color.
func displayStartMessage(host string, port int) {
	bold, green, cyan, reset := "\033[1m", "\033[32m", "\033[36m", "\033[0m"
	if parameters.Debug.NoColor != nil && *parameters.Debug.NoColor {
		bold, green, cyan, reset = "", "", "", ""
	}
	hostInHintText := host
	if host == "0.0.0.0" || host == "127.0.0.1" {
		hostInHintText = "localhost"
	}
	fmt.Printf("%s%s%s Starting KHI server with listening %s:%d%s", reset, bold, green, host, port, reset)
	if hostInHintText == "localhost" {
		fmt.Printf(`
%sFor Cloud Shell users:
	Click this address >> %shttp://%s:%d%s%s << Click this address

%s(For users of the other environments: Access %shttp://%s:%d%s%s with your browser. Consider SSH port-forwarding when you run KHI over SSH.)
%s`,
			reset, cyan, hostInHintText, port, reset, bold,
			reset, cyan, hostInHintText, port, reset, bold, reset)
	}
}

func main() {
	// Recover from panics and report errors.
	defer errorreport.CheckAndReportPanic()
	logger.InitGlobalKHILogger()

	// Parse command-line parameters.
	err := parameters.Parse()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
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
			slog.Error(fmt.Sprintf("Failed to start profiler: %v", err))
		} else {
			slog.Info("Cloud Profiler is enabled")
		}
	}

	lifecycle.Default.NotifyInit()
	slog.Info("Initializing Kubernetes History Inspector...")

	k8s.GenerateDefaultMergeConfig()
	if *parameters.Auth.QuotaProjectID != "" {
		api.DefaultGCPClientFactory.RegisterHeaderProvider(quotaproject.NewHeaderProvider(*parameters.Auth.QuotaProjectID))
	}

	inspectionServer, err := inspection.NewServer()
	fatal(err, "Failed to construct the inspection server")

	// In non-viewer mode, initialize the inspection tasks.
	if !*parameters.Server.ViewerMode {
		for i, registerFunc := range taskSetRegistrer {
			err = registerFunc(inspectionServer)
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to initialize taskSetRegistrer(#%d): %v", i, err))
			}
		}
	}

	// Use a context that cancels when a termination signal is received.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start in server mode.
	if !*parameters.Job.JobMode {
		slog.Info("Starting Kubernetes History Inspector server...")

		config := server.ServerConfig{
			ViewerMode:       *parameters.Server.ViewerMode,
			StaticFolderPath: *parameters.Server.FrontendAssetFolder,
			ResourceMonitor:  &server.ResourceMonitorImpl{},
			ServerBasePath:   *parameters.Server.BasePath,
		}
		engine := server.CreateKHIServer(inspectionServer, &config)

		if parameters.Auth.OAuthEnabled() {
			err := accesstoken.DefaultOAuthTokenResolver.SetServer(engine)
			fatal(err, "Failed to register the web server to OAuth Token resolver")
		}

		// Start the server in a goroutine and capture any errors.
		errCh := make(chan error, 1)
		go func() {
			errCh <- engine.Run(fmt.Sprintf("%s:%d", *parameters.Server.Host, *parameters.Server.Port))
		}()
		displayStartMessage(*parameters.Server.Host, *parameters.Server.Port)

		// Wait until a termination signal is received or the server errors.
		select {
		case <-ctx.Done():
			slog.Info("Termination signal received. Shutting down server.")
			// If your server supports graceful shutdown, call it here.
			os.Exit(0)
		case err := <-errCh:
			if err != nil {
				slog.Error(fmt.Sprintf("Server error: %v", err))
				os.Exit(1)
			}
		}
	} else {
		// Job mode
		slog.Info("Starting Kubernetes History Inspector in job mode...")

		queryParametersInJson := *parameters.Job.InspectionValues
		var values map[string]any
		err := json.Unmarshal([]byte(queryParametersInJson), &values)
		fatal(err, fmt.Sprintf("Failed to parse inspection value %s", queryParametersInJson))

		taskID, err := inspectionServer.CreateInspection(*parameters.Job.InspectionType)
		fatal(err, fmt.Sprintf("Failed to create an inspection with type %s", *parameters.Job.InspectionType))

		features := strings.Split(*parameters.Job.InspectionFeatures, ",")
		t := inspectionServer.GetTask(taskID)
		// If "ALL" is specified, enable every available feature.
		if len(features) == 1 && strings.ToUpper(features[0]) == "ALL" {
			availableFeatures, err := t.FeatureList()
			fatal(err, "Failed to obtain current feature list")
			features = make([]string, len(availableFeatures))
			for i, af := range availableFeatures {
				features[i] = af.Id
			}
		}
		err = t.SetFeatureList(features)
		fatal(err, fmt.Sprintf("Failed to set features %v", features))

		// Run the inspection task with the provided values.
		err = t.Run(ctx, &task.InspectionRequest{Values: values})
		fatal(err, "Failed to run inspection task")

		// Wait for the task to complete.
		<-t.Wait()
		result, err := t.Result()
		fatal(err, "Failed to get inspection result")

		reader, err := result.ResultStore.GetReader()
		fatal(err, "Failed to get inspection result reader")
		// Ensure that the reader is closed if it implements io.Closer.
		defer func() {
			if closer, ok := reader.(io.Closer); ok {
				closer.Close()
			}
		}()

		file, err := os.OpenFile(*parameters.Job.ExportDestination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		fatal(err, "Failed to open the destination file")
		defer file.Close()

		_, err = io.Copy(file, reader)
		fatal(err, "Failed to flush to the destination file")
		os.Exit(0)
	}
}
