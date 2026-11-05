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

import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { Component, WritableSignal, input, model, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { ActivatedRoute, convertToParamMap } from '@angular/router';
import {
  TimelineExcerptKind,
  TimelineExcerptLineKind,
  TimelineMarkerStyle,
} from 'src/app/common/schema/timeline-render-types';
import { TimelineEvidenceDeckComponent } from 'src/app/pages/timeline-render/components/timeline-evidence-deck.component';
import { TimelineRenderLayoutComponent } from 'src/app/pages/timeline-render/components/timeline-render-layout.component';
import { TimelineSceneOverlayComponent } from 'src/app/pages/timeline-render/components/timeline-scene-overlay.component';
import {
  McpTimelineRenderSmartComponent,
  TIMELINE_SESSION_URL_OPENER,
} from 'src/app/pages/timeline-render/mcp-timeline-render-smart.component';
import { TIMELINE_TEXT_WIDTH_MEASURER } from 'src/app/pages/timeline-render/types/timeline-render-sizing';
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
  readonly indexPaneSize = input<number>(300);
  readonly horizontalOverdrawInPx = input<number>(300);
  readonly showLegend = input<unknown>();
  readonly showResizerHandle = input<unknown>();
  readonly readyToRender = signal(true);
  readonly pixelsPerNs = model<number>(1e-6);
  readonly viewportLeftTimeNs = model<bigint>(0n);
  public readonly syncViewportAndInvalidate = jasmine.createSpy(
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

describe('McpTimelineRenderSmartComponent', () => {
  let component: McpTimelineRenderSmartComponent;
  let fixture: ComponentFixture<McpTimelineRenderSmartComponent>;
  let stubFrame: StubTimelineFrameComponent;
  let mockLoader: jasmine.SpyObj<InspectionDataLoaderService>;
  let sessionUrlOpenerSpy: jasmine.Spy<(url: string) => void>;
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
    sessionUrlOpenerSpy = jasmine.createSpy('sessionUrlOpener');

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
      imports: [McpTimelineRenderSmartComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
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
        {
          provide: TIMELINE_TEXT_WIDTH_MEASURER,
          useValue: (text: string) => text.length * 7,
        },
        {
          provide: TIMELINE_SESSION_URL_OPENER,
          useValue: sessionUrlOpenerSpy,
        },
      ],
    })
      .overrideComponent(McpTimelineRenderSmartComponent, {
        set: {
          imports: [TimelineRenderLayoutComponent],
          providers: [
            { provide: InspectionDataLoaderService, useValue: mockLoader },
          ],
        },
      })
      .overrideComponent(TimelineRenderLayoutComponent, {
        set: {
          imports: [
            StubTimelineFrameComponent,
            TimelineSceneOverlayComponent,
            TimelineEvidenceDeckComponent,
          ],
        },
      })
      .compileComponents();

    fixture = TestBed.createComponent(McpTimelineRenderSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    stubFrame = fixture.debugElement.query(
      By.directive(StubTimelineFrameComponent),
    ).componentInstance as StubTimelineFrameComponent;
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
    // MIN_INDEX_PANE_WIDTH_PX (300) + GUTTER_WIDTH (8) + MIN_CHART_VIEWPORT_WIDTH_PX (1280) = 1588.
    // HEADER_HEIGHT (60) + 5 rows * (height 2 * BASE_ROW_HEIGHT 25 = 50) = 310.
    expect(firstResult).toEqual({
      width: 1588,
      height: 310,
      renderedCount: 5,
      totalMatched: 5,
      hasMore: false,
      viewStartTimeMs: 1_000_000,
      viewEndTimeMs: 2_000_000,
    });
    expect(stubFrame.indexPaneSize()).toBe(300);

    await component.applyAndMeasure({
      inspectionId: 'insp-1',
      timelineQuery: 'name == "pod-b"',
    });
    expect(mockLoader.loadInspectionDataFromBackend).toHaveBeenCalledTimes(1);
  });

  const selectionCases = [
    {
      desc: 'includes ancestor timelines when filtering by timelineIds',
      params: { timelineIds: [4] },
      wantIds: [1, 2, 3, 4],
      wantResult: {
        width: 1588,
        height: 260,
        totalMatched: 4,
        renderedCount: 4,
        hasMore: false,
        viewStartTimeMs: 1_000_000,
        viewEndTimeMs: 2_000_000,
      },
    },
    {
      desc: 'excludes timelines deeper than maxDepth',
      params: { timelineIds: [4], maxDepth: 2 },
      wantIds: [1, 2, 3],
      wantResult: {
        width: 1588,
        height: 210,
        totalMatched: 3,
        renderedCount: 3,
        hasMore: false,
        viewStartTimeMs: 1_000_000,
        viewEndTimeMs: 2_000_000,
      },
    },
    {
      desc: 'truncates rendered timelines to maxNodes and sets hasMore',
      params: { timelineIds: [4], maxDepth: 2, maxNodes: 2 },
      wantIds: [1, 2],
      wantResult: {
        width: 1588,
        height: 160,
        totalMatched: 3,
        renderedCount: 2,
        hasMore: true,
        viewStartTimeMs: 1_000_000,
        viewEndTimeMs: 2_000_000,
      },
    },
  ];

  for (const tc of selectionCases) {
    it(`should ${tc.desc}`, async () => {
      const res = await component.applyAndMeasure({
        inspectionId: 'insp-1',
        ...tc.params,
      });
      fixture.detectChanges();
      expect(res).toEqual(tc.wantResult);
      expect(stubFrame.timelines().map((t) => t.id)).toEqual(tc.wantIds);
    });
  }

  it('should zoom horizontal viewport to requested viewStartTimeMs and viewEndTimeMs', async () => {
    const res = await component.applyAndMeasure({
      inspectionId: 'insp-1',
      viewStartTimeMs: 1200_000,
      viewEndTimeMs: 1400_000,
    });
    fixture.detectChanges();

    expect(res).toEqual({
      width: 1588,
      height: 310,
      renderedCount: 5,
      totalMatched: 5,
      hasMore: false,
      viewStartTimeMs: 1_200_000,
      viewEndTimeMs: 1_400_000,
    });
    expect({
      viewportLeftTimeNs: stubFrame.viewportLeftTimeNs(),
      pixelsPerNs: stubFrame.pixelsPerNs(),
    }).toEqual({
      viewportLeftTimeNs: 1_200_000_000_000n,
      pixelsPerNs: 6.4e-9,
    });
  });

  it('should expand indexPaneSize and stage width horizontally when rendered timeline names are wide while keeping minimum chart viewport width', async () => {
    const widePod = createFakeTimeline(
      6,
      [1, 2, 3, 6],
      'gke-test-cluster-e2-standard-8-pool-node-12345678-abcd',
    );
    filteredTimelinesSignal.set([
      rootTimeline,
      namespaceTimeline,
      podTimelineA,
      widePod,
    ]);

    const res = await component.applyAndMeasure({
      inspectionId: 'insp-1',
      viewStartTimeMs: 1200_000,
      viewEndTimeMs: 1400_000,
    });
    fixture.detectChanges();

    const indexPaneSize = stubFrame.indexPaneSize();
    expect(indexPaneSize).toBeGreaterThan(300);
    expect(res.width).toBe(indexPaneSize + 8 + 1280);
    expect(stubFrame.pixelsPerNs()).toBe(6.4e-9);
  });

  it('should wait for frame readiness and restore viewport signals in waitForStableFrame', async () => {
    await component.applyAndMeasure({
      inspectionId: 'insp-1',
      viewStartTimeMs: 1200_000,
      viewEndTimeMs: 1400_000,
    });
    fixture.detectChanges();

    stubFrame.readyToRender.set(false);
    stubFrame.viewportLeftTimeNs.set(999_000_000_000n);
    fixture.detectChanges();

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
    fixture.detectChanges();

    expect({
      viewportLeftTimeNs: stubFrame.viewportLeftTimeNs(),
      pixelsPerNs: stubFrame.pixelsPerNs(),
    }).toEqual({
      viewportLeftTimeNs: 1_200_000_000_000n,
      pixelsPerNs: 6.4e-9,
    });
    expect(stubFrame.syncViewportAndInvalidate.calls.allArgs()).toEqual([
      [1_200_000_000_000n, 6.4e-9],
      [1_200_000_000_000n, 6.4e-9],
    ]);
  });

  it('should throw when loading a different inspection fails on a warm page', async () => {
    await component.applyAndMeasure({
      inspectionId: 'insp-1',
    });

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

  it('should wait for document.fonts.ready in CDP bridge functions before completing', async () => {
    let resolveFontsReady!: (value: FontFaceSet) => void;
    const fontsReadyPromise = new Promise<FontFaceSet>((resolve) => {
      resolveFontsReady = resolve;
    });
    const fontsReadySpy = spyOnProperty(
      document.fonts,
      'ready',
      'get',
    ).and.returnValue(fontsReadyPromise);

    const bridge = window.__khiCDPBridge?.timelineRenderer;
    expect(bridge).toBeDefined();

    let applyCompleted = false;
    const bridgeApplyPromise = bridge!
      .applyAndMeasure({
        inspectionId: 'insp-1',
      })
      .then(() => {
        applyCompleted = true;
      });

    await Promise.resolve();
    expect(applyCompleted).toBeFalse();

    resolveFontsReady(document.fonts);
    await bridgeApplyPromise;
    expect(applyCompleted).toBeTrue();

    let resolveFrameFontsReady!: (value: FontFaceSet) => void;
    const frameFontsReadyPromise = new Promise<FontFaceSet>((resolve) => {
      resolveFrameFontsReady = resolve;
    });
    fontsReadySpy.and.returnValue(frameFontsReadyPromise);

    let frameCompleted = false;
    const bridgeFramePromise = bridge!.waitForStableFrame().then(() => {
      frameCompleted = true;
    });

    await Promise.resolve();
    expect(frameCompleted).toBeFalse();

    resolveFrameFontsReady(document.fonts);
    await bridgeFramePromise;
    expect(frameCompleted).toBeTrue();
  });

  it('should not wait for document.fonts.ready when applyAndMeasure is called directly', async () => {
    const neverResolvingFonts = new Promise<FontFaceSet>(() => undefined);
    spyOnProperty(document.fonts, 'ready', 'get').and.returnValue(
      neverResolvingFonts,
    );

    const result = await component.applyAndMeasure({
      inspectionId: 'insp-1',
    });

    expect(result.renderedCount).toBe(5);
  });

  it('should expand index pane and chart viewport width for overlay role badges and evidence deck cards', async () => {
    const res = await component.applyAndMeasure({
      inspectionId: 'insp-1',
      viewStartTimeMs: 1_200_000,
      viewEndTimeMs: 1_400_000,
      overlay: {
        rowHighlights: [
          {
            id: 'rh-trigger',
            roleLabel: 'TRIGGER · Control Plane Policy Update',
            timelineIds: [3],
            paletteId: 'default',
          },
        ],
        excerpts: [
          {
            id: 'ex-1',
            kind: TimelineExcerptKind.DIFF,
            badge: 'E1',
            paletteId: 'default',
            timeMs: 1_250_000,
            timeLabel: '00:20:50',
            timelineId: 3,
            resourcePath: 'pod/default/test-pod-a',
            title: 'Patch',
            subtitle: '',
            lines: [
              { kind: TimelineExcerptLineKind.ADDED, text: 'a'.repeat(90) },
            ],
          },
          {
            id: 'ex-2',
            kind: TimelineExcerptKind.LOG,
            badge: 'E2',
            paletteId: 'default',
            timeMs: 1_260_000,
            timeLabel: '00:21:00',
            timelineId: 3,
            resourcePath: 'pod/default/test-pod-a',
            title: 'Log',
            subtitle: '',
            lines: [{ kind: TimelineExcerptLineKind.CONTEXT, text: 'ok' }],
          },
          {
            id: 'ex-3',
            kind: TimelineExcerptKind.LOG,
            badge: 'E3',
            paletteId: 'default',
            timeMs: 1_270_000,
            timeLabel: '00:21:10',
            timelineId: 3,
            resourcePath: 'pod/default/test-pod-a',
            title: 'Log',
            subtitle: '',
            lines: [{ kind: TimelineExcerptLineKind.CONTEXT, text: 'ok' }],
          },
        ],
      },
    });
    fixture.detectChanges();

    const indexPaneSize = stubFrame.indexPaneSize();
    const chartViewportWidthPx = res.width - indexPaneSize - 8;
    expect(indexPaneSize).toBeGreaterThan(300);
    expect(chartViewportWidthPx).toBeGreaterThan(1280);
    expect(stubFrame.pixelsPerNs()).toBeCloseTo(
      chartViewportWidthPx / 200_000_000_000,
      15,
    );
  });

  it('should preserve overlay autoIncludeTimelineIds and ancestors when maxNodes truncates matched timelines', async () => {
    const res = await component.applyAndMeasure({
      inspectionId: 'insp-1',
      maxNodes: 2,
      overlay: {
        autoIncludeTimelineIds: [4],
      },
    });
    fixture.detectChanges();

    expect(stubFrame.timelines().map((t) => t.id)).toEqual([1, 2, 3, 4]);
    expect(res.renderedCount).toBe(4);
    expect(res.hasMore).toBe(true);
  });

  it('should compute composite height including top marker band and bottom deck cards', async () => {
    const res = await component.applyAndMeasure({
      inspectionId: 'insp-1',
      overlay: {
        markers: [
          {
            id: 'm-1',
            badge: '1',
            label: 'Test Marker',
            style: TimelineMarkerStyle.PIN,
            paletteId: 'default',
            timeMs: 1_200_000,
            timeLabel: '00:20:00',
            timelineId: 3,
          },
        ],
        excerpts: [
          {
            id: 'ex-1',
            kind: TimelineExcerptKind.DIFF,
            badge: 'E1',
            paletteId: 'default',
            timeMs: 1_250_000,
            timeLabel: '00:20:50',
            timelineId: 3,
            resourcePath: 'pod/default/test-pod-a',
            title: 'Patch',
            subtitle: '',
            lines: [{ kind: TimelineExcerptLineKind.ADDED, text: 'diff' }],
          },
        ],
      },
    });
    fixture.detectChanges();

    expect(res.height).toBeGreaterThan(310);
  });

  it('should load scene from backend and enable interactive mode when sceneId is in route queryParams', async () => {
    const queryParamsMap = new Map<string, string>([
      ['sceneId', 'scn_test123'],
    ]);
    TestBed.resetTestingModule();
    await TestBed.configureTestingModule({
      imports: [McpTimelineRenderSmartComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              queryParamMap: {
                get: (key: string) => queryParamsMap.get(key) ?? null,
              },
            },
          },
        },
        {
          provide: InspectionDataStore,
          useValue: {
            inspectionData: inspectionDataSignal,
            timelineView: timelineViewSignal,
          },
        },
        {
          provide: StyleOverrideService,
          useValue: {
            severities: [],
            timelineTypes: [mockTimelineType],
            getTimelineType: () => mockTimelineType,
          },
        },
        {
          provide: TIMELINE_TEXT_WIDTH_MEASURER,
          useValue: (text: string) => text.length * 7,
        },
        {
          provide: TIMELINE_SESSION_URL_OPENER,
          useValue: sessionUrlOpenerSpy,
        },
      ],
    })
      .overrideComponent(McpTimelineRenderSmartComponent, {
        set: {
          imports: [TimelineRenderLayoutComponent],
          providers: [
            { provide: InspectionDataLoaderService, useValue: mockLoader },
          ],
        },
      })
      .overrideComponent(TimelineRenderLayoutComponent, {
        set: {
          imports: [
            StubTimelineFrameComponent,
            TimelineSceneOverlayComponent,
            TimelineEvidenceDeckComponent,
          ],
        },
      })
      .compileComponents();

    const localFixture = TestBed.createComponent(
      McpTimelineRenderSmartComponent,
    );
    const localHttpMock = TestBed.inject(HttpTestingController);
    localFixture.detectChanges();

    const req = localHttpMock.expectOne('/api/v1/mcp-scenes/scn_test123');
    expect(req.request.method).toBe('GET');
    req.flush({
      sceneId: 'scn_test123',
      inspectionId: 'insp-1',
      timelineQuery: 'name == "node-3"',
    });

    await localFixture.whenStable();
    localFixture.detectChanges();

    expect(localFixture.componentInstance['interactiveMode']()).toBe(true);
    expect(localFixture.componentInstance['activeSceneId']()).toBe(
      'scn_test123',
    );
    expect(localFixture.componentInstance['errorMessage']()).toBe('');

    localFixture.componentInstance['onJumpToSession']({
      timelineId: 3,
      logId: 19,
    });
    expect(sessionUrlOpenerSpy).toHaveBeenCalledWith(
      '/session/0?inspectionId=insp-1&timelineId=3&logId=19',
    );

    localFixture.componentInstance['onOpenFullSession']();
    expect(sessionUrlOpenerSpy).toHaveBeenCalledWith(
      '/session/0?inspectionId=insp-1&timelineQuery=name+%3D%3D+%22node-3%22',
    );
  });

  it('should display expired scene message when loading scene returns HTTP 410', async () => {
    const queryParamsMap = new Map<string, string>([
      ['sceneId', 'scn_expired'],
    ]);
    TestBed.resetTestingModule();
    await TestBed.configureTestingModule({
      imports: [McpTimelineRenderSmartComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              queryParamMap: {
                get: (key: string) => queryParamsMap.get(key) ?? null,
              },
            },
          },
        },
        {
          provide: InspectionDataStore,
          useValue: {
            inspectionData: inspectionDataSignal,
            timelineView: timelineViewSignal,
          },
        },
        {
          provide: StyleOverrideService,
          useValue: {
            severities: [],
            timelineTypes: [mockTimelineType],
            getTimelineType: () => mockTimelineType,
          },
        },
        {
          provide: TIMELINE_TEXT_WIDTH_MEASURER,
          useValue: (text: string) => text.length * 7,
        },
        {
          provide: TIMELINE_SESSION_URL_OPENER,
          useValue: sessionUrlOpenerSpy,
        },
      ],
    })
      .overrideComponent(McpTimelineRenderSmartComponent, {
        set: {
          imports: [TimelineRenderLayoutComponent],
          providers: [
            { provide: InspectionDataLoaderService, useValue: mockLoader },
          ],
        },
      })
      .overrideComponent(TimelineRenderLayoutComponent, {
        set: {
          imports: [
            StubTimelineFrameComponent,
            TimelineSceneOverlayComponent,
            TimelineEvidenceDeckComponent,
          ],
        },
      })
      .compileComponents();

    const localFixture = TestBed.createComponent(
      McpTimelineRenderSmartComponent,
    );
    const localHttpMock = TestBed.inject(HttpTestingController);
    localFixture.detectChanges();

    const req = localHttpMock.expectOne('/api/v1/mcp-scenes/scn_expired');
    req.flush('Expired', { status: 410, statusText: 'Gone' });

    await localFixture.whenStable();
    localFixture.detectChanges();

    expect(localFixture.componentInstance['errorMessage']()).toContain(
      'Scene scn_expired has expired',
    );
  });

  it('should display not found scene message when loading scene returns HTTP 404', async () => {
    const queryParamsMap = new Map<string, string>([
      ['sceneId', 'scn_not_found'],
    ]);
    TestBed.resetTestingModule();
    await TestBed.configureTestingModule({
      imports: [McpTimelineRenderSmartComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: {
              queryParamMap: {
                get: (key: string) => queryParamsMap.get(key) ?? null,
              },
            },
          },
        },
        {
          provide: InspectionDataStore,
          useValue: {
            inspectionData: inspectionDataSignal,
            timelineView: timelineViewSignal,
          },
        },
        {
          provide: StyleOverrideService,
          useValue: {
            severities: [],
            timelineTypes: [mockTimelineType],
            getTimelineType: () => mockTimelineType,
          },
        },
        {
          provide: TIMELINE_TEXT_WIDTH_MEASURER,
          useValue: (text: string) => text.length * 7,
        },
        {
          provide: TIMELINE_SESSION_URL_OPENER,
          useValue: sessionUrlOpenerSpy,
        },
      ],
    })
      .overrideComponent(McpTimelineRenderSmartComponent, {
        set: {
          imports: [TimelineRenderLayoutComponent],
          providers: [
            { provide: InspectionDataLoaderService, useValue: mockLoader },
          ],
        },
      })
      .overrideComponent(TimelineRenderLayoutComponent, {
        set: {
          imports: [
            StubTimelineFrameComponent,
            TimelineSceneOverlayComponent,
            TimelineEvidenceDeckComponent,
          ],
        },
      })
      .compileComponents();

    const localFixture = TestBed.createComponent(
      McpTimelineRenderSmartComponent,
    );
    const localHttpMock = TestBed.inject(HttpTestingController);
    localFixture.detectChanges();

    const req = localHttpMock.expectOne('/api/v1/mcp-scenes/scn_not_found');
    req.flush('Not Found', { status: 404, statusText: 'Not Found' });

    await localFixture.whenStable();
    localFixture.detectChanges();

    expect(localFixture.componentInstance['errorMessage']()).toContain(
      'Scene scn_not_found was not found',
    );
  });
});
