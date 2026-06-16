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
import { MarkdownPopupComponent } from './markdown-popup.component';
import { ComponentRef } from '@angular/core';

describe('MarkdownPopupComponent', () => {
  let component: MarkdownPopupComponent;
  let fixture: ComponentFixture<MarkdownPopupComponent>;
  let componentRef: ComponentRef<MarkdownPopupComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MarkdownPopupComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(MarkdownPopupComponent);
    component = fixture.componentInstance;
    componentRef = fixture.componentRef;
    componentRef.setInput('markdown', '# Hello\nThis is a **test**.');
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should render sanitized HTML from markdown', () => {
    const element = fixture.nativeElement as HTMLElement;
    const markdownBody = element.querySelector('.markdown-body');
    expect(markdownBody?.innerHTML).toContain('Hello');
  });
});
