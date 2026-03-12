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

package googlecloudclustercomposer_impl

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
)

// ComposerLogsFieldSetReadTask reads the main message and Composer component fieldsets.
var ComposerLogsFieldSetReadTask = inspectiontaskbase.NewFieldSetReadTask(
	googlecloudclustercomposer_contract.ComposerLogsFieldSetReadTaskID,
	googlecloudclustercomposer_contract.ComposerLogsQueryTaskID.Ref(),
	[]log.FieldSetReader{
		&gcpqueryutil.GCPMainMessageFieldSetReader{},
		&googlecloudclustercomposer_contract.ComposerFieldSetReader{},
		&googlecloudclustercomposer_contract.ComposerTaskInstanceFieldSetReader{},
		&googlecloudclustercomposer_contract.ComposerWorkerTaskInstanceFieldSetReader{},
	},
)
