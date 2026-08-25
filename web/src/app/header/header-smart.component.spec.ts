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
import { HeaderSmartComponent } from './header-smart.component';
import { MenuManager } from '../services/menu/menu-manager.service';
import { BACKEND_SYNC } from '../services/api/backend-sync.service';
import {
  BackendConnectionStatus,
  BackendSyncService,
} from '../services/api/backend-sync-interface';
import { WindowConnectorService } from '../services/frame-connection/window-connector.service';
import { signal } from '@angular/core';
import { Subject } from 'rxjs';
import { ServerStat } from 'src/app/generated/api/v1/server_status_pb';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('HeaderSmartComponent', () => {
  let component: HeaderSmartComponent;
  let fixture: ComponentFixture<HeaderSmartComponent>;

  const mockConnectionStatus = signal(BackendConnectionStatus.Connected);
  const mockServerStat = signal<ServerStat | null>(null);
  const mockBackendSync: Partial<BackendSyncService> = {
    connectionStatus: mockConnectionStatus,
    serverStat: mockServerStat,
  };

  const sessionSubject = new Subject<void>();
  const mockWindowConnector = {
    sessionEstablished: sessionSubject.asObservable(),
    sessionId: 'session-123',
  };

  const mockMenuManager = {
    groups: signal([]),
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [HeaderSmartComponent, NoopAnimationsModule],
      providers: [
        { provide: MenuManager, useValue: mockMenuManager },
        { provide: BACKEND_SYNC, useValue: mockBackendSync },
        { provide: WindowConnectorService, useValue: mockWindowConnector },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(HeaderSmartComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should compute serverMemory and serverCpu from proto serverStat', () => {
    const protoStat = {
      currentMemoryUsage: BigInt(2 * 1024 * 1024 * 1024), // 2 GB
      totalMemory: BigInt(8 * 1024 * 1024 * 1024), // 8 GB
      cpuUsagePercentage: 35.5,
    } as ServerStat;

    mockServerStat.set(protoStat);
    fixture.detectChanges();

    expect(component['serverMemory']()).toBe('2.00');
    expect(component['serverMaxMemory']()).toBe('8.00');
    expect(component['serverCpu']()).toBe('35.5%');
  });
});
