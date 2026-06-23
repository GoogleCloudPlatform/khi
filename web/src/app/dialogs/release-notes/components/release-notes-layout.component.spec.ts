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
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { ReleaseNotesLayoutComponent } from 'src/app/dialogs/release-notes/components/release-notes-layout.component';
import { By } from '@angular/platform-browser';

describe('ReleaseNotesLayoutComponent', () => {
  let component: ReleaseNotesLayoutComponent;
  let fixture: ComponentFixture<ReleaseNotesLayoutComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ReleaseNotesLayoutComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(ReleaseNotesLayoutComponent);
    component = fixture.componentInstance;
  });

  it('should create and render sanitized markdown content', () => {
    fixture.componentRef.setInput('markdownContent', '# Hello World');
    fixture.detectChanges();

    expect(component).toBeTruthy();
    const content = fixture.debugElement.query(By.css('.markdown-body'));
    expect(content.nativeElement.innerHTML).toContain('<h1>Hello World</h1>');
  });

  it('should emit closed event when close button is clicked', () => {
    fixture.componentRef.setInput('markdownContent', 'Release notes');
    fixture.detectChanges();

    let closedEmitted = false;
    component.closed.subscribe(() => {
      closedEmitted = true;
    });

    const closeButton = fixture.debugElement.query(By.css('button'));
    closeButton.nativeElement.click();

    expect(closedEmitted).toBeTrue();
  });
});
