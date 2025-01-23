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

/**
 * timeline-navigator.ts
 * Timeline navigator is an UI component shown at the right bottom of the timeline view. This component shows information related to the timeline currently selected.
 */

import { Type } from '@angular/core';
import { ResourceRevision } from 'src/app/store/revision';
import { TimelineEntry } from 'src/app/store/timeline';

/**
 * DisplayableTimelineNavigatorExtension is the set of parameters and the component to be shown as the extension.
 */
export interface DisplayableTimelineNavigatorExtension {
  /**
   * The Angular component type itself to show the data from extension.
   */
  component: Type<unknown>;

  /**
   * The @Input() parameters passed to the component.
   */
  inputs: Record<string, unknown>;
}

/**
 * TimelineNavigatorExtension defines a rule to show a custom component on the timeline navigator component regarding the content of currently selected timeline.
 */
export interface TimelineNavigatorExtension {
  /**
   * show returns true only when this extension should be visible on the timeline.
   */
  show(timeline: TimelineEntry): boolean;

  /**
   * getDisplayable constructs the set of parameters and the component shown as the extension.
   * @param timeline
   */
  getDisplayable(
    timeline: TimelineEntry,
  ): DisplayableTimelineNavigatorExtension;
}

export type RevisionManifestFieldFilterPredicate = (
  fieldValue: unknown,
  revision: ResourceRevision,
  timeline: TimelineEntry,
) => boolean;

/**
 * TimelineNavigatorExtensionUtil provides utilities for timeline navigator extensions.
 */
export class TimelineNavigatorExtensionUtil {
  /**
   * anyOfManifestBodyFieldInRevisions find if there is any of the field matching the given condition in revision bodies on timeline.
   * Note: this method is not supporting checking the field having array in the path between the field and its root.
   */
  public static anyOfManifestBodyFieldInRevisions(
    timeline: TimelineEntry,
    path: string[],
    predicate: RevisionManifestFieldFilterPredicate,
  ): boolean {
    for (const revision of timeline.revisions) {
      if (
        revision.parsedManifest === undefined ||
        revision.parsedManifest === null
      )
        continue;
      let currentManifestBody: Record<string, unknown> =
        revision.parsedManifest as unknown as Record<string, unknown>; // TODO: revision.parsedManifest must not be strictly typed and it should be just a map.
      for (let pathIndex = 0; pathIndex < path.length; pathIndex++) {
        const pathElement = path[pathIndex];
        if (
          currentManifestBody === undefined ||
          revision.parsedManifest === null
        )
          break;
        const next = currentManifestBody[pathElement];
        if (pathIndex === path.length - 1) {
          if (predicate(next, revision, timeline)) {
            return true;
          }
        } else {
          if (typeof currentManifestBody !== 'object') {
            console.warn(
              `non object type found in the path to read the path "${path.join('.')}"`,
            );
            break;
          }
          currentManifestBody = next as Record<string, unknown>;
        }
      }
    }
    return false;
  }

  /**
   * getSetOfManifestBodyFieldInRevisions returns the set of field at the specified path from all revisions on the timeline.
   * Note: this method is not supporting checking the field having array in the path between the field and its root.
   */
  public static getSetOfManifestBodyFieldInRevisions(
    timeline: TimelineEntry,
    path: string[],
  ): unknown[] {
    const result = new Set<unknown>();
    for (const revision of timeline.revisions) {
      if (
        revision.parsedManifest === undefined ||
        revision.parsedManifest === null
      )
        continue;
      let currentManifestBody: Record<string, unknown> =
        revision.parsedManifest as unknown as Record<string, unknown>; // TODO: revision.parsedManifest must not be strictly typed and it should be just a map.
      for (let pathIndex = 0; pathIndex < path.length; pathIndex++) {
        const pathElement = path[pathIndex];
        if (currentManifestBody === undefined || currentManifestBody === null)
          break;
        const next = currentManifestBody[pathElement];
        if (pathIndex === path.length - 1) {
          result.add(next);
        } else {
          if (typeof currentManifestBody !== 'object') {
            console.warn(
              `non object type found in the path to read the path "${path.join('.')}"`,
            );
            break;
          }
          currentManifestBody = next as Record<string, unknown>;
        }
      }
    }
    return [...result];
  }
}
