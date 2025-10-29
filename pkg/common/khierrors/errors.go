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

package khierrors

import "errors"

// ErrNotFound indicates that a requested resource or value was not found.
var ErrNotFound = errors.New("not found")

// ErrTypeConversionFailed indicates that a type conversion operation failed.
var ErrTypeConversionFailed = errors.New("type conversion failed")

// ErrInvalidInput indicates that the provided input is invalid.
var ErrInvalidInput = errors.New("invalid input")
