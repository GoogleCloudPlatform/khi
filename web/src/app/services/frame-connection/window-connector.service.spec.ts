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

import { InMemoryWindowConnectionProvider } from './window-connection-provider.service';
import {
  KHIWindowPacket,
  WindowConnectorService,
} from './window-connector.service';

function waitFor(msec: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve();
    }, msec);
  });
}

describe('WindowConnectorService', () => {
  it('should route broadcasted packet to all frames', async () => {
    const connectionProvider = new InMemoryWindowConnectionProvider();
    const windowConnector1 = new WindowConnectorService(connectionProvider);
    expect(await windowConnector1.createSession(1)).toBe(true);
    const windowConnector2 = new WindowConnectorService(connectionProvider);
    expect(await windowConnector2.joinSession(1, 'Diagram')).toBe(true);
    const windowConnector3 = new WindowConnectorService(connectionProvider);
    expect(await windowConnector3.joinSession(1, 'Diagram')).toBe(true);
    const connector1Packets: KHIWindowPacket<unknown>[] = [];
    const connector2Packets: KHIWindowPacket<unknown>[] = [];
    const connector3Packets: KHIWindowPacket<unknown>[] = [];
    windowConnector1
      .receiver('test')
      .subscribe((packet) => connector1Packets.push(packet));
    windowConnector2
      .receiver('test')
      .subscribe((packet) => connector2Packets.push(packet));
    windowConnector3
      .receiver('test')
      .subscribe((packet) => connector3Packets.push(packet));

    windowConnector1.broadcast('test', 'bar');
    await waitFor(100);

    expect(connector1Packets.length).toBe(0);
    expect(connector2Packets.length).toBe(1);
    expect(connector2Packets[0].data).toBe('bar');
    expect(connector3Packets.length).toBe(1);
    expect(connector3Packets[0].data).toBe('bar');
  });

  it('should route unicasted packet to a destination', async () => {
    const connectionProvider = new InMemoryWindowConnectionProvider();
    const windowConnector1 = new WindowConnectorService(connectionProvider);
    await windowConnector1.createSession(1);
    const windowConnector2 = new WindowConnectorService(connectionProvider);
    await windowConnector2.createSession(1);
    const windowConnector3 = new WindowConnectorService(connectionProvider);
    await windowConnector3.createSession(1);
    const connector1Packets: KHIWindowPacket<unknown>[] = [];
    const connector2Packets: KHIWindowPacket<unknown>[] = [];
    const connector3Packets: KHIWindowPacket<unknown>[] = [];
    windowConnector1
      .receiver('test')
      .subscribe((packet) => connector1Packets.push(packet));
    windowConnector2
      .receiver('test')
      .subscribe((packet) => connector2Packets.push(packet));
    windowConnector3
      .receiver('test')
      .subscribe((packet) => connector3Packets.push(packet));

    windowConnector1.unicast('test', 'bar', windowConnector2.frameId);
    await waitFor(100);

    expect(connector1Packets.length).toBe(0);
    expect(connector2Packets.length).toBe(1);
    expect(connector2Packets[0].data).toBe('bar');
    expect(connector3Packets.length).toBe(0);
  });

  it('should ignore packet sent in the another session', async () => {
    const connectionProvider = new InMemoryWindowConnectionProvider();
    const windowConnector1 = new WindowConnectorService(connectionProvider);
    await windowConnector1.createSession(1);
    const windowConnector2 = new WindowConnectorService(connectionProvider);
    await windowConnector2.createSession(1);
    const connector1Packets: KHIWindowPacket<unknown>[] = [];
    const connector2Packets: KHIWindowPacket<unknown>[] = [];
    windowConnector1
      .receiver('test')
      .subscribe((packet) => connector1Packets.push(packet));
    windowConnector2
      .receiver('test')
      .subscribe((packet) => connector2Packets.push(packet));

    windowConnector1.broadcast('test', 'bar');
    await waitFor(100);

    expect(connector1Packets.length).toBe(0);
    expect(connector2Packets.length).toBe(0);
  });
});
