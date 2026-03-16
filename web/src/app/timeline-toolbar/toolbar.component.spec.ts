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
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

import { ToolbarComponent } from './toolbar.component';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectorService,
} from '../services/frame-connection/window-connector.service';
import { InMemoryWindowConnectionProvider } from '../services/frame-connection/window-connection-provider.service';
import {
  DEFAULT_TIMELINE_FILTER,
  TimelineFilter,
} from '../services/timeline-filter.service';
import { InspectionDataStoreService } from '../services/inspection-data-store.service';
import { ViewStateService } from '../services/view-state.service';

describe('ToolbarComponent', () => {
  let component: ToolbarComponent;
  let fixture: ComponentFixture<ToolbarComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [NoopAnimationsModule],
      providers: [
        WindowConnectorService,
        {
          provide: WINDOW_CONNECTION_PROVIDER,
          useValue: new InMemoryWindowConnectionProvider(),
        },
        {
          provide: DEFAULT_TIMELINE_FILTER,
          useValue: new TimelineFilter(
            new InspectionDataStoreService(),
            new ViewStateService(),
          ),
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ToolbarComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
