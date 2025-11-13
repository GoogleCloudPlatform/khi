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
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

// KLogTextParser parses given klog formatted string.
// Example klog:  I0929 08:20:24.205299    1949 kubelet_getters.go:219] "Pod status updated" pod="kube-system/kube-proxy-gke-p0-gke-basic-1-default-6400229f-0hgr" status="Running"
type KLogTextParser struct {
	workers *sync.Pool
}

func NewKLogTextParser(hasHeader bool) *KLogTextParser {
	return &KLogTextParser{
		workers: &sync.Pool{
			New: func() any {
				return newKLogTextParserWorker(hasHeader)
			},
		},
	}
}

// TryParse implements StructuredLogParser.
func (k *KLogTextParser) TryParse(message string) *ParseStructuredLogResult {
	worker := k.workers.Get().(*klogTextParserWorker)
	defer k.workers.Put(worker)
	return worker.parse(message)
}

var _ StructuredLogParser = (*KLogTextParser)(nil)

// Special header field keys stored in fields.
const KLogHeaderDateFieldKey = "@date"
const KLogHeaderTimeFieldKey = "@time"
const KLogHeaderThreadIDFieldKey = "@threadid"
const KLogHeaderSourceLocationFieldKey = "@source"

var klogTimestampRegex = regexp.MustCompile(`^([IWEF])(\d{4})\s+(\d{2}:\d{2}:\d{2}\.\d{6})\s+(\d+)\s+([^:]+:\d+)]\s+(.*)$`)

type klogTextParserWorker struct {
	builder   strings.Builder
	hasHeader bool
}

func newKLogTextParserWorker(hasHeader bool) *klogTextParserWorker {
	return &klogTextParserWorker{
		builder:   strings.Builder{},
		hasHeader: hasHeader,
	}
}

func (w *klogTextParserWorker) parse(message string) *ParseStructuredLogResult {
	result := &ParseStructuredLogResult{
		Fields: map[string]any{
			OriginalMessageFieldKey: message,
		},
	}
	if !w.hasHeader { // GKE control plane can omit the header of klog. example)"Starting watch" path="/apis/admissionregistration.k8s.io/v1/mutatingwebhookconfigurations" resourceVersion="1759127820246769000" labels="" fields="" timeout="9m16s"
		err := w.parseFields(message, result)
		if err != nil {
			return nil
		}
		return result
	}
	matches := klogTimestampRegex.FindStringSubmatch(message)
	if matches == nil {
		return nil
	}

	severity, err := w.parseSeverity(matches[1])
	if err != nil {
		return nil
	}
	result.Fields[SeverityStructuredFieldKey] = severity
	result.Fields[KLogHeaderDateFieldKey] = matches[2]
	result.Fields[KLogHeaderTimeFieldKey] = matches[3]
	result.Fields[KLogHeaderThreadIDFieldKey] = matches[4]
	result.Fields[KLogHeaderSourceLocationFieldKey] = matches[5]

	err = w.parseFields(matches[6], result)
	if err != nil {
		return nil
	}
	return result
}

func (w *klogTextParserWorker) parseSeverity(severityStr string) (enum.Severity, error) {
	switch severityStr {
	case "I":
		return enum.SeverityInfo, nil
	case "W":
		return enum.SeverityWarning, nil
	case "E":
		return enum.SeverityError, nil
	case "F":
		return enum.SeverityFatal, nil
	default:
		return enum.SeverityUnknown, khierrors.ErrInvalidInput
	}
}

func (w *klogTextParserWorker) parseFields(messagePart string, result *ParseStructuredLogResult) error {
	w.builder.Reset()
	escaping := false
	mainMessageStarted := false
	mainMessageEnded := false
	parsingValue := false
	var lastKey string
	var endingMark rune
	includeEndingMark := false
	for i, c := range messagePart {
		if !mainMessageStarted {
			switch c {
			case ' ':
			case '"':
				mainMessageStarted = true
			default:
				// If Klog main message starts with non-double quote, then it won't have fields after the main message.
				// example) object-"1-8-daemonsets"/"kube-root-ca.crt": Failed to watch *v1.ConfigMap: failed to list *v1.ConfigMap: configmaps "kube-root-ca.crt" is forbidden: User "system:node:gke-p0-gke-basic-1-default-6400229f-0hgr" cannot list resource "configmaps" in API group "" in the namespace "1-8-daemonsets": no relationship found between node 'gke-p0-gke-basic-1-default-6400229f-0hgr' and this object
				result.Fields[MainMessageStructuredFieldKey] = messagePart
				return nil
			}
			continue
		}
		if !mainMessageEnded {
			if escaping {
				w.builder.WriteRune(c)
				escaping = false
				continue
			}
			switch c {
			case '\\':
				escaping = true
			case '"':
				mainMessageEnded = true
				result.Fields[MainMessageStructuredFieldKey] = w.builder.String()
				w.builder.Reset()
			default:
				w.builder.WriteRune(c)
			}
			continue
		}
		if !parsingValue {
			switch c {
			case '=':
				parsingValue = true
				lastKey = w.builder.String()
				w.builder.Reset()
			case ' ':
				if w.builder.Len() > 0 {
					return fmt.Errorf("found a space in the middle of key name")
				}
			default:
				w.builder.WriteRune(c)
			}
		} else {
			if endingMark == 0 {
				switch c {
				case '"':
					endingMark = '"'
					includeEndingMark = false
					continue
				case '[':
					endingMark = ']'
					includeEndingMark = true
				case '{':
					endingMark = '}'
					includeEndingMark = true
				case '&': // This must be followed by '{'
					if i+1 < len(messagePart) && messagePart[i+1] == '{' {
						endingMark = '}'
						includeEndingMark = true
					} else {
						return fmt.Errorf("failed to parse fields. '&' must be followed by '{' in fields")
					}
				default:
					endingMark = ' '
					includeEndingMark = false
				}
			}
			if escaping {
				w.builder.WriteRune(c)
				escaping = false
				continue
			}
			switch c {
			case endingMark:
				if includeEndingMark {
					w.builder.WriteRune(c)
				}
				result.Fields[lastKey] = w.builder.String()
				w.builder.Reset()
				parsingValue = false
				lastKey = ""
				endingMark = 0
			case '\\':
				escaping = true
				continue
			default:
				w.builder.WriteRune(c)
			}
		}
	}
	if parsingValue {
		result.Fields[lastKey] = w.builder.String()
		w.builder.Reset()
	}
	return nil
}
