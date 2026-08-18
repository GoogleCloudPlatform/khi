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

import { signal, WritableSignal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { LogSmartComponent } from './log-smart.component';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectorService,
} from '../services/frame-connection/window-connector.service';
import { InMemoryWindowConnectionProvider } from '../services/frame-connection/window-connection-provider.service';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { Log } from 'src/app/store/domain/log';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

describe('LogSmartComponent', () => {
  let component: LogSmartComponent;
  let fixture: ComponentFixture<LogSmartComponent>;
  let workbenchClientSpy: jasmine.SpyObj<WorkbenchClientService>;
  let mockSelectedLog: WritableSignal<ReadonlyDomainElement<Log> | null>;
  let mockSelectedTimeline: WritableSignal<ReadonlyDomainElement<Timeline> | null>;

  beforeEach(async () => {
    workbenchClientSpy = jasmine.createSpyObj('WorkbenchClientService', [
      'readStructYAML',
    ]);
    mockSelectedLog = signal<ReadonlyDomainElement<Log> | null>(null);
    mockSelectedTimeline = signal<ReadonlyDomainElement<Timeline> | null>(null);

    await TestBed.configureTestingModule({
      providers: [
        WindowConnectorService,
        {
          provide: WINDOW_CONNECTION_PROVIDER,
          useValue: new InMemoryWindowConnectionProvider(),
        },
        {
          provide: WorkbenchClientService,
          useValue: workbenchClientSpy,
        },
        {
          provide: SelectionManager,
          useValue: {
            selectedLog: mockSelectedLog,
            selectedLogIndex: signal(-1),
            selectedTimeline: mockSelectedTimeline,
            selectedTimelinesWithChildren: signal([]),
            highlightLogIndices: signal(new Set<number>()),
            timelineSelectionShouldIncludeChildren: signal(true),
            onSelectLog: (log: ReadonlyDomainElement<Log> | null) =>
              mockSelectedLog.set(log),
            onSelectTimeline: (
              timeline: ReadonlyDomainElement<Timeline> | null,
            ) => mockSelectedTimeline.set(timeline),
            onHighlightLog: jasmine.createSpy('onHighlightLog'),
            onHighlightTimeline: jasmine.createSpy('onHighlightTimeline'),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(LogSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should return null logContentViewModel when no log is selected', () => {
    expect(component.logContentViewModel()).toBeNull();
    expect(component.isLoading()).toBeFalse();
  });

  it('should fetch and display YAML when a log with structId is selected', async () => {
    workbenchClientSpy.readStructYAML.and.returnValue(
      Promise.resolve('message: hello from backend\n'),
    );

    const mockLog = {
      id: 1,
      structId: 42,
      body: { message: 'hello from backend' },
    } as unknown as ReadonlyDomainElement<Log>;

    mockSelectedLog.set(mockLog);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(workbenchClientSpy.readStructYAML).toHaveBeenCalledWith(42);
    expect(component.logContentViewModel()?.logBody).toBe(
      'message: hello from backend\n',
    );
    expect(component.isLoading()).toBeFalse();
  });

  it('should handle readStructYAML rejection gracefully', async () => {
    workbenchClientSpy.readStructYAML.and.returnValue(
      Promise.reject(new Error('RPC failure')),
    );

    const mockLog = {
      id: 2,
      structId: 99,
      body: null,
    } as unknown as ReadonlyDomainElement<Log>;

    mockSelectedLog.set(mockLog);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(workbenchClientSpy.readStructYAML).toHaveBeenCalledWith(99);
    expect(component.logContentViewModel()?.logBody).toBe('');
    expect(component.isLoading()).toBeFalse();
  });
});
