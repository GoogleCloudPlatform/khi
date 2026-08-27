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
import { SessionTimedOutLayoutComponent } from 'src/app/dialogs/session-timed-out/components/session-timed-out-layout.component';
import { By } from '@angular/platform-browser';

describe('SessionTimedOutLayoutComponent', () => {
  let component: SessionTimedOutLayoutComponent;
  let fixture: ComponentFixture<SessionTimedOutLayoutComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SessionTimedOutLayoutComponent, NoopAnimationsModule],
    }).compileComponents();

    fixture = TestBed.createComponent(SessionTimedOutLayoutComponent);
    component = fixture.componentInstance;
  });

  it('should create and render session timed out title when errorMessage is null', () => {
    fixture.detectChanges();

    expect(component).toBeTruthy();
    const title = fixture.debugElement.query(By.css('.dialog-title'));
    expect(title.nativeElement.textContent).toContain('Session Timed Out');
  });

  it('should emit reconnect event when Reconnect button is clicked', () => {
    fixture.detectChanges();

    let emitted = false;
    component.reconnect.subscribe(() => {
      emitted = true;
    });

    const buttons = fixture.debugElement.queryAll(By.css('button'));
    const reconnectBtn = buttons.find((b) =>
      b.nativeElement.textContent.includes('Reconnect'),
    );
    expect(reconnectBtn).toBeDefined();
    reconnectBtn?.nativeElement.click();

    expect(emitted).toBeTrue();
  });

  it('should emit returnToStartup event when Return to Startup button is clicked', () => {
    fixture.detectChanges();

    let emitted = false;
    component.returnToStartup.subscribe(() => {
      emitted = true;
    });

    const buttons = fixture.debugElement.queryAll(By.css('button'));
    const startupBtn = buttons.find((b) =>
      b.nativeElement.textContent.includes('Return to Startup'),
    );
    expect(startupBtn).toBeDefined();
    startupBtn?.nativeElement.click();

    expect(emitted).toBeTrue();
  });

  it('should render error message and action buttons when errorMessage is provided', () => {
    fixture.componentRef.setInput('errorMessage', 'Connection refused');
    fixture.detectChanges();

    const title = fixture.debugElement.query(By.css('.dialog-title'));
    expect(title.nativeElement.textContent).toContain('Failed to Reconnect');

    const errorBox = fixture.debugElement.query(By.css('.error-box'));
    expect(errorBox.nativeElement.textContent).toContain('Connection refused');
  });

  it('should emit reconnect event when Retry button is clicked in error state', () => {
    fixture.componentRef.setInput('errorMessage', 'Server crashed');
    fixture.detectChanges();

    let retryEmitted = false;
    component.reconnect.subscribe(() => {
      retryEmitted = true;
    });

    const buttons = fixture.debugElement.queryAll(By.css('button'));
    const retryBtn = buttons.find((b) =>
      b.nativeElement.textContent.includes('Retry'),
    );

    retryBtn?.nativeElement.click();
    expect(retryEmitted).toBeTrue();
  });

  it('should disable both action buttons when isReconnecting is true', () => {
    fixture.componentRef.setInput('isReconnecting', true);
    fixture.detectChanges();

    const buttons = fixture.debugElement.queryAll(By.css('button'));
    const reconnectBtn = buttons.find((b) =>
      b.nativeElement.textContent.includes('Reconnect'),
    );
    const startupBtn = buttons.find((b) =>
      b.nativeElement.textContent.includes('Return to Startup'),
    );
    expect(reconnectBtn?.nativeElement.disabled).toBeTrue();
    expect(startupBtn?.nativeElement.disabled).toBeTrue();
  });
});
