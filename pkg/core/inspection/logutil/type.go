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

package logutil

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

const OriginalMessageFieldKey = "@original"

// MainMessageStructuredFieldKey is the key used to store the main log message in a structured log result.
const MainMessageStructuredFieldKey = "@msg"

// SeverityStructuredFieldKey is the key used to store the log severity in a structured log result.
const SeverityStructuredFieldKey = "@severity"

// ParseStructuredLogResult represents the result of parsing a structured log message.
type ParseStructuredLogResult struct {
	Fields map[string]any
}

func (p *ParseStructuredLogResult) Raw() string {
	if value, found := p.Fields[OriginalMessageFieldKey]; found {
		if valueStr, ok := value.(string); ok {
			return valueStr
		}
	}
	return ""
}

// MainMessage returns the main message of the structured log, or an error if not found or not a string.
func (p *ParseStructuredLogResult) MainMessage() (string, error) {
	return p.StringField(MainMessageStructuredFieldKey)
}

// Severity returns the severity of the structured log, or an error if not found or not of type enum.Severity.
func (p *ParseStructuredLogResult) Severity() (enum.Severity, error) {
	if value, found := p.Fields[SeverityStructuredFieldKey]; !found {
		return enum.SeverityUnknown, khierrors.ErrNotFound
	} else {
		if valueSeverity, ok := value.(enum.Severity); !ok {
			return enum.SeverityUnknown, khierrors.ErrTypeConversionFailed
		} else {
			return valueSeverity, nil
		}
	}
}

// StringField returns the string value of a specific field, or an error if not found or not a string.
func (p *ParseStructuredLogResult) StringField(field string) (string, error) {
	if value, found := p.Fields[field]; !found {
		return "", khierrors.ErrNotFound
	} else {
		if valueStr, ok := value.(string); !ok {
			return "", khierrors.ErrTypeConversionFailed
		} else {
			return valueStr, nil
		}
	}
}

// StructuredLogParser is the interface to parse string represented structured logs like klog.
type StructuredLogParser interface {
	// TryParse attempts to parse the given message into *ParseStructuredLogResult.
	// It returns nil when the given message is not in the format.
	TryParse(message string) *ParseStructuredLogResult
}

// FallbackRawTextLogParser uses the given message directly as the result of main message of parsing structured message.
type FallbackRawTextLogParser struct{}

// TryParse implements StructuredLogParser.
func (f *FallbackRawTextLogParser) TryParse(message string) *ParseStructuredLogResult {
	return &ParseStructuredLogResult{
		Fields: map[string]any{
			OriginalMessageFieldKey:       message,
			MainMessageStructuredFieldKey: message,
		},
	}
}

var _ StructuredLogParser = (*FallbackRawTextLogParser)(nil)

type MultiTextLogParser struct {
	parsers []StructuredLogParser
}

func NewMultiTextLogParser(parsers ...StructuredLogParser) *MultiTextLogParser {
	return &MultiTextLogParser{
		parsers: parsers,
	}
}

// TryParse implements StructuredLogParser.
func (m *MultiTextLogParser) TryParse(message string) *ParseStructuredLogResult {
	for _, parser := range m.parsers {
		result := parser.TryParse(message)
		if result != nil {
			return result
		}
	}
	return nil
}

var _ StructuredLogParser = (*MultiTextLogParser)(nil)
