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

const fs = require('fs');
const path = require('path');
const { createRequire } = require('module');

const PREDECLARED_GLOBALS = new Set([
  'undefined',
  'NaN',
  'Infinity',
  'globalThis',
  'window',
  'document',
  'console',
  'Math',
  'Number',
  'String',
  'Boolean',
  'BigInt',
  'Symbol',
  'Object',
  'Array',
  'Map',
  'Set',
  'WeakMap',
  'WeakSet',
  'Date',
  'RegExp',
  'Error',
  'TypeError',
  'RangeError',
  'Promise',
  'JSON',
  'Intl',
  'Reflect',
  'Proxy',
  'Uint8Array',
  'Uint32Array',
  'Int32Array',
  'Float32Array',
  'Float64Array',
  'ArrayBuffer',
  'DataView',
  'URL',
  'URLSearchParams',
  'setTimeout',
  'clearTimeout',
  'setInterval',
  'clearInterval',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'structuredClone',
]);

function main() {
  const rawInput = fs.readFileSync(0, 'utf8');
  const req = JSON.parse(rawInput);
  if (!req.webDir) {
    process.stderr.write('missing webDir field for typescript resolution\n');
    process.exit(2);
  }
  const webRequire = createRequire(path.join(path.resolve(req.webDir), 'package.json'));
  const ts = webRequire('typescript');

  const minStmts = req.minStmts || 0;
  const minCodeLines = req.minCodeLines || 0;
  const keysOnly = Boolean(req.keysOnly);

  const filesOut = {};
  for (const fileItem of req.files || []) {
    const relPath = fileItem.path;
    const srcText = fileItem.content || '';
    if (isGeneratedHeader(srcText)) {
      continue;
    }
    const extracted = processSourceFile(ts, relPath, srcText, minStmts, minCodeLines, keysOnly);
    if (keysOnly) {
      filesOut[relPath] = { keys: extracted.keys };
    } else if (extracted.units.length > 0) {
      filesOut[relPath] = { units: extracted.units };
    }
  }

  process.stdout.write(JSON.stringify({ files: filesOut }));
}

function isGeneratedHeader(srcText) {
  const head = srcText.slice(0, 1024);
  return (
    (head.includes('Code generated') && head.includes('DO NOT EDIT')) ||
    head.includes('@generated') ||
    head.includes('AUTO-GENERATED')
  );
}

function processSourceFile(ts, relPath, srcText, minStmts, minCodeLines, keysOnly) {
  const sf = ts.createSourceFile(
    relPath,
    srcText,
    ts.ScriptTarget.ES2022,
    /* setParentNodes */ true,
    ts.ScriptKind.TS,
  );

  const candidates = [];
  for (const stmt of sf.statements) {
    collectTopLevelCandidates(ts, sf, stmt, candidates);
  }

  if (keysOnly) {
    return { keys: candidates.map((c) => `${c.kind}:${c.name}`), units: [] };
  }

  const units = [];
  for (const cand of candidates) {
    const declSlice = srcText.slice(cand.startPos, cand.endPos);
    const codeLines = countCodeLines(ts, declSlice);
    if (cand.stmtCount < minStmts || codeLines < minCodeLines) {
      continue;
    }
    const startLine = sf.getLineAndCharacterOfPosition(cand.startPos).line + 1;
    const endLine = sf.getLineAndCharacterOfPosition(cand.endPos).line + 1;
    const normText = normalizeCandidate(ts, cand, declSlice);

    units.push({
      name: cand.name,
      kind: cand.kind,
      startLine,
      endLine,
      stmtCount: cand.stmtCount,
      codeLines,
      rawDecl: declSlice,
      normText,
    });
  }
  return { keys: [], units };
}

function collectTopLevelCandidates(ts, sf, stmt, out) {
  if (ts.isFunctionDeclaration(stmt) && stmt.body && stmt.name) {
    out.push({
      kind: 'func',
      name: stmt.name.text,
      startPos: stmt.getStart(sf),
      endPos: stmt.getEnd(),
      stmtCount: countBodyStatements(ts, stmt.body),
      wrapMode: 'top_func',
    });
    return;
  }

  if (ts.isVariableStatement(stmt)) {
    for (const decl of stmt.declarationList.declarations) {
      if (!ts.isIdentifier(decl.name) || !decl.initializer) {
        continue;
      }
      const varName = decl.name.text;
      const init = unwrapParensAndAs(ts, decl.initializer);
      const useStmtSpan = stmt.declarationList.declarations.length === 1;
      const startPos = useStmtSpan ? stmt.getStart(sf) : decl.getStart(sf);
      const endPos = useStmtSpan ? stmt.getEnd() : decl.getEnd();

      if (ts.isArrowFunction(init) || ts.isFunctionExpression(init)) {
        out.push({
          kind: 'func',
          name: varName,
          startPos,
          endPos,
          stmtCount: countFunctionLikeStatements(ts, init),
          wrapMode: useStmtSpan ? 'top_var_stmt' : 'top_var_decl',
        });
      } else if (ts.isCallExpression(init)) {
        const closureStmts = countImmediateCallClosureStatements(ts, init);
        if (closureStmts.found) {
          out.push({
            kind: 'closure',
            name: varName,
            startPos,
            endPos,
            stmtCount: closureStmts.count,
            wrapMode: useStmtSpan ? 'top_closure_stmt' : 'top_closure_decl',
          });
        }
      }
    }
    return;
  }

  if (ts.isClassDeclaration(stmt)) {
    const className = stmt.name ? stmt.name.text : 'AnonymousClass';
    for (const member of stmt.members) {
      collectClassMemberCandidate(ts, sf, className, member, out);
    }
  }
}

function collectClassMemberCandidate(ts, sf, className, member, out) {
  if (
    (ts.isMethodDeclaration(member) ||
      ts.isGetAccessorDeclaration(member) ||
      ts.isSetAccessorDeclaration(member) ||
      ts.isConstructorDeclaration(member)) &&
    member.body
  ) {
    const memberName = resolveMemberName(ts, member);
    if (!memberName) {
      return;
    }
    out.push({
      kind: 'method',
      name: `${className}.${memberName}`,
      startPos: memberStartWithoutDecorators(ts, sf, member),
      endPos: member.getEnd(),
      stmtCount: countBodyStatements(ts, member.body),
      wrapMode: 'class_member',
    });
    return;
  }

  if (ts.isPropertyDeclaration(member) && member.initializer) {
    const propName = propertyNameText(ts, member.name);
    if (!propName) {
      return;
    }
    const init = unwrapParensAndAs(ts, member.initializer);
    const startPos = memberStartWithoutDecorators(ts, sf, member);
    if (ts.isArrowFunction(init) || ts.isFunctionExpression(init)) {
      out.push({
        kind: 'method',
        name: `${className}.${propName}`,
        startPos,
        endPos: member.getEnd(),
        stmtCount: countFunctionLikeStatements(ts, init),
        wrapMode: 'class_prop_func',
      });
    } else if (ts.isCallExpression(init)) {
      const closureStmts = countImmediateCallClosureStatements(ts, init);
      if (closureStmts.found) {
        out.push({
          kind: 'closure',
          name: `${className}.${propName}`,
          startPos,
          endPos: member.getEnd(),
          stmtCount: closureStmts.count,
          wrapMode: 'class_prop_closure',
        });
      }
    }
  }
}

function memberStartWithoutDecorators(ts, sf, member) {
  if (!ts.canHaveDecorators(member)) {
    return member.getStart(sf);
  }
  const decorators = ts.getDecorators(member);
  if (!decorators || decorators.length === 0) {
    return member.getStart(sf);
  }
  const lastDecEnd = decorators[decorators.length - 1].getEnd();
  const fullText = sf.text;
  let pos = lastDecEnd;
  while (pos < member.getEnd() && /\s/.test(fullText[pos])) {
    pos++;
  }
  return pos;
}

function resolveMemberName(ts, member) {
  if (ts.isConstructorDeclaration(member)) {
    return 'constructor';
  }
  const base = propertyNameText(ts, member.name);
  if (!base) {
    return '';
  }
  if (ts.isGetAccessorDeclaration(member)) {
    return `get_${base}`;
  }
  if (ts.isSetAccessorDeclaration(member)) {
    return `set_${base}`;
  }
  return base;
}

function propertyNameText(ts, nameNode) {
  if (!nameNode) {
    return '';
  }
  if (
    ts.isIdentifier(nameNode) ||
    ts.isPrivateIdentifier(nameNode) ||
    ts.isStringLiteral(nameNode) ||
    ts.isNumericLiteral(nameNode)
  ) {
    return nameNode.text;
  }
  return '';
}

function unwrapParensAndAs(ts, expr) {
  let cur = expr;
  while (cur) {
    if (ts.isParenthesizedExpression(cur) || ts.isAsExpression(cur) || ts.isSatisfiesExpression(cur)) {
      cur = cur.expression;
    } else {
      break;
    }
  }
  return cur;
}

function countFunctionLikeStatements(ts, fnNode) {
  if (!fnNode.body) {
    return 0;
  }
  if (ts.isBlock(fnNode.body)) {
    return countBodyStatements(ts, fnNode.body);
  }
  return 1;
}

function countImmediateCallClosureStatements(ts, callExpr) {
  let total = 0;
  let found = false;

  function scanArg(node) {
    const unwrapped = unwrapParensAndAs(ts, node);
    if (ts.isArrowFunction(unwrapped) || ts.isFunctionExpression(unwrapped)) {
      found = true;
      total += countFunctionLikeStatements(ts, unwrapped);
      return;
    }
    if (ts.isCallExpression(unwrapped)) {
      for (const arg of unwrapped.arguments) {
        scanArg(arg);
      }
    }
  }

  for (const arg of callExpr.arguments) {
    scanArg(arg);
  }
  return { found, count: total };
}

function countBodyStatements(ts, blockNode) {
  let count = 0;
  function walk(node) {
    if (node !== blockNode && isStatementNode(ts, node) && !ts.isBlock(node)) {
      count++;
    }
    ts.forEachChild(node, walk);
  }
  walk(blockNode);
  return count;
}

function isStatementNode(ts, node) {
  return (
    (node.kind >= ts.SyntaxKind.FirstStatement &&
      node.kind <= ts.SyntaxKind.LastStatement) ||
    ts.isIfStatement(node) ||
    ts.isReturnStatement(node) ||
    ts.isExpressionStatement(node) ||
    ts.isVariableStatement(node) ||
    ts.isForStatement(node) ||
    ts.isForOfStatement(node) ||
    ts.isForInStatement(node) ||
    ts.isWhileStatement(node) ||
    ts.isDoStatement(node) ||
    ts.isSwitchStatement(node) ||
    ts.isTryStatement(node) ||
    ts.isThrowStatement(node) ||
    ts.isBreakStatement(node) ||
    ts.isContinueStatement(node)
  );
}

function countCodeLines(ts, snippetText) {
  const scanner = ts.createScanner(
    ts.ScriptTarget.ES2022,
    /* skipTrivia */ true,
    ts.LanguageVariant.Standard,
    snippetText,
  );
  const lineOffsets = computeLineStarts(snippetText);
  const linesWithCode = new Set();

  let tok = scanner.scan();
  while (tok !== ts.SyntaxKind.EndOfFileToken) {
    const tokStart = scanner.getTokenStart();
    const tokText = scanner.getTokenText();
    if (!/^[,;})\]\s]*$/.test(tokText)) {
      const ln = offsetToLine(lineOffsets, tokStart);
      linesWithCode.add(ln);
    }
    tok = scanner.scan();
  }
  return linesWithCode.size;
}

function computeLineStarts(text) {
  const starts = [0];
  for (let i = 0; i < text.length; i++) {
    if (text.charCodeAt(i) === 10) {
      starts.push(i + 1);
    }
  }
  return starts;
}

function offsetToLine(starts, offset) {
  let lo = 0;
  let hi = starts.length - 1;
  while (lo <= hi) {
    const mid = (lo + hi) >>> 1;
    if (starts[mid] <= offset) {
      lo = mid + 1;
    } else {
      hi = mid - 1;
    }
  }
  return hi;
}

function wrapSnippetForParsing(wrapMode, declSlice) {
  switch (wrapMode) {
    case 'top_var_decl':
    case 'top_closure_decl':
      return `const ${declSlice};`;
    case 'class_member':
    case 'class_prop_func':
    case 'class_prop_closure':
      return `class RecvType {\n${declSlice}\n}`;
    default:
      return declSlice;
  }
}

function normalizeCandidate(ts, cand, declSlice) {
  const wrapped = wrapSnippetForParsing(cand.wrapMode, declSlice);
  const sf = ts.createSourceFile(
    'snippet.ts',
    wrapped,
    ts.ScriptTarget.ES2022,
    /* setParentNodes */ true,
    ts.ScriptKind.TS,
  );
  if (sf.statements.length === 0) {
    return collapseBlankLines(declSlice);
  }

  const mapping = new Map();
  let nextID = 1;
  function registerName(name) {
    if (!name || PREDECLARED_GLOBALS.has(name)) {
      return;
    }
    if (!mapping.has(name)) {
      mapping.set(name, `v${nextID++}`);
    }
  }

  collectAllLocalBindings(ts, sf, registerName);

  const transformer = (context) => {
    const visit = (node) => {
      if (ts.isFunctionDeclaration(node) && node.parent === sf) {
        const cleanMods = filterSemanticModifiers(ts, node.modifiers);
        const visitedTypeParams = node.typeParameters
          ? node.typeParameters.map((tp) => ts.visitNode(tp, visit))
          : undefined;
        const visitedParams = node.parameters.map((p) => ts.visitNode(p, visit));
        const visitedType = node.type ? ts.visitNode(node.type, visit) : undefined;
        const visitedBody = node.body ? ts.visitNode(node.body, visit) : undefined;
        return ts.factory.updateFunctionDeclaration(
          node,
          cleanMods,
          node.asteriskToken,
          ts.factory.createIdentifier('targetFunc'),
          visitedTypeParams,
          visitedParams,
          visitedType,
          visitedBody,
        );
      }

      if (ts.isVariableStatement(node) && node.parent === sf) {
        const decls = node.declarationList.declarations.map((d, idx) => {
          const baseName = cand.kind === 'closure' ? 'targetClosure' : 'targetFunc';
          const targetName = idx === 0 ? baseName : `${baseName}${idx + 1}`;
          return ts.factory.updateVariableDeclaration(
            d,
            ts.factory.createIdentifier(targetName),
            d.exclamationToken,
            d.type ? ts.visitNode(d.type, visit) : undefined,
            d.initializer ? ts.visitNode(d.initializer, visit) : undefined,
          );
        });
        return ts.factory.updateVariableStatement(
          node,
          undefined,
          ts.factory.createVariableDeclarationList(decls, ts.NodeFlags.Const),
        );
      }

      if (ts.isClassDeclaration(node) && node.parent === sf) {
        const members = node.members.map((m) => renameClassMember(ts, m, cand.kind, visit));
        return ts.factory.updateClassDeclaration(
          node,
          undefined,
          ts.factory.createIdentifier('RecvType'),
          undefined,
          undefined,
          members,
        );
      }

      if (ts.isPropertyAccessExpression(node)) {
        const expr = ts.visitNode(node.expression, visit);
        return ts.factory.updatePropertyAccessExpression(node, expr, node.name);
      }

      if (ts.isPropertyAssignment(node) && ts.isIdentifier(node.name)) {
        const init = ts.visitNode(node.initializer, visit);
        return ts.factory.updatePropertyAssignment(node, node.name, init);
      }

      if (ts.isPropertySignature(node) && ts.isIdentifier(node.name)) {
        return ts.factory.updatePropertySignature(
          node,
          node.modifiers,
          node.name,
          node.questionToken,
          node.type ? ts.visitNode(node.type, visit) : undefined,
        );
      }

      if (ts.isMethodSignature(node) && ts.isIdentifier(node.name)) {
        return ts.factory.updateMethodSignature(
          node,
          node.modifiers,
          node.name,
          node.questionToken,
          node.typeParameters ? node.typeParameters.map((tp) => ts.visitNode(tp, visit)) : undefined,
          node.parameters.map((p) => ts.visitNode(p, visit)),
          node.type ? ts.visitNode(node.type, visit) : undefined,
        );
      }

      if (ts.isShorthandPropertyAssignment(node)) {
        const mapped = mapping.get(node.name.text);
        if (mapped) {
          return ts.factory.createPropertyAssignment(
            ts.factory.createIdentifier(node.name.text),
            ts.factory.createIdentifier(mapped),
          );
        }
        return node;
      }

      if (ts.isBindingElement(node)) {
        if (!node.propertyName && ts.isIdentifier(node.name)) {
          const origKey = node.name.text;
          const mapped = mapping.get(origKey);
          if (mapped && ts.isObjectBindingPattern(node.parent)) {
            return ts.factory.updateBindingElement(
              node,
              node.dotDotDotToken,
              ts.factory.createIdentifier(origKey),
              ts.factory.createIdentifier(mapped),
              node.initializer ? ts.visitNode(node.initializer, visit) : undefined,
            );
          }
        } else if (node.propertyName && ts.isIdentifier(node.propertyName)) {
          return ts.factory.updateBindingElement(
            node,
            node.dotDotDotToken,
            node.propertyName,
            ts.visitNode(node.name, visit),
            node.initializer ? ts.visitNode(node.initializer, visit) : undefined,
          );
        }
      }

      if (ts.isQualifiedName(node)) {
        const left = ts.visitNode(node.left, visit);
        return ts.factory.updateQualifiedName(node, left, node.right);
      }

      if (ts.isIdentifier(node)) {
        const mapped = mapping.get(node.text);
        if (mapped) {
          return ts.factory.createIdentifier(mapped);
        }
      }

      return ts.visitEachChild(node, visit, context);
    };
    return (rootNode) => ts.visitNode(rootNode, visit);
  };

  const result = ts.transform(sf, [transformer]);
  const printer = ts.createPrinter({
    removeComments: true,
    newLine: ts.NewLineKind.LineFeed,
  });
  const printed = printer.printFile(result.transformed[0]);
  result.dispose();
  return collapseBlankLines(printed);
}

function renameClassMember(ts, member, unitKind, visit) {
  const cleanMods = filterSemanticModifiers(ts, member.modifiers);
  if (ts.isMethodDeclaration(member)) {
    return ts.factory.updateMethodDeclaration(
      member,
      cleanMods,
      member.asteriskToken,
      ts.factory.createIdentifier('targetMethod'),
      member.questionToken,
      member.typeParameters ? member.typeParameters.map((tp) => ts.visitNode(tp, visit)) : undefined,
      member.parameters.map((p) => ts.visitNode(p, visit)),
      member.type ? ts.visitNode(member.type, visit) : undefined,
      member.body ? ts.visitNode(member.body, visit) : undefined,
    );
  }
  if (ts.isGetAccessorDeclaration(member)) {
    return ts.factory.updateGetAccessorDeclaration(
      member,
      cleanMods,
      ts.factory.createIdentifier('targetGetter'),
      member.parameters.map((p) => ts.visitNode(p, visit)),
      member.type ? ts.visitNode(member.type, visit) : undefined,
      member.body ? ts.visitNode(member.body, visit) : undefined,
    );
  }
  if (ts.isSetAccessorDeclaration(member)) {
    return ts.factory.updateSetAccessorDeclaration(
      member,
      cleanMods,
      ts.factory.createIdentifier('targetSetter'),
      member.parameters.map((p) => ts.visitNode(p, visit)),
      member.body ? ts.visitNode(member.body, visit) : undefined,
    );
  }
  if (ts.isConstructorDeclaration(member)) {
    return ts.factory.updateConstructorDeclaration(
      member,
      cleanMods,
      member.parameters.map((p) => ts.visitNode(p, visit)),
      member.body ? ts.visitNode(member.body, visit) : undefined,
    );
  }
  if (ts.isPropertyDeclaration(member)) {
    const targetName = unitKind === 'closure' ? 'targetClosure' : 'targetMethod';
    return ts.factory.updatePropertyDeclaration(
      member,
      cleanMods,
      ts.factory.createIdentifier(targetName),
      member.questionToken || member.exclamationToken,
      member.type ? ts.visitNode(member.type, visit) : undefined,
      member.initializer ? ts.visitNode(member.initializer, visit) : undefined,
    );
  }
  return ts.visitNode(member, visit);
}

function filterSemanticModifiers(ts, modifiers) {
  if (!modifiers) {
    return undefined;
  }
  const kept = modifiers.filter(
    (m) =>
      m.kind === ts.SyntaxKind.AsyncKeyword ||
      m.kind === ts.SyntaxKind.StaticKeyword,
  );
  return kept.length > 0 ? kept : undefined;
}

function collectAllLocalBindings(ts, sf, registerName) {
  function registerBindingName(nameNode) {
    if (!nameNode) {
      return;
    }
    if (ts.isIdentifier(nameNode)) {
      registerName(nameNode.text);
      return;
    }
    if (ts.isObjectBindingPattern(nameNode) || ts.isArrayBindingPattern(nameNode)) {
      for (const el of nameNode.elements) {
        if (ts.isBindingElement(el)) {
          registerBindingName(el.name);
        }
      }
    }
  }

  function walk(node) {
    if (ts.isTypeParameterDeclaration(node) && ts.isIdentifier(node.name)) {
      registerName(node.name.text);
    } else if (ts.isParameter(node)) {
      registerBindingName(node.name);
    } else if (ts.isVariableDeclaration(node)) {
      const isTopVar =
        node.parent &&
        ts.isVariableDeclarationList(node.parent) &&
        node.parent.parent &&
        ts.isVariableStatement(node.parent.parent) &&
        node.parent.parent.parent === sf;
      if (!isTopVar) {
        registerBindingName(node.name);
      }
    } else if (ts.isFunctionDeclaration(node) && node.parent !== sf && node.name) {
      registerName(node.name.text);
    } else if (ts.isFunctionExpression(node) && node.name) {
      registerName(node.name.text);
    }
    ts.forEachChild(node, walk);
  }

  walk(sf);
}

function collapseBlankLines(text) {
  return text
    .split('\n')
    .filter((ln) => ln.trim() !== '')
    .join('\n')
    .trim();
}

main();
