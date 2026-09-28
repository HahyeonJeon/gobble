import { mkdtemp, mkdir, rm, symlink, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';
import { assetPath, readShellAsset } from '../src/main/security/content';
import {
  DEVELOPMENT_URL,
  isTrustedSender,
  PACKAGED_URL,
  rendererURL,
} from '../src/main/security/origin';

describe('renderer origin and sender authority', () => {
  it('uses the built origin unless an exact local development origin is selected', () => {
    expect(rendererURL(false, undefined)).toBe(PACKAGED_URL);
    expect(rendererURL(false, DEVELOPMENT_URL)).toBe(DEVELOPMENT_URL);
    expect(rendererURL(true, 'https://external.test/')).toBe(PACKAGED_URL);
    expect(() => rendererURL(false, 'http://127.0.0.1:5173.attacker.test/')).toThrow();
    expect(() => rendererURL(false, 'http://localhost:5173/')).toThrow();
  });

  it.each([
    { isMainFrame: false, isWorkspaceWindow: true, url: PACKAGED_URL },
    { isMainFrame: true, isWorkspaceWindow: false, url: PACKAGED_URL },
    { isMainFrame: true, isWorkspaceWindow: true, url: 'app://gobble.attacker.test/index.html' },
    { isMainFrame: true, isWorkspaceWindow: true, url: 'app://gobble/index.html?redirect=1' },
    { isMainFrame: true, isWorkspaceWindow: true, url: 'about:blank' },
  ])('rejects untrusted sender %j', (sender) => {
    expect(isTrustedSender(sender, PACKAGED_URL)).toBe(false);
  });

  it('accepts only the registered top-level workspace document', () => {
    expect(
      isTrustedSender(
        { isMainFrame: true, isWorkspaceWindow: true, url: PACKAGED_URL },
        PACKAGED_URL,
      ),
    ).toBe(true);
  });
});

describe('built asset containment', () => {
  const directories: string[] = [];
  afterEach(async () => {
    await Promise.all(
      directories.splice(0).map((path) => rm(path, { recursive: true, force: true })),
    );
  });

  it.each([
    'file:///etc/passwd',
    'app://other/index.html',
    'app://user@gobble/index.html',
    'app://gobble:999/index.html',
    'app://gobble/%2e%2e/secret.txt',
    'app://gobble/assets/%2fsecret.js',
    'app://gobble/assets/../../package.json',
    'app://gobble/assets/source.js.map',
    'app://gobble/index.html?path=/etc/passwd',
  ])('rejects %s', (url) => {
    expect(assetPath(url)).toBeUndefined();
  });

  it('serves shell assets with strict CSP and blocks symlinks outside the build', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'gobble-content-'));
    directories.push(directory);
    const root = join(directory, 'renderer');
    await mkdir(join(root, 'assets'), { recursive: true });
    await writeFile(join(root, 'index.html'), '<main>Gobble</main>');
    await writeFile(join(directory, 'private.js'), 'private');
    await symlink(join(directory, 'private.js'), join(root, 'assets', 'leak.js'));
    const response = await readShellAsset(root, PACKAGED_URL);
    expect(response.status).toBe(200);
    expect(await response.text()).toContain('Gobble');
    expect(response.headers.get('Content-Security-Policy')).toContain("default-src 'none'");
    expect(response.headers.get('Content-Security-Policy')).not.toContain('unsafe');
    expect((await readShellAsset(root, 'app://gobble/assets/leak.js')).status).toBe(404);
    expect((await readShellAsset(root, 'app://gobble/assets/missing.js')).status).toBe(404);
  });
});
