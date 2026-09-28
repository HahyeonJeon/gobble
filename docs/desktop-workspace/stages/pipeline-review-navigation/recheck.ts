// Revisit the original actual-engine/Agent walkthrough in a copied profile.
// No Agent turn, engine action, adoption, or modification of the original profile.
import { cp, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { launch, captureWindow } from '../../../../app/desktop/tests/electron/support';

async function main() {
  const stage = resolve('../docs/desktop-workspace/stages/pipeline-review-navigation');
  const original = JSON.parse(await readFile(resolve('../docs/desktop-workspace/stages/report-live-walkthrough/fixture.json'), 'utf8'));
  const base = await mkdtemp(join(tmpdir(), 'gobble-navigation-recheck-'));
  const profile = join(base, 'profile');
  await cp(original.profile, profile, { recursive: true });
  await writeFile(join(stage, 'recheck-fixture.json'), JSON.stringify({ profile, source: original.profile }, null, 2));
  const { application, page } = await launch(profile);
  try {
    await application.evaluate(({ BrowserWindow }) => BrowserWindow.getAllWindows().find(w => w.webContents.getURL().startsWith('app://gobble/'))!.setSize(1000, 760));
    await page.getByRole('button', { name: 'Workspace', exact: true }).click();
    await page.getByRole('button', { name: 'Changes', exact: true }).click();
    const review = page.getByRole('region', { name: 'Pipeline change review' });
    await review.getByText('25 Phred', { exact: true }).waitFor();
    await page.getByRole('button', { name: 'Chat', exact: true }).click();
    await review.waitFor({ state: 'detached' });
    if (await page.getByRole('textbox', { name: 'Message draft' }).inputValue() !== 'Keep this draft while I review the proposed threshold.') throw Error('Draft changed');
    await page.getByRole('button', { name: 'Workspace', exact: true }).click();
    await review.getByText('25 Phred', { exact: true }).waitFor();
    await review.getByText('20 Phred', { exact: true }).waitFor();
    await review.getByRole('region', { name: 'Selected change' }).scrollIntoViewIfNeeded();
    await captureWindow(application, join(stage, 'actual-profile-return.png'));
    console.log('PASS: original 20 → 25 Phred comparison and draft survive compact Chat return; no Agent or engine action.');
  } finally { await application.close(); }
}
main().catch(error => { console.error(error); process.exitCode = 1; });
