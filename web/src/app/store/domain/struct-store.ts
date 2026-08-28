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
import {
  nextCapacity,
  reallocateBuffer,
  reallocateUint32Array,
} from 'src/app/store/domain/buffer-util';
import { allocateBuffer } from 'src/app/store/domain/types';

const MIN_SAFE_INTEGER_BI = BigInt(Number.MIN_SAFE_INTEGER);
const MAX_SAFE_INTEGER_BI = BigInt(Number.MAX_SAFE_INTEGER);

/**
 * Separator used when flattening nested object keys in InternedStruct.
 */
const FIELD_PATH_SEPARATOR = '\x00';

/**
 * Binary tags for value kinds in the flat struct byte stream.
 */
const enum ValueTag {
  Null = 0,
  BoolFalse = 1,
  BoolTrue = 2,
  Int64 = 3,
  Double = 4,
  StringId = 5,
  StructId = 6,
  List = 7,
  TimestampNs = 8,
}

/**
 * Kind discriminator for StructValue.
 */
export enum StructValueKind {
  Null = 'null',
  Int64 = 'int64',
  Double = 'double',
  String = 'string',
  Bool = 'bool',
  StructId = 'structId',
  List = 'list',
  Timestamp = 'timestamp',
}

/**
 * Null variant of StructValueDTO.
 */
export interface StructNullValueDTO {
  readonly kind: StructValueKind.Null;
}

/**
 * Int64 variant of StructValueDTO.
 */
export interface StructInt64ValueDTO {
  readonly kind: StructValueKind.Int64;
  readonly value: bigint;
}

/**
 * Double variant of StructValueDTO.
 */
export interface StructDoubleValueDTO {
  readonly kind: StructValueKind.Double;
  readonly value: number;
}

/**
 * String variant of StructValueDTO.
 */
export interface StructStringValueDTO {
  readonly kind: StructValueKind.String;
  readonly stringId: number;
}

/**
 * Boolean variant of StructValueDTO.
 */
export interface StructBoolValueDTO {
  readonly kind: StructValueKind.Bool;
  readonly value: boolean;
}

/**
 * StructId reference variant of StructValueDTO.
 */
export interface StructStructIdValueDTO {
  readonly kind: StructValueKind.StructId;
  readonly structId: number;
}

/**
 * List variant of StructValueDTO.
 */
export interface StructListValueDTO {
  readonly kind: StructValueKind.List;
  readonly values: readonly StructValueDTO[];
}

/**
 * Timestamp variant of StructValueDTO.
 */
export interface StructTimestampValueDTO {
  readonly kind: StructValueKind.Timestamp;
  readonly timestampNs: bigint;
}

/**
 * Variant value types for structured data in InternedStruct.
 */
export type StructValueDTO =
  | StructNullValueDTO
  | StructInt64ValueDTO
  | StructDoubleValueDTO
  | StructStringValueDTO
  | StructBoolValueDTO
  | StructStructIdValueDTO
  | StructListValueDTO
  | StructTimestampValueDTO;

/**
 * Raw FieldPathSet DTO from the assembler.
 */
export interface FieldPathSetDTO {
  readonly id: number;
  readonly fieldPathStringIds: readonly number[];
}

/**
 * Raw Struct DTO from the assembler.
 */
export interface StructDTO {
  readonly id: number;
  readonly fieldPathSetId: number;
  readonly values: readonly StructValueDTO[];
}

/**
 * Store for managing structured data (InternedStruct) using flat ArrayBuffers.
 */
export class StructStore {
  // Field path set storage
  private fieldPathSetOffsets: Uint32Array;
  private fieldPathSetLengths: Uint32Array;
  private fieldPathsBuffer: Uint32Array;
  private fieldPathsWriteOffset = 0;
  private maxFieldPathSetId = 0;

  // Struct indexing
  private structOffsets: Uint32Array;
  private structFieldPathSetIds: Uint32Array;
  private maxStructId = 0;
  private structCount = 0;

  // Binary payload storage
  private payloadBuffer: Uint8Array;
  private payloadDataView: DataView;
  private payloadWriteOffset = 0;

  private static readonly INITIAL_CAPACITY = 1024;
  private static readonly INITIAL_PAYLOAD_BYTES = 64 * 1024;

  private constructor(
    private readonly internPool: InternPoolStore,

    initialCapacity = StructStore.INITIAL_CAPACITY,
  ) {
    this.fieldPathSetOffsets = new Uint32Array(initialCapacity);
    this.fieldPathSetLengths = new Uint32Array(initialCapacity);
    this.fieldPathsBuffer = new Uint32Array(initialCapacity * 8);

    this.structOffsets = new Uint32Array(initialCapacity);
    this.structFieldPathSetIds = new Uint32Array(initialCapacity);

    const rawPayloadBuffer = allocateBuffer(StructStore.INITIAL_PAYLOAD_BYTES);
    this.payloadBuffer = new Uint8Array(rawPayloadBuffer);
    this.payloadDataView = new DataView(rawPayloadBuffer);
  }

  /**
   * Creates an empty StructStore instance.
   *
   * @param internPool The intern pool store for string resolution.
   * @param initialCapacity The initial capacity for struct index tables.
   * @returns A new StructStore instance.
   */
  public static create(
    internPool: InternPoolStore,
    initialCapacity = StructStore.INITIAL_CAPACITY,
  ): StructStore {
    return new StructStore(internPool, initialCapacity);
  }

  /**
   * Gets the total number of structs stored.
   */
  public get count(): number {
    return this.structCount;
  }

  /**
   * Appends a FieldPathSet to the store.
   *
   * @param set The FieldPathSetDTO to add.
   */
  public addFieldPathSet(set: FieldPathSetDTO): void {
    this.ensureFieldPathSetCapacity(set.id + 1);
    this.ensureFieldPathsBufferCapacity(
      this.fieldPathsWriteOffset + set.fieldPathStringIds.length,
    );

    this.fieldPathSetOffsets[set.id] = this.fieldPathsWriteOffset;
    this.fieldPathSetLengths[set.id] = set.fieldPathStringIds.length;

    for (let i = 0; i < set.fieldPathStringIds.length; i++) {
      this.fieldPathsBuffer[this.fieldPathsWriteOffset + i] =
        set.fieldPathStringIds[i];
    }
    this.fieldPathsWriteOffset += set.fieldPathStringIds.length;

    if (set.id > this.maxFieldPathSetId) {
      this.maxFieldPathSetId = set.id;
    }
  }

  /**
   * Appends a Struct to the store, writing its values into the flat payload byte stream.
   *
   * @param struct The StructDTO to add.
   */
  public addStruct(struct: StructDTO): void {
    this.ensureStructCapacity(struct.id + 1);

    // Offset + 1 so that offset 0 indicates unset/empty
    const startOffset = this.payloadWriteOffset + 1;
    this.structOffsets[struct.id] = startOffset;
    this.structFieldPathSetIds[struct.id] = struct.fieldPathSetId;

    for (const val of struct.values) {
      this.writeValue(val);
    }

    if (struct.id > this.maxStructId) {
      this.maxStructId = struct.id;
    }
    this.structCount++;
  }

  private writeValue(val: StructValueDTO): void {
    switch (val.kind) {
      case StructValueKind.Null: {
        this.ensurePayloadCapacity(this.payloadWriteOffset + 1);
        this.payloadBuffer[this.payloadWriteOffset++] = ValueTag.Null;
        break;
      }
      case StructValueKind.Bool: {
        this.ensurePayloadCapacity(this.payloadWriteOffset + 1);
        this.payloadBuffer[this.payloadWriteOffset++] = val.value
          ? ValueTag.BoolTrue
          : ValueTag.BoolFalse;
        break;
      }
      case StructValueKind.Int64: {
        this.ensurePayloadCapacity(this.payloadWriteOffset + 9);
        this.payloadBuffer[this.payloadWriteOffset++] = ValueTag.Int64;
        this.payloadDataView.setBigInt64(
          this.payloadWriteOffset,
          val.value,
          /* littleEndian= */ true,
        );
        this.payloadWriteOffset += 8;
        break;
      }
      case StructValueKind.Double: {
        this.ensurePayloadCapacity(this.payloadWriteOffset + 9);
        this.payloadBuffer[this.payloadWriteOffset++] = ValueTag.Double;
        this.payloadDataView.setFloat64(
          this.payloadWriteOffset,
          val.value,
          /* littleEndian= */ true,
        );
        this.payloadWriteOffset += 8;
        break;
      }
      case StructValueKind.String: {
        this.ensurePayloadCapacity(this.payloadWriteOffset + 5);
        this.payloadBuffer[this.payloadWriteOffset++] = ValueTag.StringId;
        this.payloadDataView.setUint32(
          this.payloadWriteOffset,
          val.stringId,
          /* littleEndian= */ true,
        );
        this.payloadWriteOffset += 4;
        break;
      }
      case StructValueKind.StructId: {
        this.ensurePayloadCapacity(this.payloadWriteOffset + 5);
        this.payloadBuffer[this.payloadWriteOffset++] = ValueTag.StructId;
        this.payloadDataView.setUint32(
          this.payloadWriteOffset,
          val.structId,
          /* littleEndian= */ true,
        );
        this.payloadWriteOffset += 4;
        break;
      }
      case StructValueKind.List: {
        this.ensurePayloadCapacity(this.payloadWriteOffset + 5);
        this.payloadBuffer[this.payloadWriteOffset++] = ValueTag.List;
        this.payloadDataView.setUint32(
          this.payloadWriteOffset,
          val.values.length,
          /* littleEndian= */ true,
        );
        this.payloadWriteOffset += 4;
        for (const item of val.values) {
          this.writeValue(item);
        }
        break;
      }
      case StructValueKind.Timestamp: {
        this.ensurePayloadCapacity(this.payloadWriteOffset + 9);
        this.payloadBuffer[this.payloadWriteOffset++] = ValueTag.TimestampNs;
        this.payloadDataView.setBigInt64(
          this.payloadWriteOffset,
          val.timestampNs,
          /* littleEndian= */ true,
        );
        this.payloadWriteOffset += 8;
        break;
      }
    }
  }

  /**
   * Retrieves and decodes a struct by ID into a nested JavaScript object on-demand.
   *
   * @param id The unique struct ID.
   * @returns Decoded nested object, or null if not found.
   */
  public getStruct(id: number): Record<string, unknown> | null {
    if (id <= 0 || id > this.maxStructId) {
      return null;
    }

    const storedOffset = this.structOffsets[id];
    if (storedOffset === 0) {
      return null;
    }

    const startOffset = storedOffset - 1;
    const fieldPathSetId = this.structFieldPathSetIds[id];
    const fieldCount = this.fieldPathSetLengths[fieldPathSetId] ?? 0;
    const fieldOffset = this.fieldPathSetOffsets[fieldPathSetId] ?? 0;

    let currentPos = startOffset;
    const root: Record<string, unknown> = Object.create(null);

    for (let i = 0; i < fieldCount; i++) {
      const stringId = this.fieldPathsBuffer[fieldOffset + i];
      const fullPath = this.internPool.getString(stringId);
      const [val, nextPos] = this.readValue(currentPos);
      currentPos = nextPos;

      this.setPathValue(root, fullPath, val);
    }

    return root;
  }

  private readValue(pos: number): [unknown, number] {
    const tag: ValueTag = this.payloadBuffer[pos++];
    switch (tag) {
      case ValueTag.Null:
        return [null, pos];
      case ValueTag.BoolFalse:
        return [false, pos];
      case ValueTag.BoolTrue:
        return [true, pos];
      case ValueTag.Int64: {
        const bigVal = this.payloadDataView.getBigInt64(
          pos,
          /* littleEndian= */ true,
        );
        const numVal = Number(bigVal);
        const isSafe =
          bigVal >= MIN_SAFE_INTEGER_BI && bigVal <= MAX_SAFE_INTEGER_BI;
        return [isSafe ? numVal : bigVal, pos + 8];
      }
      case ValueTag.Double: {
        const dVal = this.payloadDataView.getFloat64(
          pos,
          /* littleEndian= */ true,
        );
        return [dVal, pos + 8];
      }
      case ValueTag.StringId: {
        const strId = this.payloadDataView.getUint32(
          pos,
          /* littleEndian= */ true,
        );
        const resolved = this.internPool.getString(strId);
        return [resolved, pos + 4];
      }
      case ValueTag.StructId: {
        const structId = this.payloadDataView.getUint32(
          pos,
          /* littleEndian= */ true,
        );
        const nested = this.getStruct(structId);
        return [nested, pos + 4];
      }
      case ValueTag.List: {
        const count = this.payloadDataView.getUint32(
          pos,
          /* littleEndian= */ true,
        );
        let listPos = pos + 4;
        const list: unknown[] = [];
        for (let i = 0; i < count; i++) {
          const [item, nextPos] = this.readValue(listPos);
          list.push(item);
          listPos = nextPos;
        }
        return [list, listPos];
      }
      case ValueTag.TimestampNs: {
        const tsNs = this.payloadDataView.getBigInt64(
          pos,
          /* littleEndian= */ true,
        );
        return [tsNs, pos + 8];
      }
      default:
        throw new Error(`Unknown ValueTag: ${tag} at offset ${pos - 1}`);
    }
  }

  private setPathValue(
    target: Record<string, unknown>,
    fullPath: string,
    value: unknown,
  ): void {
    const parts = fullPath.split(FIELD_PATH_SEPARATOR);

    for (const part of parts) {
      if (
        part.length === 0 ||
        part === '__proto__' ||
        part === 'constructor' ||
        part === 'prototype'
      ) {
        return;
      }
    }

    let current = target;
    for (let i = 0; i < parts.length - 1; i++) {
      const part = parts[i];
      if (
        !Object.hasOwn(current, part) ||
        typeof current[part] !== 'object' ||
        current[part] === null
      ) {
        current[part] = Object.create(null);
      }
      current = current[part] as Record<string, unknown>;
    }
    current[parts[parts.length - 1]] = value;
  }

  /**
   * Ensures capacity for the FieldPathSet index tables.
   */
  private ensureFieldPathSetCapacity(minCapacity: number): void {
    if (minCapacity <= this.fieldPathSetOffsets.length) {
      return;
    }
    const newCap = nextCapacity(this.fieldPathSetOffsets.length, minCapacity);
    this.fieldPathSetOffsets = reallocateUint32Array(
      this.fieldPathSetOffsets,
      newCap,
    );
    this.fieldPathSetLengths = reallocateUint32Array(
      this.fieldPathSetLengths,
      newCap,
    );
  }

  /**
   * Ensures capacity for the contiguous field paths buffer.
   */
  private ensureFieldPathsBufferCapacity(minCapacity: number): void {
    if (minCapacity <= this.fieldPathsBuffer.length) {
      return;
    }
    const newCap = nextCapacity(this.fieldPathsBuffer.length, minCapacity);
    this.fieldPathsBuffer = reallocateUint32Array(
      this.fieldPathsBuffer,
      newCap,
      this.fieldPathsWriteOffset,
    );
  }

  /**
   * Ensures capacity for the struct index tables.
   */
  private ensureStructCapacity(minCapacity: number): void {
    if (minCapacity <= this.structOffsets.length) {
      return;
    }
    const newCap = nextCapacity(this.structOffsets.length, minCapacity);
    this.structOffsets = reallocateUint32Array(this.structOffsets, newCap);
    this.structFieldPathSetIds = reallocateUint32Array(
      this.structFieldPathSetIds,
      newCap,
    );
  }

  /**
   * Ensures capacity for the binary payload buffer.
   */
  private ensurePayloadCapacity(minByteLength: number): void {
    if (minByteLength <= this.payloadBuffer.byteLength) {
      return;
    }
    const newBytes = nextCapacity(
      this.payloadBuffer.byteLength,
      minByteLength,
      StructStore.INITIAL_PAYLOAD_BYTES,
    );
    const newBuffer = reallocateBuffer(
      this.payloadBuffer.buffer as ArrayBuffer,
      newBytes,
      this.payloadWriteOffset,
    );
    this.payloadBuffer = new Uint8Array(newBuffer);
    this.payloadDataView = new DataView(newBuffer);
  }

  /**
   * Shrinks all buffers and index arrays to minimal required lengths.
   */
  public shrinkToFit(): void {
    const neededSetCap = this.maxFieldPathSetId + 1;
    if (neededSetCap < this.fieldPathSetOffsets.length && neededSetCap > 0) {
      this.fieldPathSetOffsets = reallocateUint32Array(
        this.fieldPathSetOffsets,
        neededSetCap,
        neededSetCap,
      );
      this.fieldPathSetLengths = reallocateUint32Array(
        this.fieldPathSetLengths,
        neededSetCap,
        neededSetCap,
      );
    }

    if (this.fieldPathsWriteOffset < this.fieldPathsBuffer.length) {
      this.fieldPathsBuffer = reallocateUint32Array(
        this.fieldPathsBuffer,
        this.fieldPathsWriteOffset,
        this.fieldPathsWriteOffset,
      );
    }

    const neededStructCap = this.maxStructId + 1;
    if (neededStructCap < this.structOffsets.length && neededStructCap > 0) {
      this.structOffsets = reallocateUint32Array(
        this.structOffsets,
        neededStructCap,
        neededStructCap,
      );
      this.structFieldPathSetIds = reallocateUint32Array(
        this.structFieldPathSetIds,
        neededStructCap,
        neededStructCap,
      );
    }

    if (this.payloadWriteOffset < this.payloadBuffer.byteLength) {
      const newBuffer = reallocateBuffer(
        this.payloadBuffer.buffer as ArrayBuffer,
        this.payloadWriteOffset,
        this.payloadWriteOffset,
      );
      this.payloadBuffer = new Uint8Array(newBuffer);
      this.payloadDataView = new DataView(newBuffer);
    }
  }
}
