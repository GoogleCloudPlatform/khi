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

import { Component, WritableSignal, input, model, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { ActivatedRoute, convertToParamMap } from '@angular/router';
import { TimelineRenderSmartComponent } from 'src/app/pages/timeline-render/timeline-render-smart.component';
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
  readonly timelines = input<readonly ReadonlyDomainElement<Timeline>[]>([]);
  readonly collapsedTimelineIds = input<unknown>();
  readonly timezoneShiftHours = input<unknown>();
  readonly minQueryLogTimeNs = input<unknown>();
  readonly maxQueryLogTimeNs = input<unknown>();
  readonly initialScaleApplied = input<unknown>();
  readonly allLogs = input<unknown>();
  readonly filteredLogIds = input<unknown>();
  readonly showLegend = input<unknown>();
  readonly showResizerHandle = input<unknown>();
  readonly readyToRender = signal(true);
  readonly pixelsPerNs = model<number>(1e-6);
  readonly viewportLeftTimeNs = model<bigint>(0n);
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
  name = `node-${id}`,
): ReadonlyDomainElement<Timeline> {
  const pathNodes: TimelinePathNode[] = pathIds.map((nodeId) => ({
    id: nodeId,
    type: mockTimelineType,
    label: `node-${nodeId}`,
    rawLabel: `node-${nodeId}`,
  }));
  return {
    id,
    name,
    rawName: name,
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
  let stubFrame: StubTimelineFrameComponent;
  let mockLoader: jasmine.SpyObj<InspectionDataLoaderService>;
  let inspectionDataSignal: WritableSignal<InspectionData | null>;
  let timelineViewSignal: WritableSignal<TimelineView | null>;
  let filteredTimelinesSignal: WritableSignal<
    ReadonlyDomainElement<Timeline>[]
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
      inspectionDataSignal.set({
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
      } as unknown as InspectionData);
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
              queryParamMap: convertToParamMap({
                inspectionId: 'insp-route',
                timelineQuery: 'name == "pod-a"',
                timelineIds: '4',
              }),
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
    await fixture.whenStable();
    fixture.detectChanges();
    stubFrame = fixture.debugElement.query(
      By.directive(StubTimelineFrameComponent),
    ).componentInstance as StubTimelineFrameComponent;
  });

  afterEach(() => {
    fixture.destroy();
  });

  it('should not register window.__khiCDPBridge.timelineRenderer', () => {
    expect(window.__khiCDPBridge?.timelineRenderer).toBeUndefined();
  });

  it('should automatically apply parameters from route query params on ngOnInit', async () => {
    await fixture.whenStable();
    fixture.detectChanges();

    expect(mockLoader.loadInspectionDataFromBackend).toHaveBeenCalledWith(
      'insp-route',
    );
    expect(updateFilterParamsSpy).toHaveBeenCalledWith({
      timelineQuery: 'name == "pod-a"',
      timelineExclusionQuery: '',
      logQuery: '',
      excludeNoLogs: false,
      filterStartTime: null,
      filterEndTime: null,
    });
    expect(stubFrame.timelines().map((t) => t.id)).toEqual([1, 2, 3, 4]);
  });

  it('should render viewport container without fixed inline width or height styles', () => {
    const viewportEl: HTMLElement = fixture.debugElement.query(
      By.css('.timeline-render-viewport'),
    ).nativeElement;

    expect(viewportEl.style.width).toBe('');
    expect(viewportEl.style.height).toBe('');
  });

  it('should load inspection data once and reuse it on subsequent calls for the same inspectionId', async () => {
    await component.applyRenderParams({
      inspectionId: 'insp-1',
      timelineQuery: 'name == "pod-a"',
    });

    const callsAfterFirst =
      mockLoader.loadInspectionDataFromBackend.calls.count();

    await component.applyRenderParams({
      inspectionId: 'insp-1',
      timelineQuery: 'name == "pod-b"',
    });

    expect(mockLoader.loadInspectionDataFromBackend.calls.count()).toBe(
      callsAfterFirst,
    );
  });

  const selectionCases = [
    {
      desc: 'includes ancestor timelines when filtering by timelineIds',
      params: { timelineIds: [4] },
      wantIds: [1, 2, 3, 4],
    },
    {
      desc: 'excludes timelines deeper than maxDepth',
      params: { timelineIds: [4], maxDepth: 2 },
      wantIds: [1, 2, 3],
    },
    {
      desc: 'truncates rendered timelines to maxNodes',
      params: { timelineIds: [4], maxDepth: 2, maxNodes: 2 },
      wantIds: [1, 2],
    },
  ];

  for (const tc of selectionCases) {
    it(`should ${tc.desc}`, async () => {
      await component.applyRenderParams({
        inspectionId: 'insp-1',
        ...tc.params,
      });
      fixture.detectChanges();
      expect(stubFrame.timelines().map((t) => t.id)).toEqual(tc.wantIds);
    });
  }

  it('should zoom horizontal viewport to requested viewStartTimeMs and viewEndTimeMs', async () => {
    const res = await component.applyRenderParams({
      inspectionId: 'insp-1',
      viewStartTimeMs: 1200_000,
      viewEndTimeMs: 1400_000,
    });
    fixture.detectChanges();

    expect(res).toEqual({
      viewStartTimeMs: 1_200_000,
      viewEndTimeMs: 1_400_000,
    });
    expect(stubFrame.viewportLeftTimeNs()).toBe(1_200_000_000_000n);
    expect(stubFrame.pixelsPerNs()).toBeGreaterThan(0);
  });

  it('should throw when loading a different inspection fails on a warm page', async () => {
    await component.applyRenderParams({
      inspectionId: 'insp-1',
    });

    mockLoader.loadInspectionDataFromBackend.and.callFake(async () => {
      // Store remains set to insp-1's data.
    });

    await expectAsync(
      component.applyRenderParams({
        inspectionId: 'insp-2',
      }),
    ).toBeRejectedWithError(
      /Failed to load inspection data for inspectionId=insp-2/,
    );
  });
});
