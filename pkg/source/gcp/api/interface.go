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

package api

import (
	"context"
)

type GCPClient interface {
	GetClusterNames(ctx context.Context, projectId string) ([]string, error)
	GetAnthosAWSClusterNames(ctx context.Context, projectId string) ([]string, error)
	GetAnthosAzureClusterNames(ctx context.Context, projectId string) ([]string, error)
	GetAnthosOnBaremetalClusterNames(ctx context.Context, projectId string) ([]string, error)
	GetAnthosOnVMWareClusterNames(ctx context.Context, projectId string) ([]string, error)
	GetComposerEnvironmentNames(ctx context.Context, projectId string, location string) ([]string, error)
	ListLogEntries(ctx context.Context, projectId string, filter string, logSink chan any) error
}

type RefreshableToken interface {
	Refresh() (string, error)
}
