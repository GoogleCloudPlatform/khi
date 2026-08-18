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

import { DiffSmartComponent } from './diff-smart.component';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectorService,
} from '../services/frame-connection/window-connector.service';
import { InMemoryWindowConnectionProvider } from '../services/frame-connection/window-connection-provider.service';
import { WorkbenchClientService } from 'src/app/services/api/workbench/workbench-client.service';
import { SelectionManager } from 'src/app/services/selection-manager.service';
import { Revision, Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';

describe('DiffSmartComponent', () => {
  let component: DiffSmartComponent;
  let fixture: ComponentFixture<DiffSmartComponent>;
  let workbenchClientSpy: jasmine.SpyObj<WorkbenchClientService>;
  let mockSelectedRevision: WritableSignal<ReadonlyDomainElement<Revision> | null>;
  let mockPreviousOfSelectedRevision: WritableSignal<ReadonlyDomainElement<Revision> | null>;
  let mockSelectedTimeline: WritableSignal<ReadonlyDomainElement<Timeline> | null>;

  beforeEach(async () => {
    workbenchClientSpy = jasmine.createSpyObj('WorkbenchClientService', [
      'readStructYAML',
    ]);
    mockSelectedRevision = signal<ReadonlyDomainElement<Revision> | null>(null);
    mockPreviousOfSelectedRevision =
      signal<ReadonlyDomainElement<Revision> | null>(null);
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
            selectedRevision: mockSelectedRevision,
            previousOfSelectedRevision: mockPreviousOfSelectedRevision,
            selectedTimeline: mockSelectedTimeline,
            selectedTimelinesWithChildren: signal([]),
            selectedLog: signal(null),
            selectedLogIndex: signal(-1),
            highlightLogIndices: signal([]),
            highlightedLogIndicesOnSelectedTimeline: signal([]),
            onSelectRevision: (rev: ReadonlyDomainElement<Revision> | null) => {
              mockSelectedRevision.set(rev);
            },
            onHighlightRevision: jasmine.createSpy('onHighlightRevision'),
            onMoveRevisionSelection: jasmine.createSpy(
              'onMoveRevisionSelection',
            ),
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(DiffSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should have empty currentRevisionContent, null previousRevisionContent and isLoading false when no revision is selected', () => {
    expect(component['currentRevisionContent']()).toBe('');
    expect(component['previousRevisionContent']()).toBeNull();
    expect(component.isLoading()).toBeFalse();
  });

  it('should fetch and set current and previous revision content when revisions are selected', async () => {
    workbenchClientSpy.readStructYAML.and.callFake(async (structId: number) => {
      if (structId === 10) return 'metadata:\n  name: pod-v2\n';
      if (structId === 5) return 'metadata:\n  name: pod-v1\n';
      return '';
    });

    const mockTimeline = {
      id: 'timeline-1',
      path: [],
      revisions: [] as ReadonlyDomainElement<Revision>[],
    };

    const prevRev = {
      id: 1,
      structId: 5,
      logIndex: 0,
      timeline: mockTimeline,
    } as unknown as ReadonlyDomainElement<Revision>;

    const curRev = {
      id: 2,
      structId: 10,
      logIndex: 1,
      timeline: mockTimeline,
    } as unknown as ReadonlyDomainElement<Revision>;

    mockPreviousOfSelectedRevision.set(prevRev);
    mockSelectedRevision.set(curRev);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(workbenchClientSpy.readStructYAML).toHaveBeenCalledWith(10);
    expect(workbenchClientSpy.readStructYAML).toHaveBeenCalledWith(5);
    expect(component['currentRevisionContent']()).toBe(
      'metadata:\n  name: pod-v2\n',
    );
    expect(component['previousRevisionContent']()).toBe(
      'metadata:\n  name: pod-v1\n',
    );
    expect(component.isLoading()).toBeFalse();
  });

  it('should strip managedFields when showManagedFields is false', async () => {
    workbenchClientSpy.readStructYAML.and.returnValue(
      Promise.resolve(
        'metadata:\n  name: pod-v1\n  managedFields:\n    - manager: kubectl\n',
      ),
    );

    const curRev = {
      id: 3,
      structId: 15,
      logIndex: 2,
      timeline: { id: 'timeline-2', path: [], revisions: [] },
    } as unknown as ReadonlyDomainElement<Revision>;

    mockSelectedRevision.set(curRev);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(component['currentRevisionContent']()).not.toContain(
      'managedFields',
    );
    expect(component['currentRevisionContent']()).toContain('name: pod-v1');

    component['showManagedFields'].set(true);
    fixture.detectChanges();
    expect(component['currentRevisionContent']()).toContain('managedFields');
  });

  it('should handle readStructYAML rejection gracefully', async () => {
    workbenchClientSpy.readStructYAML.and.returnValue(
      Promise.reject(new Error('RPC failure')),
    );

    const curRev = {
      id: 5,
      structId: 25,
      logIndex: 4,
      timeline: { id: 'timeline-3', path: [], revisions: [] },
    } as unknown as ReadonlyDomainElement<Revision>;

    mockSelectedRevision.set(curRev);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(component['currentRevisionContent']()).toBe('');
    expect(component['previousRevisionContent']()).toBeNull();
    expect(component.isLoading()).toBeFalse();
  });
});
