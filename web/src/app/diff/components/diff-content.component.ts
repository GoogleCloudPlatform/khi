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
  effect,
  ElementRef,
  HostListener,
  inject,
  input,
  model,
  output,
  signal,
  viewChild,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { DiffToolbarComponent } from './diff-toolbar.component';
import { UnifiedDiffComponent } from 'ngx-diff';
import { HighlightModule } from 'ngx-highlightjs';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Clipboard } from '@angular/cdk/clipboard';
import { Revision } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { SearchBarComponent } from 'src/app/shared/components/search-bar/search-bar.component';
import { SearchScope } from 'src/app/services/view-state.service';

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
 * Component for displaying the unified diff of a resource revision.
 */
@Component({
  selector: 'khi-diff-content',
  templateUrl: './diff-content.component.html',
  styleUrls: ['./diff-content.component.scss'],
  imports: [
    CommonModule,
    DiffToolbarComponent,
    UnifiedDiffComponent,
    HighlightModule,
    SearchBarComponent,
  ],
})
export class DiffContentComponent {
  private readonly clipboard = inject(Clipboard);
  private readonly snackBar = inject(MatSnackBar);

  /**
   * The current revision being viewed.
   */
  readonly currentRevision =
    input.required<ReadonlyDomainElement<Revision> | null>();

  /**
   * The content string of the current revision.
   */
  readonly currentRevisionContent = input.required<string>();

  /**
   * The content string of the previous revision to diff against.
   */
  readonly previousRevisionContent = input.required<string>();

  /**
   * Two-way bound state for showing managed fields in the diff.
   */
  readonly showManagedFields = model.required<boolean>();

  /**
   * Emitted when requesting to open the diff in a new window/tab.
   */
  readonly openInNewTab = output<void>();

  /**
   * Emitted when the mouse hovers over or focus enters/leaves the diff content area.
   */
  readonly scopeActiveChange = output<boolean>();

  /**
   * Reference to the diff inner container element to search within.
   */
  public readonly diffContainer =
    viewChild<ElementRef<HTMLElement>>('diffContainer');

  /**
   * Reference to the search bar component for focus management.
   */
  public readonly searchBar = viewChild(SearchBarComponent);

  /**
   * Signal indicating whether the search bar is currently open.
   */
  public readonly isSearchOpen = signal(false);

  /**
   * Signal holding the current search query string.
   */
  public readonly searchQuery = signal('');

  /**
   * Signal holding the total count of matches found.
   */
  public readonly matchCount = signal(0);

  /**
   * Signal holding the 1-based index of the active match.
   */
  public readonly currentMatchIndex = signal(0);

  /** Holds the current active search scope. */
  public readonly activeSearchScope = input<SearchScope>(SearchScope.Global);

  private matchGroups: MatchGroup[] = [];

  /**
   * Initializes the component and registers an effect to sync search highlights.
   */
  constructor() {
    effect(() => {
      const query = this.searchQuery();
      this.currentRevisionContent();
      this.previousRevisionContent();
      const isOpen = this.isSearchOpen();
      setTimeout(() => {
        // applyHighlights read DOM contents that can be updated by the signals above.
        // applyHighlights can update a signal and effects shouldn't update signals.
        // This setTimeout is a workaround to avoid this issue.
        if (isOpen && query) {
          this.applyHighlights(query);
        } else {
          this.removeHighlights();
        }
      });
    });
  }

  /**
   * Intercepts keyboard shortcuts to trigger in-diff search when active.
   * @param event The keyboard event.
   */
  @HostListener('window:keydown', ['$event'])
  onKeyDown(event: KeyboardEvent) {
    if (this.activeSearchScope() !== SearchScope.Diff) {
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
   * Opens the search bar and focuses the input field.
   */
  public openSearch() {
    this.isSearchOpen.set(true);
    setTimeout(() => {
      this.searchBar()?.focus();
    });
  }

  /**
   * Closes the search bar and clears the query string.
   */
  public closeSearch() {
    this.isSearchOpen.set(false);
    this.searchQuery.set('');
  }

  /**
   * Updates the search query string.
   * @param query The new query string.
   */
  public updateSearchQuery(query: string) {
    this.searchQuery.set(query);
  }

  /**
   * Navigates to the next search match.
   */
  public nextMatch() {
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
  public prevMatch() {
    const count = this.matchCount();
    if (count === 0) {
      return;
    }
    let prev = this.currentMatchIndex() - 1;
    if (prev <= 0) {
      prev = count;
    }
    this.currentMatchIndex.set(prev);
    this.updateActiveHighlight();
  }

  /**
   * Triggers the openInNewTab output event.
   */
  protected _openInNewTab() {
    this.openInNewTab.emit();
  }

  /**
   * Copies the current revision's content to the clipboard and displays a snackbar notification.
   */
  protected copyContent() {
    const content = this.currentRevisionContent();
    let snackbarMessage = 'Copy failed';
    if (this.clipboard.copy(content)) {
      snackbarMessage = 'Copied!';
    }
    this.snackBar.open(snackbarMessage, undefined, { duration: 1000 });
  }

  /**
   * Removes all search highlights and restores original text nodes.
   */
  private removeHighlights() {
    const root = this.diffContainer()?.nativeElement;
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
   * Applies search highlights to the diff body matching the given query across text nodes.
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

    const root = this.diffContainer()?.nativeElement;
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
