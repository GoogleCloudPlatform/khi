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

import { Inject, Injectable, InjectionToken } from '@angular/core';
import {
  BehaviorSubject,
  filter,
  map,
  Observable,
  ReplaySubject,
  Subscription,
  takeUntil,
} from 'rxjs';

import { randomString } from '../../utils/random';

export const WINDOW_CONNECTION_PROVIDER = new InjectionToken(
  'window-connection-provider',
);

export type KHIPageType = 'Main' | 'Diagram' | 'Diff';

export interface KHIWindowPacket<T> {
  type: string;
  sessionId?: number;
  sourceFrameId?: string;
  destinationFrameId?: string;
  data: T;
}

export interface WindowConnectionProvider {
  send(data: KHIWindowPacket<unknown>): void;
  receive(): Observable<KHIWindowPacket<unknown>>;
}

/**
 * Represents a page meta information joinning a session.
 */
export interface KHISessionPage {
  frameId: string;
  pageType: KHIPageType;
}

interface SessionCreateMessage {
  sessionId: number;
}

interface SessionJoinMessage {
  sessionId: number;
  pageType: KHIPageType;
}

interface SessionStatusNotificationMessage {
  pages: KHISessionPage[];
}

/**
 * Provides functionality to connect frames with message passing way.
 * Each page have frame ID to identify the source or destination of inter-frame messages.
 * Pages needs to create/join a `session`. Main window creates a session and other type of pages joins them.
 * Main window will be a server of the other pages joined the session.
 */
@Injectable({ providedIn: 'root' })
export class WindowConnectorService {
  sessionId = -1;

  readonly sessionEstablished = new ReplaySubject(1);

  readonly sessionPages = new BehaviorSubject<KHISessionPage[]>([]);

  /**
   * Indicates if connected session contains main page connection or not.
   * This will be false when the main page was closed.
   */
  readonly mainPageConenctionEstablished = this.sessionPages.pipe(
    map(
      (pages) => pages.filter((page) => page.pageType === 'Main').length === 1,
    ),
  );

  readonly frameId: string = '';

  private isHost = false;

  private messageSource: Observable<KHIWindowPacket<unknown>>;

  private exclusiveSessionSubscription?: Subscription;

  private sessionAcceptionSubscription?: Subscription;

  private sessionInfoNotificationSubscription?: Subscription;

  private leaveSessionSubscription?: Subscription;

  private focusWindowSubscription?: Subscription;

  constructor(
    @Inject(WINDOW_CONNECTION_PROVIDER)
    private connectionProvider: WindowConnectionProvider,
  ) {
    this.frameId = randomString();
    this.messageSource = this.connectionProvider.receive().pipe(
      filter(
        (message) =>
          message.sessionId === undefined || // The first message to join session / create session doesn't have session ID. it should be sent to all frames.
          message.sessionId === this.sessionId,
      ),
      filter((message) => message.sourceFrameId !== this.frameId), // Ignore message if the source is itself.
      filter(
        (
          message, // Ignore message not tagreting to my frame if the destination frame is specified
        ) =>
          typeof message.destinationFrameId === 'undefined' ||
          message.destinationFrameId == this.frameId,
      ),
    );
  }

  /**
   * Attempts to create a session with given session ID.
   * If there were another frames having the same session ID already, this attempt will fails.
   * Monitor the other pages joining/creating session requests after creating session and reject/accept as responding to these requests.
   *
   * @returns If the session creation succeeded or not.
   */
  async createSession(sessionId: number): Promise<boolean> {
    this.broadcast<SessionCreateMessage>('CREATE_SESSION', { sessionId }, true);
    if ((await this.waitMessage('REJECT_CREATE_SESSION', 300)) !== null) {
      console.warn(`Session creation rejected for session ID: ${sessionId}`);
      return false;
    }

    this.sessionId = sessionId;
    this.isHost = true;
    this.sessionPages.next([
      {
        pageType: 'Main',
        frameId: this.frameId,
      },
    ]);
    this.subscribeSessionExclusiveMessages(sessionId);
    this.subscribeWindowFocusMessage();
    this.sessionAcceptionSubscription = this.receiver<SessionJoinMessage>(
      'JOIN_SESSION',
    )
      .pipe(filter((data) => data.data.sessionId === sessionId))
      .subscribe((packet) => {
        this.unicast('ACCEPT_JOIN_SESSION', {}, packet.sourceFrameId!);
        this.sessionPages.next(
          this.dedupePages([
            ...this.sessionPages.value,
            {
              frameId: packet.sourceFrameId!,
              pageType: packet.data.pageType,
            },
          ]),
        );
        this.broadcast<SessionStatusNotificationMessage>(
          'NOTIFY_SESSION_INFO',
          {
            pages: this.sessionPages.value,
          },
        );
      });
    this.leaveSessionSubscription = this.receiver<null>(
      'LEAVE_SESSION',
    ).subscribe((packet) => {
      this.sessionPages.next(
        this.sessionPages.value.filter(
          (page) => page.frameId !== packet.sourceFrameId,
        ),
      );
    });
    window.addEventListener('beforeunload', () => this.beforeUnload());
    this.sessionEstablished.next(1);
    this.sessionEstablished.complete();
    return true;
  }

  /**
   * Attempts to join the session.
   * If there were no page created the session, this attempt will fails.
   *
   * @returns If the session joining succeeded or not.
   */
  async joinSession(
    sessionId: number,
    pageType: KHIPageType,
  ): Promise<boolean> {
    this.broadcast<SessionJoinMessage>(
      'JOIN_SESSION',
      {
        sessionId,
        pageType,
      },
      true,
    );
    if ((await this.waitMessage('ACCEPT_JOIN_SESSION', 300)) === null) {
      console.warn(`Session joining rejected for session ID: ${sessionId}`);
      return false;
    }

    this.sessionId = sessionId;
    this.isHost = false;
    this.subscribeSessionExclusiveMessages(sessionId);
    this.subscribeWindowFocusMessage();
    this.sessionInfoNotificationSubscription =
      this.receiver<SessionStatusNotificationMessage>(
        'NOTIFY_SESSION_INFO',
      ).subscribe((packet) => {
        this.sessionPages.next(packet.data.pages);
      });
    window.addEventListener('beforeunload', () => this.beforeUnload());
    this.sessionEstablished.next(1);
    this.sessionEstablished.complete();
    return true;
  }

  leaveSession() {
    this.broadcast('LEAVE_SESSION', {});
    if (this.isHost) {
      this.broadcast<SessionStatusNotificationMessage>('NOTIFY_SESSION_INFO', {
        pages: [],
      });
    }
    this.sessionId = -1;
    this.exclusiveSessionSubscription?.unsubscribe();
    this.sessionAcceptionSubscription?.unsubscribe();
    this.sessionInfoNotificationSubscription?.unsubscribe();
    this.leaveSessionSubscription?.unsubscribe();
    this.focusWindowSubscription?.unsubscribe();
    this.sessionPages.next([]);
  }

  receiver<T>(messageType: string): Observable<KHIWindowPacket<T>> {
    return this.messageSource.pipe(
      filter((message) => message.type === messageType),
    ) as Observable<KHIWindowPacket<T>>;
  }

  broadcast<T>(type: string, data: T, ignoreSession = false) {
    const packet: KHIWindowPacket<T> = {
      type,
      data,
      sourceFrameId: this.frameId,
    };
    if (!ignoreSession) {
      packet.sessionId = this.sessionId;
    }
    this.connectionProvider.send(packet);
  }

  unicast<T>(type: string, data: T, destinationFrameId: string) {
    // frameId should be unique in all frame. SessionID should be ignored.
    const packet: KHIWindowPacket<T> = {
      type,
      data,
      sourceFrameId: this.frameId,
      destinationFrameId,
    };
    this.connectionProvider.send(packet);
  }

  /**
   * Waits for receiving specific message for given time period.
   *
   * @returns The received message data when it comes within the timeout period. Returns `null` if the deadline exceeded.
   */
  async waitMessage<T>(
    type: string,
    timeout = 1000,
  ): Promise<KHIWindowPacket<T> | null> {
    const disposer = new ReplaySubject(1);

    const result = await Promise.race([
      this.waitFor(timeout, null),
      new Promise<KHIWindowPacket<T>>((resolve) => {
        this.receiver<T>(type)
          .pipe(takeUntil(disposer))
          .subscribe((d) => resolve(d));
      }),
    ]);
    disposer.next({});
    disposer.complete();
    return result;
  }

  focusWindow(frameId: string) {
    window.blur();
    this.unicast('FOCUS_WINDOW', {}, frameId);
  }

  private beforeUnload() {
    this.leaveSession();
  }

  /**
   * Register a window message handler to send rejection to a window when the window requested to create a session already existed.
   */
  private subscribeSessionExclusiveMessages(sessionId: number) {
    this.exclusiveSessionSubscription = this.receiver<SessionCreateMessage>(
      'CREATE_SESSION',
    )
      .pipe(filter((packet) => packet.data.sessionId === sessionId))
      .subscribe((packet) => {
        console.warn('Rejecting creating session for ', packet.data.sessionId);
        this.unicast('REJECT_CREATE_SESSION', {}, packet.sourceFrameId!);
      });
  }

  /**
   * Monitor FOCUS_WINDOW message to focus current window when it was requested.
   */
  private subscribeWindowFocusMessage() {
    this.focusWindowSubscription = this.receiver('FOCUS_WINDOW').subscribe(
      () => {
        window.focus();
      },
    );
  }

  /**
   * Dedupe KHI session pages by checking the uniqueness of the frame IDs.
   */
  private dedupePages(pages: KHISessionPage[]): KHISessionPage[] {
    const usedFrame = new Set<string>();
    const result: KHISessionPage[] = [];
    for (const page of pages) {
      if (!usedFrame.has(page.frameId)) {
        usedFrame.add(page.frameId);
        result.push(page);
      }
    }
    return result;
  }

  private waitFor<T>(msec: number, resolutionData: T): Promise<T> {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve(resolutionData);
      }, msec);
    });
  }
}
