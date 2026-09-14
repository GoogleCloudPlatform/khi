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
  ComponentFixture,
  TestBed,
  fakeAsync,
  tick,
} from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { MetadataCodeViewerComponent } from './metadata-code-viewer.component';

describe('MetadataCodeViewerComponent', () => {
  let component: MetadataCodeViewerComponent;
  let fixture: ComponentFixture<MetadataCodeViewerComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MetadataCodeViewerComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(MetadataCodeViewerComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('code', 'SELECT * FROM logs;');
    fixture.detectChanges();
  });

  it('should create and render code text', () => {
    expect(component).toBeTruthy();
    const codeElement = fixture.nativeElement.querySelector('code');
    expect(codeElement?.textContent).toContain('SELECT * FROM logs;');
  });

  it('should render title and badge when provided', () => {
    fixture.componentRef.setInput('title', 'Query 1');
    fixture.componentRef.setInput('badge', 'Estimated: 50');
    fixture.detectChanges();

    const titleEl = fixture.nativeElement.querySelector('.code-viewer-title');
    const badgeEl = fixture.nativeElement.querySelector('.code-viewer-badge');
    expect(titleEl?.textContent).toContain('Query 1');
    expect(badgeEl?.textContent).toContain('Estimated: 50');
  });

  it('should toggle copy feedback icon on copy event', fakeAsync(() => {
    const button = fixture.debugElement.query(By.css('.copy-button'));
    expect(button).toBeTruthy();

    button.triggerEventHandler('cdkCopyToClipboardCopied', null);
    fixture.detectChanges();

    const icon = fixture.nativeElement.querySelector('mat-icon');
    expect(icon.textContent).toContain('check');

    tick(1500);
    fixture.detectChanges();
    expect(icon.textContent).toContain('content_copy');
  }));
});
