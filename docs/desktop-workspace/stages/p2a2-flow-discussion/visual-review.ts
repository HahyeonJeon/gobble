// Owned development window capture. Does not qualify OS focus or Agent tools.
import { _electron as electron, expect } from "@playwright/test";
import { mkdtemp, cp, readFile, writeFile, rm } from "node:fs/promises";
import { join, resolve } from "node:path";
import { tmpdir } from "node:os";
import {
  chooseProject,
  readyView,
  attachmentPreview,
  closeAttachmentPreview,
} from "../../../../app/desktop/tests/electron/support";

async function main() {
  const out = resolve("../docs/desktop-workspace/stages/p2a2-flow-discussion");
  const base = await mkdtemp(join(tmpdir(), "gobble-p2a2-visual-")),
    profile = join(base, "profile"),
    project = join(base, "Read quality study");
  await cp(resolve("qualification/pipeline-flow/project"), project, {
    recursive: true,
  });
  const env = { ...process.env };
  delete env.ELECTRON_RUN_AS_NODE;
  delete env.ELECTRON_RENDERER_URL;
  const application = await electron.launch({
    args: [resolve("desktop"), "--gobble-profile=" + profile],
    env,
  });
  const page = await application.firstWindow();
  page.setDefaultTimeout(15_000);
  try {
    await expect(page.locator(".app-status")).toHaveText("Workspace ready");
    const focused = await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.isFocused(),
    );
    await chooseProject(application, page, project);
    await application.evaluate(
      ({ dialog }, folder) => {
        dialog.showOpenDialog = async () => ({
          canceled: false,
          filePaths: [folder],
        });
      },
      join(project, "trim-review"),
    );
    await page
      .getByRole("button", { name: "Import pipeline", exact: true })
      .click();
    let primary = await readyView(page);
    await primary
      .getByRole("button", { name: "Check flow", exact: true })
      .click();
    await expect(
      primary.getByText("2 steps · 1 input · 2 connections"),
    ).toBeVisible({ timeout: 135_000 });
    primary = await readyView(page);
    await primary
      .getByRole("button", { name: /^Inspect Trim adapters/ })
      .click();
    await primary
      .getByRole("button", { name: "Quality threshold 25 Phred", exact: true })
      .click();
    await expect(
      primary.getByRole("region", {
        name: "Details for Quality threshold",
        exact: true,
      }),
    ).toBeVisible();
    await primary
      .getByRole("button", { name: "Add to message", exact: true })
      .click();
    await expect(
      page.getByLabel("Message attachments", { exact: true }),
    ).toContainText("Quality threshold");
    await page
      .getByRole("textbox", { name: "Message draft" })
      .fill("Can we make this quality threshold more conservative?");
    await page.screenshot({
      path: join(out, "pipeline-setting-selection.png"),
    });
    await page
      .getByLabel("Message attachments", { exact: true })
      .getByRole("button", { name: /^Trim adapters/ })
      .click();
    await expect(attachmentPreview(page)).toContainText("25 Phred");
    await page.screenshot({ path: join(out, "pipeline-setting-capture.png") });
    await closeAttachmentPreview(page);
    await primary
      .getByRole("button", {
        name: "Select declared outputs port trimmed_read1",
        exact: true,
      })
      .click();
    await expect(
      primary.getByRole("button", {
        name: "Select declared outputs port trimmed_read1",
        exact: true,
      }),
    ).toHaveAttribute("aria-pressed", "true");
    await primary
      .getByRole("button", { name: "Step list", exact: true })
      .click();
    await primary
      .getByRole("button", { name: /^Trim adapters/ })
      .first()
      .click();
    await primary
      .getByRole("button", { name: "Minimum length 40 bp", exact: true })
      .click();
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.setBounds({ width: 940, height: 760 }),
    );
    if (
      await page
        .getByRole("button", { name: "Workspace", exact: true })
        .isVisible()
    )
      await page
        .getByRole("button", { name: "Workspace", exact: true })
        .click();
    await page.screenshot({ path: join(out, "pipeline-list-compact.png") });
    const catalog = JSON.parse(
      await readFile(join(profile, "service/catalog.json"), "utf8"),
    );
    if (catalog.runs.length) throw new Error("Unexpected Run");
    await writeFile(
      join(out, "visual-review.json"),
      JSON.stringify(
        {
          focused,
          scope:
            "Owned native window rendering and User controls; no Agent or OS focus claim",
          checkedSteps: 2,
          connections: 2,
          quality: 25,
          length: 40,
          attachment: true,
          portSelection: true,
          list: true,
          runs: 0,
          temporaryProfileRemoved: true,
        },
        null,
        2,
      ) + "\n",
    );
  } finally {
    await application
      .evaluate(({ dialog }) => {
        dialog.showMessageBox = async () => ({
          response: 1,
          checkboxChecked: false,
        });
      })
      .catch(() => {});
    await application.close();
    await rm(base, { recursive: true, force: true });
  }
}
void main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
