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
import { By } from '@angular/platform-browser';
import { TimelineRenderLayoutComponent } from 'src/app/pages/timeline-render/components/timeline-render-layout.component';
import { buildTimelineSceneViewModel } from 'src/app/pages/timeline-render/timeline-render-scene-builder';
import { IdBitset } from 'src/app/store/domain/filter/id-bitset';
import { StyleStoreLike } from 'src/app/store/domain/style-store';
import { Timeline } from 'src/app/store/domain/timeline';
import { ReadonlyDomainElement } from 'src/app/store/domain/types';
import {
  generateDefaultChartStyle,
  generateDefaultRulerStyle,
} from 'src/app/timeline/components/style-model';
import { TimelineFrameComponent } from 'src/app/timeline/components/timeline-frame.component';

@Component({
  selector: 'khi-timeline-frame',
  standalone: true,
  template: '<div class="stub-frame"></div>',
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
  readonly pixelsPerNs = model<number>(1e-8);
  readonly viewportLeftTimeNs = model<bigint>(0n);
  public readonly syncViewportAndInvalidate = jasmine.createSpy(
    'syncViewportAndInvalidate',
  );
}

describe('TimelineRenderLayoutComponent', () => {
  let fixture: ComponentFixture<TimelineRenderLayoutComponent>;
  let component: TimelineRenderLayoutComponent;

  const mockStyleStore: StyleStoreLike = {
    severities: [],
    timelineTypes: [],
    getTimelineType: () => {
      throw new Error('unused');
    },
  } as unknown as StyleStoreLike;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TimelineRenderLayoutComponent],
    })
      .overrideComponent(TimelineRenderLayoutComponent, {
        remove: { imports: [TimelineFrameComponent] },
        add: { imports: [StubTimelineFrameComponent] },
      })
      .compileComponents();

    fixture = TestBed.createComponent(TimelineRenderLayoutComponent);
    component = fixture.componentInstance;

    const emptyScene = buildTimelineSceneViewModel({
      indexPaneWidthPx: 300,
      viewportLeftTimeMs: 1_000_000,
      pixelsPerMs: 0.01,
      rows: [{ id: 1, heightPx: 50 }],
      overlay: null,
    });

    fixture.componentRef.setInput('inspectionDataUniqueID', 'uid-1');
    fixture.componentRef.setInput('styleStore', mockStyleStore);
    fixture.componentRef.setInput('chartStyle', generateDefaultChartStyle());
    fixture.componentRef.setInput(
      'rulerStyle',
      generateDefaultRulerStyle(mockStyleStore),
    );
    fixture.componentRef.setInput('timelines', []);
    fixture.componentRef.setInput('collapsedTimelineIds', new Set<number>());
    fixture.componentRef.setInput('timezoneShiftHours', 0);
    fixture.componentRef.setInput('pixelsPerNs', 1e-8);
    fixture.componentRef.setInput('viewportLeftTimeNs', 1_000_000_000_000n);
    fixture.componentRef.setInput('minQueryLogTimeNs', 1_000_000_000_000n);
    fixture.componentRef.setInput('maxQueryLogTimeNs', 2_000_000_000_000n);
    fixture.componentRef.setInput('allLogs', []);
    fixture.componentRef.setInput('filteredLogIds', IdBitset.createEmpty());
    fixture.componentRef.setInput('scene', emptyScene);
  });

  it('hides interactive toolbar in headless mode and shows it in interactive mode', () => {
    fixture.detectChanges();
    expect(
      fixture.debugElement.query(By.css('.interactive-toolbar')),
    ).toBeNull();

    fixture.componentRef.setInput('interactiveMode', true);
    fixture.componentRef.setInput('sceneId', 'scn_abc123');
    fixture.componentRef.setInput('inspectionId', 'insp-1');
    fixture.detectChanges();

    expect(
      fixture.debugElement.query(By.css('.interactive-toolbar')),
    ).not.toBeNull();

    let opened = false;
    component.openFullSession.subscribe(() => {
      opened = true;
    });
    fixture.debugElement
      .query(By.css('button.open-khi-btn'))
      .triggerEventHandler('click');
    expect(opened).toBeTrue();
  });
});
