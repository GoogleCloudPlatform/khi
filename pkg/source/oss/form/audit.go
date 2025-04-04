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

package form

import (
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/form"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	oss_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/oss/taskid"
)

var AuditLogFilesForm = form.NewFileFormTaskBuilder(oss_taskid.OSSAPIServerAuditLogFileInputTask, 1000, "Audit Log Files", &upload.JSONLineUploadFileVerifier{
	MaxLineSizeInBytes: 1024 * 1024 * 1024,
}).
	WithDescription(`Upload JSONLine format kube-apiserver audit log`).
	Build()
