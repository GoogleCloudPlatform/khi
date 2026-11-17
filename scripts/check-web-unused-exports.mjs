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

import fs from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(SCRIPT_DIR, '..');
const WEB_ROOT = path.join(REPO_ROOT, 'web');
const WEB_SRC_ROOT = path.join(WEB_ROOT, 'src');
const STORYBOOK_ROOT = path.join(WEB_ROOT, '.storybook');

const require = createRequire(path.join(WEB_ROOT, 'package.json'));
const ts = require('typescript');

/**
 * Recursively collects all TypeScript source files under a directory.
 */
function collectTsFiles(dir, out = []) {
  if (!fs.existsSync(dir)) {
    return out;
  }
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (
        entry.name === 'node_modules' ||
        entry.name === '.angular' ||
        entry.name === 'dist'
      ) {
        continue;
      }
      collectTsFiles(full, out);
    } else if (entry.isFile() && entry.name.endsWith('.ts')) {
      out.push(full);
    }
  }
  return out;
}

function isGeneratedFile(fileName) {
  const normalized = fileName.replaceAll('\\', '/');
  return (
    normalized.includes('/src/app/generated/') ||
    path.basename(normalized).startsWith('zzz-') ||
    normalized.includes('/src/environments/version.')
  );
}

function isTestOrStoryFile(fileName) {
  const normalized = fileName.replaceAll('\\', '/');
  return (
    normalized.endsWith('.spec.ts') ||
    normalized.endsWith('.stories.ts') ||
    normalized.endsWith('/src/test.ts') ||
    normalized.includes('/.storybook/')
  );
}

function isEntrypointFile(fileName) {
  const normalized = fileName.replaceAll('\\', '/');
  return (
    normalized.endsWith('/src/main.ts') ||
    normalized.endsWith('/src/polyfills.ts') ||
    normalized.endsWith('.d.ts') ||
    normalized.includes('/src/environments/')
  );
}

function hasExportModifier(node) {
  return (
    ts.canHaveModifiers(node) &&
    Boolean(
      ts
        .getModifiers(node)
        ?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword),
    )
  );
}

function getKindLabel(node) {
  if (ts.isFunctionDeclaration(node)) return 'function';
  if (ts.isClassDeclaration(node)) return 'class';
  if (ts.isInterfaceDeclaration(node)) return 'interface';
  if (ts.isTypeAliasDeclaration(node)) return 'type';
  if (ts.isEnumDeclaration(node)) return 'enum';
  if (ts.isVariableDeclaration(node)) return 'const';
  if (ts.isExportSpecifier(node)) return 're-export';
  return 'declaration';
}

function collectBindingIdentifiers(nameNode, out = []) {
  if (!nameNode) return out;
  if (ts.isIdentifier(nameNode)) {
    out.push(nameNode);
    return out;
  }
  if (
    ts.isObjectBindingPattern(nameNode) ||
    ts.isArrayBindingPattern(nameNode)
  ) {
    for (const elem of nameNode.elements) {
      if (ts.isBindingElement(elem)) {
        collectBindingIdentifiers(elem.name, out);
      }
    }
  }
  return out;
}

/**
 * Collects all module specifiers imported statically, re-exported, or dynamically imported in a SourceFile.
 */
function collectModuleSpecifiers(sf) {
  const specifiers = [];
  function visit(node) {
    if (
      (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) &&
      node.moduleSpecifier &&
      ts.isStringLiteral(node.moduleSpecifier)
    ) {
      specifiers.push(node.moduleSpecifier.text);
    } else if (
      ts.isCallExpression(node) &&
      node.expression.kind === ts.SyntaxKind.ImportKeyword &&
      node.arguments.length > 0 &&
      ts.isStringLiteral(node.arguments[0])
    ) {
      specifiers.push(node.arguments[0].text);
    } else if (
      ts.isImportTypeNode(node) &&
      ts.isLiteralTypeNode(node.argument) &&
      ts.isStringLiteral(node.argument.literal)
    ) {
      specifiers.push(node.argument.literal.text);
    }
    ts.forEachChild(node, visit);
  }
  visit(sf);
  return specifiers;
}

function main() {
  const jsonOutput = process.argv.includes('--json');
  const allTsFiles = [
    ...collectTsFiles(WEB_SRC_ROOT),
    ...collectTsFiles(STORYBOOK_ROOT),
  ];

  const configPath = path.join(WEB_ROOT, 'tsconfig.json');
  const configFile = ts.readConfigFile(configPath, ts.sys.readFile);
  if (configFile.error) {
    throw new Error(
      ts.flattenDiagnosticMessageText(configFile.error.messageText, '\n'),
    );
  }
  const parsed = ts.parseJsonConfigFileContent(
    configFile.config,
    ts.sys,
    WEB_ROOT,
  );
  const compilerOptions = {
    ...parsed.options,
    baseUrl: WEB_ROOT,
  };

  const servicesHost = {
    getScriptFileNames: () => allTsFiles,
    getScriptVersion: () => '1',
    getScriptSnapshot: (fileName) => {
      if (!fs.existsSync(fileName)) return undefined;
      return ts.ScriptSnapshot.fromString(fs.readFileSync(fileName, 'utf8'));
    },
    getCurrentDirectory: () => WEB_ROOT,
    getCompilationSettings: () => compilerOptions,
    getDefaultLibFileName: (options) => ts.getDefaultLibFilePath(options),
    fileExists: ts.sys.fileExists,
    readFile: ts.sys.readFile,
    readDirectory: ts.sys.readDirectory,
    directoryExists: ts.sys.directoryExists,
    getDirectories: ts.sys.getDirectories,
  };

  const ls = ts.createLanguageService(
    servicesHost,
    ts.createDocumentRegistry(),
  );
  const program = ls.getProgram();
  if (!program) {
    throw new Error('Failed to create TypeScript program.');
  }

  const importedFiles = new Set();
  for (const sf of program.getSourceFiles()) {
    const fileName = sf.fileName;
    if (
      !fileName.startsWith(WEB_SRC_ROOT + path.sep) &&
      !fileName.startsWith(STORYBOOK_ROOT + path.sep)
    ) {
      continue;
    }
    for (const specifier of collectModuleSpecifiers(sf)) {
      const resolved = ts.resolveModuleName(
        specifier,
        fileName,
        compilerOptions,
        ts.sys,
      );
      if (resolved.resolvedModule?.resolvedFileName) {
        importedFiles.add(
          path.resolve(resolved.resolvedModule.resolvedFileName),
        );
      }
    }
  }

  const orphanFiles = [];
  const unusedExports = [];

  for (const fileName of allTsFiles) {
    if (!fileName.startsWith(WEB_SRC_ROOT + path.sep)) continue;
    if (
      isGeneratedFile(fileName) ||
      isTestOrStoryFile(fileName) ||
      isEntrypointFile(fileName)
    ) {
      continue;
    }
    const resolvedFile = path.resolve(fileName);
    const relFile = path.relative(REPO_ROOT, resolvedFile).replaceAll('\\', '/');

    if (!importedFiles.has(resolvedFile)) {
      orphanFiles.push({
        file: relFile,
        rule: 'orphan-ts-file',
        message:
          'TypeScript file is never imported by any source, test, or story file',
      });
      continue;
    }

    const sf = program.getSourceFile(fileName);
    if (!sf) continue;

    const checkExportedNameNode = (declNode, nameNode) => {
      if (!nameNode || !ts.isIdentifier(nameNode)) return;
      const pos = nameNode.getStart(sf);
      const { line, character } = sf.getLineAndCharacterOfPosition(pos);
      const refs = ls.findReferences(fileName, pos) || [];

      let externalRefs = 0;
      let sameFileNonDefRefs = 0;
      for (const group of refs) {
        for (const ref of group.references) {
          if (
            ref.isDefinition &&
            path.resolve(ref.fileName) === resolvedFile &&
            ref.textSpan.start === pos
          ) {
            continue;
          }
          if (path.resolve(ref.fileName) === resolvedFile) {
            sameFileNonDefRefs++;
          } else {
            externalRefs++;
          }
        }
      }

      if (externalRefs === 0) {
        const kind = getKindLabel(declNode);
        unusedExports.push({
          file: relFile,
          line: line + 1,
          column: character + 1,
          name: nameNode.text,
          kind,
          sameFileUses: sameFileNonDefRefs,
          rule:
            sameFileNonDefRefs === 0
              ? 'unused-export'
              : 'unnecessary-export',
          message:
            sameFileNonDefRefs === 0
              ? `exported ${kind} '${nameNode.text}' is never used anywhere`
              : `exported ${kind} '${nameNode.text}' is only used within this file (remove 'export' modifier)`,
        });
      }
    };

    sf.forEachChild((node) => {
      if (hasExportModifier(node)) {
        if (
          ts.isFunctionDeclaration(node) ||
          ts.isClassDeclaration(node) ||
          ts.isInterfaceDeclaration(node) ||
          ts.isTypeAliasDeclaration(node) ||
          ts.isEnumDeclaration(node)
        ) {
          checkExportedNameNode(node, node.name);
        } else if (ts.isVariableStatement(node)) {
          for (const decl of node.declarationList.declarations) {
            for (const ident of collectBindingIdentifiers(decl.name)) {
              checkExportedNameNode(decl, ident);
            }
          }
        }
      } else if (
        ts.isExportDeclaration(node) &&
        node.exportClause &&
        ts.isNamedExports(node.exportClause)
      ) {
        for (const elem of node.exportClause.elements) {
          checkExportedNameNode(elem, elem.name);
        }
      }
    });
  }

  orphanFiles.sort((a, b) => a.file.localeCompare(b.file));
  unusedExports.sort(
    (a, b) =>
      a.file.localeCompare(b.file) || a.line - b.line || a.column - b.column,
  );

  if (jsonOutput) {
    console.log(
      JSON.stringify(
        {
          orphanFilesCount: orphanFiles.length,
          unusedExportsCount: unusedExports.length,
          zeroUseExportsCount: unusedExports.filter((x) => x.sameFileUses === 0)
            .length,
          sameFileOnlyExportsCount: unusedExports.filter(
            (x) => x.sameFileUses > 0,
          ).length,
          orphanFiles,
          unusedExports,
        },
        null,
        2,
      ),
    );
  } else {
    for (const item of orphanFiles) {
      console.error(`${item.file}:1:1: ${item.message} (${item.rule})`);
    }
    for (const item of unusedExports) {
      console.error(
        `${item.file}:${item.line}:${item.column}: ${item.message} (${item.rule})`,
      );
    }
  }

  if (orphanFiles.length > 0 || unusedExports.length > 0) {
    process.exit(1);
  }
}

main();
