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

import { ComponentFixture, TestBed } from '@angular/core/testing';

import { DiffComponent } from './diff.component';
import { InMemoryWindowConnectionProvider } from 'src/app/services/frame-connection/window-connection-provider.service';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectorService,
} from 'src/app/services/frame-connection/window-connector.service';
import { DiffPageDataSource } from 'src/app/services/frame-connection/frames/diff-page-datasource.service';
import { TIMELINE_ANNOTATOR_RESOLVER } from 'src/app/annotator/timeline/resolver';
import { getDefaultTimelineAnnotatorResolver } from 'src/app/annotator/timeline/default';
import {
  CHANGE_PAIR_TOOL_ANNOTATOR_FOR_FLOATING_PAGE_RESOLVER,
  CHANGE_PAIR_TOOL_ANNOTATOR_RESOLVER,
} from 'src/app/annotator/change-pair-tool/resolver';
import {
  getDefaultChangePairToolAnnotatorResolver,
  getDefaultChangePairToolAnnotatorResolverForFloatingPage,
} from 'src/app/annotator/change-pair-tool/default';
import { CHANGE_PAIR_ANNOTATOR_RESOLVER } from 'src/app/annotator/change-pair/resolver';
import { getDefaultChangePairAnnotatorResolver } from 'src/app/annotator/change-pair/default';
import { BACKEND_API } from 'src/app/services/api/backend-api-interface';
import { GetConfigResponse } from 'src/app/common/schema/api-types';
import { of } from 'rxjs';

describe('DiffComponent', () => {
  let component: DiffComponent;
  let fixture: ComponentFixture<DiffComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      providers: [
        WindowConnectorService,
        {
          provide: WINDOW_CONNECTION_PROVIDER,
          useValue: new InMemoryWindowConnectionProvider(),
        },
        {
          provide: TIMELINE_ANNOTATOR_RESOLVER,
          useValue: getDefaultTimelineAnnotatorResolver(),
        },
        {
          provide: CHANGE_PAIR_TOOL_ANNOTATOR_RESOLVER,
          useValue: getDefaultChangePairToolAnnotatorResolver(),
        },
        {
          provide: CHANGE_PAIR_TOOL_ANNOTATOR_FOR_FLOATING_PAGE_RESOLVER,
          useValue: getDefaultChangePairToolAnnotatorResolverForFloatingPage(),
        },
        {
          provide: CHANGE_PAIR_ANNOTATOR_RESOLVER,
          useValue: getDefaultChangePairAnnotatorResolver(),
        },
        {
          provide: BACKEND_API,
          useValue: {
            getConfig: () => {
              return of<GetConfigResponse>({
                viewerMode: false,
              });
            },
          },
        },
        DiffPageDataSource,
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(DiffComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
