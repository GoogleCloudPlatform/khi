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

package googlecloudcommon_contract

import (
	"strings"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// ParseGCPSeverity converts a GCP Cloud Logging severity string into a timeline style Severity.
// It maps the GCP log severities defined in https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#logseverity
// to KHI's registered timeline style Severities in the core contract.
func ParseGCPSeverity(gcpSeverity string) *pb.Severity {
	gcpSeverity = strings.ToUpper(strings.TrimSpace(gcpSeverity))
	switch gcpSeverity {
	case "DEFAULT", "DEBUG", "INFO", "NOTICE":
		return inspectioncore.SeverityInfo
	case "WARNING":
		return inspectioncore.SeverityWarning
	case "ERROR":
		return inspectioncore.SeverityError
	case "CRITICAL", "ALERT", "EMERGENCY":
		return inspectioncore.SeverityFatal
	default:
		return inspectioncore.SeverityUnknown
	}
}
