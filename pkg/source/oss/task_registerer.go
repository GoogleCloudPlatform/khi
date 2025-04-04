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

package oss

import (
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/source/oss/form"
	"github.com/GoogleCloudPlatform/khi/pkg/source/oss/parser"
)

func Prepare(inspetionServer *inspection.InspectionTaskServer) error {
	err := inspetionServer.AddInspectionType(OSSKubernetesLogFilesInspectionType)
	if err != nil {
		return err
	}

	err = inspetionServer.AddTask(parser.OSSK8sEventLogParserTask)
	if err != nil {
		return err
	}
	err = parser.RegisterK8sAuditTasks(inspetionServer)
	if err != nil {
		return err
	}
	err = inspetionServer.AddTask(parser.OSSLogFileReader)
	if err != nil {
		return err
	}
	err = inspetionServer.AddTask(parser.OSSEventLogFilter)
	if err != nil {
		return err
	}
	err = inspetionServer.AddTask(parser.OSSNonEventLogFilter)
	if err != nil {
		return err
	}
	err = inspetionServer.AddTask(form.AuditLogFilesForm)
	if err != nil {
		return err
	}

	return nil
}
