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

package history

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/binarychunk"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

// Staging data types contains large string data directly on it.
// These types are just used in the arguments of ChangeSet.
// These will be converted to the corresponding serializable types with some binary data stored in binarychunk.Builder.

type StagingResourceRevision struct {
	Verb      enum.RevisionVerb
	Body      string
	Requestor string
	Partial   bool
	// If this resource existence is inferred from another logs later.
	Inferred   bool
	ChangeTime time.Time
	State      enum.RevisionState
}

func (r *StagingResourceRevision) commit(binaryBuilder *binarychunk.Builder, l *log.LogEntity) (*ResourceRevision, error) {
	bodyRef, err := binaryBuilder.Write([]byte(r.Body))
	if err != nil {
		return nil, err
	}
	requestorRef, err := binaryBuilder.Write([]byte(r.Requestor))
	if err != nil {
		return nil, err
	}
	logId := l.ID()
	if r.Inferred {
		logId = ""
	}
	return &ResourceRevision{
		Log:        logId,
		Requestor:  requestorRef,
		Verb:       r.Verb,
		Body:       bodyRef,
		Partial:    r.Partial,
		ChangeTime: r.ChangeTime,
		State:      r.State,
	}, nil
}
