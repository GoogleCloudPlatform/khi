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

// locationFetcherTask is the task to inject the reference to LocationFetcher.
// This is primarily utilized by the default fallback autocompleteLocationTask.
var locationFetcherTask = coretask.Define(gcpcommon.LocationFetcherTaskID, func(b *coretask.Binder) func(ctx context.Context) (gcpcommon.LocationFetcher, error) {
	projectID := coretask.Use(b, gcpcommon.InputProjectIdTaskID.Ref())
	clientFactory := coretask.Use(b, gcpcommon.APIClientFactoryTaskID.Ref())
	callOptionInjector := coretask.Use(b, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())
	return func(ctx context.Context) (gcpcommon.LocationFetcher, error) {
		factory := clientFactory.Get(ctx)
		injector := callOptionInjector.Get(ctx)
		pid := projectID.Get(ctx)
		regionClient, err := factory.RegionsClient(ctx, googlecloud.Project(pid))
		if err != nil {
			return nil, err
		}
		return gcpcommon.NewLocationFetcher(regionClient, injector), nil
	}
})

// loggingFetcherTask is a task to inject the reference to LogFetcher.
var loggingFetcherTask = coretask.Define(gcpcommon.LoggingFetcherTaskID, func(b *coretask.Binder) func(ctx context.Context) (gcpcommon.LogFetcher, error) {
	clientFactory := coretask.Use(b, gcpcommon.APIClientFactoryTaskID.Ref())
	callOptionInjector := coretask.Use(b, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())
	return func(ctx context.Context) (gcpcommon.LogFetcher, error) {
		factory := clientFactory.Get(ctx)
		injector := callOptionInjector.Get(ctx)
		return gcpcommon.NewLogFetcher(factory, injector, 1000), nil
	}
})
