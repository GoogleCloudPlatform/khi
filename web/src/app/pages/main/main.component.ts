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

import {
  Component,
  inject,
  OnDestroy,
  OnInit,
  AfterViewInit,
  viewChild,
  ElementRef,
  ViewContainerRef,
} from '@angular/core';
import { LayoutService } from 'src/app/services/layout/layout.service';
import { Subject, takeUntil } from 'rxjs';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import {
  POPUP_MANAGER,
  PopupManager,
} from 'src/app/services/popup/popup-manager';
import { NotificationManager } from 'src/app/services/notification/notification';
import { DiffPageDataSourceServer } from 'src/app/services/frame-connection/frames/diff-page-datasource-server.service';
import { GraphPageDataSourceServer } from 'src/app/services/frame-connection/frames/graph-page-datasource-server.service';
import {
  EXTENSION_STORE,
  ExtensionStore,
} from 'src/app/extensions/extension-common/extension-store';
import { CommonModule } from '@angular/common';
import { SidePaneComponent } from 'src/app/common/components/side-pane.component';
import { LogSmartComponent } from 'src/app/log/log-smart.component';
import { DiffSmartComponent } from 'src/app/diff/diff-smart.component';
import { MatIconModule } from '@angular/material/icon';
import { HeaderComponent } from 'src/app/header/header.component';
import { TimelineSmartComponent } from 'src/app/timeline/timeline-smart.component';
import { StartupDialogComponent } from 'src/app/dialogs/startup/startup.component';
import {
  RequestUserActionPopupComponent,
  RequestUserActionPopupRequest,
} from 'src/app/dialogs/request-user-action-popup/request-user-action-popup.component';
import { NilPopupFormRequest } from 'src/app/services/popup/popup-manager-impl';
import { KHIIconRegistrationModule } from 'src/app/shared/module/icon-registration.module';

/**
 * AppComponent serves as the main container for the application layout.
 * It initializes GoldenLayout and manages top-level dialogs and notifications.
 */
@Component({
  templateUrl: './main.component.html',
  styleUrls: ['./main.component.scss'],
  imports: [
    CommonModule,
    HeaderComponent,
    SidePaneComponent,
    LogSmartComponent,
    DiffSmartComponent,
    MatIconModule,
    KHIIconRegistrationModule,
    TimelineSmartComponent,
  ],
  providers: [LayoutService],
})
export class AppComponent implements OnInit, OnDestroy, AfterViewInit {
  /** Store for extension data. */
  private readonly extensionStore = inject<ExtensionStore>(EXTENSION_STORE);

  /** Dialog service. */
  private readonly dialog = inject(MatDialog);

  /** ViewContainerRef for creating components dynamically. */
  private readonly viewContainerRef = inject(ViewContainerRef);

  /** Service for managing GoldenLayout. */
  private readonly layoutService = inject(LayoutService);

  /** Container element for GoldenLayout. */
  readonly layoutContainer = viewChild<ElementRef>('layoutContainer');

  /** Subject for cleaning up subscriptions. */
  private readonly destroyed = new Subject<void>();

  /** Popup manager service. */
  private readonly popupManager: PopupManager = inject(POPUP_MANAGER);

  /** Data source server for diff page. */
  private readonly diffPageSourceSender: DiffPageDataSourceServer = inject(
    DiffPageDataSourceServer,
  );

  /** Data source server for graph page. */
  private readonly graphPageSourceSender: GraphPageDataSourceServer = inject(
    GraphPageDataSourceServer,
  );

  /** Notification manager service. */
  private readonly notificationManager: NotificationManager =
    inject(NotificationManager);

  /**
   * Initializes the component.
   * Checks for data in URL, opens startup dialog if needed, and starts monitoring popup requests.
   */
  ngOnInit() {
    if (!this.extensionStore.tryOpenDataFromURL()) {
      this.dialog.open(StartupDialogComponent, {
        maxWidth: '100vw',
        panelClass: 'startup-modalbox',
        disableClose: true,
      });
    }
    // Start monitoring popup request from server
    let lastDialogRef: MatDialogRef<RequestUserActionPopupComponent> | null =
      null;
    this.popupManager
      .requests()
      .pipe(takeUntil(this.destroyed))
      .subscribe((formRequest) => {
        // The last opened dialog will be closed automatically When the popup was cancelled from server side,
        if (formRequest.id === NilPopupFormRequest.id) {
          lastDialogRef?.close();
          lastDialogRef = null;
          return;
        }
        lastDialogRef = this.dialog.open<
          RequestUserActionPopupComponent,
          RequestUserActionPopupRequest
        >(RequestUserActionPopupComponent, {
          data: {
            formRequest,
          },
        });
        this.notificationManager.notify({
          title: 'KHI requests additional parameter',
          body: `Please supply ${formRequest.title} to proceed tasks`,
        });
      });
    this.diffPageSourceSender.activate();
    this.graphPageSourceSender.activate();
  }

  /**
   * Initializes GoldenLayout after the view is initialized.
   */
  ngAfterViewInit() {
    const container = this.layoutContainer()?.nativeElement;
    if (container) {
      this.layoutService.init(container, this.viewContainerRef);
      this.layoutService.loadDefaultLayout();
    }
  }

  /**
   * Cleans up subscriptions when the component is destroyed.
   */
  ngOnDestroy(): void {
    this.destroyed.next();
  }
}
