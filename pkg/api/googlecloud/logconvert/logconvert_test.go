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
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/genproto/googleapis/cloud/audit"
	ltype "google.golang.org/genproto/googleapis/logging/type"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestLogEntryToNode(t *testing.T) {
	now := time.Now()
	nowpb := timestamppb.New(now)
	nowFormatted := now.UTC().Format("2006-01-02T15:04:05Z")

	protoPayload, err := anypb.New(&audit.AuditLog{
		ServiceName: "foo.com",
	})
	if err != nil {
		t.Fatalf("failed to generate anypb from the dummy protoPayload: %v", err)
	}

	testCases := []struct {
		name     string
		logEntry *loggingpb.LogEntry
		want     structured.Node
		wantErr  bool
	}{
		{
			name: "simple text payload",
			logEntry: &loggingpb.LogEntry{
				InsertId:  "insert-1",
				LogName:   "projects/my-project/logs/my-log",
				Timestamp: nowpb,
				Payload:   &loggingpb.LogEntry_TextPayload{TextPayload: "hello world"},
			},
			want: structured.NewStandardMap(
				[]string{"insertId", "logName", "textPayload", "timestamp"},
				[]structured.Node{
					structured.NewStandardScalarNode("insert-1"),
					structured.NewStandardScalarNode("projects/my-project/logs/my-log"),
					structured.NewStandardScalarNode("hello world"),
					structured.NewStandardScalarNode(nowFormatted),
				},
			),
		},
		{
			name: "json payload",
			logEntry: &loggingpb.LogEntry{
				InsertId: "insert-2",
				Payload: &loggingpb.LogEntry_JsonPayload{
					JsonPayload: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"message": structpb.NewStringValue("hello json"),
							"user":    structpb.NewStringValue("test-user"),
						},
					},
				},
				LogName:   "projects/my-project/logs/my-log",
				Timestamp: nowpb,
			},
			want: structured.NewStandardMap(
				[]string{"insertId", "logName", "jsonPayload", "timestamp"},
				[]structured.Node{
					structured.NewStandardScalarNode("insert-2"),
					structured.NewStandardScalarNode("projects/my-project/logs/my-log"),
					structured.NewStandardMap(
						[]string{"message", "user"},
						[]structured.Node{
							structured.NewStandardScalarNode("hello json"),
							structured.NewStandardScalarNode("test-user"),
						},
					),
					structured.NewStandardScalarNode(nowFormatted),
				},
			),
		},
		{
			name: "proto payload",
			logEntry: &loggingpb.LogEntry{
				InsertId: "insert-3",
				Payload: &loggingpb.LogEntry_ProtoPayload{
					ProtoPayload: protoPayload,
				},
				LogName:   "projects/my-project/logs/my-log",
				Timestamp: nowpb,
			},
			want: structured.NewStandardMap(
				[]string{"insertId", "logName", "protoPayload", "timestamp"},
				[]structured.Node{
					structured.NewStandardScalarNode("insert-3"),
					structured.NewStandardScalarNode("projects/my-project/logs/my-log"),
					structured.NewStandardMap(
						[]string{"@type", "serviceName"},
						[]structured.Node{
							structured.NewStandardScalarNode("type.googleapis.com/google.cloud.audit.AuditLog"),
							structured.NewStandardScalarNode("foo.com"),
						},
					),
					structured.NewStandardScalarNode(nowFormatted),
				},
			),
		},
		{
			name: "with all fields",
			logEntry: &loggingpb.LogEntry{
				InsertId: "full-entry",
				Labels: map[string]string{
					"b_label": "val2",
					"a_label": "val1",
				},
				LogName: "projects/p/logs/l",
				Operation: &loggingpb.LogEntryOperation{
					Id: "op-1",
				},
				HttpRequest: &ltype.HttpRequest{
					RequestMethod: "GET",
				},
				Payload: &loggingpb.LogEntry_TextPayload{
					TextPayload: "full text",
				},
				Resource: &monitoredres.MonitoredResource{
					Type: "gce_instance",
					Labels: map[string]string{
						"zone": "us-central1-a",
					},
				},
				Severity:         ltype.LogSeverity_INFO,
				ReceiveTimestamp: nowpb,
				Timestamp:        nowpb,
				Trace:            "trace-id",
				SpanId:           "span-id",
				TraceSampled:     true,
				SourceLocation: &loggingpb.LogEntrySourceLocation{
					File: "main.go",
				},
				Split: &loggingpb.LogSplit{
					Uid: "split-uid",
				},
			},
			want: structured.NewStandardMap(
				[]string{
					"insertId", "logName", "labels", "operation", "httpRequest",
					"textPayload", "resource", "severity", "receiveTimestamp",
					"timestamp", "trace", "spanId", "traceSampled", "sourceLocation", "split",
				},
				[]structured.Node{
					structured.NewStandardScalarNode("full-entry"),
					structured.NewStandardScalarNode("projects/p/logs/l"),
					structured.NewStandardMap(
						[]string{"a_label", "b_label"},
						[]structured.Node{
							structured.NewStandardScalarNode("val1"),
							structured.NewStandardScalarNode("val2"),
						},
					),
					structured.NewStandardMap([]string{"id"}, []structured.Node{structured.NewStandardScalarNode("op-1")}),
					structured.NewStandardMap([]string{"requestMethod"}, []structured.Node{structured.NewStandardScalarNode("GET")}),
					structured.NewStandardScalarNode("full text"),
					structured.NewStandardMap([]string{"type", "labels"}, []structured.Node{structured.NewStandardScalarNode("gce_instance"),
						structured.NewStandardMap([]string{"zone"}, []structured.Node{structured.NewStandardScalarNode("us-central1-a")})}),
					structured.NewStandardScalarNode("INFO"),
					structured.NewStandardScalarNode(nowFormatted),
					structured.NewStandardScalarNode(nowFormatted),
					structured.NewStandardScalarNode("trace-id"),
					structured.NewStandardScalarNode("span-id"),
					structured.NewStandardScalarNode(true),
					structured.NewStandardMap([]string{"file"}, []structured.Node{structured.NewStandardScalarNode("main.go")}),
					structured.NewStandardMap([]string{"uid"}, []structured.Node{structured.NewStandardScalarNode("split-uid")}),
				},
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LogEntryToNode(tc.logEntry)
			if (err != nil) != tc.wantErr {
				t.Fatalf("LogEntryToNode() error = %v, wantErr %v", err, tc.wantErr)
			}

			serializer := structured.YAMLNodeSerializer{}
			gotYAML, err := serializer.Serialize(got)
			if err != nil {
				t.Fatalf("yaml serialization failed for got node: %v", err)
			}
			wantYAML, err := serializer.Serialize(tc.want)
			if err != nil {
				t.Fatalf("yaml serialization failed for want node: %v", err)
			}
			if diff := cmp.Diff(gotYAML, wantYAML); diff != "" {
				t.Errorf("LogEntryToNode() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestGetLogLabelsMap(t *testing.T) {
	testCases := []struct {
		name   string
		labels map[string]string
		want   structured.Node
	}{
		{
			name:   "empty map",
			labels: map[string]string{},
			want:   structured.NewStandardMap([]string{}, []structured.Node{}),
		},
		{
			name:   "single entry",
			labels: map[string]string{"key": "value"},
			want: structured.NewStandardMap(
				[]string{"key"},
				[]structured.Node{structured.NewStandardScalarNode("value")},
			),
		},
		{
			name: "multiple entries, should be sorted",
			labels: map[string]string{
				"zeta":  "3",
				"beta":  "2",
				"alpha": "1",
			},
			want: structured.NewStandardMap(
				[]string{"alpha", "beta", "zeta"},
				[]structured.Node{
					structured.NewStandardScalarNode("1"),
					structured.NewStandardScalarNode("2"),
					structured.NewStandardScalarNode("3"),
				},
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getLogLabelsMap(tc.labels)
			if err != nil {
				t.Fatalf("getLogLabelsMap() failed: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("getLogLabelsMap() got = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestProtoTimestampToScalar(t *testing.T) {
	tm := time.Date(2023, 10, 26, 10, 30, 0, 0, time.UTC)
	ts := timestamppb.New(tm)
	want := structured.NewStandardScalarNode("2023-10-26T10:30:00Z")

	got := protoTimestampToScalar(ts)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("protoTimestampToScalar() got = %v, want %v", got, want)
	}
}
