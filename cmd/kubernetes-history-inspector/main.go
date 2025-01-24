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
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

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

	"cloud.google.com/go/profiler"
)

func displayStartMessage(host string, port int) {
	var (
		bold  = "\033[1m"
		green = "\033[32m"
		cyan  = "\033[36m"
		reset = "\033[0m"
	)
	if parameters.Debug.NoColor != nil && *parameters.Debug.NoColor {
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

var taskSetRegistrer []inspection.PrepareInspectionServerFunc = make([]inspection.PrepareInspectionServerFunc, 0)

func init() {
	parameters.AddStore(parameters.Help)
	parameters.AddStore(parameters.Common)
	parameters.AddStore(parameters.Server)
	parameters.AddStore(parameters.Job)
	parameters.AddStore(parameters.Auth)
	parameters.AddStore(parameters.Debug)

	taskSetRegistrer = append(taskSetRegistrer, common.PrepareInspectionServer)
	taskSetRegistrer = append(taskSetRegistrer, gcp.PrepareInspectionServer)
}

func handleTerminateSignal(terminateErrorCode int) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	s := <-sig
	lifecycle.Default.NotifyTerminate(s)
	os.Exit(terminateErrorCode)
}

func main() {
	defer errorreport.CheckAndReportPanic()
	logger.InitGlobalKHILogger()
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
			slog.Error(fmt.Sprintf("Failed to start profiler\n%s", err.Error()))
		}
		slog.Info("Cloud Profiler is enabled")
	}
	lifecycle.Default.NotifyInit()
	slog.Info("Initializing Kubernetes History Inspector...")

	k8s.GenerateDefaultMergeConfig()
	if *parameters.Auth.QuotaProjectID != "" {
		api.DefaultGCPClientFactory.RegisterHeaderProvider(quotaproject.NewHeaderProvider(*parameters.Auth.QuotaProjectID))
	}
	inspectionServer, err := inspection.NewServer()
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to construct the inspection server due to unexpected error\n%v", err))
	}

	if !*parameters.Server.ViewerMode {
		for i, taskSetRegistrer := range taskSetRegistrer {
			err = taskSetRegistrer(inspectionServer)
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to call initialize calls for taskSetRegistrer(#%d)\n%v", i, err))
			}
		}
	}

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
			if err != nil {
				slog.Error("failed to register the web server to OAuth Token resolver")
				os.Exit(1)
			}
		}

		grp := sync.WaitGroup{}
		grp.Add(1)
		go func() {
			err = engine.Run(fmt.Sprintf("%s:%d", *parameters.Server.Host, *parameters.Server.Port))
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to start server\n%s", err.Error()))
				os.Exit(1)
			}
			grp.Done()
		}()
		go handleTerminateSignal(0)
		displayStartMessage(*parameters.Server.Host, *parameters.Server.Port)
		grp.Wait()
	} else {
		slog.Info("Starting Kubernetes History Inspector as job mode...")
		go handleTerminateSignal(1)
		queryParametersInJson := *parameters.Job.InspectionValues
		var values map[string]any
		err := json.Unmarshal([]byte(queryParametersInJson), &values)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to parse an inspection value %s\n%s", queryParametersInJson, err.Error()))
			os.Exit(1)
		}
		taskId, err := inspectionServer.CreateInspection(*parameters.Job.InspectionType)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to create an inspection with type %s\n%s", *parameters.Job.InspectionType, err.Error()))
			os.Exit(1)
		}

		features := strings.Split(*parameters.Job.InspectionFeatures, ",")
		t := inspectionServer.GetTask(taskId)
		// When the features env has `ALL`, it enables every features being available
		if len(features) == 1 && strings.ToUpper(features[0]) == "ALL" {
			availableFeatures, err := t.FeatureList()
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to obtain current feature list\n%s", err.Error()))
				os.Exit(1)
			}
			allFeatures := []string{}
			for _, af := range availableFeatures {
				allFeatures = append(allFeatures, af.Id)
			}
			features = allFeatures
		}
		err = t.SetFeatureList(features)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to set features %v\n%s", features, err.Error()))
			os.Exit(1)
		}
		err = t.Run(context.Background(), &task.InspectionRequest{
			Values: values,
		})
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to run inspection task \n%s", err.Error()))
			os.Exit(1)
		}
		<-t.Wait()
		result, err := t.Result()
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to get inspection result \n%s", err.Error()))
			os.Exit(1)
		}
		reader, err := result.ResultStore.GetReader()
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to get inspection result reader \n%s", err.Error()))
			os.Exit(1)
		}
		file, err := os.OpenFile(*parameters.Job.ExportDestination, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to open the destination file \n%s", err.Error()))
			os.Exit(1)
		}
		_, err = io.Copy(file, reader)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to flush to the destination file \n%s", err.Error()))
			os.Exit(1)
		}
		os.Exit(0)
	}
}
