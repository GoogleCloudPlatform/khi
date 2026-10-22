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

import { Component, input, model, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap } from '@angular/router';
import { TimelineRenderSmartComponent } from './timeline-render-smart.component';
import { InspectionDataLoaderService } from 'src/app/services/data-loader.service';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import { StyleOverrideService } from 'src/app/services/style-override.service';
import { TimelineFrameComponent } from 'src/app/timeline/components/timeline-frame.component';
import { Timeline, TimelinePathNode } from 'src/app/store/domain/timeline';
import { TimelineType } from 'src/app/store/domain/style';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { BackendFilterParams } from 'src/app/store/domain/filter/backend-filter';
import { InspectionData } from 'src/app/store/domain/inspection-data';
import { TimelineView } from 'src/app/store/domain/timeline-view';

@Component({
  selector: 'khi-timeline-frame',
  standalone: true,
  template: '',
  providers: [
    {
      provide: TimelineFrameComponent,
      useExisting: StubTimelineFrameComponent,
    },
  ],
})
class StubTimelineFrameComponent {
  readonly inspectionDataUniqueID = input<unknown>();
  readonly styleStore = input<unknown>();
  readonly chartStyle = input<unknown>();
  readonly rulerStyle = input<unknown>();
  readonly timelines = input<unknown>();
  readonly collapsedTimelineIds = input<unknown>();
  readonly timezoneShiftHours = input<unknown>();
  readonly minQueryLogTimeMS = input<unknown>();
  readonly maxQueryLogTimeMS = input<unknown>();
  readonly initialScaleApplied = input<unknown>();
  readonly allLogs = input<unknown>();
  readonly filteredLogIds = input<unknown>();
  readonly showLegend = input<unknown>();
  readonly showResizerHandle = input<unknown>();
  readonly readyToRender = signal(true);
  readonly pixelsPerMs = model<number>(1.0);
  readonly viewportLeftTimeMS = model<number>(0);
  public syncViewportAndInvalidate = jasmine.createSpy(
    'syncViewportAndInvalidate',
  );
}

const mockTimelineType: TimelineType = {
  id: 1,
  label: 'Pod',
  description: 'Pod resource',
  icon: 'pod',
  backgroundColor: { r: 0, g: 0, b: 0, a: 1 },
  foregroundColor: { r: 1, g: 1, b: 1, a: 1 },
  typeChipBackgroundColor: { r: 0, g: 0, b: 0, a: 1 },
  typeChipForegroundColor: { r: 1, g: 1, b: 1, a: 1 },
  visible: true,
  sortPriority: 0,
  height: 2,
};

function createFakeTimeline(
  id: number,
  pathIds: number[],
): ReadonlyDomainElement<Timeline> {
  const pathNodes: TimelinePathNode[] = pathIds.map((nodeId) => ({
    id: nodeId,
    type: mockTimelineType,
    label: `node-${nodeId}`,
  }));
  return {
    id,
    name: `node-${id}`,
    type: mockTimelineType,
    path: pathNodes,
    layer: pathNodes.length - 1,
    revisions: [],
    events: [],
    parent: null,
    childrenCount: 0,
  } as unknown as ReadonlyDomainElement<Timeline>;
}

describe('TimelineRenderSmartComponent', () => {
  let component: TimelineRenderSmartComponent;
  let fixture: ComponentFixture<TimelineRenderSmartComponent>;
  let mockLoader: jasmine.SpyObj<InspectionDataLoaderService>;
  let inspectionDataSignal: ReturnType<typeof signal<InspectionData | null>>;
  let timelineViewSignal: ReturnType<typeof signal<TimelineView | null>>;
  let filteredTimelinesSignal: ReturnType<
    typeof signal<ReadonlyDomainElement<Timeline>[]>
  >;
  let updateFilterParamsSpy: jasmine.Spy<(params: BackendFilterParams) => void>;
  let expandAllSpy: jasmine.Spy<() => void>;

  const rootTimeline = createFakeTimeline(1, [1]);
  const namespaceTimeline = createFakeTimeline(2, [1, 2]);
  const podTimelineA = createFakeTimeline(3, [1, 2, 3]);
  const containerTimelineA = createFakeTimeline(4, [1, 2, 3, 4]);
  const podTimelineB = createFakeTimeline(5, [1, 2, 5]);
  const allFakeTimelines = [
    rootTimeline,
    namespaceTimeline,
    podTimelineA,
    containerTimelineA,
    podTimelineB,
  ];

  beforeEach(async () => {
    filteredTimelinesSignal =
      signal<ReadonlyDomainElement<Timeline>[]>(allFakeTimelines);
    updateFilterParamsSpy = jasmine.createSpy('updateFilterParams');
    expandAllSpy = jasmine.createSpy('expandAllTimelines');

    const fakeInspectionData = {
      metadata: {
        header: {
          startTimeUnixSeconds: 1000,
          endTimeUnixSeconds: 2000,
        },
      },
      logStore: {
        logs: () => [],
      },
      timelineStore: {
        timelines: allFakeTimelines,
      },
    } as unknown as InspectionData;

    const fakeTimelineView = {
      expandAllTimelines: expandAllSpy,
      isFiltering: signal(false),
      filteredTimelines: filteredTimelinesSignal,
      filteredLogIds: signal(IdBitset.createEmpty()),
      backendFilter: {
        updateFilterParams: updateFilterParamsSpy,
      },
    } as unknown as TimelineView;

    inspectionDataSignal = signal<InspectionData | null>(null);
    timelineViewSignal = signal<TimelineView | null>(null);

    mockLoader = jasmine.createSpyObj('InspectionDataLoaderService', [
      'loadInspectionDataFromBackend',
    ]);
    mockLoader.loadInspectionDataFromBackend.and.callFake(async () => {
      inspectionDataSignal.set(fakeInspectionData);
      timelineViewSignal.set(fakeTimelineView);
    });

    const mockInspectionDataStore = {
      inspectionData: inspectionDataSignal,
      timelineView: timelineViewSignal,
    };

    const mockStyleOverrideService = {
      severities: [],
      timelineTypes: [mockTimelineType],
      getTimelineType: () => mockTimelineType,
    };

    await TestBed.configureTestingModule({
      imports: [TimelineRenderSmartComponent],
      providers: [
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              queryParamMap: convertToParamMap({}),
            },
          },
        },
        { provide: InspectionDataStore, useValue: mockInspectionDataStore },
        { provide: StyleOverrideService, useValue: mockStyleOverrideService },
      ],
    })
      .overrideComponent(TimelineRenderSmartComponent, {
        set: {
          imports: [StubTimelineFrameComponent],
          providers: [
            { provide: InspectionDataLoaderService, useValue: mockLoader },
          ],
        },
      })
      .compileComponents();

    fixture = TestBed.createComponent(TimelineRenderSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  afterEach(() => {
    fixture.destroy();
  });

  it('should register and clean up window.__khiCDPBridge.timelineRenderer on lifecycle', () => {
    expect(window.__khiCDPBridge?.timelineRenderer).toBeDefined();
    fixture.destroy();
    expect(window.__khiCDPBridge?.timelineRenderer).toBeUndefined();
  });

  it('should load inspection data once and reuse it on subsequent calls for the same inspectionId', async () => {
    const firstResult = await component.applyAndMeasure({
      inspectionId: 'insp-1',
      timelineQuery: 'name == "pod-a"',
    });

    expect(mockLoader.loadInspectionDataFromBackend).toHaveBeenCalledTimes(1);
    expect(mockLoader.loadInspectionDataFromBackend).toHaveBeenCalledWith(
      'insp-1',
    );
    expect(expandAllSpy).toHaveBeenCalled();
    expect(updateFilterParamsSpy).toHaveBeenCalledWith({
      timelineQuery: 'name == "pod-a"',
      timelineExclusionQuery: '',
      logQuery: '',
      excludeNoLogs: false,
      filterStartTime: null,
      filterEndTime: null,
    });
    expect(firstResult.totalMatched).toBe(5);
    expect(firstResult.renderedCount).toBe(5);
    expect(firstResult.hasMore).toBeFalse();
    expect(firstResult.viewStartTimeMs).toBe(1000 * 1000);
    expect(firstResult.viewEndTimeMs).toBe(2000 * 1000);
    // HEADER_HEIGHT (60) + 5 rows * (height 2 * BASE_ROW_HEIGHT 25 = 50) = 310
    expect(firstResult.height).toBe(310);

    await component.applyAndMeasure({
      inspectionId: 'insp-1',
      timelineQuery: 'name == "pod-b"',
    });
    expect(mockLoader.loadInspectionDataFromBackend).toHaveBeenCalledTimes(1);
  });

  it('should include ancestor timelines when filtering by timelineIds and enforce maxDepth and maxNodes', async () => {
    // Select containerTimelineA (id=4, path=[1, 2, 3, 4]), which should include ancestors 1, 2, 3 and exclude podTimelineB (id=5).
    const resWithAncestors = await component.applyAndMeasure({
      inspectionId: 'insp-1',
      timelineIds: [4],
    });
    expect(resWithAncestors.totalMatched).toBe(4);
    expect(resWithAncestors.renderedCount).toBe(4);
    expect(component['renderedTimelines']().map((t) => t.id)).toEqual([
      1, 2, 3, 4,
    ]);

    // Apply maxDepth=2 (layers 0, 1, 2 -> ids 1, 2, 3) and maxNodes=2.
    const resDepthAndMaxNodes = await component.applyAndMeasure({
      inspectionId: 'insp-1',
      timelineIds: [4],
      maxDepth: 2,
      maxNodes: 2,
    });
    expect(resDepthAndMaxNodes.totalMatched).toBe(3);
    expect(resDepthAndMaxNodes.renderedCount).toBe(2);
    expect(resDepthAndMaxNodes.hasMore).toBeTrue();
    expect(component['renderedTimelines']().map((t) => t.id)).toEqual([1, 2]);
  });

  it('should zoom horizontal viewport to requested viewStartTimeMs and viewEndTimeMs', async () => {
    const res = await component.applyAndMeasure({
      inspectionId: 'insp-1',
      width: 1308, // chartViewportWidth = 1308 - 308 = 1000px
      viewStartTimeMs: 1200_000,
      viewEndTimeMs: 1400_000, // duration = 200_000ms -> pixelsPerMs = 1000 / 200_000 = 0.005
    });

    expect(res.width).toBe(1308);
    expect(res.viewStartTimeMs).toBe(1200_000);
    expect(res.viewEndTimeMs).toBe(1400_000);
    expect(component['viewportLeftTimeMs']()).toBe(1200_000);
    expect(component['pixelsPerMs']()).toBeCloseTo(0.005, 6);
  });

  it('should wait for frame readiness and restore viewport signals in waitForStableFrame', async () => {
    await component.applyAndMeasure({
      inspectionId: 'insp-1',
      width: 1308,
      viewStartTimeMs: 1200_000,
      viewEndTimeMs: 1400_000,
    });

    const stubFrame = fixture.debugElement.children[0].children[0]
      .componentInstance as StubTimelineFrameComponent;
    stubFrame.readyToRender.set(false);
    // Simulate a scroll event mutating the two-way bound model signal before waitForStableFrame runs.
    stubFrame.viewportLeftTimeMS.set(999_000);

    let rafCount = 0;
    spyOn(window, 'requestAnimationFrame').and.callFake(
      (cb: FrameRequestCallback): number => {
        rafCount++;
        if (rafCount >= 2) {
          stubFrame.readyToRender.set(true);
        }
        cb(performance.now());
        return rafCount;
      },
    );

    await component.waitForStableFrame();

    expect(component['viewportLeftTimeMs']()).toBe(1200_000);
    expect(component['pixelsPerMs']()).toBeCloseTo(0.005, 6);
    expect(stubFrame.syncViewportAndInvalidate).toHaveBeenCalledTimes(2);
    expect(stubFrame.syncViewportAndInvalidate).toHaveBeenCalledWith(
      1200_000,
      0.005,
    );
  });

  it('should throw when loading a different inspection fails on a warm page without overwriting loadedInspectionId', async () => {
    await component.applyAndMeasure({
      inspectionId: 'insp-1',
    });
    expect(mockLoader.loadInspectionDataFromBackend).toHaveBeenCalledTimes(1);

    // Simulate loadInspectionDataFromBackend('insp-2') catching an error internally without updating inspectionDataStore.
    mockLoader.loadInspectionDataFromBackend.and.callFake(async () => {
      // Store remains set to insp-1's data.
    });

    await expectAsync(
      component.applyAndMeasure({
        inspectionId: 'insp-2',
      }),
    ).toBeRejectedWithError(
      /Failed to load inspection data for inspectionId=insp-2/,
    );
  });
});
