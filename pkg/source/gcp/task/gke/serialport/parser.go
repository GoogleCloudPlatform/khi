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

package serialport

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GoogleCloudPlatform/khi/pkg/common/parserutil"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	serialport_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/serialport/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"

	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/inspectiontype"
)

var serialportSequenceConverters = []parserutil.SpecialSequenceConverter{
	&parserutil.ANSIEscapeSequenceStripper{},
	&parserutil.SequenceConverter{From: []string{"\\r", "\\n", "\\x1bM"}},
	&parserutil.UnicodeUnquoteConverter{},
	&parserutil.SequenceConverter{From: []string{"\\x2d"}, To: "-"},
	&parserutil.SequenceConverter{From: []string{"\t"}, To: " "},
}

type SerialPortLogParser struct {
}

// TargetLogType implements parser.Parser.
func (s *SerialPortLogParser) TargetLogType() enum.LogType {
	return enum.LogTypeSerialPort
}

// Description implements parser.Parser.
func (*SerialPortLogParser) Description() string {
	return `Gather serialport logs of GKE nodes. This helps detailed investigation on VM bootstrapping issue on GKE node.`
}

// GetParserName implements parser.Parser.
func (*SerialPortLogParser) GetParserName() string {
	return "Node serial port logs"
}

func (*SerialPortLogParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (*SerialPortLogParser) LogTask() taskid.TaskReference[[]*log.LogEntity] {
	return serialport_taskid.SerialPortLogQueryTaskID.Ref()
}

func (*SerialPortLogParser) Grouper() grouper.LogGrouper {
	return grouper.NewSingleStringFieldKeyLogGrouper("resource.labels.instance_id")
}

// Parse implements parser.Parser.
func (*SerialPortLogParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {

	// Label field contains `.` in its key. the value needs to be retrived from the low level API.
	nodeName := "unknown"
	nodeNameReader, err := l.Fields.ReaderFromArrayRoute([]string{"labels", "compute.googleapis.com/resource_name"})
	if err == nil && len(nodeNameReader) > 0 {
		nodeNameReadFromReader, err := nodeNameReader[0].ReadString("")
		if err == nil {
			nodeName = nodeNameReadFromReader
		}
	}

	mainMessage, err := l.MainMessage()
	if err != nil {
		yaml, err := l.Fields.ToYaml("")
		if err != nil {
			yaml = "!!ERROR failed to dump in yaml"
		}
		slog.WarnContext(ctx, fmt.Sprintf("Failed to extract main message from serial port log.\nError: %s\n\nLog content: %s", err.Error(), yaml))
		mainMessage = "(unknown)"
	}
	mainMessage = parserutil.ConvertSpecialSequences(mainMessage, serialportSequenceConverters...)
	serialPortResourcePath := resourcepath.NodeSerialport(nodeName)
	cs.RecordEvent(serialPortResourcePath)
	cs.RecordLogSummary(mainMessage)
	return nil
}

var _ parser.Parser = (*SerialPortLogParser)(nil)

var GKESerialPortLogParseTask = parser.NewParserTaskFromParser(serialport_taskid.SerialPortLogParserTaskID, &SerialPortLogParser{}, false, inspectiontype.GKEBasedClusterInspectionTypes)
