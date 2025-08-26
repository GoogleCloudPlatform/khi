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

package flag

import (
	"flag"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func setCommandlineArguments(t *testing.T, arguments []string) {
	t.Helper()
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{os.Args[0]}
	os.Args = append(os.Args, arguments...)
}

func TestString(t *testing.T) {
	testCases := []struct {
		name      string
		cmdArgKey string
		envKey    string
		value     string
		cmdArgs   []string
		before    func()
		after     func()
		want      string
	}{
		{
			name:      "from command line argument",
			cmdArgKey: "foo",
			envKey:    "",
			value:     "bar",
			cmdArgs:   []string{"--foo=baz"},
			before:    func() {},
			after:     func() {},
			want:      "baz",
		},
		{
			name:      "from environment variable",
			cmdArgKey: "foo",
			envKey:    "FOO",
			value:     "bar",
			cmdArgs:   []string{},
			before: func() {
				os.Setenv("FOO", "baz")
			},
			after: func() {
				os.Unsetenv("FOO")
			},
			want: "baz",
		},
		{
			name:      "both provided, command line argument is prioritized",
			cmdArgKey: "foo",
			envKey:    "FOO",
			value:     "bar",
			cmdArgs:   []string{"--foo=baz"},
			before: func() {
				os.Setenv("FOO", "qux")
			},
			after: func() {
				os.Unsetenv("FOO")
			},
			want: "baz",
		},
		{
			name:      "default value",
			cmdArgKey: "foo",
			envKey:    "",
			value:     "bar",
			cmdArgs:   []string{},
			before:    func() {},
			after:     func() {},
			want:      "bar",
		},
		{
			name:      "empty environment variable",
			cmdArgKey: "foo",
			envKey:    "FOO",
			value:     "bar",
			cmdArgs:   []string{},
			before: func() {
				os.Setenv("FOO", "")
			},
			after: func() {
				os.Unsetenv("FOO")
			},
			want: "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			defer tc.after()
			defer Reset()
			setCommandlineArguments(t, tc.cmdArgs)
			gotPointer := String(tc.cmdArgKey, tc.value, "", tc.envKey)
			err := Parse()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, *gotPointer); diff != "" {
				t.Errorf("unexpected result (-want +got)\n%s", diff)
			}
		})
	}
}

func TestBool(t *testing.T) {
	testCases := []struct {
		name      string
		cmdArgKey string
		envKey    string
		value     bool
		cmdArgs   []string
		before    func()
		after     func()
		want      bool
	}{
		{
			name:      "from command line argument",
			cmdArgKey: "foo",
			envKey:    "",
			value:     false,
			cmdArgs:   []string{"--foo"},
			before:    func() {},
			after:     func() {},
			want:      true,
		},
		{
			name:      "from environment variable",
			cmdArgKey: "foo",
			envKey:    "FOO",
			value:     false,
			cmdArgs:   []string{},
			before: func() {
				os.Setenv("FOO", "true")
			},
			after: func() {
				os.Unsetenv("FOO")
			},
			want: true,
		},
		{
			name:      "both provided, command line argument is prioritized",
			cmdArgKey: "foo",
			envKey:    "FOO",
			value:     false,
			cmdArgs:   []string{"--foo"},
			before: func() {
				os.Setenv("FOO", "false")
			},
			after: func() {
				os.Unsetenv("FOO")
			},
			want: true,
		},
		{
			name:      "default value",
			cmdArgKey: "foo",
			envKey:    "",
			value:     true,
			cmdArgs:   []string{},
			before:    func() {},
			after:     func() {},
			want:      true,
		},
		{
			name:      "empty environment variable",
			cmdArgKey: "foo",
			envKey:    "FOO",
			value:     true,
			cmdArgs:   []string{},
			before: func() {
				os.Setenv("FOO", "")
			},
			after: func() {
				os.Unsetenv("FOO")
			},
			want: true,
		},
		{
			name:      "falsy environment variable",
			cmdArgKey: "foo",
			envKey:    "FOO",
			value:     true,
			cmdArgs:   []string{},
			before: func() {
				os.Setenv("FOO", "false")
			},
			after: func() {
				os.Unsetenv("FOO")
			},
			want: false,
		},
		{
			name:      "falsy environment variable with mixed case",
			cmdArgKey: "foo",
			envKey:    "FOO",
			value:     true,
			cmdArgs:   []string{},
			before: func() {
				os.Setenv("FOO", "fAlSe")
			},
			after: func() {
				os.Unsetenv("FOO")
			},
			want: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			defer tc.after()
			defer Reset()
			setCommandlineArguments(t, tc.cmdArgs)
			gotPointer := Bool(tc.cmdArgKey, tc.value, "", tc.envKey)
			err := Parse()
			if err != nil {
				t.Fatal(err)
			}
			if *gotPointer != tc.want {
				t.Errorf("unexpected result, got %v, want %v", *gotPointer, tc.want)
			}
		})
	}
}

func TestInt(t *testing.T) {
	testCases := []struct {
		name           string
		cmdArgKey      string
		envKey         string
		value          int
		cmdArgs        []string
		before         func()
		after          func()
		want           int
		wantErrOnParse bool
	}{
		{
			name:      "from command line argument",
			cmdArgKey: "foo",
			envKey:    "",
			value:     1,
			cmdArgs:   []string{"--foo=2"},
			before:    func() {},
			after:     func() {},
			want:      2,
		},
		{
			name:      "from environment variable",
			cmdArgKey: "foo",
			envKey:    "FOO",
			value:     1,
			cmdArgs:   []string{},
			before: func() {
				os.Setenv("FOO", "2")
			},
			after: func() {
				os.Unsetenv("FOO")
			},
			want: 2,
		},
		{
			name:      "both provided, command line argument is prioritized",
			cmdArgKey: "foo",
			envKey:    "FOO",
			value:     1,
			cmdArgs:   []string{"--foo=2"},
			before: func() {
				os.Setenv("FOO", "3")
			},
			after: func() {
				os.Unsetenv("FOO")
			},
			want: 2,
		},
		{
			name:      "default value",
			cmdArgKey: "foo",
			envKey:    "",
			value:     1,
			cmdArgs:   []string{},
			before:    func() {},
			after:     func() {},
			want:      1,
		},
		{
			name:      "empty environment variable",
			cmdArgKey: "foo",
			envKey:    "FOO",
			value:     1,
			cmdArgs:   []string{},
			before: func() {
				os.Setenv("FOO", "")
			},
			after: func() {
				os.Unsetenv("FOO")
			},
			want:           1,
			wantErrOnParse: true,
		},
		{
			name:      "invalid environment variable",
			cmdArgKey: "foo",
			envKey:    "FOO",
			value:     1,
			cmdArgs:   []string{},
			before: func() {
				os.Setenv("FOO", "invalid")
			},
			after: func() {
				os.Unsetenv("FOO")
			},
			want:           1,
			wantErrOnParse: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			defer tc.after()
			defer Reset()
			setCommandlineArguments(t, tc.cmdArgs)
			gotPointer := Int(tc.cmdArgKey, tc.value, "", tc.envKey)
			err := Parse()
			if tc.wantErrOnParse && err == nil {
				t.Errorf("unexpected error, got nil, want error")
			}
			if !tc.wantErrOnParse && err != nil {
				t.Errorf("unexpected error, got %v, want nil", err)
			}
			if *gotPointer != tc.want {
				t.Errorf("unexpected result, got %v, want %v", *gotPointer, tc.want)
			}
		})
	}
}
