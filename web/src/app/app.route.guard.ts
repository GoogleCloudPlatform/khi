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

import { Component, inject } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import {
  ActivatedRouteSnapshot,
  Router,
  RouterStateSnapshot,
  UrlTree,
} from '@angular/router';
import { ReplaySubject, map, tap } from 'rxjs';
import {
  KHIPageType,
  WindowConnectorService,
} from './services/frame-connection/window-connector.service';
import { DiffPageDataSource } from './services/frame-connection/frames/diff-page-datasource.service';
import { GraphPageDataSource } from './services/frame-connection/frames/graph-page-datasource.service';
import { PageType } from './extensions/extension-common/extension-types/lifecycle-hook';
import { EXTENSION_STORE } from './extensions/extension-common/extension-store';

const SESSION_ID_PARAM_KEY = 'sessionId';

/**
 * CanActivate route guard for /session/:sessionId
 * `/session/:sessionId` must be opened at most 1 per session ID.
 * This guard redirect to `/` to issue another session ID.
 */
export const SessionHostGuard = (route: ActivatedRouteSnapshot) => {
  const guardResult = new ReplaySubject<boolean | UrlTree>(1);

  const snackBar = inject(MatSnackBar);
  const router = inject(Router);
  const connector = inject(WindowConnectorService);

  const sessionIdStr: string = route.params[SESSION_ID_PARAM_KEY];
  const sessionId = Number.parseInt(sessionIdStr);

  // sessionID must be an integer and not empty. Redirect "/" if session ID can't be parsed as number
  if (isNaN(sessionId) || sessionIdStr === '') {
    guardResult.next(router.createUrlTree(['/session', 0]));
    guardResult.complete();
    snackBar.open('Invalid session ID', 'OK', { duration: 1000 });
    return guardResult;
  }

  async function createAvailableSession() {
    let currentSessionId = sessionId;
    let changed = false;
    if (!(await connector.createSession(currentSessionId))) {
      currentSessionId++;
      changed = true;
    }
    if (changed) {
      const hash = location.hash.substring(1);
      guardResult.next(
        router.createUrlTree(['session', currentSessionId], {
          fragment: hash === '' ? undefined : hash,
        }),
      );
      snackBar.open(
        'This session ID is used. Creating another session...',
        'OK',
        { duration: 1000 },
      );
    } else {
      guardResult.next(true);
    }
    guardResult.complete();
  }
  if (sessionId !== connector.sessionId) {
    createAvailableSession();
  } else {
    guardResult.next(true);
    guardResult.complete();
  }

  return guardResult;
};

/**
 * CanDeactivate route guard for /session/:sessionId/?*
 * Leave session in the connector.
 */
export const SessionDeactivateGuard = (
  _: Component,
  currentRoute: ActivatedRouteSnapshot,
  currentState: RouterStateSnapshot,
  nextState: RouterStateSnapshot,
) => {
  const connector = inject(WindowConnectorService);
  const sessionIdStr: string =
    nextState.root.firstChild?.params[SESSION_ID_PARAM_KEY];
  const sessionId = Number.parseInt(sessionIdStr);
  if (sessionId !== connector.sessionId) {
    connector.leaveSession();
  }
  return true;
};

/**
 * CanActivate route guard for `/session/:sessionId/*`
 * `/session/:sessionId` window must be opened to newly open a child window.
 */
export const SessionChildGuard =
  (pageType: KHIPageType) => (route: ActivatedRouteSnapshot) => {
    const guardResult = new ReplaySubject<boolean | UrlTree>(1);
    const snackBar = inject(MatSnackBar);
    const router = inject(Router);
    const connector = inject(WindowConnectorService);

    const sessionIdStr: string = route.params[SESSION_ID_PARAM_KEY];
    const sessionId = Number.parseInt(sessionIdStr);

    // sessionID must be an integer. Redirect "/" if session ID can't be parsed as number
    if (isNaN(sessionId) || sessionIdStr === '') {
      guardResult.next(router.createUrlTree(['/session', 0]));
      guardResult.complete();
      snackBar.open('Invalid session ID', 'OK', { duration: 1000 });
      return guardResult;
    }

    async function checkAcceptanceOfSessionJoin() {
      let attempt = 0;
      while (
        !(await connector.joinSession(sessionId, pageType)) &&
        attempt < 10
      ) {
        attempt += 1;
      }
      if (attempt == 10) {
        guardResult.next(router.createUrlTree(['/session', 0]));
        guardResult.complete();
        snackBar.open('Failed to join the given session', 'OK', {
          duration: 1000,
        });
        return;
      }
      guardResult.next(true);
      guardResult.complete();
    }
    checkAcceptanceOfSessionJoin();

    return guardResult;
  };

export const DiffPageGuard = () => {
  const diffSource = inject(DiffPageDataSource);
  const connector = inject(WindowConnectorService);
  return connector.sessionEstablished.pipe(
    tap(() => {
      diffSource.enable();
    }),
    map(() => true),
  );
};

export const PageOpenLifecycleGuard = (page: PageType) => {
  return () => {
    const extension = inject(EXTENSION_STORE);
    extension.notifyLifecycleOnPageOpen(page);
    return true;
  };
};

export const DiffPageDeactivateGuard = () => {
  const diffSource = inject(DiffPageDataSource);

  diffSource.disable();
};

export const GraphPageGuard = () => {
  const graphSource = inject(GraphPageDataSource);
  const connector = inject(WindowConnectorService);
  return connector.sessionEstablished.pipe(
    tap(() => {
      graphSource.enable();
    }),
    map(() => true),
  );
};

export const GraphPageDeactiveGuard = () => {
  const graphSource = inject(GraphPageDataSource);

  graphSource.disable();
};
