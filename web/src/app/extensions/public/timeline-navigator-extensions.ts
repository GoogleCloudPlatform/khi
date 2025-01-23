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

import { TimelineEntry, TimelineLayer } from 'src/app/store/timeline';
import {
  DisplayableTimelineNavigatorExtension,
  TimelineNavigatorExtension,
  TimelineNavigatorExtensionUtil,
} from '../extension-common/extension-types/timeline-navigator';
import { CommonFieldAnnotatorComponent } from 'src/app/annotator/common-field-annotator.component';
import { of } from 'rxjs';
import {
  SUBRESOURCE_BINDING,
  TimelineFilterFacade,
} from 'src/app/store/timeline-filter';

/**
 * NodeNameBindingWithPod is a TimelineNavigatorExtension that shows node names where the pod scheduled on.
 */
export class NodeNameBindingWithPod implements TimelineNavigatorExtension {
  show(timeline: TimelineEntry): boolean {
    const bindingTimeline = timeline.children.find(
      (t) =>
        t.getNameOfLayer(TimelineLayer.Subresource) === SUBRESOURCE_BINDING,
    );
    if (!bindingTimeline) return false;
    return (
      timeline.layer === TimelineLayer.Name &&
      TimelineFilterFacade.isPodOrPodChildren(timeline)
    );
  }
  getDisplayable(
    timeline: TimelineEntry,
  ): DisplayableTimelineNavigatorExtension {
    const bindingTimeline = timeline.children.find(
      (t) =>
        t.getNameOfLayer(TimelineLayer.Subresource) === SUBRESOURCE_BINDING,
    )!; // binding subresource is bound to this timeline
    const nodeNames =
      TimelineNavigatorExtensionUtil.getSetOfManifestBodyFieldInRevisions(
        bindingTimeline,
        ['target', 'name'],
      );
    return {
      component: CommonFieldAnnotatorComponent,
      inputs: {
        label: 'Node',
        value: of(nodeNames.join(',')),
      },
    };
  }
}
