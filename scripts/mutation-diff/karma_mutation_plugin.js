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

/**
 * Karma framework plugin that orchestrates single-build multi-mutant evaluation for KHI TypeScript mutation testing.
 *
 * Reads configuration from the JSON file pointed to by process.env.KHI_MUTATION_CONFIG_FILE:
 *   {
 *     "candidateIds": [1, 2, ...],
 *     "fileByMutantId": { "1": "web/src/app/foo.ts", ... },
 *     "maxMutants": 80,
 *     "minPerMutantTimeoutMs": 3000,
 *     "overallTimeoutMs": 180000,
 *     "resultsPath": "/tmp/.../mutation-results.json"
 *   }
 */

'use strict';

const fs = require('fs');

function createKhiMutationFramework(emitter, executor, config, capturedBrowsers, launcher) {
  // Force multi-run lifecycle under plugin control even when invoked with ng test --watch=false.
  config.singleRun = false;
  config.autoWatch = false;

  const cfgPath = process.env.KHI_MUTATION_CONFIG_FILE;
  if (!cfgPath) {
    throw new Error('KHI_MUTATION_CONFIG_FILE environment variable is required.');
  }

  const runCfg = JSON.parse(fs.readFileSync(cfgPath, 'utf8'));
  const candidateIds = Array.isArray(runCfg.candidateIds) ? runCfg.candidateIds : [];
  const fileByMutantId = runCfg.fileByMutantId || {};
  const maxMutants = typeof runCfg.maxMutants === 'number' ? runCfg.maxMutants : 80;
  const minTimeoutMs =
    typeof runCfg.minPerMutantTimeoutMs === 'number' ? runCfg.minPerMutantTimeoutMs : 3000;
  const overallTimeoutMs =
    typeof runCfg.overallTimeoutMs === 'number' ? runCfg.overallTimeoutMs : 180000;
  const resultsPath = runCfg.resultsPath;

  const startTimeMs = Date.now();
  let phase = 'idle'; // 'baseline' | 'mutants' | 'restarting' | 'done'
  let baselineReport = null;
  let baselineStartMs = 0;
  let perMutantTimeoutMs = minTimeoutMs;
  let queue = [];
  let queueIdx = 0;
  let activeMutantId = 0;
  let activeRanSpecIds = [];
  let activeFailedSpecIds = [];
  let watchdogTimer = null;

  const outcomes = {}; // mutantId -> 'KILLED' | 'SURVIVED' | 'TIMED_OUT' | 'UNCOVERED' | 'SKIPPED_BUDGET'
  const mutantRuns = {}; // mutantId -> { ranSpecIds: string[], failedSpecIds: string[] }

  function clearWatchdog() {
    if (watchdogTimer !== null) {
      clearTimeout(watchdogTimer);
      watchdogTimer = null;
    }
  }

  function finishRun(baselineFailed, errorMessage) {
    if (phase === 'done') {
      return;
    }
    phase = 'done';
    clearWatchdog();

    const payload = {
      baselineFailed: Boolean(baselineFailed),
      errorMessage: errorMessage || '',
      outcomes,
      specOrder: baselineReport && Array.isArray(baselineReport.specOrder) ? baselineReport.specOrder : [],
      specPolarities:
        baselineReport && baselineReport.specPolarities ? baselineReport.specPolarities : {},
      mutantRuns,
    };
    fs.writeFileSync(resultsPath, JSON.stringify(payload, null, 2), 'utf8');

    const exitCode = baselineFailed ? 1 : 0;
    setImmediate(() => {
      if (typeof emitter._close === 'function') {
        emitter._close(exitCode);
      } else {
        emitter.emit('stop', () => process.exit(exitCode));
      }
    });
  }

  emitter.on('browser_info', (_browser, info) => {
    if (info && info.khiMutantReport && phase === 'baseline') {
      baselineReport = info.khiMutantReport;
    }
  });

  emitter.on('spec_complete', (_browser, specResult) => {
    if (phase !== 'mutants' || activeMutantId <= 0 || !specResult || specResult.skipped) {
      return;
    }
    if (specResult.id) {
      activeRanSpecIds.push(specResult.id);
      if (!specResult.success) {
        activeFailedSpecIds.push(specResult.id);
      }
    }
  });

  emitter.on('browsers_ready', () => {
    if (phase === 'idle') {
      phase = 'baseline';
      baselineStartMs = Date.now();
      config.client = config.client || {};
      config.client.args = ['--khi-mutant=0'];
      executor.schedule();
      return;
    }
    if (phase === 'restarting') {
      phase = 'mutants';
      scheduleNextMutant();
    }
  });

  emitter.on('run_complete', (_browsers, results) => {
    if (phase === 'done' || phase === 'restarting') {
      return;
    }
    clearWatchdog();

    if (phase === 'baseline') {
      const baselineDurationMs = Math.max(1, Date.now() - baselineStartMs);
      if (!results || results.failed > 0 || results.error || results.disconnected) {
        finishRun(true, 'Baseline Karma unit tests failed before mutation runs.');
        return;
      }

      perMutantTimeoutMs = Math.max(minTimeoutMs, baselineDurationMs * 5);
      buildMutantQueue(baselineReport || { specHits: {}, topLevelHits: [] });
      phase = 'mutants';
      scheduleNextMutant();
      return;
    }

    if (phase === 'mutants') {
      const killed = Boolean(
        !results ||
          results.failed > 0 ||
          results.error ||
          results.disconnected ||
          results.exitCode !== 0,
      );
      if (killed && activeFailedSpecIds.length === 0 && activeRanSpecIds.length === 1) {
        activeFailedSpecIds.push(activeRanSpecIds[0]);
      }
      mutantRuns[String(activeMutantId)] = {
        ranSpecIds: activeRanSpecIds,
        failedSpecIds: activeFailedSpecIds,
      };
      outcomes[String(activeMutantId)] = killed ? 'KILLED' : 'SURVIVED';
      scheduleNextMutant();
    }
  });

  function buildMutantQueue(report) {
    const specsByMutant = new Map();
    const specHits = report.specHits || {};
    for (const [specName, ids] of Object.entries(specHits)) {
      if (!Array.isArray(ids)) {
        continue;
      }
      for (const id of ids) {
        let list = specsByMutant.get(id);
        if (!list) {
          list = [];
          specsByMutant.set(id, list);
        }
        list.push(specName);
      }
    }
    const topLevelSet = new Set(Array.isArray(report.topLevelHits) ? report.topLevelHits : []);

    const reachedItems = [];
    for (const id of candidateIds) {
      const hitSpecs = specsByMutant.get(id) || [];
      const isTopLevel = topLevelSet.has(id);
      if (hitSpecs.length === 0 && !isTopLevel) {
        outcomes[String(id)] = 'UNCOVERED';
        continue;
      }
      reachedItems.push({
        id,
        file: fileByMutantId[String(id)] || '',
        specs: isTopLevel ? null : hitSpecs,
      });
    }

    const budgeted = selectRoundRobinBudget(reachedItems, maxMutants);
    const selectedIds = new Set(budgeted.map((item) => item.id));
    for (const item of reachedItems) {
      if (!selectedIds.has(item.id)) {
        outcomes[String(item.id)] = 'SKIPPED_BUDGET';
      }
    }
    queue = budgeted;
    queueIdx = 0;
  }

  function scheduleNextMutant() {
    if (phase !== 'mutants') {
      return;
    }
    if (queueIdx >= queue.length) {
      finishRun(false, '');
      return;
    }

    if (overallTimeoutMs > 0 && Date.now() - startTimeMs >= overallTimeoutMs) {
      while (queueIdx < queue.length) {
        const skipped = queue[queueIdx++];
        outcomes[String(skipped.id)] = 'SKIPPED_BUDGET';
      }
      finishRun(false, '');
      return;
    }

    const item = queue[queueIdx++];
    activeMutantId = item.id;
    activeRanSpecIds = [];
    activeFailedSpecIds = [];

    const clientArgs = [`--khi-mutant=${item.id}`];
    if (Array.isArray(item.specs) && item.specs.length > 0 && item.specs.length <= 50) {
      const pattern = item.specs.map(escapeRegex).join('|');
      clientArgs.push(`--grep=/^(?:${pattern})$/`);
    }
    config.client = config.client || {};
    config.client.args = clientArgs;

    watchdogTimer = setTimeout(() => {
      handleMutantTimeout(item.id);
    }, perMutantTimeoutMs);

    executor.schedule();
  }

  function handleMutantTimeout(timedOutId) {
    if (phase !== 'mutants') {
      return;
    }
    watchdogTimer = null;
    outcomes[String(timedOutId)] = 'TIMED_OUT';

    if (queueIdx >= queue.length) {
      finishRun(false, '');
      return;
    }

    phase = 'restarting';
    const browsers = [];
    capturedBrowsers.forEach((b) => browsers.push(b));
    if (browsers.length === 0) {
      finishRun(false, '');
      return;
    }

    for (const b of browsers) {
      b.remove();
      if (!launcher.restart(b.id)) {
        finishRun(false, '');
        return;
      }
    }
  }
}

createKhiMutationFramework.$inject = [
  'emitter',
  'executor',
  'config',
  'capturedBrowsers',
  'launcher',
];

function selectRoundRobinBudget(items, limit) {
  if (limit <= 0 || items.length <= limit) {
    return items;
  }
  const byFile = new Map();
  const fileOrder = [];
  for (const item of items) {
    if (!byFile.has(item.file)) {
      byFile.set(item.file, []);
      fileOrder.push(item.file);
    }
    byFile.get(item.file).push(item);
  }
  fileOrder.sort();

  const selectedIds = new Set();
  while (selectedIds.size < limit) {
    let progressed = false;
    for (const f of fileOrder) {
      const bucket = byFile.get(f);
      if (bucket.length === 0) {
        continue;
      }
      const next = bucket.shift();
      selectedIds.add(next.id);
      progressed = true;
      if (selectedIds.size >= limit) {
        break;
      }
    }
    if (!progressed) {
      break;
    }
  }
  return items.filter((item) => selectedIds.has(item.id));
}

function escapeRegex(str) {
  return str.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&');
}

module.exports = {
  'framework:khi-mutation': ['factory', createKhiMutationFramework],
};
