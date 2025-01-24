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
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

var jsonPayloadMessageFieldNames = []string{
	"MESSAGE",
	"message",
	"msg",
	"log",
}

var uniqueIdMap = sync.Map{}

type GCPCommonFieldExtractor struct{}

// LogBody implements log.CommonLogFieldExtractor.
func (GCPCommonFieldExtractor) LogBody(l *log.LogEntity) string {
	id, err := l.Fields.ToYaml("")
	if err != nil {
		return ""
	}
	return id
}

// DisplayID implements log.CommonLogFieldExtractor.
func (GCPCommonFieldExtractor) DisplayID(l *log.LogEntity) string {
	id, err := l.GetString("insertId")
	if err != nil {
		panic(err)
	}
	return id
}

// GCPCommonFieldExtractor implements log.CommonLogFieldExtractor
var _ log.CommonLogFieldExtractor = (*GCPCommonFieldExtractor)(nil)

func (GCPCommonFieldExtractor) ID(log *log.LogEntity) string {
	id, err := log.GetString("insertId")
	if err != nil {
		panic(err)
	}
	timestamp, err := log.Fields.ReadTimeAsString("timestamp")
	if err != nil {
		panic(err)
	}
	// id key can be long and it can inflate the size of KHI file.
	// Use a random ID associated to timestamp and insertId instead.
	idKey := fmt.Sprintf("%s-%s", id, timestamp)
	nextId := generateLogId()
	logId, _ := uniqueIdMap.LoadOrStore(idKey, nextId)
	return logId.(string)
}

func (GCPCommonFieldExtractor) Timestamp(log *log.LogEntity) time.Time {
	timeInStr, err := log.Fields.ReadTimeAsString("timestamp")
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

func (GCPCommonFieldExtractor) MainMessage(log *log.LogEntity) (string, error) {
	textPayload, err := log.GetString("textPayload")
	if err == nil {
		return textPayload, nil
	}

	for _, fieldName := range jsonPayloadMessageFieldNames {
		jsonPayloadMessage, err := log.GetString(fmt.Sprintf("jsonPayload.%s", fieldName))
		if err == nil {
			return jsonPayloadMessage, nil
		}
	}

	requestReader, err := log.Fields.ReaderSingle("httpRequest")
	if err == nil {
		statusInt, err1 := requestReader.ReadInt("status")
		requestUrl, err2 := requestReader.ReadString("requestUrl")
		requestMethod, err3 := requestReader.ReadString("requestMethod")
		protocol, _ := requestReader.ReadString("protocol")
		if err1 == nil && err2 == nil && err3 == nil {
			if protocol == "grpc" {
				return fmt.Sprintf("【%d】GRPC %s", statusInt, requestUrl), nil
			} else {
				return fmt.Sprintf("【%d】%s %s", statusInt, requestMethod, requestUrl), nil
			}
		}
	}

	fallbackAsJson, err := log.Fields.ReaderSingle("jsonPayload")
	if err == nil {
		jsonMessage, err := fallbackAsJson.ToJson("")
		if err == nil {
			return jsonMessage, nil
		} else {
			return "", err
		}
	}

	fallbackAsLabels, err := log.Fields.ReaderSingle("labels")
	if err == nil {
		jsonMessage, err := fallbackAsLabels.ToJson("")
		if err == nil {
			return jsonMessage, nil
		} else {
			return "", err
		}
	}
	return "", fmt.Errorf("failed to extract main message from given log")
}

// Severity implements log.CommonLogFieldExtractor.
func (GCPCommonFieldExtractor) Severity(l *log.LogEntity) (enum.Severity, error) {
	severity, err := l.GetString("severity")
	if err != nil {
		return enum.SeverityUnknown, err
	}
	return gcpSeverityToKHISeverity(severity), nil
}

func gcpSeverityToKHISeverity(severity string) enum.Severity {
	// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#logseverity
	severity = strings.ToUpper(severity)
	switch severity {
	case "DEFAULT":
		return enum.SeverityInfo
	case "DEBUG":
		return enum.SeverityInfo
	case "INFO":
		return enum.SeverityInfo
	case "NOTICE":
		return enum.SeverityInfo
	case "WARNING":
		return enum.SeverityWarning
	case "ERROR":
		return enum.SeverityError
	case "CRITICAL":
		return enum.SeverityFatal
	case "ALERT":
		return enum.SeverityFatal
	case "EMERGENCY":
		return enum.SeverityFatal
	default:
		return enum.SeverityUnknown
	}
}

func generateLogId() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomid := make([]rune, 16)
	for i := range randomid {
		randomid[i] = letters[rand.Intn(len(letters))]
	}
	return string(randomid)
}
