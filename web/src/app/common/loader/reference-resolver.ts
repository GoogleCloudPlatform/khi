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

import { Observable, of } from 'rxjs';
import {
  SingleTypeReferenceResolver,
  ReferenceType,
  TextReference,
} from './interface';
import { IsTextReferenceFromKHIFileBinary } from './reference-type';

export class ReferenceResolverStore {
  constructor(public readonly resolvers: SingleTypeReferenceResolver[]) {}

  getText(reference: TextReference): Observable<string> {
    const resolver = this.resolvers.find((r) =>
      r.isSupportedReferenceType(reference.type),
    );
    if (!resolver) throw new Error(`No resolver found for ${reference.type}`);
    return resolver.getText(reference);
  }
}

/**
 * An implementation of ReferenceResolver to resolve data from the binary part of KHI file.
 */
export class KHIFileReferenceResolver implements SingleTypeReferenceResolver {
  private decoder = new TextDecoder();

  constructor(public readonly sourceBuffers: ArrayBuffer[]) {}

  isSupportedReferenceType(type: ReferenceType): boolean {
    return type === ReferenceType.KHIFileBinary;
  }

  getText(reference: TextReference): Observable<string> {
    if (!IsTextReferenceFromKHIFileBinary(reference))
      throw new Error(
        `Unsupported reference type ${reference.type} given to KHIFileReferenceResolver`,
      );

    const bufferView = new Uint8Array(
      this.sourceBuffers[reference.buffer],
      reference.offset,
      reference.len,
    );
    return of(this.decoder.decode(bufferView));
  }
}

/**
 * A RefrenceResolver for NullReference to return a default value.
 */
export class NullReferenceResolver implements SingleTypeReferenceResolver {
  isSupportedReferenceType(type: ReferenceType): boolean {
    return type === ReferenceType.NullReference;
  }

  getText(): Observable<string> {
    return of('');
  }
}
