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

package k8s

import (
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

// severityStringNotation maps string notation of severity found in logs to the severity types used in KHI.
var severityStringNotation = map[string]enum.Severity{
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

var severityKlogFieldNames = []string{"level", "severity"}

// https://github.com/kubernetes/klog/blob/v2.80.1/klog.go#L626-L645
// TODO: We need to handle time field in later, but ignore it for now because times can be obtained from the other source.
type klogHeader struct {
	Severity enum.Severity
	Message  string
}

// ignore `file`,`threadid` and `line` part.
var klogHeaderMatcher = regexp.MustCompile(`^([IWEF])(\d{2})(\d{2}) (\d{2}):(\d{2}):(\d{2})\.(\d{6})\s+.*\](.*)$`)

func parseKLogHeader(klog string) *klogHeader {
	matches := klogHeaderMatcher.FindStringSubmatch(klog)
	if len(matches) > 0 {
		severityStr := matches[1]
		severity := enum.SeverityUnknown
		switch severityStr {
		case "I":
			severity = enum.SeverityInfo
		case "W":
			severity = enum.SeverityWarning
		case "E":
			severity = enum.SeverityError
		case "F":
			severity = enum.SeverityFatal
		}
		return &klogHeader{
			Severity: severity,
			Message:  strings.TrimSpace(matches[len(matches)-1]),
		}
	}
	return nil
}

func parseKLogMessageFragment(klogMessageFragment string) map[string]string {
	result := map[string]string{}
	inQuotes := false
	inGoBrace := false
	parsingKey := true
	escaping := false
	currentKey := ""
	currentGroup := ""
	// For the log format not starting from the double quote
	// Example:
	// Error foo" fieldWithQuotes="foo" fieldWithEscape="foo \"bar\"" fieldWithoutQuotes=qux1234
	if strings.Count(klogMessageFragment, "\"")%2 == 1 {
		klogMessageFragment = `"` + klogMessageFragment
	}
	for i := 0; i < len(klogMessageFragment); i++ {
		// For log body beginning with `"`, it should be regarded as the msg field.
		if i == 0 && klogMessageFragment[i] == '"' {
			inQuotes = true
			// `msg` is reserved for the main message
			currentKey = "msg"
			parsingKey = false
			continue
		}
		if !escaping {

			if klogMessageFragment[i] == '\\' {
				escaping = true
				continue
			}

			if klogMessageFragment[i] == '{' && !inQuotes {
				inGoBrace = true
				currentGroup += string(klogMessageFragment[i])
				continue
			}

			if klogMessageFragment[i] == '}' && !inQuotes && inGoBrace {
				inGoBrace = false
				currentGroup += string(klogMessageFragment[i])
				continue
			}

			if klogMessageFragment[i] == '"' && !inGoBrace {
				if !parsingKey && inQuotes {
					result[currentKey] = currentGroup
					parsingKey = true
					currentGroup = ""
				}
				inQuotes = !inQuotes
				continue
			}

			if klogMessageFragment[i] == '=' && !inQuotes && !inGoBrace {
				if parsingKey {
					currentKey = currentGroup
					currentGroup = ""
					parsingKey = false
					continue
				}
			}
		}

		if klogMessageFragment[i] == ' ' && !inQuotes && !inGoBrace {
			if !parsingKey {
				result[currentKey] = currentGroup
				parsingKey = true
				currentGroup = ""
			}
			continue
		}

		if escaping {
			escaping = false
		}

		currentGroup += string(klogMessageFragment[i])
	}
	if !parsingKey {
		result[currentKey] = currentGroup
	}
	return result
}

// https://kubernetes.io/docs/concepts/cluster-administration/system-logs/#klog-output
func ExtractKLogField(klogBody string, field string) (string, error) {
	header := parseKLogHeader(klogBody)
	message := klogBody
	if header != nil {
		message = header.Message
	}
	fields := parseKLogMessageFragment(message)
	if field == "" {
		if message, hasMsg := fields["msg"]; hasMsg {
			return message, nil
		}
		if header != nil {
			return header.Message, nil
		}
		return klogBody, nil
	} else {
		if fieldValue, hasField := fields[field]; hasField {
			return fieldValue, nil
		} else {
			return "", nil
		}
	}
}

// ExractKLogSeverity returns severity from klog formatted logs.
func ExractKLogSeverity(klogBody string) enum.Severity {
	header := parseKLogHeader(klogBody)
	if header != nil {
		klogBody = header.Message
	}
	fields := parseKLogMessageFragment(klogBody)
	for _, fieldName := range severityKlogFieldNames {
		if severityInStr, hasLevel := fields[fieldName]; hasLevel {
			if khiSeverity, isKnownSeverity := severityStringNotation[severityInStr]; isKnownSeverity {
				return khiSeverity
			}
		}
	}
	if header != nil {
		return header.Severity
	}
	return enum.SeverityUnknown
}
