// Copyright 2026 Google LLC
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

// Package uploadurl issues short-lived URLs that accept a file for one file form field in a single PUT request.
// Clients that cannot run the chunked upload protocol of the Web UI, such as AI agents using curl, use these URLs.
package uploadurl

import (
	"crypto/rand"
	"errors"
	"sync"
	"time"
)

// DefaultTTL is how long an issued upload URL accepts uploads.
const DefaultTTL = 30 * time.Minute

var (
	// ErrUnknownURL is returned when an upload URL token was never issued.
	ErrUnknownURL = errors.New("unknown upload URL")
	// ErrExpiredURL is returned when an upload URL token is used after it expires.
	ErrExpiredURL = errors.New("upload URL has expired")
)

// Grant describes an issued upload URL.
type Grant struct {
	// URL is the address that accepts the file body with a PUT request.
	URL string
	// FieldID is the ID of the file form field that receives the file.
	FieldID string
	// MaxSizeBytes is the largest file size the URL accepts.
	MaxSizeBytes int64
	// ExpiresAt is the time after which the URL rejects uploads.
	ExpiresAt time.Time
}

// grantTarget is what an issued URL token resolves to.
type grantTarget struct {
	uploadTokenID string
	fieldID       string
	expiresAt     time.Time
}

// Issuer issues upload URLs and resolves the URL tokens in them. Each URL is bound to the upload token of one file form field.
type Issuer struct {
	baseURL      string
	maxSizeBytes int64
	ttl          time.Duration

	mu      sync.Mutex
	targets map[string]grantTarget
}

// NewIssuer returns an Issuer whose URLs are baseURL followed by a random URL token.
// baseURL must end with the path that routes to Handler, including the trailing slash.
func NewIssuer(baseURL string, maxSizeBytes int64, ttl time.Duration) *Issuer {
	return &Issuer{
		baseURL:      baseURL,
		maxSizeBytes: maxSizeBytes,
		ttl:          ttl,
		targets:      make(map[string]grantTarget),
	}
}

// Issue returns a new upload URL that stores the file for the upload token uploadTokenID of the file form field fieldID.
func (i *Issuer) Issue(uploadTokenID, fieldID string) Grant {
	urlToken := rand.Text()
	now := time.Now()
	expiresAt := now.Add(i.ttl)

	i.mu.Lock()
	defer i.mu.Unlock()
	// Expired URL tokens can never be used again, so dropping them here keeps the map from growing with every issued URL.
	for issuedURLToken, target := range i.targets {
		if !now.Before(target.expiresAt) {
			delete(i.targets, issuedURLToken)
		}
	}
	i.targets[urlToken] = grantTarget{
		uploadTokenID: uploadTokenID,
		fieldID:       fieldID,
		expiresAt:     expiresAt,
	}

	return Grant{
		URL:          i.baseURL + urlToken,
		FieldID:      fieldID,
		MaxSizeBytes: i.maxSizeBytes,
		ExpiresAt:    expiresAt,
	}
}

// resolve returns the target of urlToken, or ErrUnknownURL or ErrExpiredURL when the URL token cannot be used.
func (i *Issuer) resolve(urlToken string) (grantTarget, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	target, found := i.targets[urlToken]
	if !found {
		return grantTarget{}, ErrUnknownURL
	}
	if !time.Now().Before(target.expiresAt) {
		delete(i.targets, urlToken)
		return grantTarget{}, ErrExpiredURL
	}
	return target, nil
}

// consume removes urlToken so that its URL accepts no further uploads.
// It returns false when another request already consumed urlToken.
func (i *Issuer) consume(urlToken string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, found := i.targets[urlToken]; !found {
		return false
	}
	delete(i.targets, urlToken)
	return true
}
