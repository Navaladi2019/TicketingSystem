import { build } from 'esbuild';
import { statSync } from 'node:fs';
import { gzipSync } from 'node:zlib';
import { readFileSync } from 'node:fs';
await build({ entryPoints: ['app.jsx'], bundle: true, minify: true, format: 'iife', outfile: 'dist/bundle.js',
  loader: { '.js': 'jsx', '.json': 'json' }, define: { 'process.env.NODE_ENV': '"production"' } });
const f = 'dist/bundle.js';
console.log(`bundle ${statSync(f).size} B, gzip ${gzipSync(readFileSync(f)).length} B`);
