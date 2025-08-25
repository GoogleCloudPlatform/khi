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

package googlecloudlogserialport_impl

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/legacyparser"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogserialport_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogserialport/contract"
)

var serialportSequenceConverters = []logutil.SpecialSequenceConverter{
	&logutil.ANSIEscapeSequenceStripper{},
	&logutil.SequenceConverter{From: []string{"\\r", "\\n", "\\x1bM"}},
	&logutil.UnicodeUnquoteConverter{},
	&logutil.SequenceConverter{From: []string{"\\x2d"}, To: "-"},
	&logutil.SequenceConverter{From: []string{"\t"}, To: " "},
}

type SerialPortLogParser struct {
}

// TargetLogType implements parsertask.Parser.
func (s *SerialPortLogParser) TargetLogType() enum.LogType {
	return enum.LogTypeSerialPort
}

// Description implements parsertask.Parser.
func (*SerialPortLogParser) Description() string {
	return `Gather serialport logs of GKE nodes. This helps detailed investigation on VM bootstrapping issue on GKE node.`
}

// GetParserName implements parsertask.Parser.
func (*SerialPortLogParser) GetParserName() string {
	return "Node serial port logs"
}

func (*SerialPortLogParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (*SerialPortLogParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogserialport_contract.SerialPortLogQueryTaskID.Ref()
}

func (*SerialPortLogParser) Grouper() grouper.LogGrouper {
	return grouper.NewSingleStringFieldKeyLogGrouper("resource.labels.instance_id")
}

// Parse implements parsertask.Parser.
func (*SerialPortLogParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	nodeName := l.ReadStringOrDefault("labels.compute\\.googleapis\\.com/resource_name", "unknown")
	mainMessageFieldSet := log.MustGetFieldSet(l, &log.MainMessageFieldSet{})
	escapedMainMessage := logutil.ConvertSpecialSequences(mainMessageFieldSet.MainMessage, serialportSequenceConverters...)
	serialPortResourcePath := resourcepath.NodeSerialport(nodeName)
	cs.RecordEvent(serialPortResourcePath)
	cs.RecordLogSummary(escapedMainMessage)
	return nil
}

var _ legacyparser.Parser = (*SerialPortLogParser)(nil)

var GKESerialPortLogParseTask = legacyparser.NewParserTaskFromParser(googlecloudlogserialport_contract.SerialPortLogParserTaskID, &SerialPortLogParser{}, 10000, false, googlecloudinspectiontypegroup_contract.GKEBasedClusterInspectionTypes)
