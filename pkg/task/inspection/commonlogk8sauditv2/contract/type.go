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

package commonlogk8sauditv2_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// ResourceChangeLog is the log with the resource change information.
type ResourceChangeLog struct {
	// Log is the log.
	Log *log.Log
	// ResourceBodyYAML is the YAML representation of the resource body.
	ResourceBodyYAML string
	// ResourceBodyReader is the reader for the resource body.
	ResourceBodyReader *structured.NodeReader
	// ResourceCreated is true if the resource is created.
	ResourceCreated bool
	// ResourceDeleted is true if the resource is deleted.
	ResourceDeleted bool
}

// ResourceChangeLogGroup is the group of the resource change logs.
type ResourceChangeLogGroup struct {
	// Group is the group name.
	Group string
	// Logs is the list of the resource change logs.
	Logs []*ResourceChangeLog
}

// ResourceChangeLogGroupMap is the map of the resource change log groups.
type ResourceChangeLogGroupMap = map[string]*ResourceChangeLogGroup
