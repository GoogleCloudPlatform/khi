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

import { TestBed } from '@angular/core/testing';
import { PopupManagerImpl } from 'src/app/services/popup/popup-manager-impl';
import { ConnectClientService } from 'src/app/services/api/connect-client.service';
import { create } from '@bufbuild/protobuf';
import {
  PopupFormSchema,
  WatchPopupResponse,
  WatchPopupResponseSchema,
} from 'src/app/generated/api/v1/popup_pb';

describe('PopupManagerImpl', () => {
  let connectClientSpy: jasmine.SpyObj<ConnectClientService>;
  let manager: PopupManagerImpl;

  beforeEach(() => {
    async function* createStream(): AsyncGenerator<
      WatchPopupResponse,
      void,
      unknown
    > {
      yield create(WatchPopupResponseSchema, {
        event: {
          case: 'popup',
          value: create(PopupFormSchema, {
            id: 'popup-1',
            title: 'Test Popup',
            description: 'Test Desc',
            payload: {
              case: 'text',
              value: { placeholder: 'Placeholder' },
            },
          }),
        },
      });
      // Keep stream alive to avoid tight loop reconnection during test
      await new Promise((resolve) => setTimeout(resolve, 5000));
    }

    connectClientSpy = jasmine.createSpyObj('ConnectClientService', [], {
      popupClient: {
        watchPopup: jasmine
          .createSpy('watchPopup')
          .and.callFake(() => createStream()),
        validatePopupAnswer: jasmine.createSpy('validatePopupAnswer'),
        submitPopupAnswer: jasmine.createSpy('submitPopupAnswer'),
      },
    });

    TestBed.configureTestingModule({
      providers: [
        PopupManagerImpl,
        { provide: ConnectClientService, useValue: connectClientSpy },
      ],
    });

    manager = TestBed.inject(PopupManagerImpl);
  });

  afterEach(() => {
    manager.ngOnDestroy();
  });

  it('should receive popup event and update currentPopup signal', async () => {
    // Wait for the async generator in startWatchLoop to process
    await new Promise((resolve) => setTimeout(resolve, 50));

    const popup = manager.currentPopup();
    expect(popup).not.toBeNull();
    expect(popup?.form.id).toBe('popup-1');
    expect(popup?.form.title).toBe('Test Popup');
    expect(popup?.client).toBe(connectClientSpy.popupClient);
  });
});
