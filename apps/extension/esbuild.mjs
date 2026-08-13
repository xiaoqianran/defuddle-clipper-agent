import { build } from 'esbuild';
import { copyFile, mkdir, rm } from 'node:fs/promises';
import { resolve } from 'node:path';

const here = new URL('.', import.meta.url).pathname;
const outdir = resolve(here, 'dist');

await rm(outdir, { recursive: true, force: true });
await mkdir(outdir, { recursive: true });

await build({
  entryPoints: {
    background: resolve(here, 'src/background.ts'),
    content: resolve(here, 'src/content.ts'),
    popup: resolve(here, 'src/popup.ts'),
    options: resolve(here, 'src/options.ts')
  },
  bundle: true,
  outdir,
  format: 'iife',
  platform: 'browser',
  target: ['chrome120'],
  sourcemap: true,
  minify: false
});

for (const file of ['manifest.json', 'popup.html', 'options.html']) {
  await copyFile(resolve(here, 'public', file), resolve(outdir, file));
}

console.log(`Extension built at ${outdir}`);
