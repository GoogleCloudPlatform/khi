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

import { Pipe, PipeTransform } from '@angular/core';
import { InspectionDataStoreService } from '../services/inspection-data-store.service';
import { Observable, switchMap } from 'rxjs';
import { TextReference } from './loader/interface';

/**
 * A pipe to resolve KHIFileTextReference type with data store.
 * Large texts shouldn't be kept in view models, resolve text from buffer source in data store.
 */
@Pipe({
  name: 'resolveText',
})
export class ResolveTextPipe implements PipeTransform {
  constructor(private dataStore: InspectionDataStoreService) {}
  transform(value: TextReference): Observable<string> {
    return this.dataStore.referenceResolver.pipe(
      switchMap((bs) => bs?.getText(value) ?? 'error'),
    );
  }
}
