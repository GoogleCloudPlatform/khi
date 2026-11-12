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

'use strict';

const fs = require('node:fs');
const path = require('node:path');
const { createRequire } = require('node:module');

/**
 * Generates zero-hit sourcemap-remapped Istanbul FileCoverage JSON for TypeScript files
 * that were not imported by any executed Karma spec bundle.
 */
async function main() {
  const repoRootArg = process.argv[2];
  const files = process.argv.slice(3);
  if (!repoRootArg || files.length === 0) {
    process.stdout.write('{}\n');
    return;
  }

  const repoRoot = path.resolve(repoRootArg);
  const webPkgPath = path.join(repoRoot, 'web', 'package.json');
  const webRequire = createRequire(webPkgPath);

  const ts = webRequire('typescript');
  const { createInstrumenter } = webRequire('istanbul-lib-instrument');
  const libCoverage = webRequire('istanbul-lib-coverage');
  const libSourceMaps = webRequire('istanbul-lib-source-maps');

  const rawCoverageMap = libCoverage.createCoverageMap({});

  for (const relOrAbs of files) {
    const absPath = path.isAbsolute(relOrAbs)
      ? relOrAbs
      : path.join(repoRoot, relOrAbs);
    if (!fs.existsSync(absPath)) {
      continue;
    }

    const sourceText = fs.readFileSync(absPath, 'utf8');
    const transpiled = ts.transpileModule(sourceText, {
      fileName: absPath,
      compilerOptions: {
        target: ts.ScriptTarget.ES2022,
        module: ts.ModuleKind.ES2020,
        experimentalDecorators: true,
        useDefineForClassFields: false,
        sourceMap: true,
        inlineSources: true,
      },
    });

    const inputSourceMap = transpiled.sourceMapText
      ? JSON.parse(transpiled.sourceMapText)
      : undefined;

    const instrumenter = createInstrumenter({
      esModules: true,
      produceSourceMap: true,
    });
    instrumenter.instrumentSync(transpiled.outputText, absPath, inputSourceMap);

    const fc = instrumenter.lastFileCoverage();
    if (fc) {
      if (inputSourceMap && !fc.inputSourceMap) {
        fc.inputSourceMap = inputSourceMap;
      }
      rawCoverageMap.addFileCoverage(fc);
    }
  }

  const mapStore = libSourceMaps.createSourceMapStore();
  const remapped = await mapStore.transformCoverage(rawCoverageMap);
  process.stdout.write(`${JSON.stringify(remapped.toJSON())}\n`);
}

main().catch((err) => {
  process.stderr.write(`${err && err.stack ? err.stack : String(err)}\n`);
  process.exit(1);
});
