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

import { Component, computed, inject } from '@angular/core';
import { StyleOverrideService } from 'src/app/services/style-override.service';
import { StyleOverrideLayoutComponent } from 'src/app/dialogs/style-override/components/style-override-layout.component';
import { HDRColor4 } from 'src/app/store/domain/style';
import {
  RevisionStateStyleOverrideViewModel,
  TimelineTypeStyleOverrideViewModel,
  TimelineTypeOverrideEvent,
  LogTypeStyleOverrideViewModel,
  LogTypeOverrideEvent,
} from 'src/app/dialogs/style-override/types/style-override-viewmodel';

/**
 * Smart component for the style override dialog.
 * Connects the StyleOverrideService color override logic with the presentation layout component.
 */
@Component({
  selector: 'khi-style-override-smart',
  standalone: true,
  imports: [StyleOverrideLayoutComponent],
  templateUrl: './style-override-smart.component.html',
  styleUrls: ['./style-override-smart.component.scss'],
})
export class StyleOverrideSmartComponent {
  private readonly styleOverrideService = inject(StyleOverrideService);

  /** Computes the list of revision state view models with overridden flags and hex colors. */
  protected readonly revisionStateViewModels = computed<
    RevisionStateStyleOverrideViewModel[]
  >(() => {
    // Track signal to trigger computed updates when styles are overridden
    this.styleOverrideService.stylesUpdated();

    return this.styleOverrideService.revisionStates.map((state) => {
      const color = state.backgroundColor;
      const goColorCode = `style.Color{R: ${color.r.toFixed(3)}, G: ${color.g.toFixed(3)}, B: ${color.b.toFixed(3)}, A: ${color.a.toFixed(1)}}`;
      return {
        id: state.id,
        label: state.label,
        icon: state.icon,
        description: state.description,
        hexColor: this.hdrColorToHex(state.backgroundColor),
        isOverridden: this.styleOverrideService.isRevisionStateOverridden(
          state.id,
        ),
        style: state.style,
        goColorCode,
      };
    });
  });

  /** Computes the list of timeline type view models with overridden flags and hex colors. */
  protected readonly timelineTypeViewModels = computed<
    TimelineTypeStyleOverrideViewModel[]
  >(() => {
    // Track signal to trigger computed updates when styles are overridden
    this.styleOverrideService.stylesUpdated();

    return this.styleOverrideService.timelineTypes.map((type) => {
      const bg = type.backgroundColor;
      const fg = type.foregroundColor;
      const chipBg = type.typeChipBackgroundColor;
      const chipFg = type.typeChipForegroundColor;
      const goColorCode = `style.Color{R: ${bg.r.toFixed(3)}, G: ${bg.g.toFixed(3)}, B: ${bg.b.toFixed(3)}, A: ${bg.a.toFixed(1)}}`;
      return {
        id: type.id,
        label: type.label,
        icon: type.icon,
        description: type.description,
        hexColor: this.hdrColorToHex(bg),
        hexForegroundColor: this.hdrColorToHex(fg),
        hexChipBackgroundColor: this.hdrColorToHex(chipBg),
        hexChipForegroundColor: this.hdrColorToHex(chipFg),
        height: type.height,
        isOverridden: this.styleOverrideService.isTimelineTypeOverridden(
          type.id,
        ),
        goColorCode,
      };
    });
  });

  /** Computes the list of log type view models with overridden flags and hex colors. */
  protected readonly logTypeViewModels = computed<
    LogTypeStyleOverrideViewModel[]
  >(() => {
    // Track signal to trigger computed updates when styles are overridden
    this.styleOverrideService.stylesUpdated();

    return this.styleOverrideService.logTypes.map((type) => {
      const bg = type.backgroundColor;
      const fg = type.foregroundColor;
      const goColorCode = `style.Color{R: ${bg.r.toFixed(3)}, G: ${bg.g.toFixed(3)}, B: ${bg.b.toFixed(3)}, A: ${bg.a.toFixed(1)}}`;
      return {
        id: type.id,
        label: type.label,
        description: type.description,
        hexColor: this.hdrColorToHex(bg),
        hexForegroundColor: this.hdrColorToHex(fg),
        isOverridden: this.styleOverrideService.isLogTypeOverridden(type.id),
        goColorCode,
      };
    });
  });

  /**
   * Handles revision state color override emission.
   * @param event The event payload containing state ID and hex color.
   */
  protected onRevisionStateColorChange(event: {
    readonly id: number;
    readonly hexColor: string;
  }): void {
    const originalState = this.styleOverrideService.getRevisionState(event.id);
    this.styleOverrideService.overrideRevisionState({
      ...originalState,
      backgroundColor: this.hexToHDRColor(event.hexColor),
    });
  }

  /**
   * Handles resetting revision state color override emission.
   * @param id The ID of the revision state to reset.
   */
  protected onRevisionStateResetColor(id: number): void {
    this.styleOverrideService.resetRevisionState(id);
  }

  /**
   * Handles timeline type property override emission.
   * @param event The event payload containing timeline type ID and modified style properties.
   */
  protected onTimelineTypePropertyChange(
    event: TimelineTypeOverrideEvent,
  ): void {
    const originalType = this.styleOverrideService.getTimelineType(event.id);
    const updatedType = {
      ...originalType,
    };
    if (event.backgroundColor !== undefined) {
      updatedType.backgroundColor = this.hexToHDRColor(event.backgroundColor);
    }
    if (event.foregroundColor !== undefined) {
      updatedType.foregroundColor = this.hexToHDRColor(event.foregroundColor);
    }
    if (event.typeChipBackgroundColor !== undefined) {
      updatedType.typeChipBackgroundColor = this.hexToHDRColor(
        event.typeChipBackgroundColor,
      );
    }
    if (event.typeChipForegroundColor !== undefined) {
      updatedType.typeChipForegroundColor = this.hexToHDRColor(
        event.typeChipForegroundColor,
      );
    }
    if (event.height !== undefined) {
      updatedType.height = event.height;
    }
    this.styleOverrideService.overrideTimelineType(updatedType);
  }

  /**
   * Handles resetting timeline type color override emission.
   * @param id The ID of the timeline type to reset.
   */
  protected onTimelineTypeResetColor(id: number): void {
    this.styleOverrideService.resetTimelineType(id);
  }

  /**
   * Handles log type property override emission.
   * @param event The event payload containing log type ID and modified style properties.
   */
  protected onLogTypePropertyChange(event: LogTypeOverrideEvent): void {
    const originalType = this.styleOverrideService.getLogType(event.id);
    const updatedType = {
      ...originalType,
    };
    if (event.backgroundColor !== undefined) {
      updatedType.backgroundColor = this.hexToHDRColor(event.backgroundColor);
    }
    if (event.foregroundColor !== undefined) {
      updatedType.foregroundColor = this.hexToHDRColor(event.foregroundColor);
    }
    this.styleOverrideService.overrideLogType(updatedType);
  }

  /**
   * Handles resetting log type color override emission.
   * @param id The ID of the log type to reset.
   */
  protected onLogTypeResetColor(id: number): void {
    this.styleOverrideService.resetLogType(id);
  }

  private hdrColorToHex(color: HDRColor4): string {
    const r = Math.round(color.r * 255)
      .toString(16)
      .padStart(2, '0');
    const g = Math.round(color.g * 255)
      .toString(16)
      .padStart(2, '0');
    const b = Math.round(color.b * 255)
      .toString(16)
      .padStart(2, '0');
    return `#${r}${g}${b}`;
  }

  private hexToHDRColor(hex: string): HDRColor4 {
    const r = parseInt(hex.substring(1, 3), 16) / 255;
    const g = parseInt(hex.substring(3, 5), 16) / 255;
    const b = parseInt(hex.substring(5, 7), 16) / 255;
    return { r, g, b, a: 1.0 };
  }
}
