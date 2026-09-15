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

package gcpcommon_impl

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
)

// LocationFetcherTask is the task to inject the reference to LocationFetcher.
// This is primarily utilized by the default fallback AutocompleteLocationTask.
var LocationFetcherTask = coretask.NewTask(gcpcommon.LocationFetcherTaskID, []coretask.Dependency{
	gcpcommon.InputProjectIdTaskID.Ref(),
	gcpcommon.APIClientFactoryTaskID.Ref(),
	gcpcommon.APIClientCallOptionsInjectorTaskID.Ref(),
}, func(ctx context.Context) (gcpcommon.LocationFetcher, error) {
	clientFactory := coretask.GetTaskResult(ctx, gcpcommon.APIClientFactoryTaskID.Ref())
	callOptionInjector := coretask.GetTaskResult(ctx, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())
	projectID := coretask.GetTaskResult(ctx, gcpcommon.InputProjectIdTaskID.Ref())
	regionClient, err := clientFactory.RegionsClient(ctx, googlecloud.Project(projectID))
	if err != nil {
		return nil, err
	}
	return gcpcommon.NewLocationFetcher(regionClient, callOptionInjector), nil
})

// LoggingFetcherTask is a task to inject the reference to LogFetcher.
var LoggingFetcherTask = coretask.NewTask(gcpcommon.LoggingFetcherTaskID, []coretask.Dependency{
	gcpcommon.APIClientFactoryTaskID.Ref(),
	gcpcommon.APIClientCallOptionsInjectorTaskID.Ref(),
}, func(ctx context.Context) (gcpcommon.LogFetcher, error) {
	clientFactory := coretask.GetTaskResult(ctx, gcpcommon.APIClientFactoryTaskID.Ref())
	callOptionInjector := coretask.GetTaskResult(ctx, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())
	return gcpcommon.NewLogFetcher(clientFactory, callOptionInjector, 1000), nil
})
