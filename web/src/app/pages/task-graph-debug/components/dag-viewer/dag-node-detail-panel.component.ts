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

import { Component, computed, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';
import {
  DagViewerNode,
  getTaskDescription,
  ProvidedTagItem,
} from 'src/app/pages/task-graph-debug/components/dag-viewer/dag-viewer.model';

/**
 * Key-value pair entry for labels display.
 */
export interface NodeLabelEntry {
  /** Label key name. */
  readonly key: string;
  /** Label value string. */
  readonly value: string;
}

/**
 * Slide-out detail side panel showing full metadata, labels, and dependencies of a selected node.
 */
@Component({
  selector: 'khi-dag-node-detail-panel',
  imports: [
    CommonModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    KHIIconRegistrationModule,
  ],
  templateUrl: './dag-node-detail-panel.component.html',
  styleUrls: ['./dag-node-detail-panel.component.scss'],
})
export class DagNodeDetailPanelComponent {
  /**
   * Currently selected task node, or null if no node is selected.
   */
  readonly node = input<DagViewerNode | null>(null);

  /**
   * List of upstream predecessor task nodes.
   */
  readonly upstreamNodes = input<readonly DagViewerNode[]>([]);

  /**
   * List of downstream consumer task nodes.
   */
  readonly downstreamNodes = input<readonly DagViewerNode[]>([]);

  /**
   * Emits when the user requests closing the detail panel.
   */
  readonly closePanel = output<void>();

  /**
   * Emits when the user clicks on a connected neighbor node to navigate to it.
   */
  readonly selectNode = output<string>();

  /**
   * List of provided tag identifiers extracted from providedTags.
   */
  readonly providedTags = computed<readonly ProvidedTagItem[]>(() => {
    const currentNode = this.node();
    if (!currentNode || !currentNode.providedTags) {
      return [];
    }
    return [...currentNode.providedTags]
      .map((pt) => ({
        tag: pt.tag,
        outputType: pt.outputType,
      }))
      .sort((a, b) => a.tag.localeCompare(b.tag));
  });

  /**
   * Array of label entries extracted from the node's labels dictionary.
   */
  readonly labelEntries = computed<readonly NodeLabelEntry[]>(() => {
    const currentNode = this.node();
    if (!currentNode || !currentNode.labels) {
      return [];
    }
    return Object.entries(currentNode.labels).map(([key, value]) => ({
      key,
      value,
    }));
  });

  /**
   * Human-readable description of the selected task node.
   */
  readonly taskDescription = computed<string>(() => {
    const currentNode = this.node();
    if (!currentNode) {
      return '';
    }
    return getTaskDescription(currentNode.labels);
  });
}
