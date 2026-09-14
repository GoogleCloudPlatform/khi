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
  ChangeDetectionStrategy,
  Component,
  computed,
  input,
  output,
} from '@angular/core';
import {
  DagNodeRunPhase,
  DagPositionedNode,
  getTaskDescription,
  ProvidedTagItem,
} from 'src/app/shared/components/dag-viewer/dag-viewer.model';
import { formatDurationMs } from 'src/app/utils/time-format-util';

const CHAR_WIDTH_ESTIMATE_PX = 7.5;
const NODE_HORIZONTAL_PADDING_PX = 28;
const DURATION_CHAR_WIDTH_PX = 6.5;
const DURATION_GAP_PX = 16;
const FEATURE_BADGE_OFFSET_X = 62;
const TAG_CHAR_WIDTH_PX = 6.2;
const TAG_PADDING_PX = 14;
const TAG_GAP_PX = 6;

/**
 * Visual badge representation for a provided tag on a task node card.
 */
export interface TagBadge {
  /** Descriptive tooltip text for the badge. */
  readonly tooltip: string;
  /** Formatted display text prefixed with hash. */
  readonly displayText: string;
  /** Width of the badge card in pixels. */
  readonly width: number;
  /** SVG transform string for positioning. */
  readonly transform: string;
}

/**
 * Renders an individual task node card inside the SVG DAG canvas.
 */
@Component({
  // eslint-disable-next-line @angular-eslint/component-selector
  selector: 'g[khi-dag-node]',
  templateUrl: './dag-node.component.html',
  styleUrls: ['./dag-node.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    class: 'dag-node-group',
    '[class.selected]': 'isSelected()',
    '[class.highlighted]': 'isHighlighted()',
    '[class.dimmed]': 'isDimmed()',
    '[class.feature]': 'node().isFeature',
    '[class.form-task]': 'node().isFormTask',
    '[attr.data-run-phase]': 'runPhaseAttribute()',
    '[attr.transform]': 'transform()',
    '(click)': 'onClick($event)',
  },
})
export class DagNodeComponent {
  /**
   * Positioned node containing task metadata and coordinates.
   */
  readonly node = input.required<DagPositionedNode>();

  /**
   * Whether this node is currently selected by the user.
   */
  readonly isSelected = input<boolean>(false);

  /**
   * Whether this node is part of the highlighted upstream/downstream dependency path.
   */
  readonly isHighlighted = input<boolean>(false);

  /**
   * Whether this node is dimmed because another node is active.
   */
  readonly isDimmed = input<boolean>(false);

  /**
   * Emits when the user clicks on this node.
   */
  readonly selectNode = output<DagPositionedNode>();

  /**
   * SVG transform string positioning the node group at its top-left coordinates.
   */
  readonly transform = computed(
    () => `translate(${this.node().x}, ${this.node().y})`,
  );

  /**
   * Run phase exposed as a host attribute, or null when the node is not tied to a run.
   */
  readonly runPhaseAttribute = computed(() => {
    const runPhase = this.node().runPhase;
    return runPhase === DagNodeRunPhase.NONE ? null : runPhase;
  });

  /**
   * Formatted run duration label, or empty string when the duration is unknown.
   */
  readonly runDurationLabel = computed(() => {
    const runDurationMs = this.node().runDurationMs;
    return runDurationMs > 0 ? formatDurationMs(runDurationMs) : '';
  });

  /**
   * Domain prefix extracted from the reference ID, such as "khi.google.com/".
   * Returns empty string if the reference ID does not contain a slash.
   */
  readonly displayDomain = computed(() => {
    const ref = this.node().referenceId;
    const slashIndex = ref.indexOf('/');
    if (slashIndex === -1) {
      return '';
    }
    return ref.slice(0, slashIndex + 1);
  });

  /**
   * Main task identifier after the domain prefix.
   */
  readonly displayTaskName = computed(() => {
    const ref = this.node().referenceId;
    const slashIndex = ref.indexOf('/');
    const name = slashIndex === -1 ? ref : ref.slice(slashIndex + 1);
    const maxChars = Math.max(
      10,
      Math.floor(
        (this.node().width - NODE_HORIZONTAL_PADDING_PX) /
          CHAR_WIDTH_ESTIMATE_PX,
      ),
    );
    if (name.length > maxChars) {
      return `${name.slice(0, maxChars - 3)}...`;
    }
    return name;
  });

  /**
   * Vertical coordinate (Y) for the task name SVG text element.
   */
  readonly taskTextY = computed(() => (this.displayDomain() ? 35 : 26));

  /**
   * Human-readable description extracted from task labels.
   */
  readonly taskDescription = computed(() =>
    getTaskDescription(this.node().labels),
  );

  /**
   * Short display string for output type, truncated to avoid overlapping the run duration label.
   */
  readonly displayOutputType = computed(() => {
    const outputType = this.node().outputType;
    if (!outputType) {
      return '';
    }
    const durationLabel = this.runDurationLabel();
    const durationReservePx =
      durationLabel.length > 0
        ? durationLabel.length * DURATION_CHAR_WIDTH_PX + DURATION_GAP_PX
        : 0;
    const availableWidth =
      this.node().width - NODE_HORIZONTAL_PADDING_PX - durationReservePx;
    const maxChars = Math.max(
      10,
      Math.floor(availableWidth / CHAR_WIDTH_ESTIMATE_PX),
    );
    if (outputType.length > maxChars) {
      return `${outputType.slice(0, maxChars - 3)}...`;
    }
    return outputType;
  });

  /**
   * Title text for the SVG title element tooltip.
   */
  readonly nodeTitle = computed(() => {
    const desc = this.taskDescription();
    const ref = this.node().referenceId;
    const outputType = this.node().outputType;
    const base = desc ? `${ref} - ${desc}` : ref;
    return outputType ? `${base}\nOutput: ${outputType}` : base;
  });

  /**
   * Implementation hash identifier for the secondary label.
   */
  readonly displayImplementationId = computed(() => {
    const id = this.node().id;
    const parts = id.split('#');
    if (parts.length > 1) {
      return `#${parts[1]}`;
    }
    return id;
  });

  /**
   * SVG transform for the Input badge.
   */
  readonly inputBadgeTransform = computed(() => {
    const offsetX = this.node().isFeature ? FEATURE_BADGE_OFFSET_X : 0;
    return `translate(${offsetX}, 0)`;
  });

  /**
   * Provided tags attached to this task, converted into visual badge models.
   */
  readonly tagBadges = computed<readonly TagBadge[]>(() => {
    const providedTags = this.node().providedTags ?? [];
    const sortedProvidedTags: readonly ProvidedTagItem[] = [...providedTags]
      .map((pt) => ({
        tag: pt.tag,
        outputType: pt.outputType,
      }))
      .sort((a, b) => a.tag.localeCompare(b.tag));

    const maxTagsRowWidth = this.node().width - NODE_HORIZONTAL_PADDING_PX;
    const badges: TagBadge[] = [];
    let currentX = 0;
    for (let i = 0; i < sortedProvidedTags.length; i++) {
      const item = sortedProvidedTags[i];
      const text = `#${item.tag}`;
      const tooltip = item.outputType
        ? `${item.tag} (${item.outputType})`
        : item.tag;
      const badgeWidth = Math.ceil(
        text.length * TAG_CHAR_WIDTH_PX + TAG_PADDING_PX,
      );
      if (currentX + badgeWidth > maxTagsRowWidth && badges.length > 0) {
        const remaining = sortedProvidedTags.length - i;
        const moreText = `+${remaining}`;
        const moreWidth = Math.ceil(
          moreText.length * TAG_CHAR_WIDTH_PX + TAG_PADDING_PX,
        );
        if (currentX + moreWidth <= maxTagsRowWidth) {
          badges.push({
            tooltip: `${remaining} more tag(s)`,
            displayText: moreText,
            width: moreWidth,
            transform: `translate(${currentX}, 0)`,
          });
        }
        break;
      }
      badges.push({
        tooltip,
        displayText: text,
        width: badgeWidth,
        transform: `translate(${currentX}, 0)`,
      });
      currentX += badgeWidth + TAG_GAP_PX;
    }
    return badges;
  });

  /**
   * Handles click events on the node group.
   */
  onClick(event: MouseEvent): void {
    event.stopPropagation();
    this.selectNode.emit(this.node());
  }
}
