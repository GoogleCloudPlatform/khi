/**
 * Copyright 2026 Google LLC
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

import { InternPoolStore } from 'src/app/store/domain/intern-pool-store';

/**
 * Represents an interned structured object converted from Protobuf.
 */
export interface InternedStructDTO {
  /**
   * The ID of the field path set.
   */
  readonly fieldPathSetId: number;
  /**
   * The list of interned field values.
   */
  readonly values: readonly InternedValueDTO[];
}

/**
 * Represents an interned field value.
 */
export interface InternedValueDTO {
  /**
   * The specific variant type and its associated payload.
   */
  readonly kind:
    | { readonly case: 'nullValue'; readonly value: unknown }
    | { readonly case: 'int64Value'; readonly value: number }
    | { readonly case: 'doubleValue'; readonly value: number }
    | { readonly case: 'stringValue'; readonly value: number }
    | { readonly case: 'boolValue'; readonly value: boolean }
    | { readonly case: 'structValue'; readonly value: InternedStructDTO }
    | {
        readonly case: 'listValue';
        readonly value: { readonly values: readonly InternedValueDTO[] };
      }
    | {
        readonly case: 'timestampValue';
        readonly value: bigint;
      }
    | { readonly case: undefined; readonly value?: undefined };
}

/**
 * Utility to decode InternedStruct into standard JavaScript objects.
 */
export class InternedStructDecoder {
  constructor(private readonly internPool: InternPoolStore) {}

  /**
   * Decodes an InternedStruct into a nested Record.
   */
  public decode(struct: InternedStructDTO): Record<string, unknown> {
    const result: Record<string, unknown> = {};
    const fieldPathSet = this.internPool.getFieldPathSet(struct.fieldPathSetId);

    if (fieldPathSet.length !== struct.values.length) {
      throw new Error(
        `Field path length (${fieldPathSet.length}) does not match values length (${struct.values.length})`,
      );
    }

    for (let i = 0; i < fieldPathSet.length; i++) {
      const fieldPathStr = fieldPathSet[i];
      const value = this.decodeValue(struct.values[i]);

      // Handle flattened keys separated by \0 (e.g. "metadata\0name")
      const parts = fieldPathStr.split('\0');
      let current = result;
      for (let j = 0; j < parts.length - 1; j++) {
        const part = parts[j];
        if (!current[part] || typeof current[part] !== 'object') {
          current[part] = {};
        }
        current = current[part] as Record<string, unknown>;
      }
      current[parts[parts.length - 1]] = value;
    }

    return result;
  }

  private decodeValue(value: InternedValueDTO): unknown {
    switch (value.kind.case) {
      case 'nullValue':
        return null;
      case 'int64Value':
        return value.kind.value;
      case 'doubleValue':
        return value.kind.value;
      case 'stringValue':
        return this.internPool.getString(value.kind.value);
      case 'boolValue':
        return value.kind.value;
      case 'structValue':
        return this.decode(value.kind.value);
      case 'listValue':
        return value.kind.value.values.map((v) => this.decodeValue(v));
      case 'timestampValue':
        return value.kind.value;
      case undefined:
        throw new Error('InternedValue kind is undefined');
      default: {
        const caseName = (value.kind as unknown as { case?: string }).case;
        throw new Error(
          `Unsupported InternedValue kind: ${caseName ?? 'unknown'}`,
        );
      }
    }
  }
}
