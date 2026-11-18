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

import * as assert from 'node:assert/strict';
import * as fs from 'node:fs';
import * as os from 'node:os';
import * as path from 'node:path';
import { describe, it } from 'node:test';
import {
  CliEnvironment,
  ExportedDeclarationKind,
  OrphanFileViolation,
  ProjectPaths,
  UnusedExportRuleId,
  UnusedExportsReport,
  UnusedExportViolation,
  analyzeUnusedExports,
  buildProjectPaths,
  formatDiagnosticLines,
  normalizePath,
  runCli,
} from 'scripts/check-unused-exports';

/**
 * Writes a temporary fixture project under `<tmp>/web` and runs `fn` with its `ProjectPaths`.
 */
function withFixtureProject(
  files: Record<string, string>,
  fn: (paths: ProjectPaths) => void,
): void {
  const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'khi-unused-ts-'));
  try {
    const defaultFiles: Record<string, string> = {
      'web/package.json': '{"name":"fixture-web"}\n',
      'web/tsconfig.json': JSON.stringify({
        compilerOptions: {
          baseUrl: './',
          target: 'es2022',
          module: 'es2020',
          moduleResolution: 'bundler',
          strict: true,
        },
      }),
      ...files,
    };
    for (const [relPath, content] of Object.entries(defaultFiles)) {
      const absPath = path.join(tmpRoot, relPath);
      fs.mkdirSync(path.dirname(absPath), { recursive: true });
      fs.writeFileSync(absPath, content, 'utf8');
    }
    fn(buildProjectPaths(tmpRoot));
  } finally {
    fs.rmSync(tmpRoot, { recursive: true, force: true });
  }
}

describe('check-unused-exports', () => {
  it('normalizes separators and Windows drive letters consistently', () => {
    assert.equal(
      normalizePath('C:\\Users\\Dev\\..\\project\\web\\src\\main.ts'),
      'c:/Users/project/web/src/main.ts',
    );
    assert.equal(normalizePath('/tmp/project/web'), '/tmp/project/web');

    const paths = buildProjectPaths('/tmp/sample-repo');
    assert.equal(paths.webRoot, `${paths.repoRoot}/web`);
    assert.equal(paths.webSrcRoot, `${paths.repoRoot}/web/src`);
    assert.equal(paths.storybookRoot, `${paths.repoRoot}/web/.storybook`);
    assert.equal(paths.webScriptsRoot, `${paths.repoRoot}/web/scripts`);
  });

  it('detects orphan files, unused exports, and same-file unnecessary exports across declaration kinds', () => {
    withFixtureProject(
      {
        'web/src/main.ts': [
          "import { UsedClass, UsedFn, UsedType, UsedEnum, usedConst, usedTupleItem, reExportedVal } from 'src/app/feature';",
          "const lazy = () => import('./app/lazy-loaded');",
          'type DynamicType = import("./app/type-only").ImportedType;',
          'const instance = new UsedClass();',
          'const check: UsedType = { field: UsedEnum.A };',
          'const checkDyn: DynamicType = { ok: true };',
          'console.log(UsedFn(), usedConst, usedTupleItem, reExportedVal, instance, check, lazy, checkDyn);',
        ].join('\n'),
        'web/src/app/re-export-source.ts':
          'export const reExportedVal = 10;\nexport const unusedReExportVal = 20;\n',
        'web/src/app/feature.ts': [
          "export { reExportedVal, unusedReExportVal } from './re-export-source';",
          'export class UsedClass {}',
          'export class UnusedClass {}',
          'export function UsedFn(): number { return internalHelper(); }',
          'export function internalHelper(): number { return 1; }',
          'export interface UsedType { readonly field: UsedEnum; }',
          'export interface LocalOnlyInterface { readonly id: string; }',
          'export type UnusedAlias = string;',
          "export enum UsedEnum { A = 'a' }",
          "export enum UnusedEnum { B = 'b' }",
          'export const [usedTupleItem, unusedTupleItem] = [1, 2];',
          'export const { usedConst } = { usedConst: usedTupleItem };',
          'const localItem: LocalOnlyInterface = { id: "x" };',
          'console.log(localItem);',
        ].join('\n'),
        'web/src/app/lazy-loaded.ts': 'export const lazyFlag = true;\n',
        'web/src/app/type-only.ts':
          'export interface ImportedType { readonly ok: boolean; }\n',
        'web/src/app/orphan.ts': 'export const neverImported = 42;\n',
        'web/src/app/generated/skip.ts': 'export const generatedUnused = 1;\n',
        'web/src/app/zzz-skip.ts': 'export const zzzUnused = 1;\n',
        'web/src/environments/environment.ts': 'export const envUnused = 1;\n',
        'web/dist/ignored.ts': 'export const ignoredDist = 1;\n',
        'web/.storybook/preview.ts':
          "import { lazyFlag } from '../src/app/lazy-loaded';\nconsole.log(lazyFlag);\n",
      },
      (paths: ProjectPaths) => {
        const report: UnusedExportsReport = analyzeUnusedExports(paths);

        const expectedOrphans: OrphanFileViolation[] = [
          {
            file: 'web/src/app/orphan.ts',
            rule: UnusedExportRuleId.OrphanTsFile,
            message:
              'TypeScript file is never imported by any source, test, or story file',
          },
        ];
        assert.deepEqual(report.orphanFiles, expectedOrphans);
        assert.equal(report.orphanFilesCount, 1);

        const summary = report.unusedExports.map(
          (v: UnusedExportViolation) => ({
            file: v.file,
            name: v.name,
            kind: v.kind,
            rule: v.rule,
            sameFileUses: v.sameFileUses,
          }),
        );
        assert.deepEqual(summary, [
          {
            file: 'web/src/app/feature.ts',
            name: 'unusedReExportVal',
            kind: ExportedDeclarationKind.ReExport,
            rule: UnusedExportRuleId.UnusedExport,
            sameFileUses: 0,
          },
          {
            file: 'web/src/app/feature.ts',
            name: 'UnusedClass',
            kind: ExportedDeclarationKind.Class,
            rule: UnusedExportRuleId.UnusedExport,
            sameFileUses: 0,
          },
          {
            file: 'web/src/app/feature.ts',
            name: 'internalHelper',
            kind: ExportedDeclarationKind.Function,
            rule: UnusedExportRuleId.UnnecessaryExport,
            sameFileUses: 1,
          },
          {
            file: 'web/src/app/feature.ts',
            name: 'LocalOnlyInterface',
            kind: ExportedDeclarationKind.Interface,
            rule: UnusedExportRuleId.UnnecessaryExport,
            sameFileUses: 1,
          },
          {
            file: 'web/src/app/feature.ts',
            name: 'UnusedAlias',
            kind: ExportedDeclarationKind.Type,
            rule: UnusedExportRuleId.UnusedExport,
            sameFileUses: 0,
          },
          {
            file: 'web/src/app/feature.ts',
            name: 'UnusedEnum',
            kind: ExportedDeclarationKind.Enum,
            rule: UnusedExportRuleId.UnusedExport,
            sameFileUses: 0,
          },
          {
            file: 'web/src/app/feature.ts',
            name: 'unusedTupleItem',
            kind: ExportedDeclarationKind.Const,
            rule: UnusedExportRuleId.UnusedExport,
            sameFileUses: 0,
          },
        ]);
        assert.equal(report.unusedExportsCount, 7);
        assert.equal(report.zeroUseExportsCount, 5);
        assert.equal(report.sameFileOnlyExportsCount, 2);

        const lines = formatDiagnosticLines(report);
        assert.equal(lines.length, 8);
        assert.match(
          lines[0],
          /^web\/src\/app\/orphan\.ts:1:1: .* \(orphan-ts-file\)$/,
        );
        assert.match(
          lines[1],
          /^web\/src\/app\/feature\.ts:\d+:\d+: exported re-export 'unusedReExportVal' is never used anywhere \(unused-export\)$/,
        );
        assert.match(
          lines[3],
          /^web\/src\/app\/feature\.ts:\d+:\d+: exported function 'internalHelper' is only used within this file \(remove 'export' modifier\) \(unnecessary-export\)$/,
        );
      },
    );
  });

  it('executes CLI in both diagnostic and JSON modes from repo root or web directory', () => {
    withFixtureProject(
      {
        'web/src/main.ts':
          "import { activeValue } from './used';\nconsole.log(activeValue);\n",
        'web/src/used.ts': 'export const activeValue = 1;\n',
      },
      (paths: ProjectPaths) => {
        const stdoutLines: string[] = [];
        const stderrLines: string[] = [];
        const cleanEnv: CliEnvironment = {
          cwd: paths.webRoot,
          writeStdout: (line: string) => stdoutLines.push(line),
          writeStderr: (line: string) => stderrLines.push(line),
        };
        assert.equal(runCli(['--json'], cleanEnv), 0);
        assert.equal(stderrLines.length, 0);
        const parsed = JSON.parse(
          stdoutLines.join('\n'),
        ) as UnusedExportsReport;
        assert.equal(parsed.orphanFilesCount, 0);
        assert.equal(parsed.unusedExportsCount, 0);

        fs.writeFileSync(
          path.join(paths.webSrcRoot, 'unused-file.ts'),
          'export const dead = true;\n',
          'utf8',
        );
        const failEnv: CliEnvironment = {
          cwd: paths.repoRoot,
          writeStdout: (line: string) => stdoutLines.push(line),
          writeStderr: (line: string) => stderrLines.push(line),
        };
        assert.equal(runCli([], failEnv), 1);
        assert.equal(stderrLines.length, 1);
        assert.match(stderrLines[0], /web\/src\/unused-file\.ts:1:1:/);
      },
    );
  });
});
