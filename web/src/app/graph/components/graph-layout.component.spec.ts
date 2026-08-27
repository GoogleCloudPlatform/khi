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

import { By } from '@angular/platform-browser';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { DEFAULT_DELETION_THRESHOLD_SECONDS } from 'src/app/common/schema/graph-schema';
import { GraphLayoutComponent } from 'src/app/graph/components/graph-layout.component';
import { GraphToolbarComponent } from './graph-toolbar.component';

interface ComponentWithRenderer {
  graphRenderer: {
    fitToView: () => void;
    downloadSvg: () => void;
    downloadPng: () => void;
  };
}

function asComponentWithRenderer(
  cmp: GraphLayoutComponent,
): ComponentWithRenderer {
  return cmp as unknown as ComponentWithRenderer;
}

describe('GraphLayoutComponent', () => {
  let component: GraphLayoutComponent;
  let fixture: ComponentFixture<GraphLayoutComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [GraphLayoutComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(GraphLayoutComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize deletionThresholdSeconds with default value', () => {
    expect(component.deletionThresholdSeconds()).toBe(
      DEFAULT_DELETION_THRESHOLD_SECONDS,
    );
  });

  it('should delegate fitToView when toolbar emits fitToView event', () => {
    const wrapped = asComponentWithRenderer(component);
    spyOn(wrapped.graphRenderer, 'fitToView');
    const toolbar = fixture.debugElement.query(
      By.directive(GraphToolbarComponent),
    );
    toolbar.componentInstance.fitToView.emit();
    expect(wrapped.graphRenderer.fitToView).toHaveBeenCalled();
  });

  it('should delegate downloadSvg when toolbar emits downloadSvg event', () => {
    const wrapped = asComponentWithRenderer(component);
    spyOn(wrapped.graphRenderer, 'downloadSvg');
    const toolbar = fixture.debugElement.query(
      By.directive(GraphToolbarComponent),
    );
    toolbar.componentInstance.downloadSvg.emit();
    expect(wrapped.graphRenderer.downloadSvg).toHaveBeenCalled();
  });

  it('should delegate downloadPng when toolbar emits downloadPng event', () => {
    const wrapped = asComponentWithRenderer(component);
    spyOn(wrapped.graphRenderer, 'downloadPng');
    const toolbar = fixture.debugElement.query(
      By.directive(GraphToolbarComponent),
    );
    toolbar.componentInstance.downloadPng.emit();
    expect(wrapped.graphRenderer.downloadPng).toHaveBeenCalled();
  });

  it('should toggle loading overlay and hide container when isLoading is true', () => {
    fixture.componentRef.setInput('isLoading', true);
    fixture.detectChanges();

    const svgContainer = fixture.nativeElement.querySelector('.svg-container');
    const loadingOverlay =
      fixture.nativeElement.querySelector('.loading-overlay');

    expect(svgContainer.classList.contains('hidden')).toBeTrue();
    expect(loadingOverlay).toBeTruthy();
    expect(loadingOverlay.textContent).toContain(
      'Loading architecture graph...',
    );
  });
});
