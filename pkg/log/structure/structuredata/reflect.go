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

package structuredata

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func ReadReflect(sd StructureData, target interface{}) error {
	jsonStr, err := ToJson(sd)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(jsonStr), target)
}

func ReadReflectK8sManifest(sd StructureData, target runtime.Object) error {
	jsonStr, err := ToJson(sd)
	if err != nil {
		return err
	}
	scheme := runtime.NewScheme()
	codecFactory := serializer.NewCodecFactory(scheme)
	deserializer := codecFactory.UniversalDeserializer()
	_, _, err = deserializer.Decode([]byte(jsonStr), nil, target)
	if err != nil {
		return err
	}
	return nil
}
