package structured

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type childItem struct {
	Index int
	Key   string
	Value any
}

func getChildren(node Node) ([]childItem, error) {
	var children []childItem
	var err error
	node.Children()(func(key NodeChildrenKey, value Node) bool {
		var val any
		if value.Type() == ScalarNodeType {
			val, err = value.NodeScalarValue()
			if err != nil {
				return false
			}
		} else {
			val = value.Type()
		}
		children = append(children, childItem{
			Index: key.Index,
			Key:   key.Key,
			Value: val,
		})
		return true
	})
	return children, err
}

func TestFieldFilterNode(t *testing.T) {
	orderProvider := &AlphabeticalGoMapKeyOrderProvider{}

	testCases := []struct {
		name          string
		originalVal   any
		ignoredKeys   []string
		wantType      NodeType
		wantScalar    any
		wantScalarErr bool
		wantLen       int
		wantChildren  []childItem
	}{
		{
			name:          "scalar node - no filtering",
			originalVal:   "hello",
			ignoredKeys:   []string{"hello"},
			wantType:      ScalarNodeType,
			wantScalar:    "hello",
			wantScalarErr: false,
			wantLen:       0,
			wantChildren:  nil,
		},
		{
			name:          "map node - filter one key",
			originalVal:   map[string]any{"a": 1, "b": 2, "c": 3},
			ignoredKeys:   []string{"b"},
			wantType:      MapNodeType,
			wantScalar:    nil,
			wantScalarErr: true,
			wantLen:       2,
			wantChildren: []childItem{
				{Index: 0, Key: "a", Value: 1},
				{Index: 1, Key: "c", Value: 3},
			},
		},
		{
			name:          "map node - filter multiple keys",
			originalVal:   map[string]any{"a": 1, "b": 2, "c": 3},
			ignoredKeys:   []string{"a", "c"},
			wantType:      MapNodeType,
			wantScalar:    nil,
			wantScalarErr: true,
			wantLen:       1,
			wantChildren: []childItem{
				{Index: 0, Key: "b", Value: 2},
			},
		},
		{
			name:          "map node - filter non-existent key",
			originalVal:   map[string]any{"a": 1, "b": 2},
			ignoredKeys:   []string{"non-existent"},
			wantType:      MapNodeType,
			wantScalar:    nil,
			wantScalarErr: true,
			wantLen:       2,
			wantChildren: []childItem{
				{Index: 0, Key: "a", Value: 1},
				{Index: 1, Key: "b", Value: 2},
			},
		},
		{
			name:          "sequence node - ignored key is ignored on sequences",
			originalVal:   []any{"x", "y"},
			ignoredKeys:   []string{"x"},
			wantType:      SequenceNodeType,
			wantScalar:    nil,
			wantScalarErr: true,
			wantLen:       2,
			wantChildren: []childItem{
				{Index: 0, Key: "", Value: "x"},
				{Index: 1, Key: "", Value: "y"},
			},
		},
		{
			name:          "sequence node - filter with empty key",
			originalVal:   []any{"x", "y"},
			ignoredKeys:   []string{""},
			wantType:      SequenceNodeType,
			wantScalar:    nil,
			wantScalarErr: true,
			wantLen:       0,
			wantChildren:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalNode, err := FromGoValue(tc.originalVal, orderProvider)
			if err != nil {
				t.Fatalf("FromGoValue failed: %v", err)
			}

			filterNode := NewFieldFilterNode(originalNode, tc.ignoredKeys)

			if gotType := filterNode.Type(); gotType != tc.wantType {
				t.Errorf("Type() mismatch: got %v, want %v", gotType, tc.wantType)
			}

			gotScalar, gotScalarErr := filterNode.NodeScalarValue()
			if tc.wantScalarErr {
				if gotScalarErr == nil {
					t.Errorf("NodeScalarValue() expected error, got nil")
				}
			} else {
				if gotScalarErr != nil {
					t.Errorf("NodeScalarValue() unexpected error: %v", gotScalarErr)
				}
				if diff := cmp.Diff(tc.wantScalar, gotScalar); diff != "" {
					t.Errorf("NodeScalarValue() mismatch (-want +got):\n%s", diff)
				}
			}

			if gotLen := filterNode.Len(); gotLen != tc.wantLen {
				t.Errorf("Len() mismatch: got %d, want %d", gotLen, tc.wantLen)
			}

			gotChildren, err := getChildren(filterNode)
			if err != nil {
				t.Fatalf("failed to get children: %v", err)
			}

			if diff := cmp.Diff(tc.wantChildren, gotChildren); diff != "" {
				t.Errorf("Children() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
