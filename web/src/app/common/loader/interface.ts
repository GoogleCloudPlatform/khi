/**
 * Copyright 2024 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Observable } from 'rxjs';

/**
 * SingleTypeReferenceResolver is an interface to get the content pointed by TextReference for a specific type of TextReference.
 */
export interface SingleTypeReferenceResolver {
  /**
   * Get the referenced string data from given reference.
   */
  getText(reference: TextReference): Observable<string>;

  /**
   * Return true when the implementation supports parsing the given reference type.
   */
  isSupportedReferenceType(type: ReferenceType): boolean;
}

/**
 * TextReference is a pointer to the text to read.
 * Currently expected implemntation of TextReference is to get it from binary part of the KHI file.
 */
export interface TextReference {
  type: ReferenceType;
}

/**
 * The types of references used in TextReference.
 */
export enum ReferenceType {
  /**
   * The reference is not pointing anything. Resolver should return a default string.
   */
  NullReference = -1,
  /**
   * The reference is to read it from the binary part of KHI file.
   */
  KHIFileBinary = 0,
}
