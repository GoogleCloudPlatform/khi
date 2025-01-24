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

package resourcepath

import "github.com/GoogleCloudPlatform/khi/pkg/model/enum"

// ResourcePath contains the path representing location of a timeline in the history.
type ResourcePath struct {
	// Path is the the raw resource path represented in string like `A#B#C`. This means `the C under the B under the A at the root`.
	// KHI uses `#` as the delimiter of resource paths, this is because the root element(API version) can contain `.` or `/`.
	Path string

	// ParentRelationship explains between the location represented with this ResourcePath and its parent.
	// KHI shows various resources in a single history with mixing many types of children. It's not only like child-parent relationship, but also pseudo relationship like node-node's component relationship.
	ParentRelationship enum.ParentRelationship
}
