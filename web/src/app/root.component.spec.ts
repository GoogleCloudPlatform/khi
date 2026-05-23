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

import { signal, WritableSignal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { BACKEND_SYNC } from './services/api/backend-sync.service';
import {
  BackendConnectionStatus,
  BackendSyncService,
} from './services/api/backend-sync-interface';
import { RootComponent } from './root.component';

describe('RootComponent', () => {
  let connectionStatus: WritableSignal<BackendConnectionStatus>;
  let addEventListenerSpy: jasmine.Spy<typeof window.addEventListener>;
  let removeEventListenerSpy: jasmine.Spy<typeof window.removeEventListener>;

  beforeEach(async () => {
    connectionStatus = signal(BackendConnectionStatus.Connecting);
    addEventListenerSpy = spyOn(window, 'addEventListener');
    removeEventListenerSpy = spyOn(window, 'removeEventListener');

    await TestBed.configureTestingModule({
      imports: [RootComponent],
      providers: [
        provideRouter([]),
        {
          provide: BACKEND_SYNC,
          useValue: {
            connectionStatus: connectionStatus.asReadonly(),
          } as unknown as BackendSyncService,
        },
      ],
    }).compileComponents();
  });

  it('should create the app root', () => {
    const fixture = TestBed.createComponent(RootComponent);

    expect(fixture.componentInstance).toBeTruthy();
  });

  it('should request confirmation before unload when backend is disconnected', () => {
    const fixture = TestBed.createComponent(RootComponent);
    fixture.detectChanges();

    connectionStatus.set(BackendConnectionStatus.Disconnected);
    fixture.detectChanges();

    const handler = getBeforeUnloadHandler(addEventListenerSpy);
    const event = {
      preventDefault: jasmine.createSpy('preventDefault'),
      returnValue: 'initial',
    } as unknown as BeforeUnloadEvent;
    handler(event);

    expect(event.preventDefault).toHaveBeenCalled();
    expect(event.returnValue).toBe('');
  });

  it('should remove confirmation before unload when backend reconnects', () => {
    const fixture = TestBed.createComponent(RootComponent);
    fixture.detectChanges();
    connectionStatus.set(BackendConnectionStatus.Disconnected);
    fixture.detectChanges();
    const handler = getBeforeUnloadHandler(addEventListenerSpy);

    connectionStatus.set(BackendConnectionStatus.Connected);
    fixture.detectChanges();

    expect(removeEventListenerSpy).toHaveBeenCalledWith(
      'beforeunload',
      handler,
    );
  });

  it('should remove confirmation before unload when destroyed', () => {
    const fixture = TestBed.createComponent(RootComponent);
    fixture.detectChanges();
    connectionStatus.set(BackendConnectionStatus.Disconnected);
    fixture.detectChanges();
    const handler = getBeforeUnloadHandler(addEventListenerSpy);

    fixture.destroy();

    expect(removeEventListenerSpy).toHaveBeenCalledWith(
      'beforeunload',
      handler,
    );
  });
});

function getBeforeUnloadHandler(
  addEventListenerSpy: jasmine.Spy<typeof window.addEventListener>,
): (event: BeforeUnloadEvent) => void {
  const call = addEventListenerSpy.calls
    .all()
    .find((call) => call.args[0] === 'beforeunload');
  expect(call).toBeDefined();
  return call!.args[1] as (event: BeforeUnloadEvent) => void;
}
