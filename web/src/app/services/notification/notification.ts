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

import { Injectable } from '@angular/core';
import { combineLatest, filter, from, Subject } from 'rxjs';

/**
 * The information to be shown as a notification
 */
export interface KHINotification {
  /**
   * Title of the notification
   */
  title: string;
  /**
   * Body of the notification
   */
  body: string;
}

/**
 * NotificationManager provides features to show notifications with WebNotification API.
 * The initialize method must be called at the beginning of this app.
 */
@Injectable({
  providedIn: 'root',
})
export class NotificationManager {
  private notificationQueue: Subject<KHINotification> =
    new Subject<KHINotification>();

  /**
   * Notify the given information as the Notification.
   * It will be shown after user accepted the notification permission.
   * @param notification
   */
  notify(notification: KHINotification) {
    this.notificationQueue.next(notification);
  }

  /**
   * Initialize NotificationManager. This will request the notification permission from user.
   */
  initialize(): void {
    combineLatest([
      from(Notification.requestPermission()).pipe(
        filter((result) => result === 'granted'),
      ),
      this.notificationQueue,
    ]).subscribe(([, queue]) => {
      new Notification(queue.title, {
        body: queue.body,
        icon: 'assets/icons/khi.png',
      });
    });
  }
}
