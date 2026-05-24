import { readdir, readFile } from 'node:fs/promises';
import path from 'node:path';
import { put } from '@vercel/blob';

const dir = process.argv[2];
if (!dir) {
  throw new Error('Usage: node scripts/upload-release-to-blob.mjs <artifact-dir>');
}

const token = process.env.F4RGE_DOWNLOADS_BLOB_READ_WRITE_TOKEN;
if (!token) {
  throw new Error('F4RGE_DOWNLOADS_BLOB_READ_WRITE_TOKEN is required');
}

const entries = await readdir(dir, { withFileTypes: true });
for (const entry of entries) {
  if (!entry.isFile()) continue;
  const filePath = path.join(dir, entry.name);
  const body = await readFile(filePath);
  const blobPath = `cli/${entry.name}`;
  const result = await put(blobPath, body, {
    access: 'public',
    addRandomSuffix: false,
    allowOverwrite: true,
    token,
  });
  console.log(`uploaded ${entry.name} -> ${result.url}`);
}
