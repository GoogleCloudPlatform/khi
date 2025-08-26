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

package history

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/idgenerator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"

	"github.com/google/go-cmp/cmp/cmpopts"
)

type testCommonFieldSetReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (t *testCommonFieldSetReader) FieldSetKind() string {
	return (&log.CommonFieldSet{}).Kind()
}

// Read implements log.FieldSetReader.
func (t *testCommonFieldSetReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	result := &log.CommonFieldSet{}
	result.DisplayID = reader.ReadStringOrDefault("insertId", "unknown")
	result.Severity = enum.SeverityUnknown
	ts, err := reader.ReadTimestamp("timestamp")
	if err != nil {
		return nil, fmt.Errorf("failed to read timestmap from given log")
	}
	result.Timestamp = ts
	return result, nil

}

var _ log.FieldSetReader = (*testCommonFieldSetReader)(nil)

func TestHistoryEnsureResourceHistory(t *testing.T) {
	t.Run("generates resource histories when it is absent", func(t *testing.T) {
		want := &History{
			Resources: []*Resource{
				{
					ResourceName: "foo",
					Relationship: enum.RelationshipChild,
					Children: []*Resource{
						{
							ResourceName: "bar",
							Relationship: enum.RelationshipChild,
							Children: []*Resource{
								{
									ResourceName: "qux",
									Relationship: enum.RelationshipChild,
									Children:     []*Resource{},
								},
							},
						},
					},
				},
			},
		}

		builder := NewBuilder("/tmp")
		builder.ensureResourcePath("foo#bar#qux")
		builder.sortData()

		if diff := cmp.Diff(want, builder.history,
			cmpopts.IgnoreFields(History{}, "Logs", "Version", "Timelines"),
			cmpopts.IgnoreFields(ResourceTimeline{}, "Revisions", "Events"),
			cmpopts.IgnoreFields(Resource{}, "FullResourcePath")); diff != "" {
			t.Errorf("(-want,+got)\n%s", diff)
		}
	})

	t.Run("generates resource histories only for absent layer", func(t *testing.T) {
		want := &History{
			Resources: []*Resource{
				{
					ResourceName: "foo",
					Relationship: enum.RelationshipChild,
					Children: []*Resource{
						{
							ResourceName: "bar",
							Relationship: enum.RelationshipChild,
							Children: []*Resource{
								{
									ResourceName: "qux",
									Relationship: enum.RelationshipChild,
									Children:     []*Resource{},
								},
							},
						}, {
							ResourceName: "baz",
							Relationship: enum.RelationshipChild,
							Children: []*Resource{
								{
									ResourceName: "quux",
									Relationship: enum.RelationshipChild,
									Children:     []*Resource{},
								},
							},
						},
					},
				},
			},
		}
		builder := NewBuilder("/tmp")
		builder.ensureResourcePath("foo#bar#qux")

		builder.ensureResourcePath("foo#baz#quux")
		builder.sortData()

		if diff := cmp.Diff(want, builder.history,
			cmpopts.IgnoreFields(History{}, "Logs", "Version", "Timelines"),
			cmpopts.IgnoreFields(ResourceTimeline{}, "Revisions", "Events"),
			cmpopts.IgnoreFields(Resource{}, "FullResourcePath")); diff != "" {
			t.Errorf("(-want, +got)\n%s", diff)
		}
	})
}

func TestGetLog(t *testing.T) {
	t.Run("returns error when the specified log id was not found", func(t *testing.T) {
		builder := NewBuilder("/tmp")

		log, err := builder.GetLog("non-existing-id")

		if err == nil {
			t.Errorf("Expected an error but nothing returned as an error")
		}
		if log != nil {
			t.Errorf("Expected log to be nil but found a log")
		}
	})

	t.Run("returns an log when the specified log id was found", func(t *testing.T) {
		builder := NewBuilder("/tmp")
		err := builder.PrepareParseLogs(context.Background(), []*log.Log{
			testlog.MustLogFromYAML(`insertId: foo
severity: INFO
textPayload: fooTextPayload
timestamp: "2024-01-01T00:00:00Z"`, &testCommonFieldSetReader{}),
		}, func() {})
		if err != nil {
			t.Fatal(err.Error())
		}

		logExpected := builder.history.Logs[0]

		logActual, err := builder.GetLog(logExpected.ID)
		if err != nil {
			t.Errorf("Unexpected error %s", err.Error())
		}
		if logActual != logExpected {
			t.Errorf("Log is not matching")
		}
	})
}

func TestPrepareParseLogs(t *testing.T) {
	testCase := []struct {
		Name              string
		LogBody           string
		ExpectedDisplayId string
		ExpectedLogType   enum.LogType
	}{
		{
			Name: "Must fill the default parameters for SerializableLog",
			LogBody: `insertId: foo
severity: INFO
textPayload: fooTextPayload
timestamp: "2024-01-01T00:00:00Z"`,
			ExpectedDisplayId: "foo",
			ExpectedLogType:   enum.LogTypeUnknown,
		},
	}
	for _, tc := range testCase {
		t.Run(tc.Name, func(t *testing.T) {
			builder := NewBuilder("/tmp")
			err := builder.PrepareParseLogs(context.Background(), []*log.Log{
				testlog.MustLogFromYAML(tc.LogBody, &testCommonFieldSetReader{}),
			}, func() {})
			if err != nil {
				t.Fatal(err.Error())
			}

			sl := builder.history.Logs[0]

			if sl.DisplayId != tc.ExpectedDisplayId {
				t.Errorf("DisplayId is not matching")
			}
			if sl.Type != tc.ExpectedLogType {
				t.Errorf("LogType is not matching")
			}
		})
	}
}

func TestGetTimelineBuilder(t *testing.T) {
	t.Run("generates resource histories when it is absent", func(t *testing.T) {
		builder := NewBuilder("/tmp")
		tb := builder.GetTimelineBuilder("foo#bar#baz")

		if len(tb.builder.history.Timelines) != 1 {
			t.Errorf("Length of timeline doesn't match: expect 1, given %d", len(tb.builder.history.Timelines))
		}

		resource := builder.ensureResourcePath("foo#bar#baz")
		if tb.builder.history.Timelines[0].ID != resource.Timeline {
			t.Errorf("Given timeline ID in Resource is not matching the generated timeline instance")
		}
	})
}

func TestGetChildResources(t *testing.T) {
	testCases := []struct {
		Resources         []string
		ExpectedTimelines []string
		Parent            string
	}{
		{
			Resources: []string{
				"core/v1#pods#default#foo",
				"core/v1#pods#default#bar",
				"core/v1#pods#default#foo#binding",
				"core/v1#pods#default#foo",
				"core/v1#pods#kube-system#qux",
			},
			ExpectedTimelines: []string{
				"core/v1#pods#kube-system",
				"core/v1#pods#default",
			},
			Parent: "core/v1#pods",
		},
		{
			Resources: []string{
				"core/v1#pods#default#foo",
				"core/v1#pods#default#bar",
				"core/v1#pods#default#foo#binding",
				"core/v1#pods#default#foo",
				"core/v1#pods#kube-system#qux",
			},
			ExpectedTimelines: []string{
				"core/v1#pods#default#foo",
				"core/v1#pods#default#bar",
			},
			Parent: "core/v1#pods#default",
		},
		{
			Resources: []string{
				"core/v1#pods#default#foo",
				"core/v1#pods#default#bar",
				"core/v1#pods#default#foo#binding",
				"core/v1#pods#default#foo",
				"core/v1#pods#kube-system#qux",
			},
			ExpectedTimelines: []string{},
			Parent:            "core/v1#pods#non-existing",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Parent, func(t *testing.T) {
			builder := generateBuilderWithTimelines(testCase.Resources)
			resources := builder.GetChildResources(testCase.Parent)
			actualTimelineResourcePaths := []string{}
			for _, resource := range resources {
				actualTimelineResourcePaths = append(actualTimelineResourcePaths, resource.FullResourcePath)
			}
			if diff := cmp.Diff(actualTimelineResourcePaths, testCase.ExpectedTimelines, cmpopts.SortSlices(func(a string, b string) bool {
				return strings.Compare(a, b) > 0
			})); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func generateBuilderWithTimelines(resourcePaths []string) *Builder {
	builder := NewBuilder("/tmp")
	for _, resourcePath := range resourcePaths {
		builder.GetTimelineBuilder(resourcePath)
	}
	builder.sortData()
	return builder
}

func TestGetTimelineBuilderThreadSafety(t *testing.T) {
	builder := NewBuilder("/tmp")
	apiVersionGenerator := idgenerator.NewPrefixIDGenerator("apiversion-")
	kindGenerator := idgenerator.NewPrefixIDGenerator("kind-")
	namespaceGenerator := idgenerator.NewPrefixIDGenerator("namespace-")
	nameGenerator := idgenerator.NewPrefixIDGenerator("name-")
	subresourceGenerator := idgenerator.NewPrefixIDGenerator("subresource-")
	threadCount := 100
	timelineCountPerThread := 100000
	pool := worker.NewPool(threadCount)
	pool.Run(func() {
		for i := 0; i < timelineCountPerThread; i++ {
			randomAPIVersion := apiVersionGenerator.Generate()
			randomKind := kindGenerator.Generate()
			randomNamespace := namespaceGenerator.Generate()
			randomResourceName := nameGenerator.Generate()
			randomSubresource := subresourceGenerator.Generate()
			builder.GetTimelineBuilder(resourcepath.SubresourceLayerGeneralItem(randomAPIVersion, randomKind, randomNamespace, randomResourceName, randomSubresource).Path)
		}
	})
	pool.Wait()
}
