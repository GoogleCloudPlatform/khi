package cel

import (
	"testing"

	"github.com/google/cel-go/cel"
)

func TestPathMap(t *testing.T) {
	env, err := cel.NewEnv(cel.Variable("path", cel.MapType(cel.StringType, cel.StringType)))
	if err != nil {
		t.Fatalf("failed to create CEL env: %v", err)
	}
	levels := map[string]string{
		"namespace": "default",
		"kind":      "Pod",
	}

	testCases := []struct {
		name       string
		expression string
		want       bool
		wantErr    bool
	}{
		{
			name:       "index with existing level",
			expression: `path["namespace"] == "default"`,
			want:       true,
		},
		{
			name:       "index with missing level evaluates to empty string",
			expression: `path["pod"] == ""`,
			want:       true,
		},
		{
			name:       "index with missing level does not match a name",
			expression: `path["pod"] == "pod-a"`,
			want:       false,
		},
		{
			name:       "field selection with missing level evaluates to empty string",
			expression: `path.pod == ""`,
			want:       true,
		},
		{
			name:       "has with existing level",
			expression: `has(path.kind)`,
			want:       true,
		},
		{
			name:       "has with missing level",
			expression: `has(path.pod)`,
			want:       false,
		},
		{
			name:       "in with existing level",
			expression: `"namespace" in path`,
			want:       true,
		},
		{
			name:       "in with missing level",
			expression: `"pod" in path`,
			want:       false,
		},
		{
			name:       "size counts only existing levels",
			expression: `size(path) == 2`,
			want:       true,
		},
		{
			name:       "exists iterates over existing levels",
			expression: `path.exists(k, path[k] == "Pod")`,
			want:       true,
		},
		{
			name:       "equality with map literal",
			expression: `path == {"namespace": "default", "kind": "Pod"}`,
			want:       true,
		},
		{
			name:       "index with non-string key is an error",
			expression: `path[dyn(1)] == ""`,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, iss := env.Compile(tc.expression)
			if iss.Err() != nil {
				t.Fatalf("failed to compile %q: %v", tc.expression, iss.Err())
			}
			prg, err := env.Program(ast)
			if err != nil {
				t.Fatalf("failed to build program for %q: %v", tc.expression, err)
			}
			out, _, err := prg.Eval(map[string]any{"path": newPathMap(levels)})
			if (err != nil) != tc.wantErr {
				t.Fatalf("Eval(%q) error = %v, wantErr = %v", tc.expression, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			got, ok := out.Value().(bool)
			if !ok {
				t.Fatalf("Eval(%q) = %v, want bool", tc.expression, out)
			}
			if got != tc.want {
				t.Errorf("Eval(%q) = %v, want %v", tc.expression, got, tc.want)
			}
		})
	}
}
