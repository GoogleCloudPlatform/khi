#!/usr/bin/env node
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
 * Extracts TypeScript AST mutation sites and raw comments for KHI differential mutation testing.
 *
 * Usage:
 *   node scripts/codequality/internal/tscheck/js/mutate_ts.js <repoRoot> <relFile1> [relFile2...]
 *
 * Emits a JSON object keyed by repository-relative file path containing comments and candidate mutation sites.
 */

'use strict';

const fs = require('fs');
const path = require('path');
const { createRequire } = require('module');

const UPPER_SNAKE_RE = /^[A-Z][A-Z0-9_]*$/;

const ANGULAR_DECLARATIVE_HELPERS = new Set([
  'inject',
  'input',
  'output',
  'outputFromObservable',
  'model',
  'viewChild',
  'viewChildren',
  'contentChild',
  'contentChildren',
]);

const CONSOLE_DIAG_METHODS = new Set([
  'log',
  'warn',
  'error',
  'info',
  'debug',
  'trace',
  'assert',
]);

const ERROR_CTORS = new Set([
  'Error',
  'TypeError',
  'RangeError',
  'SyntaxError',
  'ReferenceError',
  'URIError',
]);

const LIFECYCLE_TEARDOWN_CALLS = new Set([
  'onCleanup',
  'takeUntilDestroyed',
  'cancelAnimationFrame',
  'clearTimeout',
  'clearInterval',
]);

const LIFECYCLE_TEARDOWN_METHODS = new Set([
  'onDestroy',
  'unsubscribe',
]);

const STATEFUL_VOID_METHODS = new Set([
  'set',
  'update',
  'mutate',
  'push',
  'pop',
  'shift',
  'unshift',
  'splice',
  'delete',
  'clear',
  'add',
  'emit',
  'next',
  'complete',
  'replaceSearch',
]);

function main() {
  const args = process.argv.slice(2);
  if (args.length < 2) {
    process.stderr.write(
      'Usage: mutate_ts.js <repoRoot> <relFile1> [relFile2...]\n',
    );
    process.exit(2);
  }

  const repoRoot = path.resolve(args[0]);
  const relFiles = args.slice(1);
  const webPkgPath = path.join(repoRoot, 'web', 'package.json');
  const webRequire = createRequire(webPkgPath);
  const ts = webRequire('typescript');

  const output = {};

  for (const relFile of relFiles) {
    const absFile = path.join(repoRoot, relFile);
    if (!fs.existsSync(absFile)) {
      continue;
    }
    const srcText = fs.readFileSync(absFile, 'utf8');
    const utf8Offsets = buildUtf16ToUtf8Table(srcText);
    const sf = ts.createSourceFile(
      absFile,
      srcText,
      ts.ScriptTarget.ES2022,
      /* setParentNodes */ true,
      ts.ScriptKind.TS,
    );

    if (relFile.endsWith('.spec.ts')) {
      output[relFile] = {
        comments: [],
        sites: [],
        specCases: extractSpecCases(ts, sf),
      };
      continue;
    }

    const comments = extractAllComments(ts, sf, srcText);
    const blockStmts = collectBlockListStatements(ts, sf);
    const sites = [];

    visitNode(ts, sf, sf, srcText, utf8Offsets, blockStmts, sites);

    output[relFile] = {
      comments,
      sites,
    };
  }

  process.stdout.write(JSON.stringify(output));
}

/**
 * Builds a lookup table mapping UTF-16 code unit indices in text to UTF-8 byte offsets.
 */
function buildUtf16ToUtf8Table(text) {
  const len = text.length;
  const table = new Uint32Array(len + 1);
  let bytePos = 0;
  let i = 0;
  while (i < len) {
    table[i] = bytePos;
    const code = text.charCodeAt(i);
    if (code <= 0x7f) {
      bytePos += 1;
      i += 1;
    } else if (code <= 0x7ff) {
      bytePos += 2;
      i += 1;
    } else if (code >= 0xd800 && code <= 0xdbff && i + 1 < len) {
      const low = text.charCodeAt(i + 1);
      if (low >= 0xdc00 && low <= 0xdfff) {
        table[i + 1] = bytePos + 2;
        bytePos += 4;
        i += 2;
        continue;
      }
      bytePos += 3;
      i += 1;
    } else {
      bytePos += 3;
      i += 1;
    }
  }
  table[len] = bytePos;
  return table;
}

/**
 * Extracts all single-line and multi-line comments from trivia intervals between AST leaf tokens.
 */
function extractAllComments(ts, sf, srcText) {
  const leafRanges = [];
  function collectLeaves(node) {
    if (node.getChildCount(sf) === 0) {
      const start = node.getStart(sf, false);
      const end = node.getEnd();
      if (end > start) {
        leafRanges.push([start, end]);
      }
      return;
    }
    for (const child of node.getChildren(sf)) {
      collectLeaves(child);
    }
  }
  collectLeaves(sf);

  const comments = [];
  let cursor = 0;
  for (const [tokenStart, tokenEnd] of leafRanges) {
    if (tokenStart > cursor) {
      scanCommentsInTrivia(ts, sf, srcText, cursor, tokenStart, comments);
    }
    if (tokenEnd > cursor) {
      cursor = tokenEnd;
    }
  }
  if (cursor < srcText.length) {
    scanCommentsInTrivia(ts, sf, srcText, cursor, srcText.length, comments);
  }
  return comments;
}

function scanCommentsInTrivia(ts, sf, srcText, start, end, out) {
  let i = start;
  while (i < end) {
    const ch = srcText.charCodeAt(i);
    if (ch === 0x2f /* '/' */ && i + 1 < end) {
      const next = srcText.charCodeAt(i + 1);
      if (next === 0x2f /* '/' */) {
        let j = i + 2;
        while (j < end && srcText.charCodeAt(j) !== 0x0a && srcText.charCodeAt(j) !== 0x0d) {
          j++;
        }
        const line = sf.getLineAndCharacterOfPosition(i).line + 1;
        out.push({ line, text: srcText.slice(i, j) });
        i = j;
        continue;
      }
      if (next === 0x2a /* '*' */) {
        let j = i + 2;
        while (j + 1 < end && !(srcText.charCodeAt(j) === 0x2a && srcText.charCodeAt(j + 1) === 0x2f)) {
          j++;
        }
        j = Math.min(end, j + 2);
        const line = sf.getLineAndCharacterOfPosition(i).line + 1;
        out.push({ line, text: srcText.slice(i, j) });
        i = j;
        continue;
      }
    }
    i++;
  }
}

function collectBlockListStatements(ts, sf) {
  const set = new Set();
  function walk(node) {
    if (ts.isBlock(node) || ts.isCaseClause(node) || ts.isDefaultClause(node)) {
      for (const st of node.statements) {
        set.add(st);
      }
    }
    ts.forEachChild(node, walk);
  }
  walk(sf);
  return set;
}

function visitNode(ts, sf, node, srcText, utf8Offsets, blockStmts, sites) {
  if (shouldSkipSubtree(ts, node, sf)) {
    return;
  }

  extractCandidateSites(ts, sf, node, srcText, utf8Offsets, blockStmts, sites);

  ts.forEachChild(node, (child) => {
    visitNode(ts, sf, child, srcText, utf8Offsets, blockStmts, sites);
  });
}

function shouldSkipSubtree(ts, node, sf) {
  if (
    ts.isTypeNode(node) ||
    ts.isInterfaceDeclaration(node) ||
    ts.isTypeAliasDeclaration(node) ||
    ts.isEnumDeclaration(node) ||
    ts.isImportDeclaration(node) ||
    ts.isImportEqualsDeclaration(node) ||
    ts.isDecorator(node)
  ) {
    return true;
  }

  if (ts.isExportDeclaration(node) && node.isTypeOnly) {
    return true;
  }

  if (hasModifier(ts, node, ts.SyntaxKind.DeclareKeyword)) {
    return true;
  }

  // Skip UPPER_SNAKE_CASE constant and readonly literal initializers.
  if (ts.isVariableDeclaration(node) && node.initializer) {
    if (isConstDeclarationList(ts, node.parent) && ts.isIdentifier(node.name)) {
      if (UPPER_SNAKE_RE.test(node.name.text) && isPureLiteralExpr(ts, node.initializer)) {
        return true;
      }
    }
    if (isAngularDeclarativeInit(ts, node.initializer)) {
      return true;
    }
  }

  if (ts.isPropertyDeclaration(node) && node.initializer) {
    if (
      hasModifier(ts, node, ts.SyntaxKind.ReadonlyKeyword) &&
      ts.isIdentifier(node.name) &&
      UPPER_SNAKE_RE.test(node.name.text) &&
      isPureLiteralExpr(ts, node.initializer)
    ) {
      return true;
    }
    if (isAngularDeclarativeInit(ts, node.initializer)) {
      return true;
    }
  }

  // Skip console.* diagnostic calls, Error(...) constructor messages, and Angular/RxJS lifecycle teardown hooks.
  if (ts.isCallExpression(node)) {
    if (
      isConsoleCall(ts, node.expression) ||
      isErrorConstructorTarget(ts, node.expression) ||
      isLifecycleTeardownCall(ts, node.expression)
    ) {
      return true;
    }
  }
  if (ts.isNewExpression(node) && isErrorConstructorTarget(ts, node.expression)) {
    return true;
  }

  // Skip standalone Angular DestroyRef teardown guards: if (this.destroyRef.destroyed) { return ...; }.
  if (ts.isIfStatement(node) && !node.elseStatement && isDestroyRefDestroyedExpr(ts, node.expression)) {
    return true;
  }

  return false;
}

function extractCandidateSites(ts, sf, node, srcText, utf8Offsets, blockStmts, sites) {
  // 1. Binary expressions: ROR, LCR (&&, ||, non-trivial ??), AOR.
  if (ts.isBinaryExpression(node)) {
    const opKind = node.operatorToken.kind;
    if (!isAssignmentOperator(ts, opKind) && !isPureLiteralExpr(ts, node)) {
      const opLine = lineOfPos(sf, node.operatorToken.getStart(sf));
      const startOffset = utf8Offsets[node.getStart(sf)];
      const endOffset = utf8Offsets[node.getEnd()];
      const leftStart = utf8Offsets[node.left.getStart(sf)];
      const leftEnd = utf8Offsets[node.left.getEnd()];
      const rightStart = utf8Offsets[node.right.getStart(sf)];
      const rightEnd = utf8Offsets[node.right.getEnd()];
      const leftTxt = compactWs(srcText.slice(node.left.getStart(sf), node.left.getEnd()));
      const rightTxt = compactWs(srcText.slice(node.right.getStart(sf), node.right.getEnd()));
      const origTxt = compactWs(srcText.slice(node.getStart(sf), node.getEnd()));

      switch (opKind) {
        case ts.SyntaxKind.LessThanToken:
          sites.push({
            kind: 'ordered_cmp',
            op: '<',
            operator: 'ROR',
            line: opLine,
            startOffset,
            endOffset,
            leftStart,
            leftEnd,
            rightStart,
            rightEnd,
            candidates: [
              { origExpr: origTxt, mutExpr: `${leftTxt} <= ${rightTxt}`, replacementPreview: `${leftTxt} <= ${rightTxt}` },
              { origExpr: origTxt, mutExpr: `${leftTxt} >= ${rightTxt}`, replacementPreview: `${leftTxt} >= ${rightTxt}` },
            ],
          });
          return;

        case ts.SyntaxKind.LessThanEqualsToken:
          sites.push({
            kind: 'ordered_cmp',
            op: '<=',
            operator: 'ROR',
            line: opLine,
            startOffset,
            endOffset,
            leftStart,
            leftEnd,
            rightStart,
            rightEnd,
            candidates: [
              { origExpr: origTxt, mutExpr: `${leftTxt} < ${rightTxt}`, replacementPreview: `${leftTxt} < ${rightTxt}` },
              { origExpr: origTxt, mutExpr: `${leftTxt} > ${rightTxt}`, replacementPreview: `${leftTxt} > ${rightTxt}` },
            ],
          });
          return;

        case ts.SyntaxKind.GreaterThanToken:
          sites.push({
            kind: 'ordered_cmp',
            op: '>',
            operator: 'ROR',
            line: opLine,
            startOffset,
            endOffset,
            leftStart,
            leftEnd,
            rightStart,
            rightEnd,
            candidates: [
              { origExpr: origTxt, mutExpr: `${leftTxt} >= ${rightTxt}`, replacementPreview: `${leftTxt} >= ${rightTxt}` },
              { origExpr: origTxt, mutExpr: `${leftTxt} <= ${rightTxt}`, replacementPreview: `${leftTxt} <= ${rightTxt}` },
            ],
          });
          return;

        case ts.SyntaxKind.GreaterThanEqualsToken:
          sites.push({
            kind: 'ordered_cmp',
            op: '>=',
            operator: 'ROR',
            line: opLine,
            startOffset,
            endOffset,
            leftStart,
            leftEnd,
            rightStart,
            rightEnd,
            candidates: [
              { origExpr: origTxt, mutExpr: `${leftTxt} > ${rightTxt}`, replacementPreview: `${leftTxt} > ${rightTxt}` },
              { origExpr: origTxt, mutExpr: `${leftTxt} < ${rightTxt}`, replacementPreview: `${leftTxt} < ${rightTxt}` },
            ],
          });
          return;

        case ts.SyntaxKind.EqualsEqualsEqualsToken:
        case ts.SyntaxKind.ExclamationEqualsEqualsToken:
        case ts.SyntaxKind.EqualsEqualsToken:
        case ts.SyntaxKind.ExclamationEqualsToken: {
          const origOp = node.operatorToken.getText(sf);
          const flippedOp = flipEqOperator(origOp);
          sites.push({
            kind: 'eq_cmp',
            op: origOp,
            operator: 'ROR',
            line: opLine,
            startOffset,
            endOffset,
            leftStart,
            leftEnd,
            rightStart,
            rightEnd,
            candidates: [
              {
                origExpr: origTxt,
                mutExpr: `${leftTxt} ${flippedOp} ${rightTxt}`,
                replacementPreview: `${leftTxt} ${flippedOp} ${rightTxt}`,
              },
            ],
          });
          return;
        }

        case ts.SyntaxKind.AmpersandAmpersandToken:
        case ts.SyntaxKind.BarBarToken: {
          if (
            containsAwaitOrYield(ts, node) ||
            isDestroyRefDestroyedExpr(ts, node.left) ||
            isDestroyRefDestroyedExpr(ts, node.right)
          ) {
            return;
          }
          const isAnd = opKind === ts.SyntaxKind.AmpersandAmpersandToken;
          const flipped = isAnd ? '||' : '&&';
          sites.push({
            kind: 'logical',
            op: isAnd ? '&&' : '||',
            operator: 'LCR',
            line: opLine,
            startOffset,
            endOffset,
            leftStart,
            leftEnd,
            rightStart,
            rightEnd,
            candidates: [
              {
                origExpr: origTxt,
                mutExpr: `${leftTxt} ${flipped} ${rightTxt}`,
                replacementPreview: `${leftTxt} ${flipped} ${rightTxt}`,
              },
            ],
          });
          return;
        }

        case ts.SyntaxKind.QuestionQuestionToken: {
          if (containsAwaitOrYield(ts, node) || isTrivialFallbackLiteral(ts, node.right)) {
            return;
          }
          sites.push({
            kind: 'nullish',
            op: '??',
            operator: 'LCR',
            line: opLine,
            startOffset,
            endOffset,
            leftStart,
            leftEnd,
            rightStart,
            rightEnd,
            candidates: [
              { origExpr: origTxt, mutExpr: leftTxt, replacementPreview: leftTxt },
              { origExpr: origTxt, mutExpr: rightTxt, replacementPreview: rightTxt },
            ],
          });
          return;
        }

        case ts.SyntaxKind.PlusToken: {
          const keepRight = isZeroOrEmptyLiteral(ts, node.left) && !isZeroOrEmptyLiteral(ts, node.right);
          const chosen = keepRight ? rightTxt : leftTxt;
          sites.push({
            kind: 'arith',
            op: '+',
            addKeepRight: keepRight,
            operator: 'AOR',
            line: opLine,
            startOffset,
            endOffset,
            leftStart,
            leftEnd,
            rightStart,
            rightEnd,
            candidates: [
              { origExpr: origTxt, mutExpr: chosen, replacementPreview: chosen },
            ],
          });
          return;
        }

        case ts.SyntaxKind.MinusToken:
        case ts.SyntaxKind.AsteriskToken:
        case ts.SyntaxKind.SlashToken: {
          const origOp = node.operatorToken.getText(sf);
          const mutOp = origOp === '-' ? '+' : origOp === '*' ? '/' : '*';
          sites.push({
            kind: 'arith',
            op: origOp,
            operator: 'AOR',
            line: opLine,
            startOffset,
            endOffset,
            leftStart,
            leftEnd,
            rightStart,
            rightEnd,
            candidates: [
              {
                origExpr: origTxt,
                mutExpr: `${leftTxt} ${mutOp} ${rightTxt}`,
                replacementPreview: `${leftTxt} ${mutOp} ${rightTxt}`,
              },
            ],
          });
          return;
        }
      }
    }
  }

  // 2. Branch conditions: non-comparison if (cond) and ternary cond ? a : b (BCR).
  if (ts.isIfStatement(node) && shouldApplyBCR(ts, node.expression)) {
    const cond = node.expression;
    const condLine = lineOfPos(sf, cond.getStart(sf));
    const condTxt = compactWs(srcText.slice(cond.getStart(sf), cond.getEnd()));
    sites.push({
      kind: 'cond',
      operator: 'BCR',
      line: condLine,
      startOffset: utf8Offsets[cond.getStart(sf)],
      endOffset: utf8Offsets[cond.getEnd()],
      candidates: [
        { origExpr: `if (${condTxt})`, mutExpr: 'if (true)', replacementPreview: 'true' },
        { origExpr: `if (${condTxt})`, mutExpr: 'if (false)', replacementPreview: 'false' },
      ],
    });
    return;
  }

  if (ts.isConditionalExpression(node) && shouldApplyBCR(ts, node.condition)) {
    const cond = node.condition;
    const condLine = lineOfPos(sf, cond.getStart(sf));
    const condTxt = compactWs(srcText.slice(cond.getStart(sf), cond.getEnd()));
    sites.push({
      kind: 'cond',
      operator: 'BCR',
      line: condLine,
      startOffset: utf8Offsets[cond.getStart(sf)],
      endOffset: utf8Offsets[cond.getEnd()],
      candidates: [
        { origExpr: `${condTxt} ? ...`, mutExpr: 'true ? ...', replacementPreview: 'true' },
        { origExpr: `${condTxt} ? ...`, mutExpr: 'false ? ...', replacementPreview: 'false' },
      ],
    });
    return;
  }

  // 3. Prefix !expr unary negation (excluding !!expr boolean coercion) (UOI).
  if (ts.isPrefixUnaryExpression(node) && node.operator === ts.SyntaxKind.ExclamationToken) {
    const parentUnwrapped = node.parent && ts.isParenthesizedExpression(node.parent) ? node.parent.parent : node.parent;
    if (
      parentUnwrapped &&
      ts.isPrefixUnaryExpression(parentUnwrapped) &&
      parentUnwrapped.operator === ts.SyntaxKind.ExclamationToken
    ) {
      return;
    }
    const inner = unwrapParens(ts, node.operand);
    if (
      (ts.isPrefixUnaryExpression(inner) && inner.operator === ts.SyntaxKind.ExclamationToken) ||
      inner.kind === ts.SyntaxKind.TrueKeyword ||
      inner.kind === ts.SyntaxKind.FalseKeyword ||
      isDestroyRefDestroyedExpr(ts, inner)
    ) {
      return;
    }
    const line = lineOfPos(sf, node.getStart(sf));
    const origTxt = compactWs(srcText.slice(node.getStart(sf), node.getEnd()));
    const operandTxt = compactWs(srcText.slice(node.operand.getStart(sf), node.operand.getEnd()));
    sites.push({
      kind: 'unary_not',
      operator: 'UOI',
      line,
      startOffset: utf8Offsets[node.getStart(sf)],
      endOffset: utf8Offsets[node.getEnd()],
      leftStart: utf8Offsets[node.operand.getStart(sf)],
      leftEnd: utf8Offsets[node.operand.getEnd()],
      candidates: [
        { origExpr: origTxt, mutExpr: operandTxt, replacementPreview: operandTxt },
      ],
    });
    return;
  }

  // 4. Boolean literal returns (return true/false and concise arrow () => true/false) (BLR).
  if (ts.isReturnStatement(node) && node.expression) {
    const expr = unwrapParens(ts, node.expression);
    if (expr.kind === ts.SyntaxKind.TrueKeyword || expr.kind === ts.SyntaxKind.FalseKeyword) {
      const boolVal = expr.kind === ts.SyntaxKind.TrueKeyword;
      const flipped = boolVal ? 'false' : 'true';
      sites.push({
        kind: 'bool_lit',
        boolVal,
        operator: 'BLR',
        line: lineOfPos(sf, expr.getStart(sf)),
        startOffset: utf8Offsets[expr.getStart(sf)],
        endOffset: utf8Offsets[expr.getEnd()],
        candidates: [
          { origExpr: `return ${boolVal}`, mutExpr: `return ${flipped}`, replacementPreview: flipped },
        ],
      });
      return;
    }
  }

  if (ts.isArrowFunction(node) && !ts.isBlock(node.body)) {
    const expr = unwrapParens(ts, node.body);
    if (expr.kind === ts.SyntaxKind.TrueKeyword || expr.kind === ts.SyntaxKind.FalseKeyword) {
      const boolVal = expr.kind === ts.SyntaxKind.TrueKeyword;
      const flipped = boolVal ? 'false' : 'true';
      sites.push({
        kind: 'bool_lit',
        boolVal,
        operator: 'BLR',
        line: lineOfPos(sf, expr.getStart(sf)),
        startOffset: utf8Offsets[expr.getStart(sf)],
        endOffset: utf8Offsets[expr.getEnd()],
        candidates: [
          { origExpr: `=> ${boolVal}`, mutExpr: `=> ${flipped}`, replacementPreview: flipped },
        ],
      });
      return;
    }
  }

  // 5. Block-member side-effect expression statements: assignments, ++/--, and stateful void calls (SBR).
  if (ts.isExpressionStatement(node) && blockStmts.has(node) && isStatefulSkipTarget(ts, node.expression)) {
    if (containsAwaitOrYield(ts, node)) {
      return;
    }
    const line = lineOfPos(sf, node.getStart(sf));
    const origStmt = compactWs(srcText.slice(node.getStart(sf), node.getEnd()));
    sites.push({
      kind: 'stmt_skip',
      operator: 'SBR',
      line,
      startOffset: utf8Offsets[node.getStart(sf)],
      endOffset: utf8Offsets[node.getEnd()],
      candidates: [
        {
          origExpr: origStmt,
          mutExpr: `/* skipped ${origStmt} */`,
          replacementPreview: `/* skipped: ${origStmt} */`,
        },
      ],
    });
  }
}

function isStatefulSkipTarget(ts, expr) {
  const e = unwrapParens(ts, expr);
  if (ts.isBinaryExpression(e) && isAssignmentOperator(ts, e.operatorToken.kind)) {
    return true;
  }
  if (
    (ts.isPrefixUnaryExpression(e) || ts.isPostfixUnaryExpression(e)) &&
    (e.operator === ts.SyntaxKind.PlusPlusToken || e.operator === ts.SyntaxKind.MinusMinusToken)
  ) {
    return true;
  }
  if (ts.isCallExpression(e) && !isConsoleCall(ts, e.expression)) {
    const callee = unwrapParens(ts, e.expression);
    if (ts.isPropertyAccessExpression(callee) && ts.isIdentifier(callee.name)) {
      return STATEFUL_VOID_METHODS.has(callee.name.text);
    }
  }
  return false;
}

function shouldApplyBCR(ts, expr) {
  const e = unwrapParens(ts, expr);
  if (ts.isBinaryExpression(e)) {
    return false;
  }
  if (ts.isPrefixUnaryExpression(e) && e.operator === ts.SyntaxKind.ExclamationToken) {
    return false;
  }
  if (e.kind === ts.SyntaxKind.TrueKeyword || e.kind === ts.SyntaxKind.FalseKeyword) {
    return false;
  }
  if (isDestroyRefDestroyedExpr(ts, e)) {
    return false;
  }
  return true;
}

function isDestroyRefDestroyedExpr(ts, expr) {
  let e = unwrapParens(ts, expr);
  if (ts.isPrefixUnaryExpression(e) && e.operator === ts.SyntaxKind.ExclamationToken) {
    e = unwrapParens(ts, e.operand);
  }
  if (!ts.isPropertyAccessExpression(e) || !ts.isIdentifier(e.name) || e.name.text !== 'destroyed') {
    return false;
  }
  const recv = unwrapParens(ts, e.expression);
  if (ts.isIdentifier(recv)) {
    return /destroyRef$/i.test(recv.text);
  }
  if (ts.isPropertyAccessExpression(recv) && ts.isIdentifier(recv.name)) {
    return /destroyRef$/i.test(recv.name.text);
  }
  return false;
}

function isLifecycleTeardownCall(ts, calleeExpr) {
  const e = unwrapParens(ts, calleeExpr);
  if (ts.isIdentifier(e)) {
    return LIFECYCLE_TEARDOWN_CALLS.has(e.text);
  }
  if (ts.isPropertyAccessExpression(e) && ts.isIdentifier(e.name)) {
    return LIFECYCLE_TEARDOWN_METHODS.has(e.name.text);
  }
  return false;
}

function isTrivialFallbackLiteral(ts, expr) {
  const e = unwrapParens(ts, expr);
  if (
    ts.isNumericLiteral(e) ||
    ts.isBigIntLiteral(e) ||
    ts.isStringLiteral(e) ||
    ts.isNoSubstitutionTemplateLiteral(e) ||
    e.kind === ts.SyntaxKind.TrueKeyword ||
    e.kind === ts.SyntaxKind.FalseKeyword ||
    e.kind === ts.SyntaxKind.NullKeyword
  ) {
    return true;
  }
  if (ts.isIdentifier(e) && e.text === 'undefined') {
    return true;
  }
  if (ts.isArrayLiteralExpression(e) && e.elements.length === 0) {
    return true;
  }
  if (ts.isObjectLiteralExpression(e) && e.properties.length === 0) {
    return true;
  }
  return false;
}

function isZeroOrEmptyLiteral(ts, expr) {
  const e = unwrapParens(ts, expr);
  if (ts.isNumericLiteral(e)) {
    return e.text === '0' || e.text === '0.0';
  }
  if (ts.isBigIntLiteral(e)) {
    return e.text === '0n';
  }
  if (ts.isStringLiteral(e) || ts.isNoSubstitutionTemplateLiteral(e)) {
    return e.text === '';
  }
  return false;
}

function isPureLiteralExpr(ts, expr) {
  const e = unwrapParens(ts, expr);
  if (
    ts.isNumericLiteral(e) ||
    ts.isBigIntLiteral(e) ||
    ts.isStringLiteral(e) ||
    ts.isNoSubstitutionTemplateLiteral(e) ||
    e.kind === ts.SyntaxKind.TrueKeyword ||
    e.kind === ts.SyntaxKind.FalseKeyword ||
    e.kind === ts.SyntaxKind.NullKeyword
  ) {
    return true;
  }
  if (
    ts.isPrefixUnaryExpression(e) &&
    (e.operator === ts.SyntaxKind.MinusToken || e.operator === ts.SyntaxKind.PlusToken)
  ) {
    return isPureLiteralExpr(ts, e.operand);
  }
  if (ts.isBinaryExpression(e)) {
    return isPureLiteralExpr(ts, e.left) && isPureLiteralExpr(ts, e.right);
  }
  return false;
}

function isAngularDeclarativeInit(ts, expr) {
  const e = unwrapParens(ts, expr);
  if (ts.isNewExpression(e) && ts.isIdentifier(e.expression) && e.expression.text === 'InjectionToken') {
    return true;
  }
  if (!ts.isCallExpression(e)) {
    return false;
  }
  const callee = unwrapParens(ts, e.expression);
  if (ts.isIdentifier(callee)) {
    return ANGULAR_DECLARATIVE_HELPERS.has(callee.text);
  }
  if (ts.isPropertyAccessExpression(callee) && ts.isIdentifier(callee.expression)) {
    return ANGULAR_DECLARATIVE_HELPERS.has(callee.expression.text);
  }
  return false;
}

function isConsoleCall(ts, calleeExpr) {
  const e = unwrapParens(ts, calleeExpr);
  return (
    ts.isPropertyAccessExpression(e) &&
    ts.isIdentifier(e.expression) &&
    e.expression.text === 'console' &&
    ts.isIdentifier(e.name) &&
    CONSOLE_DIAG_METHODS.has(e.name.text)
  );
}

function isErrorConstructorTarget(ts, calleeExpr) {
  const e = unwrapParens(ts, calleeExpr);
  return ts.isIdentifier(e) && ERROR_CTORS.has(e.text);
}

function containsAwaitOrYield(ts, root) {
  let found = false;
  function walk(n) {
    if (found) {
      return;
    }
    if (n !== root && ts.isFunctionLike(n)) {
      return;
    }
    if (ts.isAwaitExpression(n) || ts.isYieldExpression(n)) {
      found = true;
      return;
    }
    ts.forEachChild(n, walk);
  }
  walk(root);
  return found;
}

function isAssignmentOperator(ts, kind) {
  return kind >= ts.SyntaxKind.FirstAssignment && kind <= ts.SyntaxKind.LastAssignment;
}

function flipEqOperator(op) {
  switch (op) {
    case '===':
      return '!==';
    case '!==':
      return '===';
    case '==':
      return '!=';
    default:
      return '==';
  }
}

function unwrapParens(ts, expr) {
  let cur = expr;
  while (cur && ts.isParenthesizedExpression(cur)) {
    cur = cur.expression;
  }
  return cur;
}

function isConstDeclarationList(ts, parent) {
  return Boolean(parent && ts.isVariableDeclarationList(parent) && (parent.flags & ts.NodeFlags.Const) !== 0);
}

function hasModifier(ts, node, kind) {
  if (!ts.canHaveModifiers(node)) {
    return false;
  }
  const mods = ts.getModifiers(node);
  return Boolean(mods && mods.some((m) => m.kind === kind));
}

function lineOfPos(sf, pos) {
  return sf.getLineAndCharacterOfPosition(pos).line + 1;
}

function compactWs(s) {
  return s.replace(/\s+/g, ' ').trim();
}

/**
 * Extracts Jasmine spec declarations (including table-driven loops) and assertion strength from a *.spec.ts AST.
 */
function extractSpecCases(ts, sf) {
  const specCases = [];
  const suiteStack = [];
  const tableScopes = [];

  function walkScopeStatements(stmts, activeLoop) {
    const scopeMap = new Map();
    for (const st of stmts) {
      if (!ts.isVariableStatement(st)) {
        continue;
      }
      for (const decl of st.declarationList.declarations) {
        if (!ts.isIdentifier(decl.name) || !decl.initializer) {
          continue;
        }
        const rows = extractArrayLiteralRows(ts, sf, decl.initializer);
        if (rows !== null) {
          scopeMap.set(decl.name.text, rows);
        }
      }
    }
    tableScopes.push(scopeMap);
    for (const st of stmts) {
      walkSpecAst(st, activeLoop);
    }
    tableScopes.pop();
  }

  function walkSpecAst(node, activeLoop) {
    if (!node || ts.isTypeNode(node)) {
      return;
    }

    if (ts.isBlock(node)) {
      walkScopeStatements(node.statements, activeLoop);
      return;
    }

    if (ts.isForOfStatement(node)) {
      const rows = resolveTableRows(ts, sf, node.expression, tableScopes);
      const bindingSpec = extractForOfBinding(ts, node.initializer);
      if (rows !== null && bindingSpec !== null) {
        for (const row of rows) {
          walkSpecAst(node.statement, buildActiveLoopBinding(bindingSpec, row));
        }
        return;
      }
    }

    if (ts.isCallExpression(node)) {
      const callee = unwrapTypeAssertionsAndParens(ts, node.expression);

      if (ts.isIdentifier(callee)) {
        const fnName = callee.text;
        if (fnName === 'xdescribe' || fnName === 'xit') {
          return;
        }
        if ((fnName === 'describe' || fnName === 'fdescribe') && node.arguments.length >= 2) {
          const suiteTitle = evalSpecTitleExpr(ts, node.arguments[0], activeLoop);
          const cb = unwrapTypeAssertionsAndParens(ts, node.arguments[1]);
          if (suiteTitle !== null && isCallbackFunction(ts, cb)) {
            suiteStack.push({
              title: suiteTitle,
              headerLine: lineOfPos(sf, node.getStart(sf)),
            });
            walkSpecAst(cb.body, activeLoop);
            suiteStack.pop();
            return;
          }
        }
        if ((fnName === 'it' || fnName === 'fit') && node.arguments.length >= 1) {
          const specTitle = evalSpecTitleExpr(ts, node.arguments[0], activeLoop);
          if (specTitle !== null) {
            const fullName = joinJasmineFullName(suiteStack, specTitle);
            const itStartLine = lineOfPos(sf, node.getStart(sf));
            const itEndLine = lineOfPos(sf, node.getEnd());
            const startLine = activeLoop ? activeLoop.startLine : itStartLine;
            const endLine = activeLoop ? activeLoop.endLine : itEndLine;
            const cb = node.arguments.length >= 2 ? unwrapTypeAssertionsAndParens(ts, node.arguments[1]) : null;
            const weakAssertion = cb && isCallbackFunction(ts, cb) ? isWeakJasmineSpecCallback(ts, cb) : true;
            specCases.push({
              fullName,
              line: startLine,
              startLine,
              endLine,
              itStartLine,
              itEndLine,
              suiteHeaderLines: suiteStack.map((s) => s.headerLine),
              weakAssertion,
            });
          }
          return;
        }
      }

      if (
        ts.isPropertyAccessExpression(callee) &&
        ts.isIdentifier(callee.name) &&
        callee.name.text === 'forEach' &&
        node.arguments.length >= 1
      ) {
        const rows = resolveTableRows(ts, sf, callee.expression, tableScopes);
        const cb = unwrapTypeAssertionsAndParens(ts, node.arguments[0]);
        if (rows !== null && isCallbackFunction(ts, cb) && cb.parameters.length >= 1) {
          const bindingSpec = extractParamBinding(ts, cb.parameters[0].name);
          if (bindingSpec !== null) {
            for (const row of rows) {
              walkSpecAst(cb.body, buildActiveLoopBinding(bindingSpec, row));
            }
            return;
          }
        }
      }
    }

    ts.forEachChild(node, (child) => walkSpecAst(child, activeLoop));
  }

  walkScopeStatements(sf.statements, null);
  return specCases;
}

function joinJasmineFullName(suiteStack, specTitle) {
  const parts = [];
  for (const s of suiteStack) {
    if (s.title) {
      parts.push(s.title);
    }
  }
  if (specTitle) {
    parts.push(specTitle);
  }
  return parts.join(' ');
}

function resolveTableRows(ts, sf, expr, tableScopes) {
  const unwrapped = unwrapTypeAssertionsAndParens(ts, expr);
  if (ts.isIdentifier(unwrapped)) {
    for (let i = tableScopes.length - 1; i >= 0; i--) {
      if (tableScopes[i].has(unwrapped.text)) {
        return tableScopes[i].get(unwrapped.text);
      }
    }
    return null;
  }
  return extractArrayLiteralRows(ts, sf, unwrapped);
}

function extractArrayLiteralRows(ts, sf, expr) {
  const unwrapped = unwrapTypeAssertionsAndParens(ts, expr);
  if (!ts.isArrayLiteralExpression(unwrapped) || unwrapped.elements.length === 0) {
    return null;
  }
  const rows = [];
  for (const elem of unwrapped.elements) {
    const obj = unwrapTypeAssertionsAndParens(ts, elem);
    if (!ts.isObjectLiteralExpression(obj)) {
      return null;
    }
    const props = Object.create(null);
    for (const prop of obj.properties) {
      if (!ts.isPropertyAssignment(prop)) {
        continue;
      }
      const key = ts.isIdentifier(prop.name) || ts.isStringLiteral(prop.name) ? prop.name.text : null;
      if (!key) {
        continue;
      }
      const val = evalSpecTitleExpr(ts, prop.initializer, null);
      if (val !== null) {
        props[key] = val;
      }
    }
    rows.push({
      props,
      startLine: lineOfPos(sf, obj.getStart(sf)),
      endLine: lineOfPos(sf, obj.getEnd()),
    });
  }
  return rows;
}

function extractForOfBinding(ts, initializer) {
  if (!ts.isVariableDeclarationList(initializer) || initializer.declarations.length !== 1) {
    return null;
  }
  return extractParamBinding(ts, initializer.declarations[0].name);
}

function extractParamBinding(ts, bindingName) {
  if (ts.isIdentifier(bindingName)) {
    return { kind: 'ident', name: bindingName.text };
  }
  if (ts.isObjectBindingPattern(bindingName)) {
    const fields = [];
    for (const elem of bindingName.elements) {
      if (!ts.isIdentifier(elem.name)) {
        continue;
      }
      const localName = elem.name.text;
      const propName =
        elem.propertyName && (ts.isIdentifier(elem.propertyName) || ts.isStringLiteral(elem.propertyName))
          ? elem.propertyName.text
          : localName;
      fields.push({ localName, propName });
    }
    return { kind: 'destructured', fields };
  }
  return null;
}

function buildActiveLoopBinding(bindingSpec, row) {
  const bareVars = Object.create(null);
  let paramName = '';
  if (bindingSpec.kind === 'ident') {
    paramName = bindingSpec.name;
  } else {
    for (const f of bindingSpec.fields) {
      if (Object.prototype.hasOwnProperty.call(row.props, f.propName)) {
        bareVars[f.localName] = row.props[f.propName];
      }
    }
  }
  return {
    paramName,
    props: row.props,
    bareVars,
    startLine: row.startLine,
    endLine: row.endLine,
  };
}

function evalSpecTitleExpr(ts, expr, activeLoop) {
  const e = unwrapTypeAssertionsAndParens(ts, expr);
  if (!e) {
    return null;
  }
  if (ts.isStringLiteral(e) || ts.isNoSubstitutionTemplateLiteral(e)) {
    return e.text;
  }
  if (ts.isNumericLiteral(e) || ts.isBigIntLiteral(e)) {
    return e.text;
  }
  if (e.kind === ts.SyntaxKind.TrueKeyword) {
    return 'true';
  }
  if (e.kind === ts.SyntaxKind.FalseKeyword) {
    return 'false';
  }
  if (activeLoop) {
    if (ts.isIdentifier(e) && Object.prototype.hasOwnProperty.call(activeLoop.bareVars, e.text)) {
      return activeLoop.bareVars[e.text];
    }
    if (
      activeLoop.paramName &&
      ts.isPropertyAccessExpression(e) &&
      ts.isIdentifier(e.expression) &&
      e.expression.text === activeLoop.paramName &&
      ts.isIdentifier(e.name) &&
      Object.prototype.hasOwnProperty.call(activeLoop.props, e.name.text)
    ) {
      return activeLoop.props[e.name.text];
    }
  }
  if (ts.isTemplateExpression(e)) {
    let out = e.head.text;
    for (const span of e.templateSpans) {
      const part = evalSpecTitleExpr(ts, span.expression, activeLoop);
      if (part === null) {
        return null;
      }
      out += part + span.literal.text;
    }
    return out;
  }
  if (ts.isBinaryExpression(e) && e.operatorToken.kind === ts.SyntaxKind.PlusToken) {
    const left = evalSpecTitleExpr(ts, e.left, activeLoop);
    const right = evalSpecTitleExpr(ts, e.right, activeLoop);
    if (left !== null && right !== null) {
      return left + right;
    }
  }
  return null;
}

function isCallbackFunction(ts, node) {
  return Boolean(node && (ts.isArrowFunction(node) || ts.isFunctionExpression(node)));
}

function unwrapTypeAssertionsAndParens(ts, expr) {
  let cur = expr;
  while (
    cur &&
    (ts.isParenthesizedExpression(cur) ||
      ts.isAsExpression(cur) ||
      ts.isTypeAssertionExpression(cur) ||
      ts.isNonNullExpression(cur) ||
      (typeof ts.isSatisfiesExpression === 'function' && ts.isSatisfiesExpression(cur)))
  ) {
    cur = cur.expression;
  }
  return cur;
}

/**
 * Reports whether a Jasmine spec callback contains only trivial instance smoke checks (or zero assertions).
 */
function isWeakJasmineSpecCallback(ts, fnNode) {
  const localDecls = new Set();
  function collectLocals(n) {
    if (!n || ts.isTypeNode(n)) {
      return;
    }
    if (ts.isParameter(n) || ts.isVariableDeclaration(n)) {
      collectBindingIdentifiers(ts, n.name, localDecls);
    } else if (ts.isFunctionDeclaration(n) && n.name) {
      localDecls.add(n.name.text);
    }
    ts.forEachChild(n, collectLocals);
  }
  for (const p of fnNode.parameters) {
    collectLocals(p);
  }
  collectLocals(fnNode.body);

  let hasStrongAssertion = false;
  function walk(n) {
    if (!n || hasStrongAssertion || ts.isTypeNode(n)) {
      return;
    }
    if (ts.isCallExpression(n)) {
      const callee = unwrapTypeAssertionsAndParens(ts, n.expression);
      if (ts.isIdentifier(callee)) {
        const name = callee.text;
        if (name === 'fail' || name === 'expectAsync' || /^(assert|verify|check)[A-Z_]/.test(name)) {
          hasStrongAssertion = true;
          return;
        }
        if (name === 'expect') {
          if (!isTrivialSmokeExpectCall(ts, n, localDecls)) {
            hasStrongAssertion = true;
            return;
          }
        }
      }
    }
    ts.forEachChild(n, walk);
  }
  walk(fnNode.body);
  return !hasStrongAssertion;
}

function collectBindingIdentifiers(ts, nameNode, out) {
  if (!nameNode) {
    return;
  }
  if (ts.isIdentifier(nameNode)) {
    out.add(nameNode.text);
    return;
  }
  if (ts.isObjectBindingPattern(nameNode) || ts.isArrayBindingPattern(nameNode)) {
    for (const el of nameNode.elements) {
      if (ts.isBindingElement(el)) {
        collectBindingIdentifiers(ts, el.name, out);
      }
    }
  }
}

function isTrivialSmokeExpectCall(ts, expectCall, localDecls) {
  if (expectCall.arguments.length !== 1) {
    return false;
  }
  const target = unwrapTypeAssertionsAndParens(ts, expectCall.arguments[0]);
  if (!ts.isIdentifier(target) || localDecls.has(target.text)) {
    return false;
  }
  let cur = expectCall;
  while (
    cur.parent &&
    ts.isPropertyAccessExpression(cur.parent) &&
    ts.isIdentifier(cur.parent.name) &&
    cur.parent.name.text === 'withContext' &&
    cur.parent.parent &&
    ts.isCallExpression(cur.parent.parent)
  ) {
    cur = cur.parent.parent;
  }
  if (!cur.parent || !ts.isPropertyAccessExpression(cur.parent) || !ts.isIdentifier(cur.parent.name)) {
    return false;
  }
  const matcherName = cur.parent.name.text;
  if (matcherName !== 'toBeTruthy' && matcherName !== 'toBeDefined') {
    return false;
  }
  const matcherCall = cur.parent.parent;
  return Boolean(matcherCall && ts.isCallExpression(matcherCall) && matcherCall.arguments.length === 0);
}

main();
