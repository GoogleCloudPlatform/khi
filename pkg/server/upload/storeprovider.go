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
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type UploadFileStoreProvider interface {
	// Generate a UploadToken for frontend
	GetUploadToken(id string) UploadToken
	// Read returns the io.ReadCloser interface to read the file with the given ID.
	// The caller MUST close the returned ReadCloser.
	Read(token UploadToken) (io.ReadCloser, error)
}

type DirectWritableUploadFileStoreProvider interface {
	// Write writes file with given io.Writer interaface to the file with the given ID.
	Write(token UploadToken, reader io.Reader) error
}

// LocalUploadFileStoreProvider is an implementation of UploadFileStore that stores files
// in the local file system.
type LocalUploadFileStoreProvider struct {
	// directoryPath is the folder name where uploaded files are stored.
	directoryPath string
}

// NewLocalUploadFileStoreProvider creates a new LocalUploadFileStore.
func NewLocalUploadFileStoreProvider(directoryPath string) *LocalUploadFileStoreProvider {
	return &LocalUploadFileStoreProvider{directoryPath: directoryPath}
}

// GetUploadToken implements UploadFileStoreProvider.
func (l *LocalUploadFileStoreProvider) GetUploadToken(id string) UploadToken {
	return &DirectUploadToken{ID: id}
}

func (l *LocalUploadFileStoreProvider) Read(token UploadToken) (io.ReadCloser, error) {
	err := l.validateTokenFormat(token)
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(l.directoryPath, token.GetID())
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	return file, nil // os.File implements io.ReadCloser
}

func (l *LocalUploadFileStoreProvider) Write(token UploadToken, reader io.Reader) error {
	err := l.validateTokenFormat(token)
	if err != nil {
		return err
	}
	err = l.ensureFolderExists()
	if err != nil {
		return err
	}
	filePath := filepath.Join(l.directoryPath, token.GetID())
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		_ = os.Remove(filePath)
		return err
	}

	return nil
}

func (l *LocalUploadFileStoreProvider) ensureFolderExists() error {
	// Create the directory (and any parent directories) if it doesn't exist.
	// os.MkdirAll will not return an error if the directory already exists.
	return os.MkdirAll(l.directoryPath, 0700)
}

func (l *LocalUploadFileStoreProvider) validateTokenFormat(token UploadToken) error {
	id := token.GetID()
	if strings.Contains(id, "/") {
		return errors.New("token id must not contain `/`")
	}
	return nil
}

var _ UploadFileStoreProvider = &LocalUploadFileStoreProvider{}
var _ DirectWritableUploadFileStoreProvider = &LocalUploadFileStoreProvider{}
