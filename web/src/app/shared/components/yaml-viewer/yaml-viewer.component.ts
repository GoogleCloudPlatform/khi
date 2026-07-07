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

import {
  Component,
  computed,
  input,
  signal,
  ElementRef,
  effect,
  output,
  untracked,
  inject,
  AfterViewInit,
  OnDestroy,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import * as yaml from 'js-yaml';
import * as jsondiffpatch from 'jsondiffpatch';
import { DiffStatus } from 'src/app/shared/components/yaml-viewer/lcs';
import {
  buildMergeTree,
  MergeNode,
  ValueType,
} from 'src/app/shared/components/yaml-viewer/diff-util';
import {
  YamlLine,
  RenderSegment,
  YamlMatch,
  renderNode,
  postRender,
  getRenderSegments as diffGetRenderSegments,
} from 'src/app/shared/components/yaml-viewer/diff-renderer';
import {
  YamlAnnotationProvider,
  YamlFieldAnnotation,
  AnnotationSeverity,
} from 'src/app/shared/components/yaml-viewer/yaml-annotation';
import { DynamicTooltipDirective } from 'src/app/shared/components/yaml-viewer/dynamic-tooltip.directive';

/**
 * Component for displaying YAML content with preview and diff capabilities.
 * Supports syntax highlighting, line numbers, path-based tooltips, and text searching.
 */
@Component({
  selector: 'khi-yaml-viewer',
  templateUrl: './yaml-viewer.component.html',
  styleUrls: ['./yaml-viewer.component.scss'],
  imports: [CommonModule, DynamicTooltipDirective],
})
export class YamlViewerComponent implements AfterViewInit, OnDestroy {
  protected readonly AnnotationSeverity = AnnotationSeverity;
  protected readonly DiffStatus = DiffStatus;
  protected readonly ValueType = ValueType;

  /**
   * 1-based index of the active match to highlight and scroll to.
   */
  readonly activeMatchIndex = input<number>(0);

  /**
   * Emits the total number of matches found.
   */
  readonly matchCount = output<number>();

  /**
   * 1-based index of the active diff block to scroll to.
   */
  readonly activeDiffIndex = input<number>(0);

  /**
   * Emits the total number of diff blocks found.
   */
  readonly diffCount = output<number>();

  private readonly el = inject(ElementRef);

  /**
   * The original YAML content. If not provided or equal to rightYaml, the component
   * renders in single preview mode showing only the updated YAML.
   */
  readonly leftYaml = input<string | null>(null);

  /**
   * The updated/new YAML content to display.
   */
  readonly rightYaml = input.required<string>();

  /**
   * Providers to generate field-level annotations (like tooltips).
   */
  readonly annotationProviders = input<YamlAnnotationProvider[]>([]);

  /**
   * Search query string. Occurrences of this query in the rendered YAML
   * will be wrapped in highlights.
   */
  readonly searchQuery = input<string>('');

  /**
   * Set of JSON paths that are currently collapsed.
   */
  readonly collapsedPaths = signal<Set<string>>(new Set());

  /**
   * The currently hovered move ID.
   */
  readonly hoveredMoveId = signal<string | null>(null);

  /**
   * All calculated relation arrows.
   */
  readonly allArrows = signal<{ moveId: string; path: string }[]>([]);

  private resizeObserver?: ResizeObserver;

  /**
   * Computed list of all YamlLines representing the document structure and diff status.
   */
  readonly rawLines = computed<YamlLine[]>(() => {
    const leftRaw = this.leftYaml();
    const rightRaw = this.rightYaml();
    const providers = this.annotationProviders();

    let leftObj: unknown = null;
    let rightObj: unknown = null;

    try {
      if (leftRaw) {
        leftObj = yaml.load(leftRaw);
      }
    } catch (e) {
      console.warn('Failed to parse left YAML:', e);
    }

    try {
      rightObj = yaml.load(rightRaw);
    } catch (e) {
      console.warn('Failed to parse right YAML:', e);
      return [];
    }

    const isDiffMode =
      leftObj !== null && JSON.stringify(leftObj) !== JSON.stringify(rightObj);

    let mergeTree: MergeNode;
    if (isDiffMode && leftObj !== null) {
      // Configure jsondiffpatch to identify objects in arrays by their properties (e.g., 'type' for conditions).
      const patcher = jsondiffpatch.create({
        objectHash: (obj: unknown, index?: number) => {
          if (obj && typeof obj === 'object') {
            const record = obj as Record<string, unknown>;
            // Kubernetes status.conditions elements are identified by their 'type'
            if ('type' in record) {
              return String(record['type']);
            }
            // Fallback to name if available
            if ('name' in record) {
              return String(record['name']);
            }
          }
          return obj !== undefined && obj !== null
            ? String(obj)
            : index !== undefined
              ? String(index)
              : undefined;
        },
      });
      const delta = patcher.diff(leftObj, rightObj);
      mergeTree = buildMergeTree(leftObj, rightObj, delta, '', '');
    } else {
      mergeTree = buildMergeTree(rightObj, rightObj, undefined, '', '');
    }

    const flatLines: YamlLine[] = [];
    renderNode(mergeTree, 0, false, flatLines);
    postRender(flatLines);

    // Apply annotations and line numbers
    let currentLineNumber = 1;

    // Compute annotations from rightObj.
    const annotationMap = new Map<string, YamlFieldAnnotation[]>();
    for (const provider of providers) {
      if (rightObj) {
        for (const ann of provider.getAnnotations(rightObj)) {
          // Join the path array to match diff-util's string format (e.g. "metadata.name")
          const pathStr = ann.path
            .map((p) =>
              typeof p === 'number' ||
              (typeof p === 'string' && p.startsWith('['))
                ? `[${p}]`
                : p,
            )
            .join('.');
          if (!annotationMap.has(pathStr)) {
            annotationMap.set(pathStr, []);
          }
          annotationMap.get(pathStr)!.push(ann);
        }
      }
    }

    return flatLines.map((line) => {
      const lineCopy = { ...line };
      if (lineCopy.path && annotationMap.has(lineCopy.path)) {
        const annotations = annotationMap.get(lineCopy.path)!;
        lineCopy.annotations = annotations;

        let maxSeverity = AnnotationSeverity.Low;
        for (const ann of annotations) {
          const sev = ann.severity ?? AnnotationSeverity.Low;
          if (sev > maxSeverity) {
            maxSeverity = sev;
          }
        }
        lineCopy.maxSeverity = maxSeverity;
      }
      if (
        lineCopy.diffStatus !== DiffStatus.Deleted &&
        lineCopy.diffStatus !== DiffStatus.MovedOut
      ) {
        lineCopy.lineNumber = currentLineNumber++;
      }
      return lineCopy;
    });
  });

  /**
   * Computed list of visible YamlLines after applying collapsed states.
   */
  readonly lines = computed<YamlLine[]>(() => {
    const raw = this.rawLines();
    const collapsed = this.collapsedPaths();
    if (collapsed.size === 0) {
      return raw;
    }

    const result: YamlLine[] = [];
    let skipUntilPath: string | null = null;

    for (const line of raw) {
      if (skipUntilPath) {
        if (
          line.path === skipUntilPath ||
          line.path.startsWith(skipUntilPath + '.')
        ) {
          continue;
        }
        skipUntilPath = null;
      }

      if (
        line.isCollapsible &&
        line.collapsedPath &&
        collapsed.has(line.collapsedPath)
      ) {
        const modifiedLine = { ...line };

        // If we are collapsing an array element start (e.g. "- name: web"),
        // we clear the key so it renders as "- ...".
        if (
          modifiedLine.isArrayElementStart &&
          modifiedLine.collapsedPath === line.collapsedPath
        ) {
          modifiedLine.key = undefined;
        }

        // Set valueSegments to '...' to render the collapsed indicator.
        modifiedLine.valueSegments = [
          { text: '...', diffStatus: DiffStatus.Unchanged },
        ];
        modifiedLine.valueText = '...';
        modifiedLine.valueType = ValueType.None;

        result.push(modifiedLine);
        skipUntilPath = line.collapsedPath;
      } else {
        result.push(line);
      }
    }

    return result;
  });

  /**
   * List of matches found for the current search query.
   */
  readonly matches = computed<YamlMatch[]>(() => {
    const query = this.searchQuery();
    const currentLines = this.lines();
    if (!query) {
      return [];
    }
    const list: YamlMatch[] = [];
    const lowerQuery = query.toLowerCase();
    currentLines.forEach((line, lineIndex) => {
      const text =
        (line.key ? line.key + ': ' : '') +
        (line.valueSegments
          ? line.valueSegments.map((s) => s.text).join('')
          : '');
      const lowerText = text.toLowerCase();
      let idx = 0;
      while (idx < text.length) {
        const matchIdx = lowerText.indexOf(lowerQuery, idx);
        if (matchIdx === -1) {
          break;
        }
        list.push({
          lineIndex,
          startChar: matchIdx,
          endChar: matchIdx + query.length,
        });
        idx = matchIdx + query.length;
      }
    });
    return list;
  });

  /**
   * The currently active match.
   */
  readonly activeMatch = computed<YamlMatch | null>(() => {
    const index = this.activeMatchIndex();
    return this.matches()[index - 1] ?? null;
  });

  /**
   * Computed starting line indices of contiguous diff blocks (lines with non-unchanged diffStatus).
   */
  readonly diffLineIndices = computed<number[]>(() => {
    const currentLines = this.lines();
    const indices: number[] = [];
    let inBlock = false;

    currentLines.forEach((line, index) => {
      const isChanged = line.diffStatus !== DiffStatus.Unchanged;
      if (isChanged) {
        if (!inBlock) {
          indices.push(index);
          inBlock = true;
        }
      } else {
        inBlock = false;
      }
    });

    return indices;
  });

  constructor() {
    effect(() => {
      const count = this.matches().length;
      untracked(() => {
        this.matchCount.emit(count);
      });
    });

    effect(() => {
      const active = this.activeMatch();
      if (active) {
        setTimeout(() => {
          const lineEl =
            this.el.nativeElement.querySelectorAll('.yaml-line-wrapper')[
              active.lineIndex
            ];
          lineEl?.scrollIntoView({ behavior: 'smooth', block: 'center' });
        });
      }
    });

    effect(() => {
      const count = this.diffLineIndices().length;
      untracked(() => {
        this.diffCount.emit(count);
      });
    });

    effect(() => {
      const indices = this.diffLineIndices();
      const activeIdx = this.activeDiffIndex();
      if (activeIdx > 0 && activeIdx <= indices.length) {
        const lineIndex = indices[activeIdx - 1];
        setTimeout(() => {
          const lineEl =
            this.el.nativeElement.querySelectorAll('.yaml-line-wrapper')[
              lineIndex
            ];
          lineEl?.scrollIntoView({ behavior: 'smooth', block: 'center' });
        });
      }
    });

    effect(() => {
      this.lines();
      // Wait for DOM to update before rendering arrows
      setTimeout(() => {
        this.renderAllArrows();
      });
    });
  }

  /**
   * Handles mouse enter on a moved block to highlight its arrow.
   */
  protected onMoveBlockMouseEnter(moveId: string) {
    this.hoveredMoveId.set(moveId);
  }

  /**
   * Handles mouse leave to clear the highlight.
   */
  protected onMoveBlockMouseLeave() {
    this.hoveredMoveId.set(null);
  }

  ngAfterViewInit() {
    const container = this.el.nativeElement.querySelector(
      '.yaml-code-container',
    );
    if (container) {
      this.resizeObserver = new ResizeObserver(() => {
        this.renderAllArrows();
      });
      this.resizeObserver.observe(container);
    }
  }

  ngOnDestroy() {
    this.resizeObserver?.disconnect();
  }

  /**
   * Calculates and updates the SVG paths for all arrows connecting MovedOut and MovedIn.
   */
  private renderAllArrows() {
    const currentLines = this.lines();
    const moveIds = new Set<string>();
    currentLines.forEach((l) => {
      if (l.moveId) {
        moveIds.add(l.moveId);
      }
    });

    const container = this.el.nativeElement.querySelector(
      '.yaml-code-container',
    );
    if (!container) {
      this.allArrows.set([]);
      return;
    }

    const containerRect = container.getBoundingClientRect();
    const newArrows: { moveId: string; path: string }[] = [];

    moveIds.forEach((moveId) => {
      const outEl = this.el.nativeElement.querySelector(
        `[data-move-id="${moveId}"][data-move-type="out"] .line-content`,
      ) as HTMLElement;
      const inEl = this.el.nativeElement.querySelector(
        `[data-move-id="${moveId}"][data-move-type="in"] .line-content`,
      ) as HTMLElement;

      if (outEl && inEl) {
        const outRect = outEl.getBoundingClientRect();
        const inRect = inEl.getBoundingClientRect();

        // Align to the left of the line content (after the gutter)
        const startX = outRect.left - containerRect.left;
        const startY = outRect.top - containerRect.top + outRect.height / 2;
        const endX = inRect.left - containerRect.left;
        const endY = inRect.top - containerRect.top + inRect.height / 2;

        const distY = Math.abs(endY - startY);
        // Bulge to the left. Larger distance = larger bulge.
        const bulge = Math.min(20, 5 + distY * 0.1);

        // Add a straight horizontal segment at the start and end to ensure
        // the line enters the arrowhead horizontally.
        const straightLength = 4;
        const startX2 = startX - straightLength;
        const endX2 = endX - straightLength;

        const controlX = Math.min(startX2, endX2) - bulge;

        // Path: line to startX2, cubic bezier to endX2, line to endX
        const path = `M ${startX} ${startY} L ${startX2} ${startY} C ${controlX} ${startY}, ${controlX} ${endY}, ${endX2} ${endY} L ${endX} ${endY}`;
        newArrows.push({ moveId, path });
      }
    });

    this.allArrows.set(newArrows);
  }

  /**
   * Checks if a given line is collapsed.
   */
  protected isCollapsed(line: YamlLine): boolean {
    return (
      !!line.collapsedPath && this.collapsedPaths().has(line.collapsedPath)
    );
  }

  /**
   * Toggles the collapsed state of a line.
   */
  protected toggleCollapsed(line: YamlLine, event: MouseEvent): void {
    event.stopPropagation();
    if (!line.collapsedPath) {
      return;
    }
    const current = this.collapsedPaths();
    const next = new Set(current);
    if (next.has(line.collapsedPath)) {
      next.delete(line.collapsedPath);
    } else {
      next.add(line.collapsedPath);
    }
    this.collapsedPaths.set(next);
  }

  /**
   * Computes the flat render segments for a line's content, splitting them by search query matches.
   */
  protected getRenderSegments(
    line: YamlLine,
    query: string,
    lineIndex: number,
  ): RenderSegment[] {
    return diffGetRenderSegments(line, query, lineIndex, this.activeMatch());
  }

  /**
   * Custom copy event handler. Reconstructs the copied text line-by-line
   * from the DOM selection to avoid newlines introduced by HTML template formatting.
   */
  protected onCopy(event: ClipboardEvent): void {
    const selection = window.getSelection();
    if (!selection || selection.isCollapsed) {
      return;
    }

    const range = selection.getRangeAt(0);
    const container = range.commonAncestorContainer;
    const rootElement =
      container.nodeType === Node.ELEMENT_NODE
        ? (container as HTMLElement)
        : container.parentElement;

    if (!rootElement) {
      return;
    }

    // Find all line content elements within the selection range
    const lineElements = Array.from(
      rootElement.querySelectorAll('.yaml-line-wrapper'),
    ).filter((lineEl) => selection.containsNode(lineEl, true));

    if (lineElements.length === 0) {
      return;
    }

    const copiedLines: string[] = [];

    for (const lineEl of lineElements) {
      const contentEl = lineEl.querySelector('.line-content');
      if (!contentEl) {
        continue;
      }

      // Check for diff indicator (+, -, or space)
      const indicatorEl = lineEl.querySelector('.diff-indicator');
      const indicatorText = indicatorEl?.textContent?.trim() || '';

      // Reconstruct the text content of the line by concatenating all text nodes.
      let lineText = '';
      const walker = document.createTreeWalker(
        contentEl,
        NodeFilter.SHOW_TEXT,
        null,
      );

      let currentNode = walker.nextNode();
      while (currentNode) {
        // If the selection partially covers this line, only copy the selected part.
        if (selection.containsNode(currentNode, true)) {
          let text = currentNode.nodeValue || '';
          if (currentNode === range.startContainer) {
            text = text.substring(range.startOffset);
          }
          if (currentNode === range.endContainer) {
            text = text.substring(0, range.endOffset);
          }
          lineText += text;
        }
        currentNode = walker.nextNode();
      }

      // Clean up any double/extra newlines that might have leaked from the template structure
      // within the line itself, but preserve spaces (like indentation).
      const cleanedLineText = lineText.replace(/\r?\n/g, '');
      if (cleanedLineText || indicatorText === '+' || indicatorText === '-') {
        // Prepend the diff indicator (+ or -) if it exists, otherwise prepend a space to align.
        const prefix =
          indicatorText === '+' || indicatorText === '-' ? indicatorText : ' ';
        copiedLines.push(prefix + cleanedLineText);
      }
    }

    if (copiedLines.length > 0) {
      event.clipboardData?.setData('text/plain', copiedLines.join('\n'));
      event.preventDefault();
    }
  }
}
