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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/ioconfig"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	log_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/log"
	"github.com/google/go-cmp/cmp"

	"github.com/google/go-cmp/cmp/cmpopts"
)

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

		builder := NewBuilder(&ioconfig.IOConfig{TemporaryFolder: "/tmp/"})
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
		builder := NewBuilder(&ioconfig.IOConfig{TemporaryFolder: "/tmp/"})
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
		builder := NewBuilder(&ioconfig.IOConfig{TemporaryFolder: "/tmp"})

		log, err := builder.GetLog("non-existing-id")

		if err == nil {
			t.Errorf("Expected an error but nothing returned as an error")
		}
		if log != nil {
			t.Errorf("Expected log to be nil but found a log")
		}
	})

	t.Run("returns an log when the specified log id was found", func(t *testing.T) {
		builder := NewBuilder(&ioconfig.IOConfig{TemporaryFolder: "/tmp"})
		builder.PrepareParseLogs(context.Background(), []*log.LogEntity{
			log_test.MustLogEntity(`insertId: foo
severity: INFO
textPayload: fooTextPayload
timestamp: "2024-01-01T00:00:00Z"`),
		}, func() {})

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
		ExpectedSeverity  enum.Severity
	}{
		{
			Name: "Must fill the default parameters for SerializableLog",
			LogBody: `insertId: foo
severity: INFO
textPayload: fooTextPayload
timestamp: "2024-01-01T00:00:00Z"`,
			ExpectedDisplayId: "foo",
			ExpectedLogType:   enum.LogTypeUnknown,
			ExpectedSeverity:  enum.SeverityInfo,
		},
		{
			Name: "Set the unknown severity when the given severity is not supported without returning an error",
			LogBody: `insertId: foo
severity: FOOOOOOO
textPayload: fooTextPayload
timestamp: "2024-01-01T00:00:00Z"`,
			ExpectedDisplayId: "foo",
			ExpectedLogType:   enum.LogTypeUnknown,
			ExpectedSeverity:  enum.SeverityUnknown,
		},
	}
	for _, tc := range testCase {
		t.Run(tc.Name, func(t *testing.T) {
			builder := NewBuilder(&ioconfig.IOConfig{TemporaryFolder: "/tmp"})
			builder.PrepareParseLogs(context.Background(), []*log.LogEntity{
				log_test.MustLogEntity(tc.LogBody),
			}, func() {})

			sl := builder.history.Logs[0]

			if sl.DisplayId != tc.ExpectedDisplayId {
				t.Errorf("DisplayId is not matching")
			}
			if sl.Type != tc.ExpectedLogType {
				t.Errorf("LogType is not matching")
			}
			if sl.Severity != tc.ExpectedSeverity {
				t.Errorf("Severity is not matching")
			}

		})
	}
}

func TestGetTimelineBuilder(t *testing.T) {
	t.Run("generates resource histories when it is absent", func(t *testing.T) {
		builder := NewBuilder(&ioconfig.IOConfig{TemporaryFolder: "/tmp"})
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
				t.Errorf(diff)
			}
		})
	}
}

func generateBuilderWithTimelines(resourcePaths []string) *Builder {
	builder := NewBuilder(&ioconfig.IOConfig{TemporaryFolder: "/tmp"})
	for _, resourcePath := range resourcePaths {
		builder.GetTimelineBuilder(resourcePath)
	}
	builder.sortData()
	return builder
}

func TestGetTimelineBuilderThreadSafety(t *testing.T) {
	builder := NewBuilder(&ioconfig.IOConfig{TemporaryFolder: "/tmp"})
	threadCount := 100
	timelineCountPerThread := 1000000
	pool := worker.NewPool(threadCount)
	pool.Run(func() {
		for i := 0; i < timelineCountPerThread; i++ {
			uuid1 := common.NewUUID()
			uuid2 := common.NewUUID()
			uuid3 := common.NewUUID()
			uuid4 := common.NewUUID()
			uuid5 := common.NewUUID()
			builder.GetTimelineBuilder(resourcepath.SubresourceLayerGeneralItem(uuid1[:3], uuid2[:3], uuid3[:3], uuid4[:3], uuid5[:3]).Path)
		}
	})
	pool.Wait()
}
