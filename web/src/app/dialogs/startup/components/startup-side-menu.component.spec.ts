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
import { StartupSideMenuComponent } from './startup-side-menu.component';
import { By } from '@angular/platform-browser';

describe('StartupSideMenuComponent', () => {
  let component: StartupSideMenuComponent;
  let fixture: ComponentFixture<StartupSideMenuComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [StartupSideMenuComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(StartupSideMenuComponent);
    component = fixture.componentInstance;

    // Set required inputs
    fixture.componentRef.setInput('version', 'V1.0.0');
    fixture.componentRef.setInput('links', [
      {
        icon: 'description',
        label: 'Documentation',
        url: 'https://github.com/GoogleCloudPlatform/kubernetes-history-inspector#doc',
      },
      {
        icon: 'bug_report',
        label: 'Report Bug',
        url: 'https://github.com/GoogleCloudPlatform/kubernetes-history-inspector/issues',
      },
      {
        icon: 'code',
        label: 'GitHub',
        url: 'https://github.com/GoogleCloudPlatform/kubernetes-history-inspector#github',
      },
    ]);

    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should display the version', () => {
    const versionEl = fixture.debugElement.query(
      By.css('.version'),
    ).nativeElement;
    expect(versionEl.textContent).toContain('V1.0.0');
  });

  it('should emit newInvestigation when New Inspection button is clicked', () => {
    const emitSpy = spyOn(component.newInvestigation, 'emit');

    // Find the button with text "New Inspection"
    const buttons = fixture.debugElement.queryAll(
      By.css('button[mat-flat-button]'),
    );
    const btn = buttons.find((b) =>
      b.nativeElement.textContent.includes('New Inspection'),
    );

    expect(btn).toBeTruthy();
    btn!.nativeElement.click();

    expect(emitSpy).toHaveBeenCalled();
  });

  it('should emit openKhiFile when Open .khi file button is clicked', () => {
    const emitSpy = spyOn(component.openKhiFile, 'emit');

    // Find the button with text "Open .khi file"
    const buttons = fixture.debugElement.queryAll(
      By.css('button[mat-stroked-button]'),
    );
    const btn = buttons.find((b) =>
      b.nativeElement.textContent.includes('Open .khi file'),
    );

    expect(btn).toBeTruthy();
    btn!.nativeElement.click();

    expect(emitSpy).toHaveBeenCalled();
  });

  it('should have correct external links', () => {
    const links = fixture.debugElement.queryAll(By.css('.footer-link'));
    expect(links.length).toBe(3);

    const docLink = links.find((l) =>
      l.nativeElement.textContent.includes('Documentation'),
    );
    expect(docLink).toBeTruthy();
    expect(docLink!.nativeElement.getAttribute('href')).toBe(
      'https://github.com/GoogleCloudPlatform/kubernetes-history-inspector#doc',
    );
    expect(docLink!.nativeElement.getAttribute('target')).toBe('_blank');
    expect(docLink!.nativeElement.getAttribute('rel')).toBe(
      'noopener noreferrer',
    );

    const bugLink = links.find((l) =>
      l.nativeElement.textContent.includes('Report Bug'),
    );
    expect(bugLink).toBeTruthy();
    expect(bugLink!.nativeElement.getAttribute('href')).toBe(
      'https://github.com/GoogleCloudPlatform/kubernetes-history-inspector/issues',
    );

    const githubLink = links.find((l) =>
      l.nativeElement.textContent.includes('GitHub'),
    );
    expect(githubLink).toBeTruthy();
    expect(githubLink!.nativeElement.getAttribute('href')).toBe(
      'https://github.com/GoogleCloudPlatform/kubernetes-history-inspector#github',
    );
  });
});
