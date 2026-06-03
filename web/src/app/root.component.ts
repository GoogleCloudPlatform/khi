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
import { Component, inject, OnDestroy } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { BACKEND_SYNC } from 'src/app/services/api/backend-sync.service';
import { BackendConnectionStatus } from 'src/app/services/api/backend-sync-interface';

@Component({
  selector: 'khi-root',
  templateUrl: './root.component.html',
  styleUrls: ['./root.component.scss'],
  imports: [RouterOutlet],
})
export class RootComponent implements OnDestroy {
  private readonly backendSync = inject(BACKEND_SYNC);

  private readonly beforeUnloadHandler = (event: BeforeUnloadEvent) => {
    if (
      this.backendSync.connectionStatus() !==
      BackendConnectionStatus.Disconnected
    ) {
      return;
    }

    event.preventDefault();
    event.returnValue = '';
  };

  constructor() {
    window.addEventListener('beforeunload', this.beforeUnloadHandler);
  }

  ngOnDestroy() {
    window.removeEventListener('beforeunload', this.beforeUnloadHandler);
  }
}
