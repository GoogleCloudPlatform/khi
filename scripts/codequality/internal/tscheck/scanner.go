// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tscheck

type tsScanMode int

const (
	tsModeCode tsScanMode = iota
	tsModeLineComment
	tsModeBlockComment
	tsModeSingleQuote
	tsModeDoubleQuote
	tsModeTemplate
)

type tsTokenScanner struct {
	src      []byte
	numLines int
	mode     tsScanMode
	line     int
	pos      int
	hasToken []bool
}

func classifyExecutableTSLines(src []byte, numLines int) []bool {
	sc := &tsTokenScanner{
		src:      src,
		numLines: numLines,
		mode:     tsModeCode,
		line:     1,
		hasToken: make([]bool, numLines+1),
	}
	sc.scanAll()
	return sc.hasToken
}

func (s *tsTokenScanner) scanAll() {
	for s.pos < len(s.src) && s.line <= s.numLines {
		ch := s.src[s.pos]
		if ch == '\n' {
			s.onNewline()
			s.pos++
			continue
		}
		s.stepNonNewline(ch)
		s.pos++
	}
}

func (s *tsTokenScanner) onNewline() {
	if s.mode == tsModeLineComment || s.mode == tsModeSingleQuote || s.mode == tsModeDoubleQuote {
		s.mode = tsModeCode
	}
	s.line++
}

func (s *tsTokenScanner) stepNonNewline(ch byte) {
	switch s.mode {
	case tsModeLineComment:
	case tsModeBlockComment:
		if ch == '*' && s.peekNext() == '/' {
			s.mode = tsModeCode
			s.pos++
		}
	case tsModeSingleQuote:
		s.scanQuoteChar(ch, '\'')
	case tsModeDoubleQuote:
		s.scanQuoteChar(ch, '"')
	case tsModeTemplate:
		s.scanTemplateChar(ch)
	case tsModeCode:
		s.scanCodeChar(ch)
	}
}

func (s *tsTokenScanner) scanQuoteChar(ch byte, quote byte) {
	if ch == '\\' && s.peekNext() != '\n' {
		s.pos++
		return
	}
	if ch == quote {
		s.mode = tsModeCode
	}
}

func (s *tsTokenScanner) scanTemplateChar(ch byte) {
	if ch != ' ' && ch != '\t' && ch != '\r' {
		s.hasToken[s.line] = true
	}
	if ch == '\\' && s.peekNext() != '\n' {
		s.pos++
		return
	}
	if ch == '`' {
		s.mode = tsModeCode
	}
}

func (s *tsTokenScanner) scanCodeChar(ch byte) {
	if ch == ' ' || ch == '\t' || ch == '\r' {
		return
	}
	if ch == '/' && (s.peekNext() == '/' || s.peekNext() == '*') {
		if s.peekNext() == '/' {
			s.mode = tsModeLineComment
		} else {
			s.mode = tsModeBlockComment
		}
		s.pos++
		return
	}
	switch ch {
	case '\'':
		s.mode = tsModeSingleQuote
		s.hasToken[s.line] = true
	case '"':
		s.mode = tsModeDoubleQuote
		s.hasToken[s.line] = true
	case '`':
		s.mode = tsModeTemplate
		s.hasToken[s.line] = true
	case '}', ')', ']', ';', ',':
	default:
		s.hasToken[s.line] = true
	}
}

func (s *tsTokenScanner) peekNext() byte {
	if s.pos+1 < len(s.src) {
		return s.src[s.pos+1]
	}
	return 0
}
