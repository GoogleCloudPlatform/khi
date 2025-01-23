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

import { Observable, Subject, filter, map } from 'rxjs';
import {
  KHIWindowPacket,
  WindowConnectionProvider,
} from './window-connector.service';

const KHI_APPLICATION_TOKEN = 'kubernetes-history-inspector';

/**
 * Packet data wrapper for BroadcastChannel
 * To verify if the data is sent from KHI
 */
interface BroadcastChannelPacketWrap {
  packet: KHIWindowPacket<unknown>;
  applicationToken: string;
}

/**
 * WindowConnectionProvider using BroadcastChannel
 */
export class BroadcastChannelWindowConnectionProvider
  implements WindowConnectionProvider
{
  private readonly channel: BroadcastChannel;

  private readonly messageReceiver: Subject<BroadcastChannelPacketWrap> =
    new Subject();

  constructor(channelName = KHI_APPLICATION_TOKEN) {
    this.channel = new BroadcastChannel(channelName);
    this.channel.addEventListener('message', (message) => {
      const data = message.data;
      this.messageReceiver.next(data);
    });
  }

  send(data: KHIWindowPacket<unknown>): void {
    this.channel.postMessage({
      packet: data,
      applicationToken: KHI_APPLICATION_TOKEN,
    } as BroadcastChannelPacketWrap);
  }

  receive(): Observable<KHIWindowPacket<unknown>> {
    return this.messageReceiver.pipe(
      filter(
        (packetWrap) => packetWrap.applicationToken === KHI_APPLICATION_TOKEN,
      ),
      map((packetWrap) => packetWrap.packet),
    );
  }
}

/**
 * WindowConnectionProvider used for tests.
 * This connection provider allows connecting to the other in the same frame
 */
export class InMemoryWindowConnectionProvider
  implements WindowConnectionProvider
{
  private readonly messageReceiver: Subject<KHIWindowPacket<unknown>> =
    new Subject();

  send(data: KHIWindowPacket<unknown>): void {
    setTimeout(() => {
      this.messageReceiver.next(data);
    }, 10);
  }

  receive(): Observable<KHIWindowPacket<unknown>> {
    return this.messageReceiver;
  }
}
