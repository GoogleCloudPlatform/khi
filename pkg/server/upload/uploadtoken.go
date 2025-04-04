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

package upload

// UploadToken is the type given to the frontend to receive the file.
// This currently exects files are uploaded to API directly, but in future this may support the upload using signed URLs as well.
// All token implements this type must be serializable as JSON.
type UploadToken interface {
	// GetType returns the type of token specifying the methods to upload the file.
	GetType() string
	// GetID returns the unique identifier of upload files.
	GetID() string
	// GetHash returns a unique string calculated from all the field of the implementation.
	// This must be calculated from ALL the field because this is for checking if 2 instances are identical.
	GetHash() string
}

// DirectUploadToken is a UploadToken for uploading the target file to API directly.
type DirectUploadToken struct {
	// ID identiies the file location uploade to this server directly.
	ID string `json:"id"`
}

// GetHash implements UploadToken.
func (d *DirectUploadToken) GetHash() string {
	return d.ID
}

// GetID implements UploadToken.
func (d *DirectUploadToken) GetID() string {
	return d.ID
}

// GetType implements UploadToken.
func (d *DirectUploadToken) GetType() string {
	return "direct"
}

var _ UploadToken = &DirectUploadToken{}
