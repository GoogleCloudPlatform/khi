// Copyright 2026 Google LLC
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

package khifilev6

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestToInternedStruct(t *testing.T) {
	testCases := []struct {
		name    string
		yaml    string
		want    func(pool *InternPool) *pb.InternedStruct
		wantErr bool
	}{
		{
			name: "simple map",
			yaml: `
foo: value1
bar: 123
`,
			want: func(pool *InternPool) *pb.InternedStruct {
				return &pb.InternedStruct{
					FieldPathSetId: &pool.InternFieldSet([]string{"foo", "bar"}).id,
					Values: []*pb.InternedValue{
						{
							Kind: &pb.InternedValue_StringValue{
								StringValue: pool.InternString("value1").id,
							},
						},
						{
							Kind: &pb.InternedValue_Int64Value{
								Int64Value: 123,
							},
						},
					},
				}
			},
		},
		{
			name: "nested map and list",
			yaml: `
map:
  key: true
list:
  - null
  - hello
`,
			want: func(pool *InternPool) *pb.InternedStruct {
				return &pb.InternedStruct{
					FieldPathSetId: &pool.InternFieldSet([]string{"map\x00key", "list"}).id,
					Values: []*pb.InternedValue{
						{
							Kind: &pb.InternedValue_BoolValue{
								BoolValue: true,
							},
						},
						{
							Kind: &pb.InternedValue_ListValue{
								ListValue: &pb.InternedListValue{
									Values: []*pb.InternedValue{
										{
											Kind: &pb.InternedValue_NullValue{
												NullValue: structpb.NullValue_NULL_VALUE,
											},
										},
										{
											Kind: &pb.InternedValue_StringValue{
												StringValue: pool.InternString("hello").id,
											},
										},
									},
								},
							},
						},
					},
				}
			},
		},
		{
			name: "map inside list",
			yaml: `
list:
  - a: 1
    b: 2
`,
			want: func(pool *InternPool) *pb.InternedStruct {
				return &pb.InternedStruct{
					FieldPathSetId: &pool.InternFieldSet([]string{"list"}).id,
					Values: []*pb.InternedValue{
						{
							Kind: &pb.InternedValue_ListValue{
								ListValue: &pb.InternedListValue{
									Values: []*pb.InternedValue{
										{
											Kind: &pb.InternedValue_StructValue{
												StructValue: &pb.InternedStruct{
													FieldPathSetId: &pool.InternFieldSet([]string{"a", "b"}).id,
													Values: []*pb.InternedValue{
														{
															Kind: &pb.InternedValue_Int64Value{
																Int64Value: 1,
															},
														},
														{
															Kind: &pb.InternedValue_Int64Value{
																Int64Value: 2,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}
			},
		},
		{
			name: "empty map",
			yaml: `
empty_map: {}
`,
			want: func(pool *InternPool) *pb.InternedStruct {
				return &pb.InternedStruct{
					FieldPathSetId: &pool.InternFieldSet([]string{"empty_map"}).id,
					Values: []*pb.InternedValue{
						{
							Kind: &pb.InternedValue_StructValue{
								StructValue: &pb.InternedStruct{
									FieldPathSetId: &pool.InternFieldSet([]string{}).id,
									Values:         nil,
								},
							},
						},
					},
				}
			},
		},
		{
			name: "float and time",
			yaml: `
float_val: 1.23
time_val: 2026-04-20T03:00:00Z
`,
			want: func(pool *InternPool) *pb.InternedStruct {
				t, _ := time.Parse(time.RFC3339, "2026-04-20T03:00:00Z")
				return &pb.InternedStruct{
					FieldPathSetId: &pool.InternFieldSet([]string{"float_val", "time_val"}).id,
					Values: []*pb.InternedValue{
						{
							Kind: &pb.InternedValue_DoubleValue{
								DoubleValue: 1.23,
							},
						},
						{
							Kind: &pb.InternedValue_TimestampValue{
								TimestampValue: timestamppb.New(t),
							},
						},
					},
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idGenGot := &IDGenerator{}
			poolGot := NewInternPool(idGenGot)

			node, err := structured.FromYAML(tc.yaml)
			if err != nil {
				t.Fatalf("FromYAML() error = %v", err)
			}

			got, err := ToInternedStruct(node, poolGot)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ToInternedStruct() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}

			idGenWant := &IDGenerator{}
			poolWant := NewInternPool(idGenWant)
			want := tc.want(poolWant)

			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("ToInternedStruct() mismatch (-want +got):\n%s", diff)
			}

			// Round-trip test
			reverted, err := FromInternedStruct(got, poolGot)
			if err != nil {
				t.Fatalf("FromInternedStruct() error = %v", err)
			}

			serializer := &structured.JSONNodeSerializer{}
			origJSON, err := serializer.Serialize(node)
			if err != nil {
				t.Fatalf("Failed to serialize original node: %v", err)
			}
			revJSON, err := serializer.Serialize(reverted)
			if err != nil {
				t.Fatalf("Failed to serialize reverted node: %v", err)
			}

			if diff := cmp.Diff(string(origJSON), string(revJSON)); diff != "" {
				t.Errorf("Round-trip mismatch (-orig +rev):\n%s", diff)
			}
		})
	}
}

func TestFromInternedValue(t *testing.T) {
	idGen := &IDGenerator{}
	pool := NewInternPool(idGen)
	helloStrID := pool.InternString("hello").id
	timeVal := time.Date(2026, 4, 20, 3, 0, 0, 0, time.UTC)

	testCases := []struct {
		name    string
		value   *pb.InternedValue
		want    func() structured.Node
		wantErr bool
	}{
		{
			name:    "nil value",
			value:   nil,
			want:    func() structured.Node { return nil },
			wantErr: true,
		},
		{
			name:  "null value",
			value: &pb.InternedValue{Kind: &pb.InternedValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
			want:  func() structured.Node { return structured.NewStandardScalarNode[any](nil) },
		},
		{
			name:  "bool value",
			value: &pb.InternedValue{Kind: &pb.InternedValue_BoolValue{BoolValue: true}},
			want:  func() structured.Node { return structured.NewStandardScalarNode(true) },
		},
		{
			name:  "string value",
			value: &pb.InternedValue{Kind: &pb.InternedValue_StringValue{StringValue: helloStrID}},
			want:  func() structured.Node { return structured.NewStandardScalarNode("hello") },
		},
		{
			name:  "int64 value",
			value: &pb.InternedValue{Kind: &pb.InternedValue_Int64Value{Int64Value: 123}},
			want:  func() structured.Node { return structured.NewStandardScalarNode(int(123)) },
		},
		{
			name:  "double value",
			value: &pb.InternedValue{Kind: &pb.InternedValue_DoubleValue{DoubleValue: 1.23}},
			want:  func() structured.Node { return structured.NewStandardScalarNode(1.23) },
		},
		{
			name:  "timestamp value",
			value: &pb.InternedValue{Kind: &pb.InternedValue_TimestampValue{TimestampValue: timestamppb.New(timeVal)}},
			want:  func() structured.Node { return structured.NewStandardScalarNode(timeVal) },
		},
		{
			name: "list value",
			value: &pb.InternedValue{
				Kind: &pb.InternedValue_ListValue{
					ListValue: &pb.InternedListValue{
						Values: []*pb.InternedValue{
							{Kind: &pb.InternedValue_Int64Value{Int64Value: 1}},
							{Kind: &pb.InternedValue_Int64Value{Int64Value: 2}},
						},
					},
				},
			},
			want: func() structured.Node {
				return structured.NewStandardSequenceNode([]structured.Node{
					structured.NewStandardScalarNode(int(1)),
					structured.NewStandardScalarNode(int(2)),
				})
			},
		},
		{
			name: "struct value",
			value: &pb.InternedValue{
				Kind: &pb.InternedValue_StructValue{
					StructValue: &pb.InternedStruct{
						FieldPathSetId: &pool.InternFieldSet([]string{"a"}).id,
						Values: []*pb.InternedValue{
							{Kind: &pb.InternedValue_Int64Value{Int64Value: 1}},
						},
					},
				},
			},
			want: func() structured.Node {
				return structured.NewStandardMap([]string{"a"}, []structured.Node{
					structured.NewStandardScalarNode(int(1)),
				})
			},
		},
		{
			name:    "unknown kind",
			value:   &pb.InternedValue{Kind: nil},
			want:    func() structured.Node { return nil },
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := FromInternedValue(tc.value, pool)
			if (err != nil) != tc.wantErr {
				t.Fatalf("FromInternedValue() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}

			serializer := &structured.JSONNodeSerializer{}
			gotJSON, _ := serializer.Serialize(got)
			wantJSON, _ := serializer.Serialize(tc.want())
			if diff := cmp.Diff(string(wantJSON), string(gotJSON)); diff != "" {
				t.Errorf("FromInternedValue() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFlattenNode(t *testing.T) {
	testCases := []struct {
		name       string
		yaml       string
		wantKeys   []string
		wantValues []string // JSON representations of the values for easy comparison
		wantErr    bool
	}{
		{
			name: "simple map",
			yaml: `
a: 1
b: 2
`,
			wantKeys:   []string{"a", "b"},
			wantValues: []string{"1", "2"},
		},
		{
			name: "nested map",
			yaml: `
a:
  b: 1
  c: 2
d: 3
`,
			wantKeys:   []string{"a\x00b", "a\x00c", "d"},
			wantValues: []string{"1", "2", "3"},
		},
		{
			name: "map with sequence",
			yaml: `
a:
  - 1
  - 2
b: 3
`,
			wantKeys:   []string{"a", "b"},
			wantValues: []string{"[1,2]", "3"},
		},
		{
			name: "empty map",
			yaml: `
a: {}
b: 3
`,
			wantKeys:   []string{"a", "b"},
			wantValues: []string{"{}", "3"},
		},
		{
			name: "empty key nested",
			yaml: `
"":
  a: 1
`,
			wantKeys:   []string{"\x00a"},
			wantValues: []string{"1"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := structured.FromYAML(tc.yaml)
			if err != nil {
				t.Fatalf("FromYAML() error = %v", err)
			}

			var gotKeys []string
			var gotValues []structured.Node
			err = flattenNode(node, "", true, &gotKeys, &gotValues)
			if (err != nil) != tc.wantErr {
				t.Fatalf("flattenNode() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tc.wantKeys, gotKeys); diff != "" {
				t.Errorf("flattenNode() keys mismatch (-want +got):\n%s", diff)
			}

			var gotValuesJSON []string
			serializer := &structured.JSONNodeSerializer{}
			for _, v := range gotValues {
				j, _ := serializer.Serialize(v)
				gotValuesJSON = append(gotValuesJSON, string(j))
			}

			if diff := cmp.Diff(tc.wantValues, gotValuesJSON); diff != "" {
				t.Errorf("flattenNode() values mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUnflattenNodes(t *testing.T) {
	testCases := []struct {
		name    string
		keys    []string
		values  []structured.Node
		want    func() structured.Node
		wantErr bool
	}{
		{
			name:   "simple map",
			keys:   []string{"a", "b"},
			values: []structured.Node{structured.NewStandardScalarNode(1), structured.NewStandardScalarNode(2)},
			want: func() structured.Node {
				return structured.NewStandardMap(
					[]string{"a", "b"},
					[]structured.Node{structured.NewStandardScalarNode(1), structured.NewStandardScalarNode(2)},
				)
			},
		},
		{
			name:   "nested map",
			keys:   []string{"a\x00b", "a\x00c", "d"},
			values: []structured.Node{structured.NewStandardScalarNode(1), structured.NewStandardScalarNode(2), structured.NewStandardScalarNode(3)},
			want: func() structured.Node {
				nested := structured.NewStandardMap(
					[]string{"b", "c"},
					[]structured.Node{structured.NewStandardScalarNode(1), structured.NewStandardScalarNode(2)},
				)
				return structured.NewStandardMap(
					[]string{"a", "d"},
					[]structured.Node{nested, structured.NewStandardScalarNode(3)},
				)
			},
		},
		{
			name:    "conflict keys",
			keys:    []string{"a", "a\x00b"},
			values:  []structured.Node{structured.NewStandardScalarNode(1), structured.NewStandardScalarNode(2)},
			want:    func() structured.Node { return nil },
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := unflattenNodes(tc.keys, tc.values)
			if (err != nil) != tc.wantErr {
				t.Fatalf("unflattenNodes() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}

			serializer := &structured.JSONNodeSerializer{}
			gotJSON, _ := serializer.Serialize(got)
			wantJSON, _ := serializer.Serialize(tc.want())
			if diff := cmp.Diff(string(wantJSON), string(gotJSON)); diff != "" {
				t.Errorf("unflattenNodes() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
