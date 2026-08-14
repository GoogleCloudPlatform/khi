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

package logconvert

import (
	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var protojsonMarshalOptions = protojson.MarshalOptions{
	Multiline:       false,
	Resolver:        protoregistry.GlobalTypes,
	EmitUnpopulated: false,
}

// GCPLogEntryKeyOrder defines the canonical top-level field order for GCP Cloud Logging entries when serializing to YAML.
var GCPLogEntryKeyOrder = []string{
	"insertId",
	"logName",
	"trace",
	"spanId",
	"traceSampled",
	"sourceLocation",
	"split",
	"labels",
	"operation",
	"httpRequest",
	"protoPayload",
	"jsonPayload",
	"textPayload",
	"resource",
	"receiveTimestamp",
	"timestamp",
	"severity",
}

// LogEntryToNode converts a Google Cloud Logging LogEntry protobuf message into a structured.Node.
// It directly marshals the LogEntry into JSON bytes via protojson and wraps it in a LazyJSONNode.
func LogEntryToNode(l *loggingpb.LogEntry) (structured.Node, error) {
	jsonBytes, err := protojsonMarshalOptions.Marshal(l)
	if err != nil {
		return nil, err
	}
	return structured.NewLazyJSONNodeFromBytes(jsonBytes), nil
}
