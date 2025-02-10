package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToGithubAnchorHash(t *testing.T) {
	testCases := []struct {
		input string
		want  string
	}{
		{
			input: "Simple Text",
			want:  "simple-text",
		},
		{
			input: "Simple_Text",
			want:  "simpletext",
		},
		{
			input: " simple text ",
			want:  "simple-text",
		},
		{
			input: "Simple(Text)",
			want:  "simpletext",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual := ToGithubAnchorHash(tc.input)
			assert.Equal(t, tc.want, actual)
		})
	}
}
