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

/**
 * Represents an entry in the interned string pool.
 */
export interface StringEntryDTO {
  /**
   * The unique ID of the interned string.
   */
  readonly id: number;
  /**
   * The actual string value.
   */
  readonly value: string;
}

/**
 * Represents an entry defining a set of field path names.
 */
export interface FieldPathSetEntryDTO {
  /**
   * The unique ID of the field path set.
   */
  readonly id: number;
  /**
   * The array of string IDs representing list of field paths.
   */
  readonly fieldPathStringIds: readonly number[];
}

/**
 * Manages the interned strings and field names used in structured data.
 */
export class InternPoolStore {
  /**
   * The interned string directly referencable with string ID.
   */
  private readonly strings: string[] = [];

  /**
   * The interned structured data field path set.
   * It maps fieldPath id to an list of string IDs representing field paths.
   * Field paths can be flatten, it could be `a\0b` if a.b used in the structured data.
   */
  private readonly fieldPathSets: number[][] = [];

  /**
   * Adds multiple strings to the pool.
   * @param strings An iterable of objects containing id and value.
   */
  public addStrings(strings: Iterable<StringEntryDTO>): void {
    for (const { id, value } of strings) {
      this.strings[id] = value;
    }
  }

  /**
   * Adds field path sets to the pool.
   * @param sets An iterable of objects containing id and an array of string IDs.
   */
  public addFieldPathSets(sets: Iterable<FieldPathSetEntryDTO>): void {
    for (const { id, fieldPathStringIds: fieldNames } of sets) {
      this.fieldPathSets[id] = Array.from(fieldNames);
    }
  }

  /**
   * Retrieves a string value by its ID from the pool.
   * @param id The ID of the string.
   * @returns The string value.
   * @throws Error if the ID is not found in the pool.
   */
  public getString(id: number): string {
    const value = this.strings[id];
    if (value === undefined) {
      throw new Error(`String ID ${id} not found in pool`);
    }
    return value;
  }

  /**
   * Retrieves a field path set by its ID, resolving string IDs to string values.
   * @param id The ID of the field path set.
   * @returns An array of string values representing the field path.
   * @throws Error if the ID is not found.
   */
  public getFieldPathSet(id: number): string[] {
    const set = this.fieldPathSets[id];
    if (set === undefined) {
      throw new Error(`FieldPathSet ID ${id} not found in pool`);
    }
    return set.map((strId) => this.getString(strId));
  }
}
