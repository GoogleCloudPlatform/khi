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

package testlog

import (
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structurev2"
)

// StringField returns a TestLogOpt modifying the field at the specified fieldPath to the value.
// It creates maps in ancestor when it doens't exist.
func StringField(fieldPath string, value string) TestLogOpt {
	return func(original structurev2.Node) (structurev2.Node, error) {
		fieldPathInArray := strings.Split(fieldPath, ".")
		return structurev2.WithScalarField(original, fieldPathInArray, value)
	}
}

// IntField returns a TestLogOpt modifying the field at the specified fieldPath to the value.
// It creates maps in ancestor when it doens't exist.
func IntField(fieldPath string, value int) TestLogOpt {
	return func(original structurev2.Node) (structurev2.Node, error) {
		fieldPathInArray := strings.Split(fieldPath, ".")
		return structurev2.WithScalarField(original, fieldPathInArray, value)
	}
}
