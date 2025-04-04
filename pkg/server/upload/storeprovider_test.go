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
	"io"
	"os"
	"strings"
	"testing"
)

// MockLocalUploadFileStoreProvider is a mock implementation of LocalUploadFileStoreProvider for testing purposes.
type MockLocalUploadFileStoreProvider struct {
	Data string
}

func (t *MockLocalUploadFileStoreProvider) Read(token UploadToken) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(t.Data)), nil
}

func (t *MockLocalUploadFileStoreProvider) GetUploadToken(id string) UploadToken {
	return &DirectUploadToken{ID: id}
}

var _ UploadFileStoreProvider = &MockLocalUploadFileStoreProvider{}

func TestLocalUploadFileStoreProvider_Essential(t *testing.T) {
	// Create a temporary directory for testing.
	tempDir, err := os.MkdirTemp("", "uploadtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir) // Clean up after the test.

	// Create a new LocalUploadFileStore.
	store := NewLocalUploadFileStoreProvider(tempDir)

	t.Run("WriteAndRead_Basic", func(t *testing.T) {
		token := store.GetUploadToken("test-token")
		content := "This is some test content."
		reader := strings.NewReader(content)

		// Write the file.
		err = store.Write(token, reader)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		// Read the file.
		readCloser, err := store.Read(token)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		defer readCloser.Close()

		// Read the content and verify.
		readContent, err := io.ReadAll(readCloser)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}
		if string(readContent) != content {
			t.Errorf("Expected content: %q, got: %q", content, string(readContent))
		}
	})

	t.Run("Read_NonExistentFile", func(t *testing.T) {
		token := store.GetUploadToken("not-uploaded")
		_, err := store.Read(token)
		if !os.IsNotExist(err) {
			t.Errorf("Expected os.ErrNotExist, got: %v", err)
		}
	})

	t.Run("WriteAndRead_Overwrite", func(t *testing.T) {
		token := store.GetUploadToken("test-token")
		content1 := "Initial content"
		content2 := "Overwritten content"

		// Write initial content.
		if err := store.Write(token, strings.NewReader(content1)); err != nil {
			t.Fatalf("Initial write failed: %v", err)
		}

		// Overwrite the file.
		if err := store.Write(token, strings.NewReader(content2)); err != nil {
			t.Fatalf("Overwrite write failed: %v", err)
		}

		// Read and verify overwritten content.
		readCloser, err := store.Read(token)
		if err != nil {
			t.Fatalf("Read after overwrite failed: %v", err)
		}
		defer readCloser.Close()

		readContent, err := io.ReadAll(readCloser)
		if err != nil {
			t.Fatalf("ReadAll after overwrite failed: %v", err)
		}
		if string(readContent) != content2 {
			t.Errorf("Expected content: %q, got: %q", content2, string(readContent))
		}
	})
}
