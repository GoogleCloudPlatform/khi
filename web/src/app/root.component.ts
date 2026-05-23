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
import { Component, effect, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { BACKEND_SYNC } from './services/api/backend-sync.service';
import { BackendConnectionStatus } from './services/api/backend-sync-interface';

@Component({
  selector: 'khi-root',
  templateUrl: './root.component.html',
  styleUrls: ['./root.component.scss'],
  imports: [RouterOutlet],
})
export class RootComponent {
  private readonly backendSync = inject(BACKEND_SYNC);

  private readonly beforeUnloadHandler = (event: BeforeUnloadEvent) => {
    event.preventDefault();
    event.returnValue = '';
  };

  constructor() {
    effect((onCleanup) => {
      if (
        this.backendSync.connectionStatus() !==
        BackendConnectionStatus.Disconnected
      ) {
        return;
      }

      window.addEventListener('beforeunload', this.beforeUnloadHandler);
      onCleanup(() => {
        window.removeEventListener('beforeunload', this.beforeUnloadHandler);
      });
    });
  }
}
