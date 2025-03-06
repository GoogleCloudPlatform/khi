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
  EnvironmentInjector,
  Inject,
  Input,
  inject,
} from '@angular/core';
import { InspectionDataStoreService } from '../services/inspection-data-store.service';
import {
  ReplaySubject,
  map,
  shareReplay,
  startWith,
  withLatestFrom,
} from 'rxjs';
import {
  LOG_ANNOTATOR_RESOLVER,
  LogAnnotatorResolver,
} from '../annotator/log/resolver';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'khi-log-header',
  templateUrl: './header.component.html',
  styleUrls: ['./header.component.sass'],
  imports: [CommonModule],
})
export class LogHeaderComponent {
  private readonly envInjector = inject(EnvironmentInjector);

  @Input()
  public set logIndex(index: number) {
    this.logIndexObservable.next(index);
  }

  private logIndexObservable = new ReplaySubject<number>(1);

  public logEntryObservable = this.logIndexObservable.pipe(
    startWith(0),
    withLatestFrom(this._inspectionDataStore.allLogs),
    map(([i, all]) => all[i]),
    shareReplay(1),
  );

  public logAnnotators = this.logAnnotatorResolver.getResolvedAnnotators(
    this.logEntryObservable,
    this.envInjector,
  );

  constructor(
    @Inject(LOG_ANNOTATOR_RESOLVER)
    private logAnnotatorResolver: LogAnnotatorResolver,
    private _inspectionDataStore: InspectionDataStoreService,
  ) {}
}
