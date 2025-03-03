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

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// UploadFileVerifier verifies uploaded files (e.g., file type checks).
type UploadFileVerifier interface {
	// Verify checks the file. This returns an error if invalid.
	Verify(storeProvider UploadFileStoreProvider, token UploadToken) error
}

// NopWaitUploadFileVerifier just waits the specified time without doing anything on the file.
type NopWaitUploadFileVerifier struct {
	// WaitTimeInMs is the time to wait before returning the veification result.
	WaitTimeInMs int
	// Error is the error returned from Verify function after waiting the WaitTimeInMs.
	Error error
}

// Verify implements UploadFileVerifier.
func (n *NopWaitUploadFileVerifier) Verify(storeProvider UploadFileStoreProvider, token UploadToken) error {
	<-time.After(time.Duration(n.WaitTimeInMs) * time.Millisecond)
	return n.Error
}

var _ UploadFileVerifier = &NopWaitUploadFileVerifier{}

type JSONLineUploadFileVerifier struct {
	MaxLineSizeInBytes int
}

// Verify implements UploadFileVerifier.
func (j *JSONLineUploadFileVerifier) Verify(storeProvider UploadFileStoreProvider, token UploadToken) error {
	reader, err := storeProvider.Read(token)
	if err != nil {
		return fmt.Errorf("failed to read the uploded file")
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, j.MaxLineSizeInBytes), j.MaxLineSizeInBytes)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()

		// Check if the line is empty or just whitespace. Skip empty lines.
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		if !json.Valid(line) {
			// json.Valid only returns valid or not, gets the error message with deserializing it.
			var data interface{}
			if err := json.Unmarshal(line, &data); err != nil {
				return fmt.Errorf("invalid JSON on line %d: %w", lineNumber, err)
			} else {
				// The JSON is not valid but deserialized.
				return fmt.Errorf("unreachable")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}

var _ UploadFileVerifier = &JSONLineUploadFileVerifier{}
