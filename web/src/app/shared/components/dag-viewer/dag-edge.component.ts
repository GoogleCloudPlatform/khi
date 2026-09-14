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

import { Component, computed, input } from '@angular/core';
import { TaskDependencyCardinality } from 'src/app/generated/api/v1/inspection_task_graph_pb';
import { DagPositionedEdge } from 'src/app/shared/components/dag-viewer/dag-viewer.model';

/**
 * Renders an individual directed dependency edge with arrow head and optional tag chip in SVG.
 */
@Component({
  // eslint-disable-next-line @angular-eslint/component-selector
  selector: 'g[khi-dag-edge]',
  templateUrl: './dag-edge.component.html',
  styleUrls: ['./dag-edge.component.scss'],
  host: {
    class: 'dag-edge-group',
    '[class.highlighted]': 'isHighlighted()',
    '[class.dimmed]': 'isDimmed()',
    '[class.fan-in]': 'isFanIn()',
  },
})
export class DagEdgeComponent {
  /**
   * Export TaskDependencyCardinality enum to the template.
   */
  protected readonly TaskDependencyCardinality = TaskDependencyCardinality;

  /**
   * Positioned edge with layout coordinates and Bezier path data.
   */
  readonly edge = input.required<DagPositionedEdge>();

  /**
   * Whether this edge is part of the active dependency path highlight.
   */
  readonly isHighlighted = input<boolean>(false);

  /**
   * Whether this edge is dimmed because another node is selected.
   */
  readonly isDimmed = input<boolean>(false);

  /**
   * Whether this edge represents a fan-in tag aggregation.
   */
  readonly isFanIn = computed(
    () => this.edge().cardinality === TaskDependencyCardinality.FAN_IN,
  );

  /**
   * URL reference pointing to the appropriate SVG marker def ID.
   */
  readonly markerUrl = computed(() => {
    if (this.isHighlighted()) {
      return 'url(#arrow-marker-highlighted)';
    }
    if (this.isFanIn()) {
      return 'url(#arrow-marker-fan-in)';
    }
    return 'url(#arrow-marker)';
  });

  /**
   * SVG transform string positioning the tag chip at the center of the curve.
   */
  readonly tagTransform = computed(
    () => `translate(${this.edge().labelX}, ${this.edge().labelY})`,
  );

  /**
   * Domain prefix extracted from the tag, including trailing slash, such as "khi.google.com/".
   * Returns empty string if the tag does not contain a slash.
   */
  readonly tagDomain = computed(() => {
    const tag = this.edge().tag;
    if (!tag) {
      return '';
    }
    const slashIdx = tag.indexOf('/');
    if (slashIdx === -1) {
      return '';
    }
    return tag.slice(0, slashIdx + 1);
  });

  /**
   * Main tag identifier after the domain prefix, prefixed with a hashtag.
   */
  readonly tagName = computed(() => {
    const tag = this.edge().tag;
    if (!tag) {
      return '';
    }
    const slashIdx = tag.indexOf('/');
    const name = slashIdx === -1 ? tag : tag.slice(slashIdx + 1);
    return `#${name}`;
  });

  /**
   * Computed height of the tag badge background.
   */
  readonly tagBadgeHeight = computed(() => (this.tagDomain() ? 34 : 22));

  /**
   * Corner radius (rx) for the tag badge background rectangle.
   */
  readonly tagBadgeRx = computed(() => (this.tagDomain() ? 6 : 11));

  /**
   * Y offset for the centered tag badge background rectangle.
   */
  readonly tagBadgeY = computed(() => -this.tagBadgeHeight() / 2);

  /**
   * Computed width of the tag badge background based on text length.
   */
  readonly tagBadgeWidth = computed(() => {
    if (!this.edge().tag) {
      return 0;
    }
    if (this.tagDomain()) {
      const domainWidth = this.tagDomain().length * 5.8;
      const nameWidth = this.tagName().length * 7.2;
      return Math.max(48, Math.ceil(Math.max(domainWidth, nameWidth) + 20));
    }
    return Math.max(48, Math.ceil(this.tagName().length * 7.2 + 20));
  });

  /**
   * X offset for the centered tag badge background rectangle.
   */
  readonly tagBadgeX = computed(() => -this.tagBadgeWidth() / 2);
}
