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

(function () {
  window.__khiVRTPageErrors = [];
  var recordError = function (msg) {
    if (msg && window.__khiVRTPageErrors.length < 20) {
      window.__khiVRTPageErrors.push(String(msg));
    }
  };
  window.addEventListener(
    'error',
    function (ev) {
      if (ev && ev.message) {
        recordError('uncaught error: ' + ev.message);
      } else if (ev && ev.target && (ev.target.src || ev.target.href)) {
        recordError('resource load failed: ' + (ev.target.src || ev.target.href));
      }
    },
    true,
  );
  window.addEventListener('unhandledrejection', function (ev) {
    var reason = ev && ev.reason;
    var text = reason && reason.message ? reason.message : String(reason);
    recordError('unhandled rejection: ' + text);
  });
  var origFetch = window.fetch;
  if (typeof origFetch === 'function') {
    window.fetch = function (input, init) {
      return origFetch.call(window, input, init).then(function (resp) {
        if (
          resp &&
          !resp.ok &&
          resp.url &&
          resp.url.indexOf(window.location.origin) === 0
        ) {
          recordError('HTTP ' + resp.status + ' fetching ' + resp.url);
        }
        return resp;
      });
    };
  }

  // Instrument Canvas2D and WebGL draw calls so readiness detection observes real rendered frames.
  var wrapDrawMethods = function (proto, methods) {
    if (!proto) {
      return;
    }
    for (var i = 0; i < methods.length; i++) {
      var name = methods[i];
      var orig = proto[name];
      if (typeof orig !== 'function') {
        continue;
      }
      proto[name] = (function (fn) {
        return function () {
          if (this && this.canvas) {
            this.canvas.__khiDrawCount = (this.canvas.__khiDrawCount || 0) + 1;
          }
          return fn.apply(this, arguments);
        };
      })(orig);
    }
  };
  wrapDrawMethods(window.CanvasRenderingContext2D && window.CanvasRenderingContext2D.prototype, [
    'clearRect',
    'fillRect',
    'strokeRect',
    'fill',
    'stroke',
    'fillText',
    'strokeText',
    'drawImage',
    'putImageData',
  ]);
  wrapDrawMethods(window.WebGLRenderingContext && window.WebGLRenderingContext.prototype, [
    'clear',
    'drawArrays',
    'drawElements',
  ]);
  wrapDrawMethods(window.WebGL2RenderingContext && window.WebGL2RenderingContext.prototype, [
    'clear',
    'drawArrays',
    'drawElements',
    'drawArraysInstanced',
    'drawElementsInstanced',
  ]);

  var raf = function () {
    return new Promise(function (resolve) {
      requestAnimationFrame(function () {
        resolve();
      });
    });
  };

  var waitForImages = function () {
    var imgs = Array.prototype.slice.call(document.querySelectorAll('img'));
    return Promise.all(
      imgs.map(function (img) {
        if (img.complete) {
          return typeof img.decode === 'function'
            ? img.decode().catch(function () {})
            : Promise.resolve();
        }
        return new Promise(function (resolve) {
          var done = function () {
            resolve();
          };
          img.addEventListener('load', done, { once: true });
          img.addEventListener('error', done, { once: true });
        });
      }),
    );
  };

  var captureLayoutSignature = function (root) {
    var rootRect = root.getBoundingClientRect();
    var parts = [
      Math.round(rootRect.width) + 'x' + Math.round(rootRect.height),
    ];
    var canvases = document.querySelectorAll('canvas');
    var allCanvasesPainted = true;
    for (var i = 0; i < canvases.length; i++) {
      var c = canvases[i];
      var rect = c.getBoundingClientRect();
      var style = window.getComputedStyle(c);
      var visible =
        rect.width > 0 &&
        rect.height > 0 &&
        style.display !== 'none' &&
        style.visibility !== 'hidden';
      if (!visible) {
        continue;
      }
      var draws = c.__khiDrawCount || 0;
      if (c.width <= 0 || c.height <= 0 || draws === 0) {
        allCanvasesPainted = false;
      }
      parts.push(c.width + 'x' + c.height + ':' + draws);
    }
    return {
      signature: parts.join('|'),
      allCanvasesPainted: allCanvasesPainted,
    };
  };

  var pauseAllAnimations = function () {
    if (typeof document.getAnimations !== 'function') {
      return;
    }
    document.getAnimations().forEach(function (a) {
      try {
        a.pause();
        a.currentTime = 0;
      } catch (e) {
        // Ignore animations whose timeline does not allow seeking.
      }
    });
  };

  // Waits until fonts, images, DOM mutations, layout dimensions, and visible canvases have settled.
  window.__khiVRTWaitSettled = async function (timeoutMs) {
    var limit = timeoutMs || 10000;
    var deadline = performance.now() + limit;
    await document.fonts.ready;
    await waitForImages();

    var root = document.getElementById('storybook-root') || document.body;
    var mutated = false;
    var observer = new MutationObserver(function () {
      mutated = true;
    });
    observer.observe(root, {
      subtree: true,
      childList: true,
      attributes: true,
      characterData: true,
    });

    try {
      var prevSig = null;
      while (performance.now() < deadline) {
        await raf();
        if (mutated) {
          mutated = false;
          prevSig = null;
          await waitForImages();
          continue;
        }
        var snap = captureLayoutSignature(root);
        if (!snap.allCanvasesPainted) {
          prevSig = null;
          continue;
        }
        if (prevSig !== null && snap.signature === prevSig) {
          break;
        }
        prevSig = snap.signature;
      }
    } finally {
      observer.disconnect();
    }

    pauseAllAnimations();
    await raf();
  };

  var injectStyle = function () {
    if (!document.head || document.getElementById('khi-vrt-freeze-style')) {
      return;
    }
    var style = document.createElement('style');
    style.id = 'khi-vrt-freeze-style';
    style.textContent = [
      '*, *::before, *::after {',
      '  animation-duration: 0.001s !important;',
      '  animation-delay: 0s !important;',
      '  animation-iteration-count: 1 !important;',
      '  animation-play-state: paused !important;',
      '  transition-duration: 0s !important;',
      '  transition-delay: 0s !important;',
      '  caret-color: transparent !important;',
      '}',
    ].join('\n');
    document.head.appendChild(style);
  };
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', injectStyle, { once: true });
  } else {
    injectStyle();
  }
})();
