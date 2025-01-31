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

import { DiffViewComponent } from './diff-view.component';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectorService,
} from '../services/frame-connection/window-connector.service';
import { InMemoryWindowConnectionProvider } from '../services/frame-connection/window-connection-provider.service';
import { TIMELINE_ANNOTATOR_RESOLVER } from '../annotator/timeline/resolver';
import { getDefaultTimelineAnnotatorResolver } from '../annotator/timeline/default';
import { CHANGE_PAIR_TOOL_ANNOTATOR_RESOLVER } from '../annotator/change-pair-tool/resolver';
import { getDefaultChangePairToolAnnotatorResolver } from '../annotator/change-pair-tool/default';
import { CHANGE_PAIR_ANNOTATOR_RESOLVER } from '../annotator/change-pair/resolver';
import { getDefaultChangePairAnnotatorResolver } from '../annotator/change-pair/default';

describe('DiffViewComponent', () => {
  let component: DiffViewComponent;
  let fixture: ComponentFixture<DiffViewComponent>;

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
          provide: CHANGE_PAIR_ANNOTATOR_RESOLVER,
          useValue: getDefaultChangePairAnnotatorResolver(),
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(DiffViewComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
