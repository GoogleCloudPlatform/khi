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
	"syscall"

	"github.com/GoogleCloudPlatform/khi/pkg/common/errorreport"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/lifecycle"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/server"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/gin-gonic/gin"

	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"

	_ "github.com/GoogleCloudPlatform/khi/pkg/core/init/default"
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

func handleTerminateSignal(exitCh chan<- int) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	s := <-sig
	lifecycle.Default.NotifyTerminate(s)

	exitCh <- 128 + int(s.(syscall.Signal))
}

func main() {
	// main() shouldn't have the other line other than this line, for os.Exit(N) to prevent calling `defer errorreport.CheckAndReportPanic`
	os.Exit(run())
}

func run() int {
	defer errorreport.CheckAndReportPanic()
	defer func() {
		err := coreinit.CallInitExtension(func(e coreinit.InitExtension) error {
			return e.BeforeTerminate()
		})
		if err != nil {
			slog.Error("Calling termination handler on InitExtension failed", "error", err)
		}
	}()

	err := coreinit.CallInitExtension(func(e coreinit.InitExtension) error {
		return e.BeforeAll()
	})
	if err != nil {
		slog.Error("Initializing KHI failed at BeforeAll() step", "error", err)
		return 1
	}

	err = coreinit.CallInitExtension(func(e coreinit.InitExtension) error {
		return e.ConfigureParameterStore()
	})
	if err != nil {
		slog.Error("Initializing KHI failed at ConfigureParameterStore() step", "error", err)
		return 1
	}

	err = parameters.Parse()
	if err != nil {
		slog.Error("Initializing KHI failed at parameters.Parse()", "error", err)
		return 1
	}

	err = coreinit.CallInitExtension(func(e coreinit.InitExtension) error {
		return e.AfterParsingParameters()
	})
	if err != nil {
		slog.Error("Initializing KHI failed at AfterParsingParameters() step", "error", err)
		return 1
	}

	ioconfig, err := inspectioncore_contract.NewIOConfigFromParameter(parameters.Common)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to construct the IOConfig from parameter\n%v", err))
		return 1
	}
	inspectionServer, err := coreinspection.NewServer(ioconfig)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to construct the inspection server due to unexpected error\n%v", err))
	}
	err = coreinit.CallInitExtension(func(e coreinit.InitExtension) error {
		return e.ConfigureInspectionTaskServer(inspectionServer)
	})
	if err != nil {
		slog.Error("Initializing KHI failed at ConfigureInspectionTaskServer() step", "error", err)
		return 1
	}

	lifecycle.Default.NotifyInit()

	// Channel to receive exit codes from concurrent goroutines
	exitCh := make(chan int, 1)
	// Start signal handler
	go handleTerminateSignal(exitCh)

	if !*parameters.Job.JobMode {
		slog.Info("Starting Kubernetes History Inspector server...")

		// Setting up options or parameters needed to instanciate gin.Engine
		serverMode := gin.ReleaseMode

		if *parameters.Debug.Verbose {
			serverMode = gin.DebugMode
		}

		uploadFileStoreFolder := "/tmp"

		if parameters.Common.UploadFileStoreFolder != nil {
			uploadFileStoreFolder = *parameters.Common.UploadFileStoreFolder
		}

		upload.DefaultUploadFileStore = upload.NewUploadFileStore(upload.NewLocalUploadFileStoreProvider(uploadFileStoreFolder))

		err = coreinit.CallInitExtension(func(e coreinit.InitExtension) error {
			return e.ConfigureKHIWebServerFactory(server.DefaultServerFactory)
		})
		if err != nil {
			slog.Error("Initializing KHI failed at ConfigureKHIWebServerFactory() step", "error", err)
			return 1
		}

		config := server.ServerConfig{
			ViewerMode:       *parameters.Server.ViewerMode,
			StaticFolderPath: *parameters.Server.FrontendAssetFolder,
			ResourceMonitor:  &server.ResourceMonitorImpl{},
			ServerBasePath:   *parameters.Server.BasePath,
			UploadFileStore:  upload.DefaultUploadFileStore,
		}
		engine, err := server.DefaultServerFactory.CreateInstance(serverMode)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to create a server instance\n%v", err))
			return 1
		}
		engine = server.CreateKHIServer(engine, inspectionServer, &config)

		go func() {
			err = engine.Run(fmt.Sprintf("%s:%d", *parameters.Server.Host, *parameters.Server.Port))
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to start server\n%s", err.Error()))
				exitCh <- 1
			} else {
				slog.Error("Hitting the unreachable code. Server terminated unexpectedly")
				exitCh <- 1
			}
		}()

		displayStartMessage(*parameters.Server.Host, *parameters.Server.Port)
	} else {
		slog.Info("Starting Kubernetes History Inspector as job mode...")

		go func() {
			queryParametersInJson := *parameters.Job.InspectionValues
			var values map[string]any
			err := json.Unmarshal([]byte(queryParametersInJson), &values)
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to parse an inspection value %s\n%s", queryParametersInJson, err.Error()))
				exitCh <- 1
				return
			}
			inspectionID, err := inspectionServer.CreateInspection(*parameters.Job.InspectionType)
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to create an inspection with type %s\n%s", *parameters.Job.InspectionType, err.Error()))
				exitCh <- 1
				return
			}

			features := strings.Split(*parameters.Job.InspectionFeatures, ",")
			t := inspectionServer.GetInspection(inspectionID)
			// When the features env has `ALL`, it enables every features being available
			if len(features) == 1 && strings.ToUpper(features[0]) == "ALL" {
				availableFeatures, err := t.FeatureList()
				if err != nil {
					slog.Error(fmt.Sprintf("Failed to obtain current feature list\n%s", err.Error()))
					exitCh <- 1

					return
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
				exitCh <- 1
				return
			}
			err = t.Run(context.Background(), &inspectioncore_contract.InspectionRequest{
				Values: values,
			})
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to run inspection task \n%s", err.Error()))
				exitCh <- 1
				return
			}
			<-t.Wait()
			result, err := t.Result()
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to get inspection result \n%s", err.Error()))
				exitCh <- 1
				return
			}
			reader, err := result.ResultStore.GetReader()
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to get inspection result reader \n%s", err.Error()))
				exitCh <- 1
				return
			}
			defer reader.Close()
			file, err := os.OpenFile(*parameters.Job.ExportDestination, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to open the destination file \n%s", err.Error()))
				exitCh <- 1
				return
			}
			_, err = io.Copy(file, reader)
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to flush to the destination file \n%s", err.Error()))
				exitCh <- 1
				return
			}
		}()
	}
	return <-exitCh
}
