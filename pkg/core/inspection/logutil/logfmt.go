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
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

// logfmtSeverityStringNotation maps string notation of severity found in logs to the severity types used in KHI.
var logfmtSeverityStringNotation = map[string]enum.Severity{
	"INFO":    enum.SeverityInfo,
	"info":    enum.SeverityInfo,
	"WARN":    enum.SeverityWarning,
	"warn":    enum.SeverityWarning,
	"WARNING": enum.SeverityWarning,
	"warning": enum.SeverityWarning,
	"ERROR":   enum.SeverityError,
	"error":   enum.SeverityError,
	"ERR":     enum.SeverityError,
	"err":     enum.SeverityError,
	"FATAL":   enum.SeverityFatal,
	"fatal":   enum.SeverityFatal,
	"panic":   enum.SeverityFatal,
}

var severityLogfmtFieldNames = []string{"level", "severity"}

// LogfmtTextParser parses given logfmt formatted string.
// Reference: https://github.com/hynek/structlog/issues/511#issuecomment-1916426273
type LogfmtTextParser struct {
	workers *sync.Pool
}

func NewLogfmtTextParser() *LogfmtTextParser {
	return &LogfmtTextParser{
		workers: &sync.Pool{
			New: func() any {
				return newLogfmtTextParserWorker()
			},
		},
	}
}

// TryParse implements StructuredLogParser.
func (l *LogfmtTextParser) TryParse(message string) *ParseStructuredLogResult {
	worker := l.workers.Get().(*logfmtTextParserWorker)
	defer l.workers.Put(worker)
	result, err := worker.parse(message)
	if err != nil {
		return nil
	}
	return result
}

var _ StructuredLogParser = (*LogfmtTextParser)(nil)

type logfmtTextParserWorker struct {
	builder strings.Builder
}

func newLogfmtTextParserWorker() *logfmtTextParserWorker {
	return &logfmtTextParserWorker{
		builder: strings.Builder{},
	}
}

func (w *logfmtTextParserWorker) parse(message string) (*ParseStructuredLogResult, error) {
	w.builder.Reset()
	result := &ParseStructuredLogResult{
		Fields: map[string]any{
			OriginalMessageFieldKey: message,
		},
	}
	parsingKey := true
	lastKey := ""
	var endingMark rune
	escaping := false
	for _, c := range message {
		if parsingKey {
			switch c {
			case ' ':
				if w.builder.Len() > 0 {
					return nil, khierrors.ErrInvalidInput
				}
				continue
			case '=':
				parsingKey = false
				lastKey = w.builder.String()
				w.builder.Reset()
			default:
				w.builder.WriteRune(c)
			}
		} else {
			if endingMark == 0 {
				switch c {
				case '"':
					endingMark = '"'
				default:
					w.builder.WriteRune(c)
					endingMark = ' '
				}
				continue
			}
			if escaping {
				w.builder.WriteRune(c)
				escaping = false
				continue
			}
			switch c {
			case '\\':
				if endingMark == '"' {
					escaping = true // escape is valid only within ""
				} else {
					w.builder.WriteRune(c)
				}
				continue
			case endingMark:
				value := w.builder.String()
				result.Fields[lastKey] = value
				w.builder.Reset()
				parsingKey = true
				lastKey = ""
				endingMark = 0
			default:
				w.builder.WriteRune(c)
			}
		}
	}
	if !parsingKey {
		result.Fields[lastKey] = w.builder.String()
		w.builder.Reset()
	}
	if msg, ok := result.Fields["msg"]; ok {
		result.Fields[MainMessageStructuredFieldKey] = msg
	}
	for _, severityField := range severityLogfmtFieldNames {
		if severityAny, ok := result.Fields[severityField]; ok {
			if severityStr, ok := severityAny.(string); ok {
				if severity, found := logfmtSeverityStringNotation[severityStr]; found {
					result.Fields[SeverityStructuredFieldKey] = severity
					break
				}
			}
		}
	}
	return result, nil
}
