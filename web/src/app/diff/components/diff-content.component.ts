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
  HostListener,
  inject,
  input,
  model,
  output,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { DiffToolbarComponent } from './diff-toolbar.component';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Clipboard } from '@angular/cdk/clipboard';
import { Revision } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { SearchBarComponent } from 'src/app/shared/components/search-bar/search-bar.component';
import { SearchScope } from 'src/app/services/view-state.service';
import { YamlViewerComponent } from 'src/app/shared/components/yaml-viewer/yaml-viewer.component';
import { ManagedFieldsAnnotationProvider } from 'src/app/shared/components/yaml-viewer/managed-fields-annotation.provider';
import { RevisionFieldAnnotationProvider } from 'src/app/shared/components/yaml-viewer/revision-field-annotation.provider';
import * as yaml from 'js-yaml';
import { isEventFromOverlay, isSearchShortcut } from 'src/app/common/dom-util';

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
    SearchBarComponent,
    YamlViewerComponent,
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
  readonly previousRevisionContent = input<string | null>(null);

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

  /**
   * Signal holding the total count of diff blocks.
   */
  public readonly diffCount = signal(0);

  /**
   * Signal holding the 1-based index of the active diff block.
   */
  public readonly currentDiffIndex = signal(0);

  /**
   * The timezone shift in hours from UTC.
   */
  readonly timezoneShift = input.required<number>();

  /**
   * Providers for YAML annotations like tooltips.
   */
  public readonly annotationProviders = computed(() => {
    // If showManagedFields is true, the YAML viewer's parsed yaml will contain metadata.managedFields,
    // so the provider can extract it naturally.
    if (this.showManagedFields()) {
      return [new ManagedFieldsAnnotationProvider(this.timezoneShift())];
    }

    // If showManagedFields is false, the managedFields are stripped from the displayed YAML.
    // To still show tooltips on other fields, we extract the managedFields from the original unstripped YAML
    // and provide them explicitly to the provider.
    const revision = this.currentRevision();
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    let overrideFields: any[] | undefined = undefined;

    if (revision && revision.bodyYAML) {
      try {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const yamlData = yaml.load(revision.bodyYAML) as any;
        if (
          yamlData &&
          yamlData['metadata'] &&
          yamlData['metadata']['managedFields']
        ) {
          overrideFields = yamlData['metadata']['managedFields'];
        }
      } catch (e) {
        console.warn('Failed to parse original YAML for managedFields:', e);
      }
    }

    return [
      new ManagedFieldsAnnotationProvider(this.timezoneShift(), overrideFields),
      ...(revision ? [new RevisionFieldAnnotationProvider(revision)] : []),
    ];
  });

  /**
   * Initializes the component.
   */
  constructor() {
    effect(() => {
      const count = this.diffCount();
      untracked(() => {
        const currentIdx = this.currentDiffIndex();
        if (count > 0) {
          if (currentIdx === 0 || currentIdx > count) {
            this.currentDiffIndex.set(1);
          }
        } else {
          this.currentDiffIndex.set(0);
        }
      });
    });

    effect(() => {
      // Watch currentRevision changes to reset the active diff index.
      this.currentRevision();
      untracked(() => {
        this.currentDiffIndex.set(this.diffCount() > 0 ? 1 : 0);
      });
    });
  }

  /**
   * Intercepts keyboard shortcuts to trigger in-diff search when active.
   * @param event The keyboard event.
   */
  @HostListener('window:keydown', ['$event'])
  onKeyDown(event: KeyboardEvent) {
    if (isEventFromOverlay(event)) {
      return;
    }
    if (this.activeSearchScope() !== SearchScope.Diff) {
      return;
    }
    if (isSearchShortcut(event)) {
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
  }

  /**
   * Navigates to the next diff block.
   */
  public nextDiff() {
    const count = this.diffCount();
    if (count === 0) {
      return;
    }
    const next = Math.min(this.currentDiffIndex() + 1, count);
    this.currentDiffIndex.set(next);
  }

  /**
   * Navigates to the previous diff block.
   */
  public prevDiff() {
    if (this.diffCount() === 0) {
      return;
    }
    const prev = Math.max(this.currentDiffIndex() - 1, 1);
    this.currentDiffIndex.set(prev);
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
