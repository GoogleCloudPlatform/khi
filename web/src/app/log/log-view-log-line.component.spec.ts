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
import { LogViewLogLineComponent } from './log-view-log-line.component';
import { KHICommonModule } from '../common/common.module';
import { LogType, Severity } from '../generated';
import { MatTooltipModule } from '@angular/material/tooltip';
import { LogEntry } from '../store/log';
import { ReferenceType } from '../common/loader/interface';

describe('LogViewLogLineComponent', () => {
  let component: LogViewLogLineComponent;
  let fixture: ComponentFixture<LogViewLogLineComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [LogViewLogLineComponent],
      imports: [KHICommonModule, MatTooltipModule],
      providers: [],
    }).compileComponents();
    fixture = TestBed.createComponent(LogViewLogLineComponent);
    component = fixture.componentInstance;
    component.log = new LogEntry(
      0,
      'foo',
      LogType.LogTypeAudit,
      Severity.SeverityInfo,
      0,
      'foo',
      { type: ReferenceType.NullReference },
      [],
    );
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
