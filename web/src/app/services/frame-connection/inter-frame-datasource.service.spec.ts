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

import { Observable, Subject, firstValueFrom } from 'rxjs';
import { InterframeDatasource } from './inter-frame-datasource.service';

describe('InterframeDatasource', () => {
  class TestingInterframeDataSource extends InterframeDatasource<string> {
    override enable(): void {
      return;
    }
    override disable(): void {
      return;
    }

    constructor(dataSource: Observable<string>) {
      super();
      dataSource.subscribe(this.rawUpdateRequest$);
    }
  }

  it('should emit the data when observer is registered', async () => {
    const dataSourceSubject = new Subject<string>();
    const datasource = new TestingInterframeDataSource(dataSourceSubject);

    dataSourceSubject.next('foo');
    dataSourceSubject.next('bar');

    expect(await firstValueFrom(datasource.data$)).toBe('bar');
  });

  it('should not emit the data when bound$ is false', async () => {
    const dataSourceSubject = new Subject<string>();
    const datasource = new TestingInterframeDataSource(dataSourceSubject);

    dataSourceSubject.next('foo');
    datasource.bound$.next(false);
    dataSourceSubject.next('bar');

    expect(await firstValueFrom(datasource.data$)).toBe('foo');
  });
});
