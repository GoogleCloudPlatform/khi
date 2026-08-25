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

// UploadFileStoreProvider defines operations for reading uploaded files.
type UploadFileStoreProvider interface {
	// GetUploadToken generates an UploadToken for the given ID.
	GetUploadToken(id string) UploadToken

	// Read returns the io.ReadCloser to read the file with the given token.
	// The caller MUST close the returned ReadCloser.
	Read(token UploadToken) (io.ReadCloser, error)
}

// DirectWritableUploadFileStoreProvider defines write operations for store providers.
type DirectWritableUploadFileStoreProvider interface {
	// Write writes file data from the given reader to the file identified by token.
	Write(token UploadToken, reader io.Reader) error

	// GetDestinationPath returns the absolute file system path for storing the uploaded file.
	GetDestinationPath(token UploadToken) (string, error)
}

// LocalUploadFileStoreProvider is an implementation of UploadFileStore that stores files in the local file system.
type LocalUploadFileStoreProvider struct {
	directoryPath string
}

// NewLocalUploadFileStoreProvider creates a new LocalUploadFileStoreProvider.
func NewLocalUploadFileStoreProvider(directoryPath string) *LocalUploadFileStoreProvider {
	return &LocalUploadFileStoreProvider{directoryPath: directoryPath}
}

// GetUploadToken implements UploadFileStoreProvider.
func (l *LocalUploadFileStoreProvider) GetUploadToken(id string) UploadToken {
	return &DirectUploadToken{ID: id}
}

// Read implements UploadFileStoreProvider.
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
	return file, nil
}

// Write implements DirectWritableUploadFileStoreProvider.
func (l *LocalUploadFileStoreProvider) Write(token UploadToken, reader io.Reader) error {
	destPath, err := l.GetDestinationPath(token)
	if err != nil {
		return err
	}
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		_ = os.Remove(destPath)
		return err
	}

	return nil
}

// GetDestinationPath returns the local filesystem destination path for the given token.
func (l *LocalUploadFileStoreProvider) GetDestinationPath(token UploadToken) (string, error) {
	err := l.validateTokenFormat(token)
	if err != nil {
		return "", err
	}
	err = l.ensureFolderExists()
	if err != nil {
		return "", err
	}
	return filepath.Join(l.directoryPath, token.GetID()), nil
}

func (l *LocalUploadFileStoreProvider) ensureFolderExists() error {
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

// InPlaceUploadFileStoreProvider is an implementation of UploadFileStore that reads files in place.
type InPlaceUploadFileStoreProvider struct{}

// GetUploadToken implements UploadFileStoreProvider.
func (i *InPlaceUploadFileStoreProvider) GetUploadToken(id string) UploadToken {
	return &LocalFileUploadToken{FilePath: id}
}

// Read implements UploadFileStoreProvider.
func (i *InPlaceUploadFileStoreProvider) Read(token UploadToken) (io.ReadCloser, error) {
	filePath := token.GetID()
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	return file, nil
}

var _ UploadFileStoreProvider = &InPlaceUploadFileStoreProvider{}
