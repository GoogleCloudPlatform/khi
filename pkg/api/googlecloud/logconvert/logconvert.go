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
	"slices"
	"time"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	ltype "google.golang.org/genproto/googleapis/logging/type"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// LogEntryToNode converts a Google Cloud Logging LogEntry protobuf message into a structured.Node.
// It extracts all fields from the LogEntry and organizes them into a map-like structured.Node in defined order.
func LogEntryToNode(l *loggingpb.LogEntry) (structured.Node, error) {
	keys := make([]string, 0)
	values := make([]structured.Node, 0)

	keys = append(keys, "insertId")
	values = append(values, structured.NewStandardScalarNode(l.GetInsertId()))

	keys = append(keys, "logName")
	values = append(values, structured.NewStandardScalarNode(l.GetLogName()))

	if len(l.Labels) > 0 {
		keys = append(keys, "labels")
		labelsMap, err := getLogLabelsMap(l.Labels)
		if err != nil {
			return nil, err
		}
		values = append(values, labelsMap)
	}

	if l.Operation != nil {
		err := addProtoField(&keys, &values, "operation", l.Operation)
		if err != nil {
			return nil, err
		}
	}

	if l.HttpRequest != nil {
		err := addProtoField(&keys, &values, "httpRequest", l.HttpRequest)
		if err != nil {
			return nil, err
		}
	}

	if protoPayload := l.GetProtoPayload(); protoPayload != nil {
		err := addProtoField(&keys, &values, "protoPayload", protoPayload)
		if err != nil {
			return nil, err
		}
	} else if jsonPayload := l.GetJsonPayload(); jsonPayload != nil {
		err := addProtoField(&keys, &values, "jsonPayload", jsonPayload)
		if err != nil {
			return nil, err
		}
	} else if textPayload := l.GetTextPayload(); textPayload != "" {
		keys = append(keys, "textPayload")
		values = append(values, structured.NewStandardScalarNode(textPayload))
	}

	if l.Resource != nil {
		err := addProtoField(&keys, &values, "resource", l.Resource)
		if err != nil {
			return nil, err
		}
	}

	if l.Severity != ltype.LogSeverity_DEFAULT {
		keys = append(keys, "severity")
		values = append(values, structured.NewStandardScalarNode(l.Severity.String()))
	}

	if l.ReceiveTimestamp != nil {
		keys = append(keys, "receiveTimestamp")
		values = append(values, protoTimestampToScalar(l.ReceiveTimestamp))
	}

	if l.Timestamp != nil {
		keys = append(keys, "timestamp")
		values = append(values, protoTimestampToScalar(l.Timestamp))
	}

	if l.Trace != "" {
		keys = append(keys, "trace")
		values = append(values, structured.NewStandardScalarNode(l.Trace))
	}

	if l.SpanId != "" {
		keys = append(keys, "spanId")
		values = append(values, structured.NewStandardScalarNode(l.SpanId))
	}

	if l.Trace != "" || l.SpanId != "" {
		keys = append(keys, "traceSampled")
		values = append(values, structured.NewStandardScalarNode(l.TraceSampled))
	}

	if l.SourceLocation != nil {
		err := addProtoField(&keys, &values, "sourceLocation", l.SourceLocation)
		if err != nil {
			return nil, err
		}
	}

	if l.Split != nil {
		err := addProtoField(&keys, &values, "split", l.Split)
		if err != nil {
			return nil, err
		}
	}

	return structured.NewStandardMap(keys, values), nil
}

// getLogLabelsMap converts a map of string labels into a structured.Node representing a map.
// The keys in the resulting structured.Node are sorted alphabetically.
func getLogLabelsMap(l map[string]string) (structured.Node, error) {
	keys := make([]string, 0, len(l))
	for k := range l {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	values := make([]structured.Node, 0, len(l))
	for _, k := range keys {
		values = append(values, structured.NewStandardScalarNode(l[k]))
	}
	return structured.NewStandardMap(keys, values), nil
}

// addProtoField adds a map node parsed from given proto data to given key and value pointers.
// Users must assert p is not nil before calling this func.
func addProtoField(keys *[]string, values *[]structured.Node, key string, p proto.Message) error {
	node, err := protoToMapNode(p)
	if err != nil {
		return err
	}
	*keys = append(*keys, key)
	*values = append(*values, node)
	return nil
}

// protoToMapNode converts a protobuf message into a structured.Node.
// It marshals the protobuf message to JSON using protojson and then parses the JSON string
// into a structured.Node using structured.FromYAML.
func protoToMapNode(protoAny proto.Message) (structured.Node, error) {
	opt := protojson.MarshalOptions{
		Multiline: false,
		Resolver:  protoregistry.GlobalTypes,
	}
	jsonBytes, err := opt.Marshal(protoAny)
	if err != nil {
		return nil, err
	}
	return structured.FromYAML(string(jsonBytes))
}

// protoTimestampToScalar converts a timestamppb.Timestamp protobuf message into a structured.Node
// containing a scalar string representation of the timestamp in RFC3339Nano format.
func protoTimestampToScalar(pbTime *timestamppb.Timestamp) structured.Node {
	return structured.NewStandardScalarNode(pbTime.AsTime().UTC().Format(time.RFC3339Nano))
}
