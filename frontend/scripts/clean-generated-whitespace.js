import { readFileSync, writeFileSync, existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const target = join(here, '..', 'wailsjs', 'go', 'models.ts');

if (existsSync(target)) {
  const before = readFileSync(target, 'utf8');
  const after = before.replace(/[ \t]+(?=\r?$)/gm, '');
  if (after !== before) {
    writeFileSync(target, after);
  }
}
