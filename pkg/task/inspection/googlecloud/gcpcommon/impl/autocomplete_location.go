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
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// autocompleteLocationTask is a task that provides a list of available locations for autocomplete.
// This serves as the default fallback implementation for generic Google Cloud environments.
// When an inspection type specific implementation is available (e.g. Kubernetes cluster or Cloud Composer),
// this implementation is shadowed by the higher-priority task.
var autocompleteLocationTask = inspectiontaskbase.DefineCachedTask(gcpcommon.AutocompleteLocationTaskID,
	func(b *coretask.Binder) inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[string]] {
		projectID := coretask.Use(b, gcpcommon.InputProjectIdTaskID.Ref()) // for API restriction
		locationFetcher := coretask.Use(b, gcpcommon.LocationFetcherTaskID.Ref())
		return inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[string]]{
			Scope: inspectiontaskbase.CacheScopeGlobal,
			InputDigest: func(ctx context.Context) string {
				return fmt.Sprintf("location-%s", projectID.Get(ctx))
			},
			Compute: func(ctx context.Context) (*inspectioncore.AutocompleteResult[string], error) {
				pid := projectID.Get(ctx)
				if pid == "" {
					return &inspectioncore.AutocompleteResult[string]{Values: []string{}}, nil
				}
				regions, err := locationFetcher.Get(ctx).FetchRegions(ctx, pid)
				if err != nil {
					return &inspectioncore.AutocompleteResult[string]{Values: []string{}}, nil
				}
				return &inspectioncore.AutocompleteResult[string]{Values: regions}, nil
			},
		}
	},
)
