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

import { Delta } from 'jsondiffpatch';
import { DiffStatus } from 'src/app/shared/components/yaml-viewer/lcs';

/**
 * Represents the data type of a YAML value.
 */
export enum ValueType {
  Object = 'object',
  Array = 'array',
  String = 'string',
  Number = 'number',
  Boolean = 'boolean',
  Null = 'null',
  None = 'none',
}

/**
 * Internal tree representation used to merge two objects and compute diff status.
 */
export interface MergeNode {
  key: string;
  path: string;
  status: DiffStatus;
  valueType: ValueType;
  oldValueType?: ValueType;
  oldValue?: unknown;
  newValue?: unknown;
  children?: MergeNode[];
  movedFrom?: string;
  movedTo?: string;
}

import {
  isAddedDelta,
  isDeletedDelta,
  isModifiedDelta,
  isMovedDelta,
} from 'src/app/shared/components/yaml-viewer/jsondiffpatch-util';

/**
 * Converts a JSON Patch path (e.g. /metadata/annotations/cloud.google.com~1neg)
 * to an array of path segments.
 */
export function convertJsonPatchPathToArray(path: string): string[] {
  if (!path) {
    return [];
  }

  // Remove leading slash
  if (path.startsWith('/')) {
    path = path.slice(1);
  }

  // Split by slash and decode JSON Pointer escape sequences
  return path
    .split('/')
    .map((segment) => segment.replace(/~1/g, '/').replace(/~0/g, '~'));
}

/**
 * Checks if the given ValueType is a scalar type (not Object or Array).
 */
export function isScalarType(type: ValueType): boolean {
  return type !== ValueType.Object && type !== ValueType.Array;
}

/**
 * Helper to determine the ValueType of a given value.
 */
export function getValueType(value: unknown): ValueType {
  if (value === null) {
    return ValueType.Null;
  }
  if (Array.isArray(value)) {
    return ValueType.Array;
  }
  if (value && typeof value === 'object') {
    return ValueType.Object;
  }
  if (typeof value === 'string') {
    return ValueType.String;
  }
  if (typeof value === 'number') {
    return ValueType.Number;
  }
  if (typeof value === 'boolean') {
    return ValueType.Boolean;
  }
  return ValueType.None;
}

/**
 * Generates a hash string for an array element to pair them.
 */
export function getHash(obj: unknown, index: number): string {
  if (obj && typeof obj === 'object') {
    const record = obj as Record<string, unknown>;
    if ('type' in record) {
      return `type:${record['type']}`;
    }
    if ('name' in record) {
      return `name:${record['name']}`;
    }
  }
  return obj !== undefined && obj !== null ? String(obj) : `idx:${index}`;
}

/**
 * Determines if the entire value should be highlighted as modified,
 * rather than performing character-level diffing.
 *
 * This is typically true for nulls, booleans, or boolean-like strings.
 */
export function shouldHighlightEntireValue(
  oldVal: unknown,
  newVal: unknown,
): boolean {
  if (oldVal === null || newVal === null) {
    return true;
  }
  if (typeof oldVal === 'boolean' || typeof newVal === 'boolean') {
    return true;
  }
  const isBoolStr = (val: unknown) =>
    typeof val === 'string' &&
    ['true', 'false', 'yes', 'no'].includes(val.toLowerCase());
  if (isBoolStr(oldVal) || isBoolStr(newVal)) {
    return true;
  }
  return false;
}

/**
 * Recursive function to traverse left/right objects alongside the diff delta to build a merged node tree.
 */
export function buildMergeTree(
  left: unknown,
  right: unknown,
  delta: Delta | undefined,
  path: string,
  key: string,
): MergeNode {
  const nextPath = path ? (key ? `${path}.${key}` : path) : key;
  const leftType = getValueType(left);
  const rightType = getValueType(right);

  // Handle Type Modification (Type change)
  if (left !== undefined && right !== undefined && leftType !== rightType) {
    const node: MergeNode = {
      key,
      path: nextPath,
      status: DiffStatus.Modified,
      valueType: rightType,
      oldValueType: leftType,
    };

    let deletedChildren: MergeNode[] | undefined;
    let addedChildren: MergeNode[] | undefined;

    if (isScalarType(leftType)) {
      node.oldValue = left;
    } else {
      const leftNode = buildMergeTree(left, undefined, undefined, path, key);
      deletedChildren = markAllAs(leftNode.children, DiffStatus.Deleted);
    }

    if (isScalarType(rightType)) {
      node.newValue = right;
    } else {
      const rightNode = buildMergeTree(undefined, right, undefined, path, key);
      addedChildren = markAllAs(rightNode.children, DiffStatus.Added);
    }

    if (deletedChildren || addedChildren) {
      node.children = [...(deletedChildren || []), ...(addedChildren || [])];
    }

    return node;
  }

  let forcedStatus: DiffStatus | undefined = undefined;

  if (isAddedDelta(delta)) {
    const type = getValueType(delta[0]);
    if (isScalarType(type)) {
      return {
        key,
        path: nextPath,
        status: DiffStatus.Added,
        valueType: type,
        newValue: delta[0],
      };
    }
    right = delta[0];
    delta = undefined;
    forcedStatus = DiffStatus.Added;
  }
  if (isModifiedDelta(delta)) {
    const type = getValueType(delta[1]);
    if (isScalarType(type)) {
      return {
        key,
        path: nextPath,
        status: DiffStatus.Modified,
        valueType: type,
        oldValue: delta[0],
        newValue: delta[1],
      };
    }
    left = delta[0];
    right = delta[1];
    delta = undefined;
  }
  if (isDeletedDelta(delta)) {
    const type = getValueType(delta[0]);
    if (isScalarType(type)) {
      return {
        key,
        path: nextPath,
        status: DiffStatus.Deleted,
        valueType: type,
        oldValue: delta[0],
      };
    }
    left = delta[0];
    delta = undefined;
    forcedStatus = DiffStatus.Deleted;
  }

  // Determine type
  const activeObj = right !== undefined ? right : left;
  const valueType = getValueType(activeObj);

  const node: MergeNode = {
    key,
    path: nextPath,
    status:
      forcedStatus || (delta ? DiffStatus.Modified : DiffStatus.Unchanged),
    valueType,
  };

  if (valueType === ValueType.Object) {
    mergeObjectNode(node, left, right, delta, nextPath);
  } else if (valueType === ValueType.Array) {
    mergeArrayNode(node, left, right, delta, nextPath);
  } else {
    node.oldValue = left !== undefined ? left : undefined;
    node.newValue = right !== undefined ? right : undefined;
    if (delta) {
      node.status = DiffStatus.Modified;
    } else if (left !== undefined && right !== undefined && left !== right) {
      node.status = DiffStatus.Modified;
    } else {
      node.status = DiffStatus.Unchanged;
    }
  }

  return node;
}

/**
 * Merges object (map) nodes recursively and populates the node's children and status.
 */
function mergeObjectNode(
  node: MergeNode,
  left: unknown,
  right: unknown,
  delta: Delta | undefined,
  nextPath: string,
): void {
  const leftObj = (left && typeof left === 'object' ? left : {}) as Record<
    string,
    unknown
  >;
  const rightObj = (right && typeof right === 'object' ? right : {}) as Record<
    string,
    unknown
  >;
  const objDelta = delta as Record<string, Delta> | undefined;

  const keys = Array.from(
    new Set([...Object.keys(leftObj), ...Object.keys(rightObj)]),
  ).sort();

  if (keys.length === 0) {
    node.oldValue = left !== undefined ? left : undefined;
    node.newValue = right !== undefined ? right : undefined;
  }

  node.children = keys.map((k) => {
    const itemDelta = objDelta ? objDelta[k] : undefined;
    let itemStatus: DiffStatus = DiffStatus.Unchanged;

    if (itemDelta) {
      itemStatus = DiffStatus.Modified;
    } else if (left !== undefined && k in rightObj && !(k in leftObj)) {
      itemStatus = DiffStatus.Added;
    } else if (right !== undefined && k in leftObj && !(k in rightObj)) {
      itemStatus = DiffStatus.Deleted;
    }

    const childNode = buildMergeTree(
      leftObj[k],
      rightObj[k],
      itemDelta,
      nextPath,
      k,
    );
    if (
      itemStatus !== DiffStatus.Unchanged &&
      itemStatus !== DiffStatus.Modified
    ) {
      childNode.status = itemStatus;
    }
    return childNode;
  });

  // If all children are unchanged, parent is unchanged.
  if (delta && node.children.every((c) => c.status === DiffStatus.Unchanged)) {
    node.status = DiffStatus.Unchanged;
  }
}

/**
 * Merges array nodes recursively and populates the node's children and status.
 */
function mergeArrayNode(
  node: MergeNode,
  left: unknown,
  right: unknown,
  delta: Delta | undefined,
  nextPath: string,
): void {
  const leftArr = (left || []) as unknown[];
  const rightArr = (right || []) as unknown[];
  const arrDelta = delta as Record<string, Delta> | undefined;

  if (leftArr.length === 0 && rightArr.length === 0) {
    node.oldValue = left !== undefined ? left : undefined;
    node.newValue = right !== undefined ? right : undefined;
  }

  const children: MergeNode[] = [];

  const moves = new Map<number, number>();
  const reverseMoves = new Map<number, number>();
  if (arrDelta) {
    for (const key of Object.keys(arrDelta)) {
      if (key.startsWith('_') && key !== '_t') {
        const val = arrDelta[key];
        if (isMovedDelta(val)) {
          const originalIndex = parseInt(key.substring(1), 10);
          const destinationIndex = val[1];
          moves.set(originalIndex, destinationIndex);
          reverseMoves.set(destinationIndex, originalIndex);
        }
      }
    }
  }

  // Map left elements by their hash to a list of original indices (multimap).
  const leftHashMap = new Map<string, number[]>();
  leftArr.forEach((val, idx) => {
    const hash = getHash(val, idx);
    if (!leftHashMap.has(hash)) {
      leftHashMap.set(hash, []);
    }
    leftHashMap.get(hash)!.push(idx);
  });

  // Phase 1: Pre-compute pairings.
  const rightToLeft = new Map<number, number>();
  const leftToRight = new Map<number, number>();
  const pairedLeftIndices = new Set<number>();

  for (let rIdx = 0; rIdx < rightArr.length; rIdx++) {
    const rightItem = rightArr[rIdx];
    const rightHash = getHash(rightItem, rIdx);

    if (reverseMoves.has(rIdx)) {
      const lIdx = reverseMoves.get(rIdx)!;
      rightToLeft.set(rIdx, lIdx);
      leftToRight.set(lIdx, rIdx);
      pairedLeftIndices.add(lIdx);
    } else {
      const candidates = leftHashMap.get(rightHash) || [];
      const unusedCandidate = candidates.find(
        (idx) => !pairedLeftIndices.has(idx) && !moves.has(idx),
      );
      if (unusedCandidate !== undefined) {
        rightToLeft.set(rIdx, unusedCandidate);
        leftToRight.set(unusedCandidate, rIdx);
        pairedLeftIndices.add(unusedCandidate);
      }
    }
  }

  // Phase 2: Merge and Interleave.
  const addedLeftOut = new Set<number>();

  const addLeftElement = (l: number) => {
    const leftItem = leftArr[l];
    if (moves.has(l)) {
      const destIdx = moves.get(l)!;
      const rightItem = rightArr[destIdx];
      const childNode = buildMergeTree(
        leftItem,
        rightItem, // Pass the destination item to detect modifications.
        undefined,
        nextPath,
        `[${l}]`,
      );
      childNode.status = DiffStatus.MovedOut;
      childNode.movedTo = `[${destIdx}]`;
      children.push(childNode);
    } else {
      const childNode = buildMergeTree(
        leftItem,
        undefined,
        undefined,
        nextPath,
        `[${l}]`,
      );
      childNode.status = DiffStatus.Deleted;
      children.push(childNode);
    }
    addedLeftOut.add(l);
  };

  for (let rIdx = 0; rIdx < rightArr.length; rIdx++) {
    const idxKey = String(rIdx);
    const itemDelta = arrDelta ? arrDelta[idxKey] : undefined;
    const rightItem = rightArr[rIdx];

    if (rightToLeft.has(rIdx)) {
      const lIdx = rightToLeft.get(rIdx)!;

      // Insert any pending deleted/moved-out elements that originally appeared before lIdx.
      for (let l = 0; l < lIdx; l++) {
        if (!addedLeftOut.has(l) && (!leftToRight.has(l) || moves.has(l))) {
          addLeftElement(l);
        }
      }

      const leftItem = leftArr[lIdx];
      const childNode = buildMergeTree(
        leftItem,
        rightItem,
        itemDelta,
        nextPath,
        `[${rIdx}]`,
      );

      if (reverseMoves.has(rIdx)) {
        childNode.status = DiffStatus.MovedIn;
        childNode.movedFrom = `[${lIdx}]`;
      }
      children.push(childNode);
    } else {
      // Newly added element.
      const childNode = buildMergeTree(
        undefined,
        rightItem,
        itemDelta,
        nextPath,
        `[${rIdx}]`,
      );
      childNode.status = DiffStatus.Added;
      children.push(childNode);
    }
  }

  // Process any remaining left elements (Deleted or MovedOut at the end).
  for (let l = 0; l < leftArr.length; l++) {
    if (!addedLeftOut.has(l) && (!leftToRight.has(l) || moves.has(l))) {
      addLeftElement(l);
    }
  }

  node.children = children;
}

/**
 * Recursively marks all nodes in the given tree with the specified status.
 */
function markAllAs(
  nodes: MergeNode[] | undefined,
  status: DiffStatus,
): MergeNode[] | undefined {
  if (!nodes) {
    return undefined;
  }
  return nodes.map((node) => {
    const updatedNode: MergeNode = {
      ...node,
      status,
    };
    if (node.children) {
      updatedNode.children = markAllAs(node.children, status);
    }
    return updatedNode;
  });
}
