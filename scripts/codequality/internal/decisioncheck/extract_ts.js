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

function main() {
  const rawInput = fs.readFileSync(0, 'utf8');
  const req = JSON.parse(rawInput);
  if (!req.webDir) {
    process.stderr.write('missing webDir field for typescript resolution\n');
    process.exit(2);
  }
  const webRequire = createRequire(
    path.join(path.resolve(req.webDir), 'package.json'),
  );
  const ts = webRequire('typescript');

  const isSpecMode = req.mode === 'specs';
  const filesOut = {};
  for (const fileItem of req.files || []) {
    const relPath = fileItem.path;
    const srcText = fileItem.content || '';
    if (!isSpecMode && isGeneratedHeader(srcText)) {
      continue;
    }
    const sf = ts.createSourceFile(
      relPath,
      srcText,
      ts.ScriptTarget.ES2022,
      /* setParentNodes */ true,
      ts.ScriptKind.TS,
    );
    const items = isSpecMode
      ? extractSpecItems(ts, sf, srcText)
      : extractProdItems(ts, sf, srcText);
    if (items.length > 0) {
      filesOut[relPath] = items;
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

function extractProdItems(ts, sf, srcText) {
  const items = [];
  for (const stmt of sf.statements) {
    collectTopLevelDecl(ts, sf, srcText, stmt, items);
  }
  return items;
}

function collectTopLevelDecl(ts, sf, srcText, stmt, out) {
  if (ts.isClassDeclaration(stmt)) {
    if (!stmt.name || !stmt.name.text) {
      return;
    }
    const className = stmt.name.text;
    const members = [];
    for (const member of stmt.members) {
      collectClassMemberDecl(ts, sf, srcText, className, member, members);
    }
    const memberRanges = members.map((m) => [m.startLine, m.endLine]);
    const classItem = buildItem(
      ts,
      sf,
      srcText,
      stmt,
      stmt.getStart(sf),
      memberStartWithoutDecorators(ts, sf, stmt),
      stmt.getEnd(),
      className,
      'class',
      false,
    );
    classItem.memberRanges = memberRanges;
    out.push(classItem, ...members);
    return;
  }

  if (ts.isFunctionDeclaration(stmt)) {
    if (!stmt.body || !stmt.name || !stmt.name.text) {
      return;
    }
    out.push(
      buildItem(
        ts,
        sf,
        srcText,
        stmt,
        stmt.getStart(sf),
        stmt.getStart(sf),
        stmt.getEnd(),
        stmt.name.text,
        'function',
        true,
      ),
    );
    return;
  }

  if (ts.isInterfaceDeclaration(stmt)) {
    if (!stmt.name || !stmt.name.text) {
      return;
    }
    out.push(
      buildItem(
        ts,
        sf,
        srcText,
        stmt,
        stmt.getStart(sf),
        stmt.getStart(sf),
        stmt.getEnd(),
        stmt.name.text,
        'interface',
        false,
      ),
    );
    return;
  }

  if (ts.isEnumDeclaration(stmt)) {
    if (!stmt.name || !stmt.name.text) {
      return;
    }
    out.push(
      buildItem(
        ts,
        sf,
        srcText,
        stmt,
        stmt.getStart(sf),
        stmt.getStart(sf),
        stmt.getEnd(),
        stmt.name.text,
        'enum',
        false,
      ),
    );
    return;
  }

  if (ts.isTypeAliasDeclaration(stmt)) {
    if (!stmt.name || !stmt.name.text) {
      return;
    }
    out.push(
      buildItem(
        ts,
        sf,
        srcText,
        stmt,
        stmt.getStart(sf),
        stmt.getStart(sf),
        stmt.getEnd(),
        stmt.name.text,
        'type',
        false,
      ),
    );
    return;
  }

  if (ts.isVariableStatement(stmt)) {
    const useStmtSpan = stmt.declarationList.declarations.length === 1;
    for (const decl of stmt.declarationList.declarations) {
      if (!ts.isIdentifier(decl.name) || !decl.name.text || !decl.initializer) {
        continue;
      }
      if (!isCallableInitializer(ts, decl.initializer)) {
        continue;
      }
      const commentNode = useStmtSpan ? stmt : decl;
      out.push(
        buildItem(
          ts,
          sf,
          srcText,
          commentNode,
          commentNode.getStart(sf),
          decl.getStart(sf),
          commentNode.getEnd(),
          decl.name.text,
          'function',
          true,
        ),
      );
    }
  }
}

function collectClassMemberDecl(ts, sf, srcText, className, member, out) {
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
    out.push(
      buildItem(
        ts,
        sf,
        srcText,
        member,
        member.getStart(sf),
        memberStartWithoutDecorators(ts, sf, member),
        member.getEnd(),
        `${className}.${memberName}`,
        'method',
        true,
      ),
    );
    return;
  }

  if (ts.isPropertyDeclaration(member) && member.initializer) {
    const propName = propertyNameText(ts, member.name);
    if (!propName || !isCallableInitializer(ts, member.initializer)) {
      return;
    }
    out.push(
      buildItem(
        ts,
        sf,
        srcText,
        member,
        member.getStart(sf),
        memberStartWithoutDecorators(ts, sf, member),
        member.getEnd(),
        `${className}.${propName}`,
        'method',
        true,
      ),
    );
  }
}

function isCallableInitializer(ts, expr) {
  const init = unwrapParensAndAs(ts, expr);
  if (!init) {
    return false;
  }
  if (ts.isArrowFunction(init) || ts.isFunctionExpression(init)) {
    return true;
  }
  return ts.isCallExpression(init) && hasImmediateCallClosure(ts, init);
}

function hasImmediateCallClosure(ts, callExpr) {
  for (const arg of callExpr.arguments) {
    const unwrapped = unwrapParensAndAs(ts, arg);
    if (!unwrapped) {
      continue;
    }
    if (ts.isArrowFunction(unwrapped) || ts.isFunctionExpression(unwrapped)) {
      return true;
    }
    if (ts.isCallExpression(unwrapped) && hasImmediateCallClosure(ts, unwrapped)) {
      return true;
    }
  }
  return false;
}

function unwrapParensAndAs(ts, expr) {
  let cur = expr;
  while (cur) {
    if (
      ts.isParenthesizedExpression(cur) ||
      ts.isAsExpression(cur) ||
      ts.isSatisfiesExpression(cur)
    ) {
      cur = cur.expression;
    } else {
      break;
    }
  }
  return cur;
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

function extractSpecItems(ts, sf, srcText) {
  const suiteStack = [];
  const items = [];

  function walk(node) {
    if (!node) {
      return;
    }
    if (ts.isCallExpression(node) && ts.isIdentifier(node.expression)) {
      const fnName = node.expression.text;
      if (
        (fnName === 'describe' ||
          fnName === 'fdescribe' ||
          fnName === 'xdescribe') &&
        node.arguments.length >= 2 &&
        ts.isStringLiteralLike(node.arguments[0])
      ) {
        suiteStack.push(node.arguments[0].text);
        ts.forEachChild(node, walk);
        suiteStack.pop();
        return;
      }
      if (
        (fnName === 'it' || fnName === 'fit' || fnName === 'xit') &&
        node.arguments.length >= 1 &&
        ts.isStringLiteralLike(node.arguments[0])
      ) {
        const commentNode = getSpecCommentNode(ts, node);
        items.push(
          buildItem(
            ts,
            sf,
            srcText,
            commentNode,
            node.getStart(sf),
            node.getStart(sf),
            node.getEnd(),
            formatSpecName(suiteStack, node.arguments[0].text),
            'spec',
            false,
          ),
        );
        return;
      }
    }
    ts.forEachChild(node, walk);
  }

  walk(sf);
  return items;
}

function getSpecCommentNode(ts, callExpr) {
  let cur = callExpr;
  if (cur.parent && ts.isParenthesizedExpression(cur.parent)) {
    cur = cur.parent;
  }
  if (cur.parent && ts.isExpressionStatement(cur.parent)) {
    return cur.parent;
  }
  return callExpr;
}

function formatSpecName(suiteStack, specTitle) {
  return [...suiteStack, JSON.stringify(specTitle)].join(' > ');
}

function buildItem(
  ts,
  sf,
  srcText,
  commentNode,
  fallbackStartPos,
  declLinePos,
  endPos,
  name,
  kind,
  isCallable,
) {
  const fallbackStartLine = lineOfPos(sf, fallbackStartPos);
  const leading = extractLeadingComment(
    ts,
    sf,
    srcText,
    commentNode,
    fallbackStartLine,
  );
  return {
    line: lineOfPos(sf, declLinePos),
    startLine: leading.startLine,
    endLine: lineOfPos(sf, endPos),
    name,
    kind,
    doc: leading.doc,
    isCallable,
  };
}

function extractLeadingComment(
  ts,
  sf,
  srcText,
  commentNode,
  fallbackStartLine,
) {
  const ranges = ts.getLeadingCommentRanges(srcText, commentNode.getFullStart());
  if (!ranges || ranges.length === 0) {
    return { doc: '', startLine: fallbackStartLine };
  }

  const attached = [];
  let attachedStartLine = fallbackStartLine;
  let expectedNextLine = fallbackStartLine;

  for (let i = ranges.length - 1; i >= 0; i--) {
    if (isTrailingCommentOnCodeLine(srcText, ranges, i)) {
      break;
    }
    const range = ranges[i];
    const cStartLine = lineOfPos(sf, range.pos);
    const cEndLine = lineOfPos(sf, range.end);
    if (cEndLine < expectedNextLine - 1) {
      break;
    }

    const rawText = srcText.slice(range.pos, range.end);
    const isBlock = range.kind === ts.SyntaxKind.MultiLineCommentTrivia;
    const cleaned = isBlock
      ? cleanBlockComment(rawText)
      : rawText.slice(2).trim();

    if (isDirectiveOrLicense(cleaned)) {
      expectedNextLine = cStartLine;
      continue;
    }
    if (cleaned === '') {
      break;
    }
    if (isBlock) {
      if (attached.length === 0) {
        attached.push(cleaned);
        attachedStartLine = cStartLine;
      }
      break;
    }
    attached.push(cleaned);
    attachedStartLine = cStartLine;
    expectedNextLine = cStartLine;
  }

  if (attached.length === 0) {
    return { doc: '', startLine: fallbackStartLine };
  }
  attached.reverse();
  return { doc: attached.join('\n'), startLine: attachedStartLine };
}

function isTrailingCommentOnCodeLine(srcText, ranges, idx) {
  const startLineOffset = srcText.lastIndexOf('\n', ranges[idx].pos - 1) + 1;
  let firstOnLinePos = ranges[idx].pos;
  for (let j = idx - 1; j >= 0 && ranges[j].pos >= startLineOffset; j--) {
    firstOnLinePos = ranges[j].pos;
  }
  return /\S/.test(srcText.slice(startLineOffset, firstOnLinePos));
}

function cleanBlockComment(rawText) {
  const body = rawText.endsWith('*/')
    ? rawText.slice(2, -2)
    : rawText.slice(2);
  const cleaned = [];
  for (const line of body.split('\n')) {
    let trimmed = line.trim();
    if (trimmed.startsWith('*')) {
      trimmed = trimmed.slice(1).trim();
    }
    if (trimmed !== '') {
      cleaned.push(trimmed);
    }
  }
  return cleaned.join('\n');
}

function isDirectiveOrLicense(text) {
  const lower = text.trim().toLowerCase();
  return (
    lower.startsWith('eslint-') ||
    lower.startsWith('prettier-ignore') ||
    lower.startsWith('@ts-') ||
    text.includes('CODEQUALITY_ALLOW') ||
    text.includes('Licensed under the Apache License') ||
    text.includes('Copyright 20')
  );
}

function lineOfPos(sf, pos) {
  return sf.getLineAndCharacterOfPosition(pos).line + 1;
}

main();
