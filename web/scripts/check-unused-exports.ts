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

import * as fs from 'node:fs';
import * as path from 'node:path';
import ts from 'typescript';

/**
 * Identifies the lint rule violated by an unreferenced file or export.
 */
export enum UnusedExportRuleId {
  /** Flags non-entrypoint TypeScript files never imported by any file. */
  OrphanTsFile = 'orphan-ts-file',
  /** Flags exported declarations with zero references anywhere. */
  UnusedExport = 'unused-export',
  /** Flags exported declarations referenced only within their declaring file. */
  UnnecessaryExport = 'unnecessary-export',
}

/**
 * Classifies the syntactic kind of an exported TypeScript declaration.
 */
export enum ExportedDeclarationKind {
  /** Represents an exported function declaration. */
  Function = 'function',
  /** Represents an exported class declaration. */
  Class = 'class',
  /** Represents an exported interface declaration. */
  Interface = 'interface',
  /** Represents an exported type alias declaration. */
  Type = 'type',
  /** Represents an exported enum declaration. */
  Enum = 'enum',
  /** Represents an exported variable or constant binding. */
  Const = 'const',
  /** Represents a named export specifier in an export clause. */
  ReExport = 're-export',
  /** Represents any other exported declaration node. */
  Declaration = 'declaration',
}

/**
 * Describes a TypeScript source file that is never imported across the project.
 */
export interface OrphanFileViolation {
  /** Repository-relative forward-slash file path. */
  readonly file: string;
  /** Violated rule identifier. */
  readonly rule: UnusedExportRuleId;
  /** Human-readable diagnostic message. */
  readonly message: string;
}

/**
 * Describes an exported symbol that has no external module references.
 */
export interface UnusedExportViolation {
  /** Repository-relative forward-slash file path. */
  readonly file: string;
  /** 1-based source line number of the exported identifier. */
  readonly line: number;
  /** 1-based source column number of the exported identifier. */
  readonly column: number;
  /** Identifier text of the exported symbol. */
  readonly name: string;
  /** Syntactic category of the exported declaration. */
  readonly kind: ExportedDeclarationKind;
  /** Number of non-definition references within the declaring file. */
  readonly sameFileUses: number;
  /** Violated rule identifier. */
  readonly rule: UnusedExportRuleId;
  /** Human-readable diagnostic message. */
  readonly message: string;
}

/**
 * Summarizes all orphan files and unused or unnecessary exports found in a project.
 */
export interface UnusedExportsReport {
  /** Total number of orphan TypeScript files detected. */
  readonly orphanFilesCount: number;
  /** Total number of unused or unnecessary exports detected. */
  readonly unusedExportsCount: number;
  /** Count of exports with zero uses anywhere. */
  readonly zeroUseExportsCount: number;
  /** Count of exports referenced only within their declaring file. */
  readonly sameFileOnlyExportsCount: number;
  /** Sorted list of orphan file violations. */
  readonly orphanFiles: readonly OrphanFileViolation[];
  /** Sorted list of unused or unnecessary export violations. */
  readonly unusedExports: readonly UnusedExportViolation[];
}

/**
 * Holds normalized root directory paths for TypeScript unused-export analysis.
 */
export interface ProjectPaths {
  /** Normalized repository root directory path. */
  readonly repoRoot: string;
  /** Normalized frontend workspace directory path (`<repoRoot>/web`). */
  readonly webRoot: string;
  /** Normalized Angular source directory path (`<repoRoot>/web/src`). */
  readonly webSrcRoot: string;
  /** Normalized Storybook configuration directory path (`<repoRoot>/web/.storybook`). */
  readonly storybookRoot: string;
  /** Normalized frontend tooling scripts directory path (`<repoRoot>/web/scripts`). */
  readonly webScriptsRoot: string;
}

/**
 * Abstracts working directory and output streams for CLI execution.
 */
export interface CliEnvironment {
  /** Working directory from which the CLI was invoked. */
  readonly cwd: string;
  /** Writes a single line to standard output. */
  readonly writeStdout: (line: string) => void;
  /** Writes a single line to standard error. */
  readonly writeStderr: (line: string) => void;
}

/**
 * Counts external vs. same-file references for an exported declaration.
 */
interface ReferenceCounts {
  readonly externalRefs: number;
  readonly sameFileNonDefRefs: number;
}

/**
 * Bundles TypeScript Language Service context for checking a single file's exports.
 */
interface FileExportCheckContext {
  readonly ls: ts.LanguageService;
  readonly checker: ts.TypeChecker;
  readonly sf: ts.SourceFile;
  readonly fileName: string;
  readonly resolvedFile: string;
  readonly relFile: string;
  readonly violations: UnusedExportViolation[];
}

/**
 * Accumulates orphan-file and unused-export violations during project scanning.
 */
interface ViolationCollector {
  readonly ls: ts.LanguageService;
  readonly program: ts.Program;
  readonly checker: ts.TypeChecker;
  readonly paths: ProjectPaths;
  readonly importedFiles: ReadonlySet<string>;
  readonly orphanFiles: OrphanFileViolation[];
  readonly unusedExports: UnusedExportViolation[];
}

const IGNORED_DIRECTORY_NAMES = new Set<string>([
  'node_modules',
  '.angular',
  'dist',
  'out-tsc',
]);

/**
 * Normalizes a filesystem path to forward slashes and lowercases any Windows drive letter.
 */
export function normalizePath(filePath: string): string {
  const resolved = (
    /^[a-zA-Z]:[\\/]/.test(filePath)
      ? path.win32.resolve(filePath)
      : path.resolve(filePath)
  ).replaceAll('\\', '/');
  return /^[a-zA-Z]:/.test(resolved)
    ? resolved[0].toLowerCase() + resolved.slice(1)
    : resolved;
}

/**
 * Constructs normalized project directory paths from a repository root path.
 */
export function buildProjectPaths(repoRoot: string): ProjectPaths {
  const normalizedRepo = normalizePath(repoRoot);
  const webRoot = normalizePath(path.join(normalizedRepo, 'web'));
  return {
    repoRoot: normalizedRepo,
    webRoot,
    webSrcRoot: normalizePath(path.join(webRoot, 'src')),
    storybookRoot: normalizePath(path.join(webRoot, '.storybook')),
    webScriptsRoot: normalizePath(path.join(webRoot, 'scripts')),
  };
}

/**
 * Recursively collects all TypeScript source files under a directory.
 */
function collectTsFiles(dir: string, out: string[] = []): string[] {
  if (!fs.existsSync(dir)) {
    return out;
  }
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (!IGNORED_DIRECTORY_NAMES.has(entry.name)) {
        collectTsFiles(full, out);
      }
    } else if (entry.isFile() && entry.name.endsWith('.ts')) {
      out.push(normalizePath(full));
    }
  }
  return out;
}

/**
 * Checks whether a normalized file path belongs to generated source or version output.
 */
function isGeneratedFile(fileName: string): boolean {
  return (
    fileName.includes('/src/app/generated/') ||
    path.posix.basename(fileName).startsWith('zzz-') ||
    fileName.includes('/src/environments/version.')
  );
}

/**
 * Checks whether a normalized file path is a unit spec, Storybook story, or test bootstrap file.
 */
function isTestOrStoryFile(fileName: string): boolean {
  return (
    fileName.endsWith('.spec.ts') ||
    fileName.endsWith('.stories.ts') ||
    fileName.endsWith('/src/test.ts') ||
    fileName.includes('/.storybook/')
  );
}

/**
 * Checks whether a normalized file path is a runtime or ambient declaration entrypoint.
 */
function isEntrypointFile(fileName: string): boolean {
  return (
    fileName.endsWith('/src/main.ts') ||
    fileName.endsWith('/src/polyfills.ts') ||
    fileName.endsWith('.d.ts') ||
    fileName.includes('/src/environments/')
  );
}

/**
 * Reports whether a TypeScript AST node carries an explicit `export` modifier.
 */
function hasExportModifier(node: ts.Node): boolean {
  if (!ts.canHaveModifiers(node)) {
    return false;
  }
  const modifiers = ts.getModifiers(node);
  return Boolean(
    modifiers?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword),
  );
}

/**
 * Maps a TypeScript AST declaration node to its human-readable declaration kind.
 */
function getDeclarationKind(node: ts.Node): ExportedDeclarationKind {
  if (ts.isFunctionDeclaration(node)) return ExportedDeclarationKind.Function;
  if (ts.isClassDeclaration(node)) return ExportedDeclarationKind.Class;
  if (ts.isInterfaceDeclaration(node)) return ExportedDeclarationKind.Interface;
  if (ts.isTypeAliasDeclaration(node)) return ExportedDeclarationKind.Type;
  if (ts.isEnumDeclaration(node)) return ExportedDeclarationKind.Enum;
  if (ts.isVariableDeclaration(node)) return ExportedDeclarationKind.Const;
  if (ts.isExportSpecifier(node)) return ExportedDeclarationKind.ReExport;
  return ExportedDeclarationKind.Declaration;
}

/**
 * Extracts all bound identifier nodes from a variable declaration binding name.
 */
function collectBindingIdentifiers(
  nameNode: ts.BindingName,
  out: ts.Identifier[] = [],
): ts.Identifier[] {
  if (ts.isIdentifier(nameNode)) {
    out.push(nameNode);
    return out;
  }
  for (const elem of nameNode.elements) {
    if (ts.isBindingElement(elem)) {
      collectBindingIdentifiers(elem.name, out);
    }
  }
  return out;
}

/**
 * Extracts a string-literal module specifier from an import, re-export, or dynamic import AST node.
 */
function extractNodeModuleSpecifier(node: ts.Node): string | undefined {
  if (
    (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) &&
    node.moduleSpecifier &&
    ts.isStringLiteral(node.moduleSpecifier)
  ) {
    return node.moduleSpecifier.text;
  }
  if (
    ts.isCallExpression(node) &&
    node.expression.kind === ts.SyntaxKind.ImportKeyword &&
    node.arguments.length > 0 &&
    ts.isStringLiteral(node.arguments[0])
  ) {
    return node.arguments[0].text;
  }
  if (
    ts.isImportTypeNode(node) &&
    ts.isLiteralTypeNode(node.argument) &&
    ts.isStringLiteral(node.argument.literal)
  ) {
    return node.argument.literal.text;
  }
  return undefined;
}

/**
 * Collects all module specifiers imported statically, re-exported, or dynamically imported in a source file.
 */
function collectModuleSpecifiers(sf: ts.SourceFile): string[] {
  const specifiers: string[] = [];
  const visit = (node: ts.Node): void => {
    const spec = extractNodeModuleSpecifier(node);
    if (spec !== undefined) {
      specifiers.push(spec);
    }
    ts.forEachChild(node, visit);
  };
  visit(sf);
  return specifiers;
}

/**
 * Loads TypeScript compiler options from `<webRoot>/tsconfig.json`.
 */
function loadCompilerOptions(webRoot: string): ts.CompilerOptions {
  const configPath = path.join(webRoot, 'tsconfig.json');
  const configFile = ts.readConfigFile(configPath, ts.sys.readFile);
  if (configFile.error) {
    throw new Error(
      ts.flattenDiagnosticMessageText(configFile.error.messageText, '\n'),
    );
  }
  const parsed = ts.parseJsonConfigFileContent(
    configFile.config,
    ts.sys,
    webRoot,
  );
  return {
    ...parsed.options,
    baseUrl: webRoot,
  };
}

/**
 * Creates a TypeScript LanguageService backed by an in-memory snapshot cache.
 */
function createCachedLanguageService(
  allTsFiles: string[],
  webRoot: string,
  compilerOptions: ts.CompilerOptions,
): ts.LanguageService {
  const snapshotCache = new Map<string, ts.IScriptSnapshot>();
  const servicesHost: ts.LanguageServiceHost = {
    getScriptFileNames: () => allTsFiles,
    getScriptVersion: () => '1',
    getScriptSnapshot: (fileName: string) => {
      const key = normalizePath(fileName);
      let snapshot = snapshotCache.get(key);
      if (!snapshot) {
        if (!fs.existsSync(fileName)) {
          return undefined;
        }
        snapshot = ts.ScriptSnapshot.fromString(
          fs.readFileSync(fileName, 'utf8'),
        );
        snapshotCache.set(key, snapshot);
      }
      return snapshot;
    },
    getCurrentDirectory: () => webRoot,
    getCompilationSettings: () => compilerOptions,
    getDefaultLibFileName: (options: ts.CompilerOptions) =>
      ts.getDefaultLibFilePath(options),
    fileExists: ts.sys.fileExists,
    readFile: ts.sys.readFile,
    readDirectory: ts.sys.readDirectory,
    directoryExists: ts.sys.directoryExists,
    getDirectories: ts.sys.getDirectories,
  };
  return ts.createLanguageService(servicesHost, ts.createDocumentRegistry());
}

/**
 * Reports whether a normalized source file path belongs to one of the scanned project roots.
 */
function isProjectSourceFile(fileName: string, paths: ProjectPaths): boolean {
  return (
    fileName.startsWith(paths.webSrcRoot + '/') ||
    fileName.startsWith(paths.storybookRoot + '/') ||
    fileName.startsWith(paths.webScriptsRoot + '/')
  );
}

/**
 * Reports whether a normalized source file path is eligible for orphan and unused-export checks.
 */
function isCheckTargetDir(fileName: string, paths: ProjectPaths): boolean {
  return (
    fileName.startsWith(paths.webSrcRoot + '/') ||
    fileName.startsWith(paths.webScriptsRoot + '/')
  );
}

/**
 * Resolves all module specifiers across project source files into a set of normalized target paths.
 */
function collectImportedFileSet(
  program: ts.Program,
  paths: ProjectPaths,
  compilerOptions: ts.CompilerOptions,
): Set<string> {
  const importedFiles = new Set<string>();
  for (const sf of program.getSourceFiles()) {
    const fileName = normalizePath(sf.fileName);
    if (!isProjectSourceFile(fileName, paths)) {
      continue;
    }
    for (const specifier of collectModuleSpecifiers(sf)) {
      const resolved = ts.resolveModuleName(
        specifier,
        sf.fileName,
        compilerOptions,
        ts.sys,
      );
      if (resolved.resolvedModule?.resolvedFileName) {
        importedFiles.add(
          normalizePath(resolved.resolvedModule.resolvedFileName),
        );
      }
    }
  }
  return importedFiles;
}

/**
 * Collects all upstream file paths along the alias chain of a re-exported specifier.
 */
function collectReExportUpstreamFiles(
  declNode: ts.Node,
  nameNode: ts.Identifier,
  checker: ts.TypeChecker,
  currentResolvedFile: string,
): ReadonlySet<string> {
  const upstreamFiles = new Set<string>();
  if (!ts.isExportSpecifier(declNode)) {
    return upstreamFiles;
  }
  const seenSymbols = new Set<ts.Symbol>();
  let currentSymbol = checker.getSymbolAtLocation(nameNode);
  while (currentSymbol && !seenSymbols.has(currentSymbol)) {
    seenSymbols.add(currentSymbol);
    for (const decl of currentSymbol.declarations ?? []) {
      const declFile = normalizePath(decl.getSourceFile().fileName);
      if (declFile !== currentResolvedFile) {
        upstreamFiles.add(declFile);
      }
    }
    if ((currentSymbol.flags & ts.SymbolFlags.Alias) === 0) {
      break;
    }
    currentSymbol = checker.getImmediateAliasedSymbol(currentSymbol);
  }
  return upstreamFiles;
}

/**
 * Counts external and same-file non-definition references for an identifier at `pos`.
 */
function countIdentifierReferences(
  ctx: FileExportCheckContext,
  pos: number,
  upstreamReExportFiles: ReadonlySet<string>,
): ReferenceCounts {
  const refs = ctx.ls.findReferences(ctx.fileName, pos) ?? [];
  let externalRefs = 0;
  let sameFileNonDefRefs = 0;
  for (const group of refs) {
    for (const ref of group.references) {
      const refFile = normalizePath(ref.fileName);
      if (
        (ref.isDefinition &&
          refFile === ctx.resolvedFile &&
          ref.textSpan.start === pos) ||
        upstreamReExportFiles.has(refFile)
      ) {
        continue;
      }
      if (refFile === ctx.resolvedFile) {
        sameFileNonDefRefs++;
      } else {
        externalRefs++;
      }
    }
  }
  return { externalRefs, sameFileNonDefRefs };
}

/**
 * Evaluates a single exported identifier node and appends a violation when unreferenced externally.
 */
function checkExportedIdentifier(
  ctx: FileExportCheckContext,
  declNode: ts.Node,
  nameNode: ts.ModuleExportName | undefined,
): void {
  if (!nameNode || !ts.isIdentifier(nameNode)) {
    return;
  }
  const pos = nameNode.getStart(ctx.sf);
  const { line, character } = ctx.sf.getLineAndCharacterOfPosition(pos);
  const upstreamFiles = collectReExportUpstreamFiles(
    declNode,
    nameNode,
    ctx.checker,
    ctx.resolvedFile,
  );
  const counts = countIdentifierReferences(ctx, pos, upstreamFiles);
  if (counts.externalRefs > 0) {
    return;
  }
  const kind = getDeclarationKind(declNode);
  const isZeroUse = counts.sameFileNonDefRefs === 0;
  ctx.violations.push({
    file: ctx.relFile,
    line: line + 1,
    column: character + 1,
    name: nameNode.text,
    kind,
    sameFileUses: counts.sameFileNonDefRefs,
    rule: isZeroUse
      ? UnusedExportRuleId.UnusedExport
      : UnusedExportRuleId.UnnecessaryExport,
    message: isZeroUse
      ? `exported ${kind} '${nameNode.text}' is never used anywhere`
      : `exported ${kind} '${nameNode.text}' is only used within this file (remove 'export' modifier)`,
  });
}

/**
 * Inspects all top-level exported declarations in a source file and records violations.
 */
function collectFileExportViolations(ctx: FileExportCheckContext): void {
  ctx.sf.forEachChild((node: ts.Node) => {
    if (hasExportModifier(node)) {
      if (
        ts.isFunctionDeclaration(node) ||
        ts.isClassDeclaration(node) ||
        ts.isInterfaceDeclaration(node) ||
        ts.isTypeAliasDeclaration(node) ||
        ts.isEnumDeclaration(node)
      ) {
        checkExportedIdentifier(ctx, node, node.name);
      } else if (ts.isVariableStatement(node)) {
        for (const decl of node.declarationList.declarations) {
          for (const ident of collectBindingIdentifiers(decl.name)) {
            checkExportedIdentifier(ctx, decl, ident);
          }
        }
      }
    } else if (
      ts.isExportDeclaration(node) &&
      node.exportClause &&
      ts.isNamedExports(node.exportClause)
    ) {
      for (const elem of node.exportClause.elements) {
        checkExportedIdentifier(ctx, elem, elem.name);
      }
    }
  });
}

/**
 * Evaluates a single candidate file for orphan status or unused exports.
 */
function scanSingleCandidateFile(
  fileName: string,
  collector: ViolationCollector,
): void {
  if (
    !isCheckTargetDir(fileName, collector.paths) ||
    isGeneratedFile(fileName) ||
    isTestOrStoryFile(fileName) ||
    isEntrypointFile(fileName)
  ) {
    return;
  }
  const resolvedFile = normalizePath(fileName);
  const relFile = path.posix.relative(collector.paths.repoRoot, resolvedFile);

  if (!collector.importedFiles.has(resolvedFile)) {
    collector.orphanFiles.push({
      file: relFile,
      rule: UnusedExportRuleId.OrphanTsFile,
      message:
        'TypeScript file is never imported by any source, test, or story file',
    });
    return;
  }

  const sf = collector.program.getSourceFile(fileName);
  if (!sf) {
    return;
  }
  collectFileExportViolations({
    ls: collector.ls,
    checker: collector.checker,
    sf,
    fileName,
    resolvedFile,
    relFile,
    violations: collector.unusedExports,
  });
}

/**
 * Analyzes all TypeScript files under `paths` for orphan files and unused or unnecessary exports.
 */
export function analyzeUnusedExports(paths: ProjectPaths): UnusedExportsReport {
  const allTsFiles = [
    ...collectTsFiles(paths.webSrcRoot),
    ...collectTsFiles(paths.storybookRoot),
    ...collectTsFiles(paths.webScriptsRoot),
  ];
  const compilerOptions = loadCompilerOptions(paths.webRoot);
  const ls = createCachedLanguageService(
    allTsFiles,
    paths.webRoot,
    compilerOptions,
  );
  const program = ls.getProgram();
  if (!program) {
    throw new Error('Failed to create TypeScript program.');
  }

  const collector: ViolationCollector = {
    ls,
    program,
    checker: program.getTypeChecker(),
    paths,
    importedFiles: collectImportedFileSet(program, paths, compilerOptions),
    orphanFiles: [],
    unusedExports: [],
  };
  for (const fileName of allTsFiles) {
    scanSingleCandidateFile(fileName, collector);
  }

  collector.orphanFiles.sort((a, b) => a.file.localeCompare(b.file));
  collector.unusedExports.sort(
    (a, b) =>
      a.file.localeCompare(b.file) || a.line - b.line || a.column - b.column,
  );

  const zeroUseExportsCount = collector.unusedExports.filter(
    (x) => x.sameFileUses === 0,
  ).length;
  return {
    orphanFilesCount: collector.orphanFiles.length,
    unusedExportsCount: collector.unusedExports.length,
    zeroUseExportsCount,
    sameFileOnlyExportsCount:
      collector.unusedExports.length - zeroUseExportsCount,
    orphanFiles: collector.orphanFiles,
    unusedExports: collector.unusedExports,
  };
}

/**
 * Formats an `UnusedExportsReport` into standard `file:line:col: message (rule)` diagnostic lines.
 */
export function formatDiagnosticLines(report: UnusedExportsReport): string[] {
  const lines: string[] = [];
  for (const item of report.orphanFiles) {
    lines.push(`${item.file}:1:1: ${item.message} (${item.rule})`);
  }
  for (const item of report.unusedExports) {
    lines.push(
      `${item.file}:${item.line}:${item.column}: ${item.message} (${item.rule})`,
    );
  }
  return lines;
}

/**
 * Resolves the repository root directory whether invoked from the repository root or `web/`.
 */
function resolveDefaultRepoRoot(cwd: string): string {
  if (fs.existsSync(path.join(cwd, 'web', 'package.json'))) {
    return cwd;
  }
  return path.resolve(cwd, '..');
}

/**
 * Executes the CLI entrypoint for unused TypeScript file and export checks and returns the exit code.
 */
export function runCli(
  argv: readonly string[],
  env: CliEnvironment = {
    cwd: process.cwd(),
    writeStdout: (line: string) => console.log(line),
    writeStderr: (line: string) => console.error(line),
  },
): number {
  const jsonOutput = argv.includes('--json');
  const paths = buildProjectPaths(resolveDefaultRepoRoot(env.cwd));
  const report = analyzeUnusedExports(paths);

  if (jsonOutput) {
    env.writeStdout(JSON.stringify(report, null, 2));
  } else {
    for (const line of formatDiagnosticLines(report)) {
      env.writeStderr(line);
    }
  }

  return report.orphanFilesCount > 0 || report.unusedExportsCount > 0 ? 1 : 0;
}

/* c8 ignore start */
if (require.main === module) {
  process.exitCode = runCli(process.argv.slice(2));
}
/* c8 ignore stop */
