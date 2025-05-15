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

func (*SerialPortLogParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return serialport_taskid.SerialPortLogQueryTaskID.Ref()
}

func (*SerialPortLogParser) Grouper() grouper.LogGrouper {
	return grouper.NewSingleStringFieldKeyLogGrouper("resource.labels.instance_id")
}

// Parse implements parser.Parser.
func (*SerialPortLogParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	nodeName := l.ReadStringOrDefault("labels.compute\\.googleapis\\.com/resource_name", "unknown")
	mainMessageFieldSet := log.MustGetFieldSet(l, &log.MainMessageFieldSet{})
	escapedMainMessage := parserutil.ConvertSpecialSequences(mainMessageFieldSet.MainMessage, serialportSequenceConverters...)
	serialPortResourcePath := resourcepath.NodeSerialport(nodeName)
	cs.RecordEvent(serialPortResourcePath)
	cs.RecordLogSummary(escapedMainMessage)
	return nil
}

var _ parser.Parser = (*SerialPortLogParser)(nil)

var GKESerialPortLogParseTask = parser.NewParserTaskFromParser(serialport_taskid.SerialPortLogParserTaskID, &SerialPortLogParser{}, false, inspectiontype.GKEBasedClusterInspectionTypes)
