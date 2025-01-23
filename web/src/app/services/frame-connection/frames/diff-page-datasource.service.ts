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

import { Injectable } from '@angular/core';
import { InterframeDatasource } from '../inter-frame-datasource.service';
import { distinctUntilChanged, map, Subject } from 'rxjs';
import { WindowConnectorService } from '../window-connector.service';
import { Router } from '@angular/router';
import {
  DiffPageViewModel,
  UPDATE_SELECTED_RESOURCE_MESSAGE_KEY,
  UpdateSelectedResourceMessage,
} from 'src/app/common/schema/inter-window-messages';
import { TimelineEntry, TimelineLayer } from 'src/app/store/timeline';

@Injectable()
export class DiffPageDataSource extends InterframeDatasource<DiffPageViewModel> {
  private navigationCandidate: Subject<string> = new Subject();

  private enabled = false;

  constructor(
    private connector: WindowConnectorService,
    private router: Router,
  ) {
    super();

    this.navigationCandidate
      .pipe(distinctUntilChanged())
      .subscribe((sessionPath) => {
        const urlParts = this.router.url.split('/');
        this.router.navigateByUrl(`/session/${urlParts[2]}/${sessionPath}`);
      });
  }

  override enable(): void {
    if (this.enabled) {
      return;
    }
    this.enabled = true;
    this.connector
      .receiver<UpdateSelectedResourceMessage>(
        UPDATE_SELECTED_RESOURCE_MESSAGE_KEY,
      )
      .pipe(
        map((message) => ({
          timeline: TimelineEntry.clone(message.data.timeline),
          logIndex: message.data.logIndex,
        })),
      )
      .subscribe(this.rawUpdateRequest$);
    this.data$.subscribe((data) => this.updatePath(data));
    this.connector.broadcast('DIFF_PAGE_OPEN', {});
  }

  override disable(): void {
    return;
  }

  private updatePath(data: DiffPageViewModel) {
    const kind = data.timeline.getNameOfLayer(TimelineLayer.Kind) ?? '-';
    const namespace =
      data.timeline.getNameOfLayer(TimelineLayer.Namespace) ?? '-';
    const name = data.timeline.getNameOfLayer(TimelineLayer.Name) ?? '-';
    const subresource =
      data.timeline.getNameOfLayer(TimelineLayer.Subresource) ?? '-';
    const logIndex = data.logIndex;

    this.navigationCandidate.next(
      `diff/${kind}/${namespace}/${name}/${subresource}?logIndex=${logIndex}`,
    );
  }
}
