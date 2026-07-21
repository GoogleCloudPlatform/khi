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

package logutil

import (
	"encoding/json"
	"regexp"
	"strings"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// ZapConsoleTimestampFieldKey is the key stored in Fields for the timestamp in a Zap console log.
const ZapConsoleTimestampFieldKey = "@timestamp"

// ZapConsoleCallerFieldKey is the key stored in Fields for the caller or source location in a Zap console log.
const ZapConsoleCallerFieldKey = "@caller"

var zapTimestampRegex = regexp.MustCompile(`^\d{4}[-/]\d{2}[-/]\d{2}[T\s]\d{2}:\d{2}:\d{2}|^\d+(\.\d+)?$`)
var zapCallerRegex = regexp.MustCompile(`(?i)^([a-z]:)?[^\s:]+:\d+$`)

// ZapConsoleTextParser parses log lines formatted using Zap's ConsoleEncoder.
// Zap console logs start with a timestamp (part 0), followed by severity level (part 1), caller location (part 2), and message.
// An example log entry is:
// 2026-02-18T06:58:06.999Z\tinfo\tkubernetes/kubernetes.go:282\tUsing pod service account via in-cluster config\t{"kind": "receiver", "name": "prometheus", "discovery": "kubernetes"}
type ZapConsoleTextParser struct{}

// NewZapConsoleTextParser creates a new ZapConsoleTextParser instance.
func NewZapConsoleTextParser() *ZapConsoleTextParser {
	return &ZapConsoleTextParser{}
}

// TryParse attempts to parse the given message as a Zap ConsoleEncoder formatted log line.
// It requires part 0 to be a timestamp, part 1 to be a severity level, and part 2 to be a caller location.
// It returns nil when the message does not match these specifications.
func (z *ZapConsoleTextParser) TryParse(message string) *ParseStructuredLogResult {
	parts := strings.Split(message, "\t")
	if len(parts) < 4 {
		return nil
	}

	// 1. Part 0 MUST be Timestamp.
	timestamp := strings.TrimSpace(parts[0])
	if !zapTimestampRegex.MatchString(timestamp) {
		return nil
	}

	// 2. Part 1 MUST be Severity Level.
	severity, ok := parseZapSeverity(parts[1])
	if !ok {
		return nil
	}

	// 3. Part 2 MUST be Caller Location.
	caller := strings.TrimSpace(parts[2])
	if !isZapCaller(caller) {
		return nil
	}

	remaining := parts[3:]
	if len(remaining) == 0 {
		return nil
	}

	result := &ParseStructuredLogResult{
		Fields: map[string]any{
			OriginalMessageFieldKey:     message,
			SeverityStructuredFieldKey:  severity,
			ZapConsoleTimestampFieldKey: timestamp,
			ZapConsoleCallerFieldKey:    caller,
		},
	}

	var jsonParsed bool
	// Extract optional JSON context fields from the last element if present.
	lastPart := strings.TrimSpace(remaining[len(remaining)-1])
	if strings.HasPrefix(lastPart, "{") && strings.HasSuffix(lastPart, "}") {
		var m map[string]any
		decoder := json.NewDecoder(strings.NewReader(lastPart))
		decoder.UseNumber()
		if err := decoder.Decode(&m); err == nil {
			for k, v := range m {
				strVal, err := valueToString(v)
				if err == nil {
					result.Fields[k] = strVal
				}
			}
			remaining = remaining[:len(remaining)-1]
			jsonParsed = true
		}
	}

	if len(remaining) > 0 {
		mainMessage := strings.Join(remaining, "\t")
		if strings.TrimSpace(mainMessage) != "" {
			result.Fields[MainMessageStructuredFieldKey] = mainMessage
		}
	}

	_, hasMessage := result.Fields[MainMessageStructuredFieldKey]
	if !hasMessage && !jsonParsed {
		return nil
	}

	return result
}

var _ StructuredLogParser = (*ZapConsoleTextParser)(nil)

func parseZapSeverity(s string) (*pb.Severity, bool) {
	cleaned := strings.TrimSpace(s)
	if sev, ok := commonSeverityStringNotation[cleaned]; ok {
		return sev, true
	}
	switch strings.ToLower(cleaned) {
	case "debug":
		return inspectioncore_contract.SeverityInfo, true
	case "dpanic":
		return inspectioncore_contract.SeverityError, true
	}
	return nil, false
}

func isZapCaller(s string) bool {
	return zapCallerRegex.MatchString(s)
}
