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

// ComposerSchedulerFieldSetReadTask reads the main message from the scheduler log.
var ComposerSchedulerFieldSetReadTask = inspectiontaskbase.NewFieldSetReadTask(
	googlecloudclustercomposer_contract.ComposerSchedulerFieldSetReadTaskID,
	googlecloudclustercomposer_contract.ComposerSchedulerLogQueryTaskID.Ref(),
	[]log.FieldSetReader{
		&gcpqueryutil.GCPMainMessageFieldSetReader{},
		&googlecloudclustercomposer_contract.ComposerSchedulerFieldSetReader{},
		&googlecloudclustercomposer_contract.ComposerTaskInstanceFieldSetReader{},
	},
)

// ComposerDagProcessorManagerFieldSetReadTask reads the main message from the DAG processor manager log.
var ComposerDagProcessorManagerFieldSetReadTask = inspectiontaskbase.NewFieldSetReadTask(
	googlecloudclustercomposer_contract.ComposerDagProcessorManagerFieldSetReadTaskID,
	googlecloudclustercomposer_contract.ComposerDagProcessorManagerLogQueryTaskID.Ref(),
	[]log.FieldSetReader{
		&gcpqueryutil.GCPMainMessageFieldSetReader{},
		&googlecloudclustercomposer_contract.ComposerSchedulerFieldSetReader{},
	},
)

// ComposerMonitoringFieldSetReadTask reads the main message from the monitoring log.
var ComposerMonitoringFieldSetReadTask = inspectiontaskbase.NewFieldSetReadTask(
	googlecloudclustercomposer_contract.ComposerMonitoringFieldSetReadTaskID,
	googlecloudclustercomposer_contract.ComposerMonitoringLogQueryTaskID.Ref(),
	[]log.FieldSetReader{
		&gcpqueryutil.GCPMainMessageFieldSetReader{},
	},
)

// ComposerWorkerFieldSetReadTask reads the main message from the worker log.
var ComposerWorkerFieldSetReadTask = inspectiontaskbase.NewFieldSetReadTask(
	googlecloudclustercomposer_contract.ComposerWorkerFieldSetReadTaskID,
	googlecloudclustercomposer_contract.ComposerWorkerLogQueryTaskID.Ref(),
	[]log.FieldSetReader{
		&gcpqueryutil.GCPMainMessageFieldSetReader{},
		&googlecloudclustercomposer_contract.ComposerWorkerFieldSetReader{},
		&googlecloudclustercomposer_contract.ComposerWorkerTaskInstanceFieldSetReader{},
	},
)
