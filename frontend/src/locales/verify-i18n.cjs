#!/usr/bin/env node
/**
 * i18n Verification Script — verify-i18n.js
 * Checks:
 *  1. JSON format validity
 *  2. Key completeness (zh-TW ↔ en parity)
 *  3. Interpolation variable matching ({var} in both files)
 *  4. String length warnings (en > 2× zh-TW length)
 *
 * Usage: node verify-i18n.js
 * Exit code: 0 = pass, 1 = errors found
 */

const fs = require('fs');
const path = require('path');

const LOCALES_DIR = __dirname;
const SOURCE_FILE = 'zh-TW.json';
const TARGET_FILES = ['en.json'];

let errors = 0;
let warnings = 0;

function flattenKeys(obj, prefix = '') {
  const keys = {};
  for (const [k, v] of Object.entries(obj)) {
    const fullKey = prefix ? `${prefix}.${k}` : k;
    if (typeof v === 'object' && v !== null && !Array.isArray(v)) {
      Object.assign(keys, flattenKeys(v, fullKey));
    } else {
      keys[fullKey] = String(v);
    }
  }
  return keys;
}

function extractInterpolationVars(str) {
  const matches = str.match(/\{(\w+)\}/g) || [];
  return matches.sort();
}

// 1. Load and validate JSON
function loadJSON(filename) {
  const filepath = path.join(LOCALES_DIR, filename);
  try {
    const raw = fs.readFileSync(filepath, 'utf-8');
    const parsed = JSON.parse(raw);
    console.log(`  ✓ ${filename} — valid JSON`);
    return parsed;
  } catch (err) {
    console.error(`  ✗ ${filename} — INVALID JSON: ${err.message}`);
    errors++;
    return null;
  }
}

console.log('\n═══ i18n Verification ═══\n');
console.log('[1] JSON Format');
const source = loadJSON(SOURCE_FILE);
const targets = {};
for (const tf of TARGET_FILES) {
  targets[tf] = loadJSON(tf);
}

if (!source) {
  console.error('\nSource file invalid. Aborting.');
  process.exit(1);
}

const sourceKeys = flattenKeys(source);
const sourceKeyCount = Object.keys(sourceKeys).length;
console.log(`\n  Source (${SOURCE_FILE}): ${sourceKeyCount} keys`);

for (const [tf, data] of Object.entries(targets)) {
  if (!data) continue;

  const targetKeys = flattenKeys(data);
  const targetKeyCount = Object.keys(targetKeys).length;
  console.log(`  Target (${tf}): ${targetKeyCount} keys`);

  // 2. Key completeness
  console.log(`\n[2] Key Completeness: ${SOURCE_FILE} ↔ ${tf}`);
  const missingInTarget = Object.keys(sourceKeys).filter(k => !(k in targetKeys));
  const extraInTarget = Object.keys(targetKeys).filter(k => !(k in sourceKeys));

  if (missingInTarget.length === 0 && extraInTarget.length === 0) {
    console.log('  ✓ All keys match');
  } else {
    if (missingInTarget.length > 0) {
      console.error(`  ✗ Missing in ${tf} (${missingInTarget.length}):`);
      missingInTarget.forEach(k => console.error(`    - ${k}`));
      errors += missingInTarget.length;
    }
    if (extraInTarget.length > 0) {
      console.warn(`  ⚠ Extra in ${tf} (${extraInTarget.length}):`);
      extraInTarget.forEach(k => console.warn(`    - ${k}`));
      warnings += extraInTarget.length;
    }
  }

  // 3. Interpolation variable matching
  console.log(`\n[3] Interpolation Variables: ${SOURCE_FILE} ↔ ${tf}`);
  let interpErrors = 0;
  for (const key of Object.keys(sourceKeys)) {
    if (!(key in targetKeys)) continue;
    const srcVars = extractInterpolationVars(sourceKeys[key]);
    const tgtVars = extractInterpolationVars(targetKeys[key]);
    if (JSON.stringify(srcVars) !== JSON.stringify(tgtVars)) {
      console.error(`  ✗ ${key}: source={${srcVars.join(',')}} target={${tgtVars.join(',')}}`);
      interpErrors++;
    }
  }
  if (interpErrors === 0) {
    console.log('  ✓ All interpolation variables match');
  } else {
    errors += interpErrors;
  }

  // 4. Length warnings
  console.log(`\n[4] Length Warnings (en > 2× zh-TW):`);
  let lengthWarnings = 0;
  for (const key of Object.keys(sourceKeys)) {
    if (!(key in targetKeys)) continue;
    const srcLen = sourceKeys[key].length;
    const tgtLen = targetKeys[key].length;
    if (srcLen > 0 && tgtLen > srcLen * 2.5) {
      console.warn(`  ⚠ ${key}: zh-TW=${srcLen}ch, en=${tgtLen}ch (${(tgtLen/srcLen).toFixed(1)}×)`);
      lengthWarnings++;
    }
  }
  if (lengthWarnings === 0) {
    console.log('  ✓ No excessive length differences');
  } else {
    warnings += lengthWarnings;
  }
}

// Summary
console.log('\n═══ Summary ═══');
console.log(`  Errors:   ${errors}`);
console.log(`  Warnings: ${warnings}`);
console.log(`  Result:   ${errors === 0 ? '✓ PASS' : '✗ FAIL'}\n`);

process.exit(errors > 0 ? 1 : 0);
