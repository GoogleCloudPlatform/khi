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

package googlecloudlogserialport_contract

import (
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

var serialportSequenceConverters = []logutil.SpecialSequenceConverter{
	&logutil.ANSIEscapeSequenceStripper{},
	&logutil.SequenceConverter{From: []string{"\\r", "\\n", "\\x1bM"}},
	&logutil.UnicodeUnquoteConverter{},
	&logutil.SequenceConverter{From: []string{"\\x2d"}, To: "-"},
	&logutil.SequenceConverter{From: []string{"\t"}, To: " "},
	// serialport log may have timestamp at the beginning like the following format:
	// 2025-09-29T06:39:24+0000 gke-p0-gke-basic-1-default-6400229f-0hgr kubelet[1949]: I0929 06:39:24.070536    1949 flags.go:64] FLAG: --event-storage-age-limit="default=0"
	// This is replaced with
	// kubelet[1949]: I0929 06:39:24.070536    1949 flags.go:64] FLAG: --event-storage-age-limit="default=0"
	logutil.MustNewRegexSequenceConverter(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[\+\-]\d{4}\s.\S+\s`, ""),
}

type GCESerialPortLogFieldSet struct {
	Message  string
	NodeName string
	Port     string
}

// Kind implements log.FieldSet.
func (g *GCESerialPortLogFieldSet) Kind() string {
	return "gce-serialport"
}

// GetResorucePath returns the ResourcePath representing the serialport timeline.
func (g *GCESerialPortLogFieldSet) GetResourcePath() resourcepath.ResourcePath {
	return resourcepath.NodeSerialport(g.NodeName, g.Port)
}

var _ log.FieldSet = (*GCESerialPortLogFieldSet)(nil)

type GCESerialPortLogFieldSetReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (g *GCESerialPortLogFieldSetReader) FieldSetKind() string {
	return (&GCESerialPortLogFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (g *GCESerialPortLogFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	textPayload := reader.ReadStringOrDefault("textPayload", "")
	escapedTextPayload := logutil.ConvertSpecialSequences(textPayload, serialportSequenceConverters...)

	nodeName := reader.ReadStringOrDefault("labels.compute\\.googleapis\\.com/resource_name", "unknown")

	logName := reader.ReadStringOrDefault("logName", "")
	port := "unknown_port"
	if slashIndex := strings.LastIndex(logName, "%2F"); slashIndex != -1 {
		port = logName[slashIndex+len("%2F"):]
	}

	return &GCESerialPortLogFieldSet{
		Message:  escapedTextPayload,
		Port:     port,
		NodeName: nodeName,
	}, nil
}

var _ log.FieldSetReader = (*GCESerialPortLogFieldSetReader)(nil)
