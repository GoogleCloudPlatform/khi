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

import {
  Component,
  ComponentRef,
  ElementRef,
  inject,
  Type,
  viewChild,
  ViewContainerRef,
} from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialog } from '@angular/material/dialog';
import { LayoutService } from './layout.service';
import { MenuManager } from 'src/app/services/menu/menu-manager.service';
import { TimelineSmartComponent } from 'src/app/timeline/timeline-smart.component';
import { LogSmartComponent } from 'src/app/log/log-smart.component';
import { DiffSmartComponent } from 'src/app/diff/diff-smart.component';
import { GraphSmartComponent } from 'src/app/graph/graph-smart.component';

@Component({
  template: `<div
    #layoutContainer
    style="width: 1000px; height: 800px;"
  ></div>`,
  standalone: true,
})
class TestHostComponent {
  readonly layoutContainer =
    viewChild.required<ElementRef<HTMLElement>>('layoutContainer');
  readonly viewContainerRef = inject(ViewContainerRef);
}

describe('LayoutService', () => {
  let fixture: ComponentFixture<TestHostComponent>;
  let hostComponent: TestHostComponent;
  let layoutService: LayoutService;
  let menuManager: MenuManager;
  let dialogSpy: jasmine.SpyObj<MatDialog>;

  beforeEach(async () => {
    dialogSpy = jasmine.createSpyObj('MatDialog', ['open']);

    await TestBed.configureTestingModule({
      imports: [TestHostComponent],
      providers: [
        LayoutService,
        MenuManager,
        { provide: MatDialog, useValue: dialogSpy },
      ],
    })
      .overrideComponent(TimelineSmartComponent, {
        set: { template: '<div>Timeline</div>', imports: [] },
      })
      .overrideComponent(LogSmartComponent, {
        set: { template: '<div>Log</div>', imports: [] },
      })
      .overrideComponent(DiffSmartComponent, {
        set: { template: '<div>Diff</div>', imports: [] },
      })
      .overrideComponent(GraphSmartComponent, {
        set: { template: '<div>Graph</div>', imports: [] },
      })
      .compileComponents();

    fixture = TestBed.createComponent(TestHostComponent);
    hostComponent = fixture.componentInstance;
    layoutService = TestBed.inject(LayoutService);
    menuManager = TestBed.inject(MenuManager);
  });

  afterEach(() => {
    layoutService.ngOnDestroy();
    fixture.destroy();
  });

  it('should reuse TimelineSmartComponent when switching layouts', () => {
    const hostEl = hostComponent.layoutContainer().nativeElement;
    document.body.appendChild(hostEl);

    try {
      let timelineCreatedCount = 0;
      const originalCreateComponent =
        hostComponent.viewContainerRef.createComponent.bind(
          hostComponent.viewContainerRef,
        );
      spyOn(hostComponent.viewContainerRef, 'createComponent').and.callFake(
        (componentType: unknown) => {
          if (componentType === TimelineSmartComponent) {
            timelineCreatedCount++;
          }
          return originalCreateComponent(componentType as Type<unknown>);
        },
      );

      layoutService.init(hostEl, hostComponent.viewContainerRef);
      layoutService.loadDefaultLayout();

      expect(timelineCreatedCount).toBe(1);

      // Switch to State View Layout (contains Timeline and History)
      layoutService.loadStateAnalysisLayout();
      expect(timelineCreatedCount).toBe(1);

      // Switch to Topology View Layout (contains Timeline and Graph)
      layoutService.loadTopologyAnalysisLayout();
      expect(timelineCreatedCount).toBe(1);

      // Switch back to Default Layout (contains Timeline, Log, History)
      layoutService.loadDefaultLayout();
      expect(timelineCreatedCount).toBe(1);
    } finally {
      hostEl.remove();
    }
  });

  it('should reuse pooled LogSmartComponent when returning to a layout containing logs', () => {
    const hostEl = hostComponent.layoutContainer().nativeElement;
    document.body.appendChild(hostEl);

    try {
      let logCreatedCount = 0;
      const originalCreateComponent =
        hostComponent.viewContainerRef.createComponent.bind(
          hostComponent.viewContainerRef,
        );
      spyOn(hostComponent.viewContainerRef, 'createComponent').and.callFake(
        (componentType: unknown) => {
          if (componentType === LogSmartComponent) {
            logCreatedCount++;
          }
          return originalCreateComponent(componentType as Type<unknown>);
        },
      );

      layoutService.init(hostEl, hostComponent.viewContainerRef);
      layoutService.loadDefaultLayout();

      expect(logCreatedCount).toBe(1);

      // Switch to State View Layout (which does NOT include Log)
      layoutService.loadStateAnalysisLayout();

      // Switch back to Default Layout (which includes Log)
      layoutService.loadDefaultLayout();

      // Log component should have been preserved in the pool and reused
      expect(logCreatedCount).toBe(1);
    } finally {
      hostEl.remove();
    }
  });

  it('should destroy surplus timeline instances when switching layouts with multiple timelines open', () => {
    const hostEl = hostComponent.layoutContainer().nativeElement;
    document.body.appendChild(hostEl);

    try {
      const createdRefs: ComponentRef<unknown>[] = [];
      const originalCreateComponent =
        hostComponent.viewContainerRef.createComponent.bind(
          hostComponent.viewContainerRef,
        );
      spyOn(hostComponent.viewContainerRef, 'createComponent').and.callFake(
        (componentType: unknown) => {
          const ref = originalCreateComponent(componentType as Type<unknown>);
          if (componentType === TimelineSmartComponent) {
            createdRefs.push(ref);
            spyOn(ref, 'destroy').and.callThrough();
          }
          return ref;
        },
      );

      layoutService.init(hostEl, hostComponent.viewContainerRef);
      layoutService.loadDefaultLayout();

      expect(createdRefs.length).toBe(1);

      // Open a second timeline via menu action
      const viewGroup = menuManager.groups().find((g) => g.id === 'view');
      const openTimelineItem = viewGroup?.items.find(
        (item) => item.id === 'open-timeline',
      );
      expect(openTimelineItem).toBeDefined();
      openTimelineItem?.action();

      expect(createdRefs.length).toBe(2);
      const [t1, t2] = createdRefs;

      // Switch to State View Layout (which only needs 1 timeline)
      layoutService.loadStateAnalysisLayout();

      // Exactly one surplus timeline should have been destroyed
      const destroyedCount =
        (t1.destroy as jasmine.Spy).calls.count() +
        (t2.destroy as jasmine.Spy).calls.count();
      expect(destroyedCount).toBe(1);
    } finally {
      hostEl.remove();
    }
  });

  it('should destroy all active and pooled components on ngOnDestroy', () => {
    const hostEl = hostComponent.layoutContainer().nativeElement;
    document.body.appendChild(hostEl);

    try {
      const createdRefs: ComponentRef<unknown>[] = [];
      const originalCreateComponent =
        hostComponent.viewContainerRef.createComponent.bind(
          hostComponent.viewContainerRef,
        );
      spyOn(hostComponent.viewContainerRef, 'createComponent').and.callFake(
        (componentType: unknown) => {
          const ref = originalCreateComponent(componentType as Type<unknown>);
          createdRefs.push(ref);
          spyOn(ref, 'destroy').and.callThrough();
          return ref;
        },
      );

      layoutService.init(hostEl, hostComponent.viewContainerRef);
      layoutService.loadDefaultLayout();

      // Switch to State View Layout so that LogSmartComponent is pooled (inactive)
      layoutService.loadStateAnalysisLayout();

      expect(createdRefs.length).toBeGreaterThan(0);

      // Call ngOnDestroy
      layoutService.ngOnDestroy();

      // Verify all created component references were destroyed
      for (const ref of createdRefs) {
        expect((ref.destroy as jasmine.Spy).calls.count()).toBeGreaterThan(0);
      }
    } finally {
      hostEl.remove();
    }
  });
});
