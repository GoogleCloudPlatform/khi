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

package taskrecord

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
)

// TaskResultCodec serializes and deserializes intermediate task results.
type TaskResultCodec interface {
	// Serialize converts the task result value into byte slice.
	Serialize(value any) ([]byte, error)

	// Deserialize converts the byte slice into the task result value.
	Deserialize(data []byte) (any, error)
}

// DefaultJSONCodec provides generic JSON serialization and deserialization.
type DefaultJSONCodec struct{}

// Serialize implements TaskResultCodec by marshaling into indented JSON.
func (c *DefaultJSONCodec) Serialize(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}

// Deserialize implements TaskResultCodec by unmarshaling JSON into a generic map or slice.
func (c *DefaultJSONCodec) Deserialize(data []byte) (any, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("failed to deserialize json: %w", err)
	}
	return v, nil
}

var _ TaskResultCodec = (*DefaultJSONCodec)(nil)

// LogListCodec serializes and deserializes slices of log.Log objects.
type LogListCodec struct{}

// Serialize implements TaskResultCodec by converting each log's structured.Node into JSON.
func (c *LogListCodec) Serialize(value any) ([]byte, error) {
	logs, ok := value.([]*log.Log)
	if !ok {
		return nil, fmt.Errorf("LogListCodec: expected []*log.Log, got %T", value)
	}

	jsonSerializer := &structured.JSONNodeSerializer{}
	rawLogs := make([]json.RawMessage, 0, len(logs))
	for i, l := range logs {
		if l == nil || l.NodeReader == nil || l.NodeReader.Node == nil {
			continue
		}
		rawBytes, err := jsonSerializer.Serialize(l.NodeReader.Node)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize log[%d]: %w", i, err)
		}
		rawLogs = append(rawLogs, json.RawMessage(rawBytes))
	}

	return json.MarshalIndent(rawLogs, "", "  ")
}

// Deserialize implements TaskResultCodec by parsing JSON into structured.Node and wrapping in log.Log.
func (c *LogListCodec) Deserialize(data []byte) (any, error) {
	var rawLogs []json.RawMessage
	if err := json.Unmarshal(data, &rawLogs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal log list json: %w", err)
	}

	result := make([]*log.Log, 0, len(rawLogs))
	for i, raw := range rawLogs {
		node, err := structured.FromYAML(string(raw))
		if err != nil {
			return nil, fmt.Errorf("failed to parse log node[%d]: %w", i, err)
		}
		result = append(result, log.NewLog(structured.NewNodeReader(node)))
	}

	return result, nil
}

var _ TaskResultCodec = (*LogListCodec)(nil)

// CodecRegistry manages codecs keyed by Go data types (reflect.Type).
type CodecRegistry struct {
	mu     sync.RWMutex
	codecs map[reflect.Type]TaskResultCodec
}

// NewCodecRegistry creates a new CodecRegistry with default built-in type codecs.
func NewCodecRegistry() *CodecRegistry {
	r := &CodecRegistry{
		codecs: make(map[reflect.Type]TaskResultCodec),
	}
	r.Register(reflect.TypeOf([]*log.Log{}), &LogListCodec{})
	return r
}

// Register registers a custom codec for the specified reflect.Type.
func (r *CodecRegistry) Register(t reflect.Type, codec TaskResultCodec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.codecs[t] = codec
}

// GetCodec returns the registered codec for the specified type, or DefaultJSONCodec if not registered.
func (r *CodecRegistry) GetCodec(t reflect.Type) TaskResultCodec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if codec, found := r.codecs[t]; found {
		return codec
	}
	return &DefaultJSONCodec{}
}

// DeserializeForType deserializes data into the target reflect.Type.
func (r *CodecRegistry) DeserializeForType(data []byte, t reflect.Type) (any, error) {
	r.mu.RLock()
	codec, found := r.codecs[t]
	r.mu.RUnlock()

	if found {
		return codec.Deserialize(data)
	}

	// For pointer types vs non-pointer types, instantiate an instance of t.
	valPtr := reflect.New(t)
	if err := json.Unmarshal(data, valPtr.Interface()); err != nil {
		return nil, fmt.Errorf("failed to deserialize into type %s: %w", t.String(), err)
	}
	return valPtr.Elem().Interface(), nil
}

// DefaultCodecRegistry is the package-level default codec registry.
var DefaultCodecRegistry = NewCodecRegistry()

// RegisterCodec registers a custom codec for type T in DefaultCodecRegistry.
func RegisterCodec[T any](codec TaskResultCodec) {
	DefaultCodecRegistry.Register(reflect.TypeOf((*T)(nil)).Elem(), codec)
}

// GetCodec returns the codec for type t from DefaultCodecRegistry.
func GetCodec(t reflect.Type) TaskResultCodec {
	return DefaultCodecRegistry.GetCodec(t)
}
