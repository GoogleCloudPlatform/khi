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

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

const (
	tsBrowserRuntimeFileName = "zzz_khi_mutant_runtime.js"
	tsBrowserRuntimeTemplate = `(function () {
  'use strict';

  function parseSelectedMutant() {
    var k = globalThis.__karma__;
    if (!k || !k.config || !Array.isArray(k.config.args)) {
      return 0;
    }
    for (var i = 0; i < k.config.args.length; i++) {
      var arg = String(k.config.args[i]);
      if (arg.indexOf('--khi-mutant=') === 0) {
        var parsed = parseInt(arg.slice('--khi-mutant='.length), 10);
        return isNaN(parsed) ? 0 : parsed;
      }
    }
    return 0;
  }

  var selectedId = parseSelectedMutant();
  var currentSpecId = null;
  var currentSpecFullName = null;
  var currentSpecMutantSeen = null;
  var currentSpecSlotSeen = null;
  var currentSpecProbeSlotSeen = null;
  var topLevelSeen = new Uint8Array(%d);
  var topLevelHits = [];
  var specHits = Object.create(null);
  var specOrder = [];
  var specPolarities = Object.create(null);

  function recordSlot(id, slot, suffix) {
    if (id < 0) {
      var pId = -id;
      if (pId >= %d) {
        return;
      }
      if (currentSpecId !== null && currentSpecProbeSlotSeen !== null) {
        var pSlotIdx = pId * 3 + slot;
        if (!currentSpecProbeSlotSeen[pSlotIdx]) {
          currentSpecProbeSlotSeen[pSlotIdx] = 1;
          var pPolList = specPolarities[currentSpecId];
          if (!pPolList) {
            pPolList = [];
            specPolarities[currentSpecId] = pPolList;
          }
          pPolList.push('P' + String(pId) + suffix);
        }
      }
      return;
    }
    if (id === 0 || id >= %d) {
      return;
    }
    if (currentSpecFullName !== null && currentSpecMutantSeen !== null) {
      if (!currentSpecMutantSeen[id]) {
        currentSpecMutantSeen[id] = 1;
        var hitList = specHits[currentSpecFullName];
        if (!hitList) {
          hitList = [];
          specHits[currentSpecFullName] = hitList;
        }
        hitList.push(id);
      }
      if (currentSpecId !== null && currentSpecSlotSeen !== null) {
        var slotIdx = id * 3 + slot;
        if (!currentSpecSlotSeen[slotIdx]) {
          currentSpecSlotSeen[slotIdx] = 1;
          var polList = specPolarities[currentSpecId];
          if (!polList) {
            polList = [];
            specPolarities[currentSpecId] = polList;
          }
          polList.push(String(id) + suffix);
        }
      }
      return;
    }
    if (!topLevelSeen[id]) {
      topLevelSeen[id] = 1;
      topLevelHits.push(id);
    }
  }

  function recordBool(id, v) {
    if (v) {
      recordSlot(id, 2, ':T');
    } else {
      recordSlot(id, 1, ':F');
    }
  }

  function active(id) {
    if (selectedId === 0) {
      recordSlot(id, 0, ':S');
      return false;
    }
    return selectedId === id;
  }

  globalThis.__khiMut = {
    active: active,
    flip: function (id, v) {
      if (selectedId === 0) {
        recordBool(id, Boolean(v));
        return v;
      }
      return selectedId === id ? !v : v;
    },
    cond: function (idTrue, idFalse, cond) {
      if (selectedId === 0) {
        var b = Boolean(cond);
        recordBool(idTrue, b);
        recordBool(idFalse, b);
        return cond;
      }
      if (selectedId === idTrue) return true;
      if (selectedId === idFalse) return false;
      return cond;
    },
    and: function (id, lhsFn, rhsFn) {
      var l = lhsFn();
      if (selectedId === 0) {
        var res = l ? rhsFn() : l;
        recordBool(id, Boolean(res));
        return res;
      }
      if (selectedId === id) {
        return l ? l : rhsFn();
      }
      return l ? rhsFn() : l;
    },
    or: function (id, lhsFn, rhsFn) {
      var l = lhsFn();
      if (selectedId === 0) {
        var res = l ? l : rhsFn();
        recordBool(id, Boolean(res));
        return res;
      }
      if (selectedId === id) {
        return l ? rhsFn() : l;
      }
      return l ? l : rhsFn();
    },
    coal: function (idKeepLeft, idKeepRight, lhsFn, rhsFn) {
      if (selectedId === 0) {
        var l = lhsFn();
        var nonNullish = l !== null && l !== undefined;
        recordBool(idKeepLeft, nonNullish);
        recordBool(idKeepRight, nonNullish);
        return nonNullish ? l : rhsFn();
      }
      if (selectedId === idKeepLeft) {
        return lhsFn();
      }
      if (selectedId === idKeepRight) {
        return rhsFn();
      }
      var orig = lhsFn();
      return orig !== null && orig !== undefined ? orig : rhsFn();
    },
    lt: function (id1, id2, a, b) {
      if (selectedId === 0) {
        var res = a < b;
        recordBool(id1, res);
        recordBool(id2, res);
        return res;
      }
      if (selectedId === id1) return a <= b;
      if (selectedId === id2) return a >= b;
      return a < b;
    },
    le: function (id1, id2, a, b) {
      if (selectedId === 0) {
        var res = a <= b;
        recordBool(id1, res);
        recordBool(id2, res);
        return res;
      }
      if (selectedId === id1) return a < b;
      if (selectedId === id2) return a > b;
      return a <= b;
    },
    gt: function (id1, id2, a, b) {
      if (selectedId === 0) {
        var res = a > b;
        recordBool(id1, res);
        recordBool(id2, res);
        return res;
      }
      if (selectedId === id1) return a >= b;
      if (selectedId === id2) return a <= b;
      return a > b;
    },
    ge: function (id1, id2, a, b) {
      if (selectedId === 0) {
        var res = a >= b;
        recordBool(id1, res);
        recordBool(id2, res);
        return res;
      }
      if (selectedId === id1) return a > b;
      if (selectedId === id2) return a < b;
      return a >= b;
    },
    addL: function (id, a, b) {
      return active(id) ? a : a + b;
    },
    addR: function (id, a, b) {
      return active(id) ? b : a + b;
    },
    sub: function (id, a, b) {
      return active(id) ? a + b : a - b;
    },
    mul: function (id, a, b) {
      return active(id) ? a / b : a * b;
    },
    div: function (id, a, b) {
      return active(id) ? a * b : a / b;
    }
  };

  if (selectedId === 0 && globalThis.__karma__) {
    var rawStart = globalThis.__karma__.start;
    var wrapStart = function (fn) {
      return function () {
        selectedId = parseSelectedMutant();
        if (selectedId === 0 && globalThis.jasmine && typeof globalThis.jasmine.getEnv === 'function') {
          globalThis.jasmine.getEnv().addReporter({
            specStarted: function (result) {
              currentSpecId = result && result.id ? String(result.id) : null;
              currentSpecFullName = result ? result.fullName : null;
              currentSpecMutantSeen = new Uint8Array(%d);
              currentSpecSlotSeen = new Uint8Array(%d);
              currentSpecProbeSlotSeen = new Uint8Array(%d);
            },
            specDone: function (result) {
              if (
                result &&
                result.id &&
                result.fullName &&
                result.status !== 'pending' &&
                result.status !== 'disabled' &&
                result.status !== 'excluded'
              ) {
                specOrder.push({ id: String(result.id), fullName: String(result.fullName) });
              }
              currentSpecId = null;
              currentSpecFullName = null;
              currentSpecMutantSeen = null;
              currentSpecSlotSeen = null;
              currentSpecProbeSlotSeen = null;
            },
            jasmineDone: function () {
              if (globalThis.__karma__ && typeof globalThis.__karma__.info === 'function') {
                globalThis.__karma__.info({
                  khiMutantReport: {
                    specHits: specHits,
                    specOrder: specOrder,
                    specPolarities: specPolarities,
                    topLevelHits: topLevelHits
                  }
                });
              }
            }
          });
        }
        return fn.apply(this, arguments);
      };
    };
    var currentStart = typeof rawStart === 'function' ? wrapStart(rawStart) : undefined;
    Object.defineProperty(globalThis.__karma__, 'start', {
      configurable: true,
      enumerable: true,
      get: function () {
        return currentStart;
      },
      set: function (fn) {
        currentStart = typeof fn === 'function' ? wrapStart(fn) : fn;
      }
    });
  }
})();
`
)

type tsSpanBounds struct {
	start      int
	end        int
	excludeIdx int
}

// RenderTSSuperMutantSource splices activeSites into TypeScript source bytes from innermost to outermost spans.
func RenderTSSuperMutantSource(src []byte, activeSites []TSMutationSite) []byte {
	if len(activeSites) == 0 {
		return src
	}
	sorted := make([]TSMutationSite, len(activeSites))
	copy(sorted, activeSites)
	slices.SortFunc(sorted, func(a, b TSMutationSite) int {
		return cmp.Or(
			cmp.Compare(a.StartOffset, b.StartOffset),
			cmp.Compare(b.EndOffset, a.EndOffset),
		)
	})
	rendered := renderTSSpan(src, sorted, tsSpanBounds{start: 0, end: len(src), excludeIdx: -1})
	return []byte("// @ts-nocheck\n" + rendered)
}

// GenerateTSBrowserRuntimeSource returns the classic browser script defining globalThis.__khiMut and Jasmine hit/polarity tracing.
func GenerateTSBrowserRuntimeSource(maxMutantID int, maxProbeID int) []byte {
	seenCap := max(maxMutantID+1, 2)
	probeCap := max(maxProbeID+1, 2)
	script := fmt.Sprintf(tsBrowserRuntimeTemplate, seenCap, probeCap, seenCap, seenCap, seenCap*3, probeCap*3)
	return []byte(script)
}

func renderTSSpan(src []byte, sites []TSMutationSite, bounds tsSpanBounds) string {
	var sb strings.Builder
	cursor := bounds.start
	for i := range sites {
		if skipTSNestedSite(sites, i, cursor, bounds) {
			continue
		}
		s := &sites[i]
		sb.Write(src[cursor:s.StartOffset])
		sb.WriteString(renderSingleTSSite(src, sites, i))
		cursor = s.EndOffset
	}
	sb.Write(src[cursor:bounds.end])
	return sb.String()
}

func skipTSNestedSite(sites []TSMutationSite, idx int, cursor int, bounds tsSpanBounds) bool {
	if idx == bounds.excludeIdx {
		return true
	}
	s := &sites[idx]
	if s.StartOffset < cursor || s.EndOffset > bounds.end {
		return true
	}
	if bounds.excludeIdx >= 0 {
		p := &sites[bounds.excludeIdx]
		return s.StartOffset == p.StartOffset && s.EndOffset == p.EndOffset
	}
	return false
}

func renderSingleTSSite(src []byte, sites []TSMutationSite, idx int) string {
	s := &sites[idx]
	switch s.Kind {
	case "ordered_cmp", "eq_cmp", "logical", "nullish", "arith":
		return renderBinaryTSSite(src, sites, idx)
	case "cond":
		inner := renderTSSpan(src, sites, tsSpanBounds{start: s.StartOffset, end: s.EndOffset, excludeIdx: idx})
		return fmt.Sprintf("(globalThis.__khiMut.cond(%d, %d, Boolean(%s)))", s.Mutants[0].ID, s.Mutants[1].ID, inner)
	case "unary_not":
		operand := renderTSSpan(src, sites, tsSpanBounds{start: s.LeftStart, end: s.LeftEnd, excludeIdx: idx})
		return fmt.Sprintf("(globalThis.__khiMut.flip(%d, !(%s)))", s.Mutants[0].ID, operand)
	case "bool_lit":
		lit := "false"
		if s.BoolVal {
			lit = "true"
		}
		return fmt.Sprintf("(globalThis.__khiMut.flip(%d, %s))", s.Mutants[0].ID, lit)
	case "stmt_skip":
		stmtText := renderTSSpan(src, sites, tsSpanBounds{start: s.StartOffset, end: s.EndOffset, excludeIdx: idx})
		return fmt.Sprintf("if (!globalThis.__khiMut.active(%d)) { %s }", s.Mutants[0].ID, stmtText)
	default:
		return string(src[s.StartOffset:s.EndOffset])
	}
}

func renderBinaryTSSite(src []byte, sites []TSMutationSite, idx int) string {
	s := &sites[idx]
	lhs := renderTSSpan(src, sites, tsSpanBounds{start: s.LeftStart, end: s.LeftEnd, excludeIdx: idx})
	rhs := renderTSSpan(src, sites, tsSpanBounds{start: s.RightStart, end: s.RightEnd, excludeIdx: idx})
	switch s.Kind {
	case "ordered_cmp":
		return fmt.Sprintf("(globalThis.__khiMut.%s(%d, %d, (%s), (%s)))", tsOrderedMethod(s.Op), s.Mutants[0].ID, s.Mutants[1].ID, lhs, rhs)
	case "eq_cmp":
		return fmt.Sprintf("(globalThis.__khiMut.flip(%d, (%s) %s (%s)))", s.Mutants[0].ID, lhs, s.Op, rhs)
	case "logical":
		fn := "and"
		if s.Op == "||" {
			fn = "or"
		}
		return fmt.Sprintf("(globalThis.__khiMut.%s(%d, () => (%s), () => (%s)))", fn, s.Mutants[0].ID, lhs, rhs)
	case "nullish":
		return fmt.Sprintf("(globalThis.__khiMut.coal(%d, %d, () => (%s), () => (%s)))", s.Mutants[0].ID, s.Mutants[1].ID, lhs, rhs)
	default:
		return fmt.Sprintf("(globalThis.__khiMut.%s(%d, (%s), (%s)))", tsArithMethod(s), s.Mutants[0].ID, lhs, rhs)
	}
}

func tsOrderedMethod(op string) string {
	switch op {
	case "<=":
		return "le"
	case ">":
		return "gt"
	case ">=":
		return "ge"
	default:
		return "lt"
	}
}

func tsArithMethod(s *TSMutationSite) string {
	switch s.Op {
	case "+":
		if s.AddKeepRight {
			return "addR"
		}
		return "addL"
	case "-":
		return "sub"
	case "*":
		return "mul"
	default:
		return "div"
	}
}
