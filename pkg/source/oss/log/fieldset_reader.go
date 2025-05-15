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

package log

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structurev2"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

type OSSK8sAuditLogCommonFieldSetReader struct{}

// FieldSetKind implements log.FieldSetReader.
func (o *OSSK8sAuditLogCommonFieldSetReader) FieldSetKind() string {
	return (&log.CommonFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (o *OSSK8sAuditLogCommonFieldSetReader) Read(reader *structurev2.NodeReader) (log.FieldSet, error) {
	var err error
	result := &log.CommonFieldSet{}
	result.DisplayID = reader.ReadStringOrDefault("auditID", "unknown")
	result.Timestamp, err = reader.ReadTimestamp("stageTimestamp")
	if err != nil {
		return nil, fmt.Errorf("failed to read timestmap from given log")
	}
	result.Severity = enum.SeverityUnknown // TODO: handle OSS k8s audit log severity properly
	return result, nil
}

var _ log.FieldSetReader = (*OSSK8sAuditLogCommonFieldSetReader)(nil)
