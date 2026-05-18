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

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { LogListComponent } from './log-list.component';
import { Log } from '../../store/domain/log';
import { Timeline } from '../../store/domain/timeline';
import { ReadonlyDomainElement } from '../../store/domain/types';
import { ScrollingModule } from '@angular/cdk/scrolling';
import { createMockInspectionDataV2 } from 'src/app/store/mock/inspection-data.mock';

describe('LogListComponent', () => {
  let component: LogListComponent;
  let fixture: ComponentFixture<LogListComponent>;
  let mockLogs: ReadonlyDomainElement<Log>[];
  let mockTimelines: readonly ReadonlyDomainElement<Timeline>[];

  beforeEach(async () => {
    const mockData = await createMockInspectionDataV2();
    mockLogs = Array.from(mockData.logStore.logs());
    mockTimelines = mockData.timelineStore.timelines;

    await TestBed.configureTestingModule({
      imports: [LogListComponent, ScrollingModule],
    }).compileComponents();

    fixture = TestBed.createComponent(LogListComponent);
    component = fixture.componentInstance;

    // Set required inputs
    fixture.componentRef.setInput('allLogsCount', mockLogs.length);
    fixture.componentRef.setInput('filteredLogs', mockLogs);
    fixture.componentRef.setInput('selectedLogIndex', -1);
    fixture.componentRef.setInput('highlightLogIndices', new Set<number>());
    fixture.componentRef.setInput('selectedTimelinesWithChildren', []);

    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should show all logs when filterByTimeline is false', () => {
    fixture.componentRef.setInput('filterByTimeline', false);
    fixture.detectChanges();
    expect(component['shownLogs']()).toEqual(mockLogs);
  });

  it('should filter logs by timeline when filterByTimeline is true', () => {
    fixture.componentRef.setInput('filterByTimeline', true);
    fixture.componentRef.setInput('selectedTimelinesWithChildren', [
      mockTimelines[0],
    ]);
    fixture.detectChanges();

    const timeline = mockTimelines[0];
    const logIndices = new Set<number>();
    for (const revision of timeline.revisions) {
      logIndices.add(revision.logIndex);
    }
    for (const event of timeline.events) {
      logIndices.add(event.logIndex);
    }
    const expectedLogs = mockLogs.filter((log) => logIndices.has(log.logIndex));

    expect(component['shownLogs']()).toEqual(expectedLogs);
  });

  it('should emit logSelected event', () => {
    spyOn(component.logSelected, 'emit');
    component['selectLog'](mockLogs[0]);
    expect(component.logSelected.emit).toHaveBeenCalledWith(mockLogs[0]);
  });

  it('should emit logHovered event', () => {
    spyOn(component.logHovered, 'emit');
    component['onLogHover'](mockLogs[0]);
    expect(component.logHovered.emit).toHaveBeenCalledWith(mockLogs[0]);
  });
});
