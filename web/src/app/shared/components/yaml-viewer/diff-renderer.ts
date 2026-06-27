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

import { DiffStatus, ValueSegment, diffStrings } from './lcs';
import {
  ValueType,
  MergeNode,
  getValueType,
  shouldHighlightEntireValue,
} from './diff-util';

/**
 * Represents a single rendered line of the YAML document.
 */
export interface YamlLine {
  /** The full text content of this line, including indentation. */
  text: string;
  /** The indentation level (number of spaces). */
  indent: number;
  /** The property key (undefined for array elements without keys). */
  key?: string;
  /** The string representation of the value. */
  valueText: string;
  /** The Type of the value, used for syntax highlighting. */
  valueType: ValueType;
  /** Optional segments for character-level diff highlighting. */
  valueSegments?: ValueSegment[];
  /** The difference status of this line relative to the old YAML. */
  diffStatus: DiffStatus;
  /** The full JSON path of this property (e.g., "metadata.name"). */
  path: string;
  /** Optional tooltip message to display on hover. */
  tooltip?: string;
  /** For moved items, the original path or index before the move. */
  movedFrom?: string;
  /** For moved items, the destination path or index after the move. */
  movedTo?: string;
  /** Whether this line is the start of a moved block (for border styling). */
  moveBlockStart?: boolean;
  /** Whether this line is the end of a moved block (for border styling). */
  moveBlockEnd?: boolean;
  /** The calculated width of the object content in 'ch' units (approx character width). */
  contentWidthCh?: number;
  /** The line number in the final/new document. Undefined for deleted lines. */
  lineNumber?: number;
  /** Whether this line starts an array element (e.g. has a "- " prefix). */
  isArrayElementStart?: boolean;
  /** Whether this line starts a collapsible section. */
  isCollapsible?: boolean;
  /** The path that will be toggled if this line is collapsed/expanded. */
  collapsedPath?: string;
  /** Unique identifier for matching MovedIn and MovedOut blocks. */
  moveId?: string;
}

/**
 * Represents a segment of a YAML line prepared for rendering with highlight and diff information.
 */
export interface RenderSegment {
  text: string;
  diffStatus: DiffStatus;
  isMatch: boolean;
  isActiveMatch?: boolean;
  isKey?: boolean;
  isColon?: boolean;
  isValue?: boolean;
}

/**
 * Represents a match found during a search in the YAML document.
 */
export interface YamlMatch {
  lineIndex: number;
  startChar: number;
  endChar: number;
}

/**
 * Recursively renders a MergeNode into a flat list of YamlLines.
 */
export function renderNode(
  node: MergeNode,
  indent: number,
  isArrayElement: boolean,
  result: YamlLine[],
  collapsiblePath?: string,
  parentStatus?: DiffStatus,
  parentMovedFrom?: string,
  parentMovedTo?: string,
  parentMoveId?: string,
) {
  if (node.key === 'foo') {
    console.log('renderNode foo:', {
      valueType: node.valueType,
      oldValueType: node.oldValueType,
      status: node.status,
      childrenLen: node.children?.length,
      isArrayElement,
      indent,
    });
  }

  // Handle Type Modification (Type change)
  if (node.oldValueType !== undefined && node.oldValueType !== node.valueType) {
    const deletedNode: MergeNode = {
      key: node.key,
      path: node.path,
      status: DiffStatus.Deleted,
      valueType: node.oldValueType,
      oldValue: node.oldValue,
      children: node.children?.filter((c) => c.status === DiffStatus.Deleted),
    };
    const addedNode: MergeNode = {
      key: node.key,
      path: node.path,
      status: DiffStatus.Added,
      valueType: node.valueType,
      newValue: node.newValue,
      children: node.children?.filter((c) => c.status === DiffStatus.Added),
    };

    renderNode(
      deletedNode,
      indent,
      isArrayElement,
      result,
      collapsiblePath,
      parentStatus,
      parentMovedFrom,
      parentMovedTo,
      parentMoveId,
    );
    renderNode(
      addedNode,
      indent,
      isArrayElement,
      result,
      collapsiblePath,
      parentStatus,
      parentMovedFrom,
      parentMovedTo,
      parentMoveId,
    );
    return;
  }

  const isEmptyObject =
    node.valueType === ValueType.Object &&
    (!node.children || node.children.length === 0);
  const isEmptyArray =
    node.valueType === ValueType.Array &&
    (!node.children || node.children.length === 0);
  const isScalar =
    (node.valueType !== 'object' && node.valueType !== 'array') ||
    isEmptyObject ||
    isEmptyArray;

  let moveId = parentMoveId;
  if (!moveId) {
    if (node.status === DiffStatus.MovedIn && node.movedFrom) {
      const parentPath = node.path.substring(0, node.path.lastIndexOf('.'));
      moveId = `${parentPath}.${node.movedFrom}->${node.path}`;
    } else if (node.status === DiffStatus.MovedOut && node.movedTo) {
      const parentPath = node.path.substring(0, node.path.lastIndexOf('.'));
      moveId = `${node.path}->${parentPath}.${node.movedTo}`;
    }
  }

  if (isScalar) {
    renderScalarNode(
      node,
      indent,
      isArrayElement,
      result,
      collapsiblePath,
      parentStatus,
      parentMovedFrom,
      parentMovedTo,
      moveId,
    );
  } else {
    renderCompositeNode(
      node,
      indent,
      isArrayElement,
      result,
      collapsiblePath,
      parentStatus,
      parentMovedFrom,
      parentMovedTo,
      moveId,
    );
  }
}

/**
 * Performs post-processing on rendered YAML lines.
 * Identifies contiguous blocks of MovedIn/MovedOut to mark their boundaries and
 * calculates the maximum content width for each block to prevent text wrapping.
 */
export function postRender(flatLines: YamlLine[]) {
  const blocks: { start: number; end: number }[] = [];
  let currentBlockStart = -1;

  for (let i = 0; i < flatLines.length; i++) {
    const line = flatLines[i];
    if (
      line.diffStatus === DiffStatus.MovedIn ||
      line.diffStatus === DiffStatus.MovedOut
    ) {
      const prevLine = flatLines[i - 1];
      if (
        !prevLine ||
        prevLine.diffStatus !== line.diffStatus ||
        !isSameMoveBlock(prevLine, line)
      ) {
        line.moveBlockStart = true;
        currentBlockStart = i;
      }
      const nextLine = flatLines[i + 1];
      if (
        !nextLine ||
        nextLine.diffStatus !== line.diffStatus ||
        !isSameMoveBlock(line, nextLine)
      ) {
        line.moveBlockEnd = true;
        if (currentBlockStart !== -1) {
          blocks.push({ start: currentBlockStart, end: i });
          currentBlockStart = -1;
        }
      }
    }
  }

  // Calculate the maximum width for each block and assign it to the lines.
  for (const block of blocks) {
    const blockLines = flatLines.slice(block.start, block.end + 1);
    const minIndent = Math.min(...blockLines.map((l) => l.indent));

    // Calculate visual end position for each line: (indent - minIndent) + contentLength
    const lineInfos = blockLines.map((line) => {
      let relativeIndent = line.indent - minIndent;
      if (line.isArrayElementStart) {
        relativeIndent += 2; // Account for the '- ' indicator which is outside the border
      }
      let contentLength = 0;
      if (line.key) {
        contentLength += line.key.length + 2; // key + ': '
      }
      if (line.valueText) {
        contentLength += line.valueText.length;
      }
      return {
        line,
        relativeIndent,
        contentLength,
        visualEnd: relativeIndent + contentLength,
      };
    });

    const maxVisualEnd = Math.max(...lineInfos.map((info) => info.visualEnd));

    // Set width for each line in the block with a 1ch buffer to prevent wrapping due to font variations.
    for (const info of lineInfos) {
      info.line.contentWidthCh = maxVisualEnd - info.relativeIndent + 1;
    }
  }
}

/**
 * Helper to check if two lines belong to the same moved block based on their paths.
 */
function isSameMoveBlock(a: YamlLine, b: YamlLine): boolean {
  const getBlockKey = (path: string | undefined): string | null => {
    if (!path) {
      return null;
    }
    // Extract the path up to the array index (e.g., "items[2].id" -> "items[2]")
    const match = path.match(/(.*?\[\d+\])/);
    return match ? match[1] : path;
  };

  if (a.movedFrom && b.movedFrom) {
    return getBlockKey(a.movedFrom) === getBlockKey(b.movedFrom);
  }
  if (a.movedTo && b.movedTo) {
    return getBlockKey(a.movedTo) === getBlockKey(b.movedTo);
  }
  return false;
}

function isMultilineString(val: unknown): val is string {
  return typeof val === 'string' && val.includes('\n');
}

function renderScalarNode(
  node: MergeNode,
  indent: number,
  isArrayElement: boolean,
  result: YamlLine[],
  collapsiblePath?: string,
  parentStatus?: DiffStatus,
  parentMovedFrom?: string,
  parentMovedTo?: string,
  moveId?: string,
) {
  const effectiveStatus =
    parentStatus === DiffStatus.MovedIn || parentStatus === DiffStatus.MovedOut
      ? parentStatus
      : node.status === DiffStatus.Unchanged &&
          parentStatus &&
          parentStatus !== DiffStatus.Modified
        ? parentStatus
        : node.status;

  switch (effectiveStatus) {
    case DiffStatus.Modified: {
      const oldStr = formatValue(node.oldValue);
      const newStr = formatValue(node.newValue);

      let oldSegs: ValueSegment[];
      let newSegs: ValueSegment[];

      if (shouldHighlightEntireValue(node.oldValue, node.newValue)) {
        oldSegs = [{ text: oldStr, diffStatus: DiffStatus.Deleted }];
        newSegs = [{ text: newStr, diffStatus: DiffStatus.Added }];
      } else {
        const diff = diffStrings(oldStr, newStr);
        oldSegs = diff.oldSegs;
        newSegs = diff.newSegs;
      }

      renderScalar(
        node,
        node.key,
        node.oldValue,
        DiffStatus.Deleted,
        isArrayElement,
        indent,
        result,
        oldSegs,
        collapsiblePath,
        moveId,
      );
      renderScalar(
        node,
        node.key,
        node.newValue,
        DiffStatus.Added,
        isArrayElement,
        indent,
        result,
        newSegs,
        collapsiblePath,
        moveId,
      );
      break;
    }
    case DiffStatus.Deleted: {
      const valStr = formatValue(node.oldValue);
      renderScalar(
        node,
        node.key,
        node.oldValue,
        DiffStatus.Deleted,
        isArrayElement,
        indent,
        result,
        [
          {
            text: valStr,
            diffStatus: DiffStatus.Unchanged,
          },
        ],
        collapsiblePath,
        moveId,
      );
      break;
    }
    case DiffStatus.Added: {
      const valStr = formatValue(node.newValue);
      renderScalar(
        node,
        node.key,
        node.newValue,
        DiffStatus.Added,
        isArrayElement,
        indent,
        result,
        [
          {
            text: valStr,
            diffStatus: DiffStatus.Unchanged,
          },
        ],
        collapsiblePath,
        moveId,
      );
      break;
    }
    case DiffStatus.MovedOut: {
      const valStr = formatValue(node.oldValue);
      let segs: ValueSegment[];
      if (node.status === DiffStatus.Modified) {
        const oldStr = formatValue(node.oldValue);
        const newStr = formatValue(node.newValue);
        const { oldSegs } = diffStrings(oldStr, newStr);
        segs = oldSegs;
      } else {
        segs = [{ text: valStr, diffStatus: DiffStatus.Unchanged }];
      }

      renderScalar(
        node,
        node.key,
        node.oldValue,
        DiffStatus.MovedOut,
        isArrayElement,
        indent,
        result,
        segs,
        collapsiblePath,
        moveId,
      );
      const effectiveMovedTo = node.movedTo || parentMovedTo;
      if (result.length > 0 && effectiveMovedTo) {
        const lastLine = result[result.length - 1];
        lastLine.movedTo = effectiveMovedTo;
      }
      break;
    }
    case DiffStatus.MovedIn: {
      const valStr = formatValue(node.newValue);
      let segs: ValueSegment[];
      if (node.status === DiffStatus.Modified) {
        const oldStr = formatValue(node.oldValue);
        const newStr = formatValue(node.newValue);
        const { newSegs } = diffStrings(oldStr, newStr);
        segs = newSegs;
      } else {
        segs = [{ text: valStr, diffStatus: DiffStatus.Unchanged }];
      }

      renderScalar(
        node,
        node.key,
        node.newValue,
        DiffStatus.MovedIn,
        isArrayElement,
        indent,
        result,
        segs,
        collapsiblePath,
        moveId,
      );
      const effectiveMovedFrom = node.movedFrom || parentMovedFrom;
      if (result.length > 0 && effectiveMovedFrom) {
        const lastLine = result[result.length - 1];
        lastLine.movedFrom = effectiveMovedFrom;
      }
      break;
    }
    default: {
      const valStr = formatValue(node.newValue);
      renderScalar(
        node,
        node.key,
        node.newValue,
        node.status,
        isArrayElement,
        indent,
        result,
        [{ text: valStr, diffStatus: DiffStatus.Unchanged }],
        collapsiblePath,
        moveId,
      );
      break;
    }
  }
}

function renderCompositeNode(
  node: MergeNode,
  indent: number,
  isArrayElement: boolean,
  result: YamlLine[],
  collapsiblePath?: string,
  parentStatus?: DiffStatus,
  parentMovedFrom?: string,
  parentMovedTo?: string,
  moveId?: string,
) {
  const effectiveStatus =
    parentStatus === DiffStatus.MovedIn || parentStatus === DiffStatus.MovedOut
      ? parentStatus
      : node.status === DiffStatus.Unchanged &&
          parentStatus &&
          parentStatus !== DiffStatus.Modified
        ? parentStatus
        : node.status;
  const effectiveMovedFrom = node.movedFrom || parentMovedFrom;
  const effectiveMovedTo = node.movedTo || parentMovedTo;

  if (node.key && !node.key.startsWith('[')) {
    const prefix = isArrayElement ? '- ' : '';
    const actualIndent = isArrayElement ? indent - 2 : indent;
    const text = `${' '.repeat(actualIndent)}${prefix}${node.key}:`;
    const displayStatus: DiffStatus =
      effectiveStatus === DiffStatus.Added ||
      effectiveStatus === DiffStatus.Deleted ||
      effectiveStatus === DiffStatus.MovedOut ||
      effectiveStatus === DiffStatus.MovedIn
        ? effectiveStatus
        : DiffStatus.Unchanged;

    const line: YamlLine = {
      text,
      indent: actualIndent,
      key: node.key,
      valueText: '',
      valueType: ValueType.None,
      diffStatus: displayStatus,
      path: node.path,
      isArrayElementStart: isArrayElement,
      isCollapsible: true,
      collapsedPath: collapsiblePath || node.path,
      moveId,
    };

    if (effectiveStatus === DiffStatus.MovedOut && effectiveMovedTo) {
      line.movedTo = effectiveMovedTo;
    } else if (effectiveStatus === DiffStatus.MovedIn && effectiveMovedFrom) {
      line.movedFrom = effectiveMovedFrom;
    }

    result.push(line);
    collapsiblePath = undefined;
  }

  if (node.children) {
    const nextIndent = node.key ? indent + 2 : indent;
    const renderedLine = !!(node.key && !node.key.startsWith('['));
    const propagateArrayElement = isArrayElement && !renderedLine;

    node.children.forEach((child, index) => {
      const childIsArrayElement =
        node.valueType === ValueType.Array ||
        (propagateArrayElement && index === 0);

      const childCollapsiblePath = index === 0 ? collapsiblePath : undefined;
      const nextCollapsiblePath =
        childCollapsiblePath ||
        (node.valueType === ValueType.Array &&
        (child.valueType === ValueType.Object ||
          child.valueType === ValueType.Array)
          ? child.path
          : undefined);

      renderNode(
        child,
        nextIndent,
        childIsArrayElement,
        result,
        nextCollapsiblePath,
        effectiveStatus,
        effectiveMovedFrom,
        effectiveMovedTo,
        moveId,
      );
    });
  }
}

function renderScalar(
  node: MergeNode,
  key: string,
  val: unknown,
  status: DiffStatus,
  isArrEl: boolean,
  indent: number,
  result: YamlLine[],
  segs?: ValueSegment[],
  collapsiblePath?: string,
  moveId?: string,
) {
  if (isMultilineString(val)) {
    renderMultilineString(
      node,
      key,
      val,
      status,
      isArrEl,
      indent,
      result,
      segs,
      collapsiblePath,
      moveId,
    );
  } else {
    renderScalarLine(
      node,
      key,
      val,
      status,
      isArrEl,
      indent,
      result,
      segs,
      collapsiblePath,
      moveId,
    );
  }
}

function renderScalarLine(
  node: MergeNode,
  key: string,
  val: unknown,
  status: DiffStatus,
  isArrEl: boolean,
  indent: number,
  result: YamlLine[],
  segs?: ValueSegment[],
  collapsiblePath?: string,
  moveId?: string,
) {
  const valText = formatValue(val);
  const valType = getValueType(val);
  const keyPart = key ? (key.startsWith('[') ? '' : `${key}: `) : '';
  const prefix = isArrEl ? '- ' : '';
  const actualIndent = isArrEl ? indent - 2 : indent;
  const text = `${' '.repeat(actualIndent)}${prefix}${keyPart}${valText}`;

  result.push({
    text,
    indent: actualIndent,
    key: key && !key.startsWith('[') ? key : undefined,
    valueText: valText,
    valueType: valType,
    valueSegments: segs || [
      { text: valText, diffStatus: DiffStatus.Unchanged },
    ],
    diffStatus: status,
    path: node.path,
    isArrayElementStart: isArrEl,
    isCollapsible: !!collapsiblePath,
    collapsedPath: collapsiblePath,
    moveId,
  });
}

function renderMultilineString(
  node: MergeNode,
  key: string,
  val: string,
  status: DiffStatus,
  isArrEl: boolean,
  indent: number,
  result: YamlLine[],
  segs?: ValueSegment[],
  collapsiblePath?: string,
  moveId?: string,
) {
  const headerIndent = isArrEl ? indent - 2 : indent;
  const bodyIndent = isArrEl ? indent : indent + 2;
  const prefix = isArrEl ? '- ' : '';
  const keyPart = key ? (key.startsWith('[') ? '' : `${key}: `) : '';

  // 1. Render header line (e.g. "key: |" or "- |")
  const headerText = `${' '.repeat(headerIndent)}${prefix}${keyPart}|`;
  result.push({
    text: headerText,
    indent: headerIndent,
    key: key && !key.startsWith('[') ? key : undefined,
    valueText: '|',
    valueType: ValueType.None,
    valueSegments: [{ text: '|', diffStatus: DiffStatus.Unchanged }],
    diffStatus: status,
    path: node.path,
    isArrayElementStart: isArrEl,
    isCollapsible: !!collapsiblePath,
    collapsedPath: collapsiblePath,
    moveId,
  });

  // 2. Split segments by newline
  const linesSegs: ValueSegment[][] = [];
  let currentLineSegs: ValueSegment[] = [];

  const appendSeg = (text: string, diffStatus: DiffStatus) => {
    if (text) {
      currentLineSegs.push({ text, diffStatus });
    }
  };

  const segments = segs || [{ text: val, diffStatus: DiffStatus.Unchanged }];

  for (const seg of segments) {
    const parts = seg.text.split('\n');
    for (let i = 0; i < parts.length; i++) {
      if (i > 0) {
        linesSegs.push(currentLineSegs);
        currentLineSegs = [];
      }
      appendSeg(parts[i], seg.diffStatus);
    }
  }
  linesSegs.push(currentLineSegs);

  // Strip exactly one trailing newline because it is implied by the '|' indicator
  if (linesSegs.length > 1 && linesSegs[linesSegs.length - 1].length === 0) {
    linesSegs.pop();
  }

  // 3. Render body lines
  for (const lineSeg of linesSegs) {
    const lineTextContent = lineSeg.map((s) => s.text).join('');
    const text = `${' '.repeat(bodyIndent)}${lineTextContent}`;
    result.push({
      text,
      indent: bodyIndent,
      valueText: lineTextContent,
      valueType: ValueType.String,
      valueSegments: lineSeg,
      diffStatus: status,
      path: node.path,
      moveId,
    });
  }
}

/**
 * Computes the flat render segments for a line's content, splitting them by search query matches.
 */
export function getRenderSegments(
  line: YamlLine,
  query: string,
  lineIndex: number,
  activeMatch: YamlMatch | null,
): RenderSegment[] {
  const sources: {
    text: string;
    diffStatus: DiffStatus;
    isKey?: boolean;
    isColon?: boolean;
    isValue?: boolean;
  }[] = [];

  if (line.key) {
    sources.push({
      text: line.key,
      diffStatus: line.diffStatus,
      isKey: true,
    });
    sources.push({ text: ': ', diffStatus: line.diffStatus, isColon: true });
  }

  if (line.valueSegments) {
    for (const seg of line.valueSegments) {
      sources.push({
        text: seg.text,
        diffStatus: seg.diffStatus,
        isValue: true,
      });
    }
  }

  if (!query) {
    return sources.map((s) => ({ ...s, isMatch: false }));
  }

  const fullText = sources.map((s) => s.text).join('');
  const lowerText = fullText.toLowerCase();
  const lowerQuery = query.toLowerCase();

  const matchIntervals: [number, number][] = [];
  let idx = 0;
  while (idx < fullText.length) {
    const matchIdx = lowerText.indexOf(lowerQuery, idx);
    if (matchIdx === -1) {
      break;
    }
    matchIntervals.push([matchIdx, matchIdx + query.length]);
    idx = matchIdx + query.length;
  }

  const result: RenderSegment[] = [];
  let currentOffset = 0;

  for (const src of sources) {
    const srcLen = src.text.length;
    const srcStart = currentOffset;
    const srcEnd = currentOffset + srcLen;

    const splitPoints = new Set<number>([srcStart, srcEnd]);
    for (const [mStart, mEnd] of matchIntervals) {
      if (mStart > srcStart && mStart < srcEnd) {
        splitPoints.add(mStart);
      }
      if (mEnd > srcStart && mEnd < srcEnd) {
        splitPoints.add(mEnd);
      }
    }

    const sortedSplits = Array.from(splitPoints).sort((a, b) => a - b);

    for (let i = 0; i < sortedSplits.length - 1; i++) {
      const start = sortedSplits[i];
      const end = sortedSplits[i + 1];
      const subText = src.text.substring(start - srcStart, end - srcStart);
      const isMatch = matchIntervals.some(
        ([mStart, mEnd]) => start >= mStart && end <= mEnd,
      );
      const isActiveMatch =
        isMatch &&
        activeMatch !== null &&
        activeMatch.lineIndex === lineIndex &&
        start >= activeMatch.startChar &&
        end <= activeMatch.endChar;

      const segment: RenderSegment = {
        text: subText,
        diffStatus: src.diffStatus,
        isMatch,
        isActiveMatch,
      };
      if (src.isKey) {
        segment.isKey = true;
      }
      if (src.isColon) {
        segment.isColon = true;
      }
      if (src.isValue) {
        segment.isValue = true;
      }
      result.push(segment);
    }

    currentOffset = srcEnd;
  }

  return result;
}

/**
 * Helper to format values for YAML display.
 */
export function formatValue(value: unknown): string {
  if (value === null) {
    return 'null';
  }
  if (typeof value === 'string') {
    if (value.includes('\n')) {
      return value;
    }
    // Escape if needed, or wrap in quotes
    if (value === '' || value.includes(':') || value.includes(' ')) {
      return JSON.stringify(value);
    }
    return value;
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  if (Array.isArray(value) && value.length === 0) {
    return '[]';
  }
  if (value && typeof value === 'object' && Object.keys(value).length === 0) {
    return '{}';
  }
  return '';
}
