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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

type OSSAuditLogFieldExtractor struct{}

// DisplayID implements log.CommonLogFieldExtractor.
func (o *OSSAuditLogFieldExtractor) DisplayID(l *log.LogEntity) string {
	id, err := l.GetString("auditID")
	if err != nil {
		panic(err.Error())
	}
	return id
}

// ID implements log.CommonLogFieldExtractor.
func (o *OSSAuditLogFieldExtractor) ID(l *log.LogEntity) string {
	id, err := l.GetString("auditID")
	if err != nil {
		panic(err.Error())
	}
	return id
}

// LogBody implements log.CommonLogFieldExtractor.
func (o *OSSAuditLogFieldExtractor) LogBody(l *log.LogEntity) string {
	id, err := l.Fields.ToYaml("")
	if err != nil {
		return ""
	}
	return id
}

// MainMessage implements log.CommonLogFieldExtractor.
func (o *OSSAuditLogFieldExtractor) MainMessage(l *log.LogEntity) (string, error) {
	return "", nil
}

// Severity implements log.CommonLogFieldExtractor.
func (o *OSSAuditLogFieldExtractor) Severity(l *log.LogEntity) (enum.Severity, error) {
	return enum.SeverityUnknown, nil
}

// Timestamp implements log.CommonLogFieldExtractor.
func (o *OSSAuditLogFieldExtractor) Timestamp(l *log.LogEntity) time.Time {
	timeInStr, err := l.Fields.ReadTimeAsString("stageTimestamp")
	if err != nil {
		panic(fmt.Errorf("failed to decode %s", err))
	}
	t, err := time.Parse(time.RFC3339Nano, timeInStr)
	if err == nil {
		return t
	}
	t, err = time.Parse(time.RFC3339, timeInStr)
	if err != nil {
		panic(fmt.Errorf("failed to find appropriate parser for timestamp %s\n%s", timeInStr, err))
	}
	return t
}

var _ log.CommonLogFieldExtractor = (*OSSAuditLogFieldExtractor)(nil)
