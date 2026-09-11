package coretask

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

func TestWithTaskDescription(t *testing.T) {
	testCases := []struct {
		name        string
		description string
		want        string
	}{
		{
			name:        "sets simple task description",
			description: "Parses Kubernetes audit log entries.",
			want:        "Parses Kubernetes audit log entries.",
		},
		{
			name:        "sets empty task description",
			description: "",
			want:        "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			labels := NewLabelSet(WithTaskDescription(tc.description))
			got, found := typedmap.Get(labels, LabelKeyTaskDescription)
			if !found {
				t.Errorf("LabelKeyTaskDescription not found in label set")
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
