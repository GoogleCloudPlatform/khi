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

import { OverlayModule } from '@angular/cdk/overlay';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FormsModule } from '@angular/forms';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatToolbarModule } from '@angular/material/toolbar';
import { KHICommonModule } from '../common/common.module';
import { LogBodyComponent } from './body.component';

import { LogViewComponent } from './log-view.component';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectorService,
} from '../services/frame-connection/window-connector.service';
import { InMemoryWindowConnectionProvider } from '../services/frame-connection/window-connection-provider.service';
import { IconToggleButtonComponent } from './icon-toggle-button.component';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { LOG_TOOL_ANNOTATOR_RESOLVER } from '../annotator/log-tool/resolver';
import { getDefaultLogToolAnnotatorResolver } from '../annotator/log-tool/default';

describe('LogViewComponent', () => {
  let component: LogViewComponent;
  let fixture: ComponentFixture<LogViewComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [
        LogViewComponent,
        LogBodyComponent,
        IconToggleButtonComponent,
      ],
      imports: [
        OverlayModule,
        MatToolbarModule,
        MatSlideToggleModule,
        MatIconModule,
        MatTooltipModule,
        FormsModule,
        KHICommonModule,
      ],
      providers: [
        WindowConnectorService,
        {
          provide: WINDOW_CONNECTION_PROVIDER,
          useValue: new InMemoryWindowConnectionProvider(),
        },
        {
          provide: LOG_TOOL_ANNOTATOR_RESOLVER,
          useValue: getDefaultLogToolAnnotatorResolver(),
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(LogViewComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
