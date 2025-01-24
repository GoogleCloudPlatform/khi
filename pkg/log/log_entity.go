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

package log

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/parser/k8s"
)

// LogEntity corresponds to a log record.
// Log itself is just a structured data and the main functionality is provided from structure.Reader.
// LogEntity adds few methods to get the common fields in logs like timestamp,id, severity ...etc.
type LogEntity struct {
	LogType      enum.LogType
	commonFields CommonLogFieldExtractor
	Fields       structure.Reader
}

func NewLogEntity(reader *structure.Reader, commonFieldExtractor CommonLogFieldExtractor) *LogEntity {
	return &LogEntity{Fields: *reader, commonFields: commonFieldExtractor}
}

func (l *LogEntity) Has(path string) bool {
	_, err := l.Fields.ReaderSingle(path)
	return err == nil
}

func (l *LogEntity) GetString(path string) (string, error) {
	return l.Fields.ReadString(path)
}

func (l *LogEntity) GetStringOrDefault(path string, def string) string {
	return l.Fields.ReadStringOrDefault(path, def)
}

func (l *LogEntity) GetInt(path string) (int, error) {
	return l.Fields.ReadInt(path)
}

func (l *LogEntity) GetIntOrDefault(path string, def int) int {
	return l.Fields.ReadIntOrDefault(path, def)
}

func (l *LogEntity) GetChildYamlOf(path string) (string, error) {
	reader, err := l.Fields.ReaderSingle(path)
	if err != nil {
		return "", err
	}
	return reader.ToYaml("")
}

func (l *LogEntity) KLogField(klogField string) (string, error) {
	klog, err := l.MainMessage()
	if err != nil {
		return "", err
	}
	return k8s.ExtractKLogField(klog, klogField)
}

func (l *LogEntity) HasKLogField(klogField string) bool {
	klog, err := l.MainMessage()
	if err != nil {
		return false
	}
	value, err := k8s.ExtractKLogField(klog, klogField)
	return err == nil && value != ""
}

func (l *LogEntity) Timestamp() time.Time {
	return l.commonFields.Timestamp(l)
}

func (l *LogEntity) ID() string {
	return l.commonFields.ID(l)
}

func (l *LogEntity) MainMessage() (string, error) {
	return l.commonFields.MainMessage(l)
}

func (l *LogEntity) Severity() (enum.Severity, error) {
	return l.commonFields.Severity(l)
}

func (l *LogEntity) DisplayId() string {
	return l.commonFields.DisplayID(l)
}

func (l *LogEntity) LogBody() string {
	return l.commonFields.LogBody(l)
}
