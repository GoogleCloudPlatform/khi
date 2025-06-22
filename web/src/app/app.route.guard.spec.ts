/**
 * Copyright 2024 Google LLC
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

import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, RouterModule, UrlTree } from '@angular/router';
import { SessionChildGuard, SessionHostGuard } from './app.route.guard';
import { lastValueFrom } from 'rxjs';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import {
  WINDOW_CONNECTION_PROVIDER,
  WindowConnectionProvider,
  WindowConnectorService,
} from './services/frame-connection/window-connector.service';
import { InMemoryWindowConnectionProvider } from './services/frame-connection/window-connection-provider.service';

/**
 * getUniqueWindowConnectorService returns a new instance of WindowConnectorService to emulate inter-frame connection.
 * WindowConnectorService generates a frameID for each window frames.
 * This function resets the TestBed to get a new instance of WindowConnectorService because if Angular injector shares the same instance for each injection, it can't emulate the inter-frame connection.
 */
function getUniqueWindowConnectorService(
  connectionProvider: WindowConnectionProvider,
) {
  TestBed.resetTestingModule();
  TestBed.configureTestingModule({
    providers: [
      WindowConnectorService,
      { provide: WINDOW_CONNECTION_PROVIDER, useValue: connectionProvider },
    ],
  });
  const windowConnector = TestBed.inject(WindowConnectorService);
  return windowConnector;
}

function createActivateRouteSnapshotWithSessionId(
  sessionId: string,
): ActivatedRouteSnapshot {
  /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
  const snapshot = new (ActivatedRouteSnapshot as any)([], { sessionId });
  return snapshot;
}

describe('SessionHostGuard', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [RouterModule, NoopAnimationsModule],
      providers: [
        WindowConnectorService,
        {
          provide: WINDOW_CONNECTION_PROVIDER,
          useValue: new InMemoryWindowConnectionProvider(),
        },
      ],
    });
  });

  it('should redirect to / when non-number session ID was given', async () => {
    const route = createActivateRouteSnapshotWithSessionId('foo');

    const guardResult = TestBed.runInInjectionContext(() =>
      SessionHostGuard(route),
    );

    const redirectTo = (await lastValueFrom(guardResult)) as UrlTree;
    expect(redirectTo.toString()).toBe('/session/0');
  });

  it('should accept routing when no other window have same session', async () => {
    const route = createActivateRouteSnapshotWithSessionId('0');

    const guardResult = TestBed.runInInjectionContext(() =>
      SessionHostGuard(route),
    );

    const redirectTo = (await lastValueFrom(guardResult)) as boolean;
    expect(redirectTo).toBe(true);
  });

  it('should redirect with incrementing sessionId when the given session ID is used', async () => {
    const route = createActivateRouteSnapshotWithSessionId('0');
    const connectionProvider = new InMemoryWindowConnectionProvider();
    const connector = getUniqueWindowConnectorService(connectionProvider);
    await connector.createSession(0);
    getUniqueWindowConnectorService(connectionProvider); // Forcibly reset TestBed to create a new window connector in the later injection.

    const guardResult = TestBed.runInInjectionContext(() =>
      SessionHostGuard(route),
    );

    const redirectTo = (await lastValueFrom(guardResult)) as UrlTree;
    expect(redirectTo.toString()).toBe('/session/1');
  });
});

describe('SessionChildGuard', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [RouterModule, NoopAnimationsModule],
      providers: [
        WindowConnectorService,
        {
          provide: WINDOW_CONNECTION_PROVIDER,
          useValue: new InMemoryWindowConnectionProvider(),
        },
      ],
    });
  });

  it('should redirect to / when non-number session ID was given', async () => {
    const route = createActivateRouteSnapshotWithSessionId('foo');

    const guardResult = TestBed.runInInjectionContext(() =>
      SessionHostGuard(route),
    );

    const redirectTo = (await lastValueFrom(guardResult)) as UrlTree;
    expect(redirectTo.toString()).toBe('/session/0');
  });

  it('should accept when the main window with same session ID is existing', async () => {
    const route = createActivateRouteSnapshotWithSessionId('11');
    const connectionProvider = new InMemoryWindowConnectionProvider();
    const connector = getUniqueWindowConnectorService(connectionProvider);
    await connector.createSession(11);
    getUniqueWindowConnectorService(connectionProvider); // Forcibly reset TestBed to create a new window connector in the later injection.

    const guardResult = TestBed.runInInjectionContext(() =>
      SessionChildGuard('Diagram')(route),
    );

    const redirectTo = (await lastValueFrom(guardResult)) as boolean;
    expect(redirectTo).toBe(true);
  });

  it('should redirect to /session/0 when the session ID main window is missing', async () => {
    const route = createActivateRouteSnapshotWithSessionId('11');

    const guardResult = TestBed.runInInjectionContext(() =>
      SessionChildGuard('Diagram')(route),
    );

    const redirectTo = (await lastValueFrom(guardResult)) as UrlTree;
    expect(redirectTo.toString()).toBe('/session/0');
  });
});
