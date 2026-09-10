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

package inspectiontaskbase

import (
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

var (
	// TagLogIngester is the tag provided by tasks that ingest logs into KHI v6 format.
	TagLogIngester = coretask.NewTag[struct{}]("khi.google.com/task/inspection/log-ingester")

	// TagTimelineMapper is the tag provided by tasks that map logs to timeline elements.
	TagTimelineMapper = coretask.NewTag[struct{}]("khi.google.com/task/inspection/timeline-mapper")
)
