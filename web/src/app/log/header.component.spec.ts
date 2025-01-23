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

import { LogHeaderComponent } from './header.component';
import { WINDOW_CONNECTION_PROVIDER } from '../services/frame-connection/window-connector.service';
import { InMemoryWindowConnectionProvider } from '../services/frame-connection/window-connection-provider.service';
import { LOG_ANNOTATOR_RESOLVER } from '../annotator/log/resolver';
import { getDefaultLogAnnotatorResolver } from '../annotator/log/default';

describe('LogHeaderComponent', () => {
  let component: LogHeaderComponent;
  let fixture: ComponentFixture<LogHeaderComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [LogHeaderComponent],
      imports: [],
      providers: [
        {
          provide: WINDOW_CONNECTION_PROVIDER,
          useValue: new InMemoryWindowConnectionProvider(),
        },
        {
          provide: LOG_ANNOTATOR_RESOLVER,
          useValue: getDefaultLogAnnotatorResolver(),
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(LogHeaderComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
