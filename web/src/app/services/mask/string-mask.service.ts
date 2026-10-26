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

import { Injectable, computed, effect, inject, signal } from '@angular/core';
import { InspectionDataStore } from 'src/app/services/inspection-data-store.service';
import {
  InternPoolStore,
  StringMaskTransform,
} from 'src/app/store/domain/intern-pool-store';

/**
 * Minimal rule definition accepted by the single-pass string masking compiler.
 */
export interface MaskTransformRule {
  /**
   * Exact non-empty substring to match across the inspection dataset.
   */
  readonly target: string;

  /**
   * Replacement text substituted for target. An empty string deletes matching substrings.
   */
  readonly replacement: string;

  /**
   * Whether this masking rule is currently active.
   */
  readonly enabled: boolean;
}

/**
 * Represents a single substring masking rule configured by the user.
 */
export interface MaskRuleEntry extends MaskTransformRule {
  /**
   * Sequential numbered placeholder originally assigned to this rule for identity comparison.
   */
  readonly defaultReplacement: string;
}

/**
 * Formats a 1-based sequential numbered placeholder for default string masking.
 */
export function formatDefaultMaskReplacement(index: number): string {
  const normalized = Math.max(1, Math.trunc(index));
  return `<masked_${normalized}>`;
}

/**
 * Compiles a single-pass longest-match-first string masking transform from active rules.
 */
export function compileStringMaskTransform(
  rules: readonly MaskTransformRule[],
): StringMaskTransform | null {
  const active: MaskTransformRule[] = [];
  const seenTargets = new Set<string>();
  for (const rule of rules) {
    const cleanTarget = rule.target.replace(/\x00/g, '');
    if (
      !rule.enabled ||
      cleanTarget.length === 0 ||
      seenTargets.has(cleanTarget)
    ) {
      continue;
    }
    seenTargets.add(cleanTarget);
    active.push({
      target: cleanTarget,
      replacement: rule.replacement.replace(/\x00/g, ''),
      enabled: true,
    });
  }

  if (active.length === 0) {
    return null;
  }

  active.sort((a, b) => {
    const runeDiff = Array.from(b.target).length - Array.from(a.target).length;
    if (runeDiff !== 0) {
      return runeDiff;
    }
    return b.target.length - a.target.length;
  });

  const replacementByTarget = new Map<string, string>();
  const escapedAlternations: string[] = [];
  for (const rule of active) {
    replacementByTarget.set(rule.target, rule.replacement);
    escapedAlternations.push(
      rule.target.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'),
    );
  }

  const matcher = new RegExp(escapedAlternations.join('|'), 'g');
  return (value: string): string => {
    if (!value) {
      return value;
    }
    return value.replace(
      matcher,
      (matched) => replacementByTarget.get(matched) ?? matched,
    );
  };
}

/**
 * Manages active string masking rules and live preview synchronization with InternPoolStore.
 */
@Injectable({ providedIn: 'root' })
export class StringMaskService {
  private readonly inspectionDataStore = inject(InspectionDataStore);

  private readonly rulesSignal = signal<readonly MaskRuleEntry[]>([]);
  private readonly livePreviewEnabledSignal = signal<boolean>(true);
  private lastBoundPool: InternPoolStore | null = null;
  private nextMaskIndex = 1;

  /**
   * Ordered list of user-configured string masking rules.
   */
  public readonly rules = this.rulesSignal.asReadonly();

  /**
   * Whether active masking rules are previewed live across KHI views.
   */
  public readonly livePreviewEnabled =
    this.livePreviewEnabledSignal.asReadonly();

  /**
   * Subset of rules that are currently enabled and have a non-empty target.
   */
  public readonly enabledRules = computed<readonly MaskRuleEntry[]>(() =>
    this.rulesSignal().filter((r) => r.enabled && r.target.length > 0),
  );

  /**
   * Set of all target strings present in the rule list for fast selection lookups.
   */
  public readonly ruleTargets = computed<ReadonlySet<string>>(
    () => new Set(this.rulesSignal().map((r) => r.target)),
  );

  /**
   * Set of enabled target strings for fast checkbox state lookups.
   */
  public readonly enabledTargets = computed<ReadonlySet<string>>(
    () => new Set(this.enabledRules().map((r) => r.target)),
  );

  /**
   * Compiled transform function reflecting all enabled rules regardless of live preview toggle.
   */
  public readonly compiledTransform = computed<StringMaskTransform | null>(() =>
    compileStringMaskTransform(this.enabledRules()),
  );

  /**
   * Compiled transform function active in live preview mode, or null when live preview is disabled.
   */
  public readonly liveMaskTransform = computed<StringMaskTransform | null>(
    () => (this.livePreviewEnabledSignal() ? this.compiledTransform() : null),
  );

  constructor() {
    effect(() => {
      const pool =
        this.inspectionDataStore.inspectionData()?.internPool ?? null;
      const transform = this.liveMaskTransform();
      if (this.lastBoundPool && this.lastBoundPool !== pool) {
        this.lastBoundPool.setMaskTransform(null);
      }
      this.lastBoundPool = pool;
      if (pool) {
        pool.setMaskTransform(transform);
      }
    });
  }

  /**
   * Applies the active live mask transform to an arbitrary text payload such as struct YAML.
   */
  public applyMask(text: string): string {
    const transform = this.liveMaskTransform();
    if (!transform || !text) {
      return text;
    }
    return transform(text);
  }

  /**
   * Enables or disables live masking preview across KHI views.
   */
  public setLivePreviewEnabled(enabled: boolean): void {
    this.livePreviewEnabledSignal.set(enabled);
  }

  /**
   * Toggles a candidate target string in the mask rule set.
   * Adds a new enabled rule with a sequential `<masked_N>` placeholder when absent, or toggles enabled state when present.
   */
  public toggleTarget(target: string, checked: boolean): void {
    const cleanTarget = target.replace(/\x00/g, '');
    if (cleanTarget.length === 0) {
      return;
    }
    const existing = this.rulesSignal();
    const index = existing.findIndex((r) => r.target === cleanTarget);
    if (index === -1) {
      if (!checked) {
        return;
      }
      const defaultReplacement = this.allocateNextDefaultReplacement(existing);
      this.rulesSignal.set([
        ...existing,
        {
          target: cleanTarget,
          replacement: defaultReplacement,
          defaultReplacement,
          enabled: true,
        },
      ]);
      return;
    }

    const updated = [...existing];
    updated[index] = {
      ...updated[index],
      enabled: checked,
    };
    this.rulesSignal.set(updated);
  }

  /**
   * Bulk-adds or re-enables multiple target strings in a single state update with sequential `<masked_N>` placeholders.
   */
  public addTargets(targets: readonly string[]): void {
    if (targets.length === 0) {
      return;
    }
    const existing = this.rulesSignal();
    const updated: MaskRuleEntry[] = [...existing];
    const indexByTarget = new Map<string, number>();
    for (let i = 0; i < updated.length; i++) {
      indexByTarget.set(updated[i].target, i);
    }

    let changed = false;
    for (const rawTarget of targets) {
      const cleanTarget = rawTarget.replace(/\x00/g, '');
      if (cleanTarget.length === 0) {
        continue;
      }
      const existingIndex = indexByTarget.get(cleanTarget);
      if (existingIndex !== undefined) {
        if (!updated[existingIndex].enabled) {
          updated[existingIndex] = {
            ...updated[existingIndex],
            enabled: true,
          };
          changed = true;
        }
        continue;
      }

      const defaultReplacement = this.allocateNextDefaultReplacement(updated);
      indexByTarget.set(cleanTarget, updated.length);
      updated.push({
        target: cleanTarget,
        replacement: defaultReplacement,
        defaultReplacement,
        enabled: true,
      });
      changed = true;
    }

    if (changed) {
      this.rulesSignal.set(updated);
    }
  }

  /**
   * Adds a custom substring rule or re-enables it if the target already exists.
   */
  public addCustomRule(target: string, replacement?: string): boolean {
    const cleanTarget = target.replace(/\x00/g, '');
    if (cleanTarget.length === 0) {
      return false;
    }
    const existing = this.rulesSignal();
    const index = existing.findIndex((r) => r.target === cleanTarget);
    if (index === -1) {
      const defaultReplacement = this.allocateNextDefaultReplacement(existing);
      const resolvedReplacement =
        replacement !== undefined
          ? replacement.replace(/\x00/g, '')
          : defaultReplacement;
      this.rulesSignal.set([
        ...existing,
        {
          target: cleanTarget,
          replacement: resolvedReplacement,
          defaultReplacement,
          enabled: true,
        },
      ]);
      return true;
    }

    const updated = [...existing];
    updated[index] = {
      ...updated[index],
      replacement:
        replacement !== undefined
          ? replacement.replace(/\x00/g, '')
          : updated[index].replacement,
      enabled: true,
    };
    this.rulesSignal.set(updated);
    return true;
  }

  /**
   * Updates the replacement text of an existing rule. An empty string deletes matching substrings.
   */
  public updateReplacement(target: string, replacement: string): void {
    const cleanReplacement = replacement.replace(/\x00/g, '');
    this.rulesSignal.update((rules) =>
      rules.map((r) =>
        r.target === target ? { ...r, replacement: cleanReplacement } : r,
      ),
    );
  }

  /**
   * Resets a rule's replacement back to its assigned sequential `<masked_N>` placeholder,
   * allocating a fresh placeholder only if another rule manually claimed it.
   */
  public resetReplacement(target: string): void {
    const existing = this.rulesSignal();
    const index = existing.findIndex((r) => r.target === target);
    if (index === -1) {
      return;
    }

    const rule = existing[index];
    const collidesWithOtherRule = existing.some(
      (other, i) =>
        i !== index && other.replacement === rule.defaultReplacement,
    );
    const restoredPlaceholder = collidesWithOtherRule
      ? this.allocateNextDefaultReplacement(existing)
      : rule.defaultReplacement;

    const updated = [...existing];
    updated[index] = {
      ...rule,
      replacement: restoredPlaceholder,
      defaultReplacement: restoredPlaceholder,
    };
    this.rulesSignal.set(updated);
  }

  /**
   * Removes a masking rule by its target string and resets numbering when the rule list becomes empty.
   */
  public removeRule(target: string): void {
    const nextRules = this.rulesSignal().filter((r) => r.target !== target);
    if (nextRules.length === 0) {
      this.nextMaskIndex = 1;
    }
    this.rulesSignal.set(nextRules);
  }

  /**
   * Removes all configured masking rules and resets sequential placeholder numbering to `<masked_1>`.
   */
  public clearAllRules(): void {
    this.nextMaskIndex = 1;
    this.rulesSignal.set([]);
  }

  private allocateNextDefaultReplacement(
    existingRules: readonly MaskRuleEntry[],
  ): string {
    const used = new Set<string>();
    for (const rule of existingRules) {
      used.add(rule.replacement);
      used.add(rule.defaultReplacement);
    }
    while (used.has(formatDefaultMaskReplacement(this.nextMaskIndex))) {
      this.nextMaskIndex++;
    }
    const allocated = formatDefaultMaskReplacement(this.nextMaskIndex);
    this.nextMaskIndex++;
    return allocated;
  }
}
