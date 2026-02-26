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
	"encoding/json"
	"strconv"
	"strings"
)

var jsonlSeverityFieldNames = []string{"level", "severity"}
var jsonlMessageFieldNames = []string{"message", "msg"}

// JsonlTextParser parses given JSONL formatted string.
type JsonlTextParser struct{}

// NewJsonlTextParser creates a new JsonlTextParser.
func NewJsonlTextParser() *JsonlTextParser {
	return &JsonlTextParser{}
}

// TryParse implements StructuredLogParser.
func (j *JsonlTextParser) TryParse(originalMessage string) *ParseStructuredLogResult {
	message := strings.TrimSpace(originalMessage)
	if message == "" || message[0] != '{' {
		return nil
	}

	result := &ParseStructuredLogResult{
		Fields: map[string]any{
			OriginalMessageFieldKey: originalMessage,
		},
	}

	var m map[string]any
	decoder := json.NewDecoder(strings.NewReader(message))
	decoder.UseNumber()
	err := decoder.Decode(&m)
	if err != nil {
		return nil
	}

	for k, v := range m {
		strVal, err := valueToString(v)
		if err == nil {
			result.Fields[k] = strVal
		}
	}

	for _, msgField := range jsonlMessageFieldNames {
		if msgAny, ok := result.Fields[msgField]; ok {
			if msgStr, ok := msgAny.(string); ok {
				result.Fields[MainMessageStructuredFieldKey] = msgStr
				break
			}
		}
	}

	for _, severityField := range jsonlSeverityFieldNames {
		if severityAny, ok := result.Fields[severityField]; ok {
			if severityStr, ok := severityAny.(string); ok {
				if severity, found := commonSeverityStringNotation[severityStr]; found {
					result.Fields[SeverityStructuredFieldKey] = severity
					break
				}
			}
		}
	}

	return result
}

var _ StructuredLogParser = (*JsonlTextParser)(nil)

func valueToString(v any) (string, error) {
	if v == nil {
		return "null", nil
	}
	switch val := v.(type) {
	case string:
		return val, nil
	case json.Number:
		return val.String(), nil
	case bool:
		return strconv.FormatBool(val), nil
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
