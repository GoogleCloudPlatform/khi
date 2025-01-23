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

import { Component, EnvironmentInjector, inject } from '@angular/core';
import { Observable, map } from 'rxjs';
import { CHANGE_PAIR_TOOL_ANNOTATOR_FOR_FLOATING_PAGE_RESOLVER } from 'src/app/annotator/change-pair-tool/resolver';
import { CHANGE_PAIR_ANNOTATOR_RESOLVER } from 'src/app/annotator/change-pair/resolver';
import { TIMELINE_ANNOTATOR_RESOLVER } from 'src/app/annotator/timeline/resolver';
import { DiffPageDataSource } from 'src/app/services/frame-connection/frames/diff-page-datasource.service';
import { TimelineEntry } from 'src/app/store/timeline';

@Component({
  selector: 'khi-diff-page',
  templateUrl: './diff.component.html',
  styleUrls: ['./diff.component.sass'],
})
export class DiffComponent {
  private readonly envInjector = inject(EnvironmentInjector);

  private readonly timelineAnnotatorResolver = inject(
    TIMELINE_ANNOTATOR_RESOLVER,
  );

  private readonly changePairToolAnnotatorResolver = inject(
    CHANGE_PAIR_TOOL_ANNOTATOR_FOR_FLOATING_PAGE_RESOLVER,
  );

  private readonly changePairAnnotatorResolver = inject(
    CHANGE_PAIR_ANNOTATOR_RESOLVER,
  );

  timeline: Observable<TimelineEntry> = this.diffPageSource.data$.pipe(
    map((data) => data.timeline),
  );

  changePair = this.diffPageSource.data$.pipe(
    map((data) => data.timeline.getRevisionPairByLogId(data.logIndex)),
  );
  timelineAnnotators = this.timelineAnnotatorResolver.getResolvedAnnotators(
    this.timeline,
    this.envInjector,
  );

  changePairToolAnnotators =
    this.changePairToolAnnotatorResolver.getResolvedAnnotators(
      this.changePair,
      this.envInjector,
    );

  changePairAnnotators = this.changePairAnnotatorResolver.getResolvedAnnotators(
    this.changePair,
    this.envInjector,
  );

  constructor(private readonly diffPageSource: DiffPageDataSource) {}
}
