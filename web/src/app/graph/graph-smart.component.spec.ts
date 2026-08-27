/**
 * Copyright 2026 Google LLC
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

import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { GraphSmartComponent } from 'src/app/graph/graph-smart.component';
import { GraphConverterService } from 'src/app/services/graph-converter.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import {
  DEFAULT_DELETION_THRESHOLD_SECONDS,
  GraphData,
  emptyGraphData,
} from 'src/app/common/schema/graph-schema';
import { Log } from 'src/app/store/domain/log';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

function flushAsync(ms = 50): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function setMockSelectedLog(
  signalRef: { set: (v: ReadonlyDomainElement<Log> | null) => void },
  timestamp: bigint,
): void {
  signalRef.set({ timestamp } as unknown as ReadonlyDomainElement<Log>);
}

describe('GraphSmartComponent', () => {
  let component: GraphSmartComponent;
  let fixture: ComponentFixture<GraphSmartComponent>;
  let mockGraphConverter: jasmine.SpyObj<GraphConverterService>;
  const mockSelectedLog = signal<ReadonlyDomainElement<Log> | null>(null);
  const mockTimelineView = signal<{
    filteredTimelineBitset: () => unknown;
  } | null>(null);

  beforeEach(async () => {
    mockSelectedLog.set(null);
    mockTimelineView.set(null);

    mockGraphConverter = jasmine.createSpyObj('GraphConverterService', [
      'getGraphDataAt',
    ]);
    mockGraphConverter.getGraphDataAt.and.resolveTo(emptyGraphData());

    await TestBed.configureTestingModule({
      imports: [GraphSmartComponent],
      providers: [
        { provide: GraphConverterService, useValue: mockGraphConverter },
        {
          provide: SelectionManager,
          useValue: { selectedLog: mockSelectedLog },
        },
        {
          provide: InspectionDataStore,
          useValue: { timelineView: mockTimelineView },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(GraphSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should have default deletionThresholdSeconds of DEFAULT_DELETION_THRESHOLD_SECONDS', () => {
    expect(component.deletionThresholdSeconds()).toBe(
      DEFAULT_DELETION_THRESHOLD_SECONDS,
    );
  });

  it('should update deletionThresholdSeconds', () => {
    component.deletionThresholdSeconds.set(300);
    expect(component.deletionThresholdSeconds()).toBe(300);
  });

  it('should return emptyGraphData when no log is selected', () => {
    expect(component.graphData()).toEqual(emptyGraphData());
    expect(mockGraphConverter.getGraphDataAt).not.toHaveBeenCalled();
  });

  it('should query graphConverter when a log is selected', async () => {
    const mockData: GraphData = {
      ...emptyGraphData(),
      graphTime: '2026-08-27 12:00:00',
    };
    mockGraphConverter.getGraphDataAt.and.resolveTo(mockData);

    setMockSelectedLog(mockSelectedLog, 1234567890n);
    fixture.detectChanges();
    await flushAsync();

    expect(mockGraphConverter.getGraphDataAt).toHaveBeenCalledWith(
      1234567890n,
      undefined,
      DEFAULT_DELETION_THRESHOLD_SECONDS,
      jasmine.any(AbortSignal),
    );
  });

  it('should re-query graphConverter when deletionThresholdSeconds changes', async () => {
    setMockSelectedLog(mockSelectedLog, 1234567890n);
    fixture.detectChanges();
    await flushAsync();
    mockGraphConverter.getGraphDataAt.calls.reset();

    component.deletionThresholdSeconds.set(300);
    fixture.detectChanges();
    await flushAsync();

    expect(mockGraphConverter.getGraphDataAt).toHaveBeenCalledWith(
      1234567890n,
      undefined,
      300,
      jasmine.any(AbortSignal),
    );
  });

  it('should reflect isLoading state while resource is resolving', async () => {
    // Await initial empty resource resolution
    await flushAsync();
    expect(component.isLoading()).toBeFalse();

    let resolvePromise!: (val: GraphData) => void;
    const pendingPromise = new Promise<GraphData>((resolve) => {
      resolvePromise = resolve;
    });
    mockGraphConverter.getGraphDataAt.and.returnValue(pendingPromise);

    setMockSelectedLog(mockSelectedLog, 1234567890n);
    fixture.detectChanges();

    // Resource starts loading
    expect(component.isLoading()).toBeTrue();

    resolvePromise(emptyGraphData());
    await flushAsync();

    expect(component.isLoading()).toBeFalse();
  });

  it('should fallback to emptyGraphData when getGraphDataAt rejects', async () => {
    mockGraphConverter.getGraphDataAt.and.rejectWith(
      new Error('Failed to load graph data'),
    );

    setMockSelectedLog(mockSelectedLog, 1234567890n);
    fixture.detectChanges();
    await flushAsync();

    expect(component.graphData()).toEqual(emptyGraphData());
    expect(component.isLoading()).toBeFalse();
  });
});
