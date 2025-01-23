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
  distinctUntilKeyChanged,
  exhaustMap,
  interval,
  map,
  Observable,
  retry,
  shareReplay,
  throwError,
} from 'rxjs';
import {
  PopupClient,
  PopupFormRequestWithClient,
  PopupManager,
} from './popup-manager';
import {
  PopupAnswerResponse,
  PopupAnswerValidationResult,
  PopupFormRequest,
} from 'src/app/common/schema/api-types';
import { BACKEND_API, BackendAPI } from '../api/backend-api-interface';
import { Inject, Injectable } from '@angular/core';

export const NilPopupFormRequest: PopupFormRequest = {
  id: 'none',
  title: '',
  type: 'text',
  description: '',
  placeholder: '',
  options: {},
};

@Injectable({ providedIn: 'any' })
export class PopupManagerImpl implements PopupManager {
  private popupRequest = interval(1000).pipe(
    exhaustMap(
      () => this.backendAPI.getPopup() as Observable<PopupFormRequest>,
    ),
    map((req) => req ?? NilPopupFormRequest),
    distinctUntilKeyChanged('id'),
    retry(),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    }),
  );

  constructor(@Inject(BACKEND_API) private backendAPI: BackendAPI) {}

  requests(): Observable<PopupFormRequestWithClient> {
    return this.popupRequest.pipe(
      map((request) => ({
        ...request,
        client: new PopupClientImpl(request.id, this.backendAPI),
      })),
    );
  }
}

export class PopupClientImpl implements PopupClient {
  constructor(
    public readonly popupId: string,
    private backendAPI: BackendAPI,
  ) {}
  validate(data: PopupAnswerResponse): Observable<PopupAnswerValidationResult> {
    if (data.id === this.popupId) {
      return this.backendAPI.validatePopupAnswer(data);
    } else {
      return throwError(() => {
        return 'the popup id is not for this client';
      });
    }
  }
  answer(data: PopupAnswerResponse): Observable<void> {
    if (data.id === this.popupId) {
      return this.backendAPI.answerPopup(data);
    } else {
      return throwError(() => {
        return 'the popup id is not for this client';
      });
    }
  }
}
