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

const generateBMFont = require('msdf-bmfont-xml');
const fs = require('fs');

/**
 * Generates MSDF texture for Roboto numbers (0-9).
 * @param {string} destFolder
 * @returns {Promise<void>}
 */
function generateNumberMSDFTexture(destFolder) {
  return new Promise((resolve, reject) => {
    const fontBuffer = fs.readFileSync('./node_modules/@fontsource/roboto/files/roboto-latin-700-normal.ttf');
    generateBMFont(fontBuffer, {
      outputType: "json",
      charset: ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9'],
      filename: "./zzz-roboto-number-msdf",
      texturePadding: 8,
      textureSize: [128, 128],
    }, (error, textures, font) => {
      if (error) {
        reject(error);
        return;
      }
      textures.forEach((texture, index) => {
        fs.writeFileSync(destFolder + texture.filename + ".png", texture.texture);
      });
      fs.writeFileSync(destFolder + font.filename, font.data);
      resolve();
    })
  });
}

/**
 * Generates MSDF texture for Material Symbols based on used codepoints.
 * @param {string} destFolder
 * @param {string[]} usedIconCodepoints
 * @returns {Promise<void>}
 */
async function generateMaterialSymbolsMSDFTexture(destFolder, usedIconCodepoints) {
  const iconBuffer = fs.readFileSync('./node_modules/material-symbols/material-symbols-outlined.ttf');
  return new Promise((resolve, reject) => {
    generateBMFont(iconBuffer, {
      outputType: "json",
      charset: usedIconCodepoints,
      filename: "./zzz-material-icons-msdf",
      texturePadding: 4,
      textureSize: [256, 256],
    }, (error, textures, font) => {
      if (error) {
        reject(error);
        return;
      }
      textures.forEach((texture, index) => {
        fs.writeFileSync(destFolder + texture.filename + ".png", texture.texture);
      });
      fs.writeFileSync(destFolder + font.filename, font.data);
      resolve();
    });
  });
}

/**
 * Reads zzz_generated_used_icons.json, fetches codepoints, and generates MSDF texture for used icons.
 * @param {string} destFolder
 * @returns {Promise<void>}
 */
async function processMaterialSymbols(destFolder) {
  const usedIconsSetting = JSON.parse(fs.readFileSync('./zzz_generated_used_icons.json')); // This json file is generated from the backend code generation and includes the list of used icon names.

  // msdf-bmfont-xml requires codepoints thus get the code point files and convert the icon names to codepoints.
  const codePointsJSONRaw = await fetch('http://fonts.google.com/metadata/icons?incomplete=1&key=material_symbols').then(res => res.text())
  const codePointsJSON = JSON.parse(codePointsJSONRaw.replace(")]}'", ""))
  const iconNameToCodepoint = {};
  codePointsJSON.icons.forEach((icon) => {
    const code = String.fromCodePoint([icon.codepoint])
    iconNameToCodepoint[icon.name] = code;
  });

  const usedIconCodepoints = []
  const usedIconNameToCodepoint = {};
  usedIconsSetting.icons.sort((a, b) => a.localeCompare(b));
  usedIconsSetting.icons.forEach((iconName) => {
    const codepoint = iconNameToCodepoint[iconName];
    if (!codepoint) {
      throw new Error("Icon not found: " + iconName);
    } else {
      usedIconCodepoints.push(codepoint);
      usedIconNameToCodepoint[iconName] = codepoint;
    }
  });
  fs.writeFileSync(destFolder + "zzz-icon-codepoints.json", JSON.stringify(usedIconNameToCodepoint));

  await generateMaterialSymbolsMSDFTexture(destFolder, usedIconCodepoints);
}

/**
 * Main entry point.
 */
async function main() {
  const assetFolder = "../../web/src/assets/"
  await generateNumberMSDFTexture(assetFolder);
  await processMaterialSymbols(assetFolder);
}

main();
