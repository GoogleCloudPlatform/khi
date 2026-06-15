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

import {
  Component,
  computed,
  effect,
  ElementRef,
  HostListener,
  inject,
  input,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { LogContentHeaderComponent } from 'src/app/log/components/log-content-header.component';
import { HighlightModule } from 'ngx-highlightjs';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltip } from '@angular/material/tooltip';
import { Log } from 'src/app/store/domain/log';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import { MatButtonModule } from '@angular/material/button';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { Clipboard, ClipboardModule } from '@angular/cdk/clipboard';

import { ResourceRefAnnotationViewModel } from 'src/app/log/components/resource-reference-list.component';
import { SearchBarComponent } from 'src/app/shared/components/search-bar/search-bar.component';
import { SearchScope } from 'src/app/services/view-state.service';

/**
 * View model aggregating the full detailed data required to render the log content and header.
 */
export interface LogContentViewModel {
  logEntry: ReadonlyDomainElement<Log> | null;
  logBody: string;
  parsedLogBody: unknown;
  resourceRefs: ResourceRefAnnotationViewModel[];
}

interface TextNodeSpan {
  node: Text;
  start: number;
  end: number;
}

interface TextRange {
  start: number;
  end: number;
}

interface MatchGroup {
  marks: HTMLElement[];
}

/**
 * Component responsible for displaying the detailed body of a log entry.
 * Provides actions such as copying the raw log content and copying a Cloud Logging query
 * for the specific log entry.
 */
@Component({
  selector: 'khi-log-content',
  templateUrl: './log-content.component.html',
  styleUrls: ['./log-content.component.scss'],
  imports: [
    CommonModule,
    LogContentHeaderComponent,
    HighlightModule,
    MatIconModule,
    MatTooltip,
    KHIIconRegistrationModule,
    MatButtonModule,
    MatSnackBarModule,
    ClipboardModule,
    SearchBarComponent,
  ],
})
export class LogContentComponent {
  private readonly clipboard = inject(Clipboard);
  private readonly snackBar = inject(MatSnackBar);

  /**
   * Signal tracking whether the in-logbody search bar is currently open.
   */
  public readonly isSearchOpen = signal<boolean>(false);

  /**
   * Signal storing the current search query string.
   */
  public readonly searchQuery = signal<string>('');

  /**
   * Signal storing the total number of matched search highlights.
   */
  public readonly matchCount = signal<number>(0);

  /**
   * Signal tracking the 1-based index of the currently active search match.
   */
  public readonly currentMatchIndex = signal<number>(0);

  /**
   * Reference to the code element containing the highlighted log body.
   */
  public readonly codeElement =
    viewChild<ElementRef<HTMLElement>>('codeElement');

  /**
   * Reference to the search bar component for focus management.
   */
  public readonly searchBar = viewChild(SearchBarComponent);

  /** Holds the current active search scope. */
  public readonly activeSearchScope = input<SearchScope>(SearchScope.Global);

  private matchGroups: MatchGroup[] = [];

  /**
   * Initializes the component and registers an effect to sync highlights.
   */
  constructor() {
    effect(() => {
      const query = this.searchQuery();
      const vm = this.vm();
      const isOpen = this.isSearchOpen();
      setTimeout(() => {
        if (isOpen && vm && query) {
          this.applyHighlights(query);
        } else {
          this.removeHighlights();
        }
      });
    });
  }

  /**
   * The aggregated view model containing the log entry, body, and resolved references.
   */
  public readonly vm = input<LogContentViewModel | null>(null);

  /**
   * The timezone shift to apply to the timestamp.
   */
  public timezoneShift = input<number>(0);

  /**
   * Output emitted when a resource timeline is clicked from the reference list.
   */
  public timelineSelected = output<number>();

  /**
   * Output emitted when a resource timeline is hovered from the reference list.
   */
  public timelineHighlighted = output<number>();

  /**
   * Output emitted when the mouse hovers over or focus enters/leaves the log content area.
   */
  public scopeActiveChange = output<boolean>();

  /**
   * Input tracking the currently selected timeline to visually indicate selection state
   * in the resource reference list.
   */
  public selectedTimeline = input<ReadonlyDomainElement<Timeline> | null>(null);

  private readonly timestampString = computed(() => {
    const parsed = this.vm()?.parsedLogBody as
      | { [key: string]: string }
      | undefined;
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed['timestamp'] ?? null;
    }
    return null;
  });

  private readonly insertId = computed(() => {
    const log = this.vm()?.logEntry;
    if (!log) {
      return null;
    }
    const id = log.body?.['insertId'];
    return typeof id === 'string' && id.trim() !== '' ? id : null;
  });

  /**
   * Determines if the "Copy Query" button should be visible.
   * True only if both a valid timestamp and insertId can be extracted from the loaded log body.
   */
  protected readonly showCopyQueryButton = computed(() => {
    return this.timestampString() !== null && this.insertId() !== null;
  });

  /**
   * Copies the loaded log body text to the clipboard and displays a notification.
   */
  copyLog() {
    const logBody = this.vm()?.logBody;
    if (!logBody) {
      return;
    }
    this.showCopySnackbarMessage(this.clipboard.copy(logBody));
  }

  /**
   * Copies a Cloud Logging query string uniquely identifying this log to the clipboard.
   * Extracts the insertId and timestamp from the log body to build the query.
   */
  copyLogQuery() {
    const log = this.vm()?.logEntry;
    const timestampString = this.timestampString();
    const insertId = this.insertId();
    if (!log || !timestampString || !insertId) {
      return;
    }
    this.showCopySnackbarMessage(
      this.clipboard.copy(`(
-- Log query for "${log.summary}"
insertId="${insertId}"
timestamp="${timestampString}"
)`),
    );
  }

  /**
   * Displays a snackbar notification indicating the result of a copy action.
   * @param success Whether the copy to clipboard operation was successful.
   */
  private showCopySnackbarMessage(success: boolean) {
    this.snackBar.open(success ? 'Copied!' : 'Copy failed', undefined, {
      duration: 1000,
    });
  }

  /**
   * Intercepts keyboard events to handle search toggle and shortcuts.
   * @param event The keyboard event fired on the window.
   */
  @HostListener('window:keydown', ['$event'])
  onKeyDown(event: KeyboardEvent) {
    if (this.activeSearchScope() !== SearchScope.Log) {
      return;
    }
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'f') {
      event.preventDefault();
      this.openSearch();
    } else if (event.key === 'Escape' && this.isSearchOpen()) {
      event.preventDefault();
      this.closeSearch();
    }
  }

  /**
   * Opens the search bar and focuses on the search input field.
   */
  openSearch() {
    this.isSearchOpen.set(true);
    setTimeout(() => {
      this.searchBar()?.focus();
    });
  }

  /**
   * Closes the search bar, clearing highlights and resetting state.
   */
  closeSearch() {
    this.isSearchOpen.set(false);
    this.searchQuery.set('');
    this.matchCount.set(0);
    this.currentMatchIndex.set(0);
    this.removeHighlights();
  }

  /**
   * Updates the search query and applies highlights to the log body.
   * @param query The search query string.
   */
  updateSearchQuery(query: string) {
    this.searchQuery.set(query);
    this.applyHighlights(query);
  }

  /**
   * Navigates to the next search match.
   */
  nextMatch() {
    const count = this.matchCount();
    if (count === 0) {
      return;
    }
    const next = (this.currentMatchIndex() % count) + 1;
    this.currentMatchIndex.set(next);
    this.updateActiveHighlight();
  }

  /**
   * Navigates to the previous search match.
   */
  prevMatch() {
    const count = this.matchCount();
    if (count === 0) {
      return;
    }
    const current = this.currentMatchIndex();
    const prev = current - 1 > 0 ? current - 1 : count;
    this.currentMatchIndex.set(prev);
    this.updateActiveHighlight();
  }

  /**
   * Removes all search highlights and restores original text nodes.
   */
  private removeHighlights() {
    const root = this.codeElement()?.nativeElement;
    if (!root) {
      return;
    }
    const marks = root.querySelectorAll('mark.search-highlight');
    marks.forEach((mark) => {
      const parent = mark.parentNode;
      if (parent) {
        while (mark.firstChild) {
          parent.insertBefore(mark.firstChild, mark);
        }
        parent.removeChild(mark);
      }
    });
    root.normalize();
  }

  /**
   * Applies search highlights to the log body matching the given query across text nodes.
   * @param query The search query string.
   */
  private applyHighlights(query: string) {
    this.removeHighlights();
    this.matchGroups = [];
    if (!query) {
      this.matchCount.set(0);
      this.currentMatchIndex.set(0);
      return;
    }

    const root = this.codeElement()?.nativeElement;
    if (!root) {
      return;
    }

    const { textNodes, fullText } = this.extractTextNodes(root);
    const matchRanges = this.findMatchRanges(fullText, query);
    const marksForMatch = this.applyHighlightsToNodes(textNodes, matchRanges);

    this.matchGroups = marksForMatch.map((marks) => ({ marks }));
    const count = this.matchGroups.length;
    this.matchCount.set(count);
    if (count > 0) {
      this.currentMatchIndex.set(1);
      this.updateActiveHighlight();
    } else {
      this.currentMatchIndex.set(0);
    }
  }

  /**
   * Extracts text nodes and their global offsets from the root element.
   * @param root The root element to scan.
   * @returns An object containing text nodes spans and cumulative full text.
   */
  private extractTextNodes(root: HTMLElement): {
    textNodes: TextNodeSpan[];
    fullText: string;
  } {
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, null);
    const textNodes: TextNodeSpan[] = [];
    let fullText = '';

    let node = walker.nextNode();
    while (node) {
      const text = node.nodeValue || '';
      const start = fullText.length;
      fullText += text;
      textNodes.push({ node: node as Text, start, end: fullText.length });
      node = walker.nextNode();
    }

    return { textNodes, fullText };
  }

  /**
   * Finds all matching index ranges within the scanned text for the given query.
   * @param text The full text string to search within.
   * @param query The search query string.
   * @returns An array of start and end match ranges.
   */
  private findMatchRanges(text: string, query: string): TextRange[] {
    const lowerQuery = query.toLowerCase();
    const queryLen = query.length;
    const matchRanges: TextRange[] = [];
    let matchPos = text.toLowerCase().indexOf(lowerQuery);

    while (matchPos !== -1) {
      matchRanges.push({ start: matchPos, end: matchPos + queryLen });
      matchPos = text.toLowerCase().indexOf(lowerQuery, matchPos + queryLen);
    }

    return matchRanges;
  }

  /**
   * Applies mark elements to matching text nodes and groups them by match index.
   * @param textNodes The scanned text node spans.
   * @param matchRanges The matched search query ranges.
   * @returns A grouped array of created mark elements per match range.
   */
  private applyHighlightsToNodes(
    textNodes: TextNodeSpan[],
    matchRanges: TextRange[],
  ): HTMLElement[][] {
    const marksForMatch: HTMLElement[][] = matchRanges.map(() => []);

    for (const item of textNodes) {
      const highlights: {
        localStart: number;
        localEnd: number;
        matchIdx: number;
      }[] = [];

      matchRanges.forEach((m, matchIdx) => {
        const oStart = Math.max(item.start, m.start);
        const oEnd = Math.min(item.end, m.end);
        if (oStart < oEnd) {
          highlights.push({
            localStart: oStart - item.start,
            localEnd: oEnd - item.start,
            matchIdx,
          });
        }
      });

      highlights.sort((a, b) => b.localStart - a.localStart);

      for (const h of highlights) {
        const middleNode = item.node.splitText(h.localStart);
        middleNode.splitText(h.localEnd - h.localStart);

        const mark = document.createElement('mark');
        mark.className = 'search-highlight';
        mark.textContent = middleNode.nodeValue;

        if (middleNode.parentNode) {
          middleNode.parentNode.replaceChild(mark, middleNode);
        }

        marksForMatch[h.matchIdx].unshift(mark);
      }
    }

    return marksForMatch;
  }

  /**
   * Updates the visual active state of the current match and scrolls it into view.
   */
  private updateActiveHighlight() {
    const index = this.currentMatchIndex() - 1;
    this.matchGroups.forEach((group, idx) => {
      const isActive = idx === index;
      group.marks.forEach((mark) => {
        if (isActive) {
          mark.classList.add('active');
        } else {
          mark.classList.remove('active');
        }
      });
      if (isActive && group.marks.length > 0) {
        group.marks[0].scrollIntoView({ behavior: 'smooth', block: 'center' });
      }
    });
  }

  /**
   * Sets the active search scope when mouse enters or focus enters.
   */
  @HostListener('mouseenter')
  @HostListener('focusin')
  public onScopeEnter(): void {
    this.scopeActiveChange.emit(true);
  }

  /**
   * Clears the active search scope when mouse leaves or focus leaves.
   */
  @HostListener('mouseleave')
  @HostListener('focusout')
  public onScopeLeave(): void {
    this.scopeActiveChange.emit(false);
  }
}
