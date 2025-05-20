/**
 * Copyright 2025 Google LLC
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

import { CommonModule } from '@angular/common';
import { Component, computed, inject, output, signal } from '@angular/core';
import { DiagramModelFrameStore } from '../diagram-model-frame-store';
import { toSignal } from '@angular/core/rxjs-interop';
import { filter, interval, map } from 'rxjs';

/**
 * Component for controlling diagram timeline visualization
 * Provides play/pause functionality, frame navigation, and timeline seeking
 */
@Component({
  selector: 'diagram-control',
  templateUrl: './diagram-control.component.html',
  styleUrls: ['./diagram-control.component.sass'],
  standalone: true,
  imports: [CommonModule],
})
export class DiagramControlComponent {
  /**
   * Service for accessing and controlling diagram frame data
   */
  private readonly diagramModelFrameStore = inject(DiagramModelFrameStore);

  /**
   * Signal containing the current diagram model data
   */
  private readonly diagramModel = toSignal(
    this.diagramModelFrameStore.diagramModel,
  );

  /**
   * Controls the animation playback state
   * True when animation is running, false when paused
   */
  animationPlaying = signal(false);

  /**
   * Current time position in the timeline (Unix timestamp)
   */
  currentTime = toSignal(
    this.diagramModelFrameStore.currentDiagramFrame.pipe(
      map((frame) => frame.ts),
    ),
    { initialValue: 0 },
  );

  /**
   * Time range boundary of the entire frames
   * Contains start and end timestamps for the available data
   */
  totalTimeRange = toSignal(
    this.diagramModelFrameStore.diagramModel.pipe(
      filter((model) => model.frames.length > 0),
      map((model) => ({
        start: model.frames[0].ts,
        end: model.frames[model.frames.length - 1].ts,
      })),
    ),
    {
      initialValue: {
        start: 0,
        end: 0,
      },
    },
  );

  /**
   * Computed field that calculates the current time position as percentage of the timeline
   */
  currentTimePercentage = computed(() => {
    const currentTime = this.currentTime();
    const totalTimeRange = this.totalTimeRange();
    const totalMs = totalTimeRange.end - totalTimeRange.start;
    if (totalMs <= 0) return 0;

    const position = currentTime - totalTimeRange.start;
    return Math.min(100, Math.max(0, (position / totalMs) * 100));
  });

  /**
   * Event emitted when play/pause button is toggled
   * Emits the new playing state (true = playing, false = paused)
   */
  playToggled = output<boolean>();

  /**
   * Maximum number of frames available in the current diagram model
   * Used for animation and frame navigation
   */
  maxFrameCount = computed(() => {
    const model = this.diagramModel();
    if (model) {
      return model.frames.length;
    } else {
      return 0;
    }
  });

  /**
   * Handler for play/pause button click
   * Toggles animation playback and configures the animation interval
   * Updates the animation state in the store and local signal
   */
  onPlayPauseClick(): void {
    const current = this.animationPlaying();
    if (current) {
      this.diagramModelFrameStore.stopAnimation();
    } else {
      this.diagramModelFrameStore.setAnimator(
        interval(100).pipe(
          map((frame) => {
            return frame % this.maxFrameCount();
          }),
        ),
      );
    }
    this.animationPlaying.set(!current);
  }

  /**
   * Handler for previous frame button click
   * Navigates to the previous frame
   */
  onPreviousFrameClick(): void {
    this.diagramModelFrameStore.decrementFrameIndex();
  }

  /**
   * Handler for next frame button click
   * Navigates to the next frame
   */
  onNextFrameClick(): void {
    this.diagramModelFrameStore.incrementFrameIndex();
  }

  /**
   * Handler for seekbar input change
   * Sets the current frame index based on user input from the seekbar
   *
   * @param input - The input event from the seekbar element
   */
  onSeekbarChange(input: Event): void {
    const frameIndex = (input.target as HTMLInputElement).valueAsNumber;
    this.diagramModelFrameStore.setFrameIndex(frameIndex);
  }

  /**
   * Formats a Unix timestamp into a human-readable string
   *
   * @param unixTimeMs - The Unix timestamp in milliseconds
   * @returns A formatted date-time string
   */
  formatTime(unixTimeMs: number): string {
    const date = new Date(unixTimeMs);
    return date.toLocaleString();
  }
}
