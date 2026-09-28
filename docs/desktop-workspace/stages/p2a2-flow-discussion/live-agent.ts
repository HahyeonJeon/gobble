// Opt-in actual Agent review: isolated profile, synthetic Project, existing authorized account.
import { expect } from "@playwright/test";
import {
  mkdtemp,
  mkdir,
  cp,
  copyFile,
  chmod,
  readFile,
  writeFile,
  rm,
} from "node:fs/promises";
import { join, resolve } from "node:path";
import { tmpdir } from "node:os";
import { createHash } from "node:crypto";
import {
  launch,
  chooseProject,
  readyView,
  openAgentRoster,
} from "../../../../app/desktop/tests/electron/support";
async function main() {
  const out = resolve("../docs/desktop-workspace/stages/p2a2-flow-discussion");
  const base = await mkdtemp(join(tmpdir(), "gobble-p2a2-live-")),
    profile = join(base, "profile"),
    project = join(base, "Read quality study");
  const auth =
    "/Users/hahyeon/Library/Application Support/Gobble-stage4-review/codex/home/auth.json";
  const digest = async () =>
    createHash("sha256")
      .update(await readFile(auth))
      .digest("hex");
  const beforeAuth = await digest();
  await mkdir(join(profile, "codex/home"), { recursive: true, mode: 0o700 });
  await copyFile(auth, join(profile, "codex/home/auth.json"));
  await chmod(join(profile, "codex/home/auth.json"), 0o600);
  await cp(resolve("qualification/pipeline-flow/project"), project, {
    recursive: true,
  });
  delete process.env.GOBBLE_CODEX_EXECUTABLE;
  const { application, page } = await launch(profile).catch(async (error) => {
    await rm(base, { recursive: true, force: true });
    throw error;
  });
  page.setDefaultTimeout(20_000);
  try {
    await page.getByRole("button", { name: "Account", exact: true }).click();
    await page
      .getByRole("button", { name: "Refresh connection", exact: true })
      .click();
    await expect(
      page.getByText("ChatGPT connected", { exact: true }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "Close Codex account", exact: true })
      .click();
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
    await openAgentRoster(page);
    await page.getByRole("button", { name: "Add agent", exact: true }).click();
    const editor = page.getByRole("dialog", { name: "Add agent", exact: true });
    await editor.getByLabel("Name", { exact: true }).fill("Research partner");
    await editor.getByLabel("Project access").selectOption("sharedViews");
    await editor
      .getByRole("button", { name: "Add agent", exact: true })
      .click();
    await expect(editor).not.toBeVisible();
    await page
      .getByRole("combobox", { name: "Conversation recipient" })
      .selectOption({ label: "To Research partner" });
    await primary
      .getByRole("button", { name: /^Inspect Trim adapters/ })
      .click();
    await primary
      .getByRole("button", { name: "Quality threshold 25 Phred", exact: true })
      .click();
    await primary
      .getByRole("button", { name: "Add to message", exact: true })
      .click();
    await primary
      .getByRole("button", {
        name: "Select declared outputs port trimmed_read1",
        exact: true,
      })
      .click();
    await expect(
      primary.getByRole("region", {
        name: "Details for trimmed_read1",
        exact: true,
      }),
    ).toBeVisible();
    const doc = () =>
      page.evaluate(async () => {
        const projects = await window.gobble.projects.list();
        if (!projects.ok) throw new Error(projects.error.message);
        const result = await window.gobble.workspace.read({
          projectId: projects.value[0]!.projectId,
        });
        if (!result.ok) throw new Error(result.error.message);
        return result.value;
      });
    const before = await doc();
    const prompt =
      "Review the attached Quality threshold in this checked pipeline. Use workspace_observe to inspect the currently displayed pipeline and workspace_point to mark the Quality threshold setting on Trim adapters using its exact returned target and observation receipt. Briefly explain its declared quality and minimum-length values, and how the trimmed reads reach the quality check. Keep the explanation focused on the analysis UI. This is a read-only review: do not edit files or run analysis.";
    const draft = page.getByRole("textbox", { name: "Message draft" });
    await draft.fill(prompt);
    await application.evaluate(({ app, BrowserWindow }) => {
      app.focus({ steal: true });
      BrowserWindow.getAllWindows()[0]!.focus();
    });
    await expect
      .poll(() =>
        application.evaluate(({ BrowserWindow }) =>
          BrowserWindow.getAllWindows()[0]!.isFocused(),
        ),
      )
      .toBe(true);
    await page.getByRole("button", { name: "Send", exact: true }).click();
    await expect(draft).toHaveValue("");
    await draft.fill(
      "Keep this output selected while we discuss the threshold.",
    );
    await expect(
      page.locator(".message-state").getByText("completed", { exact: true }),
    ).toHaveCount(1, { timeout: 150_000 });
    const after = await doc(),
      mark = after.sharedReferences?.find(
        (r) =>
          r.author.kind === "agent" &&
          r.evidence.schemaVersion === 7 &&
          r.evidence.selection.subject.kind === "setting" &&
          r.evidence.selection.subject.key === "quality",
      );
    const nativeFocused = await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()[0]!.isFocused(),
    );
    await writeFile(
      join(out, "live-agent-observation.json"),
      JSON.stringify(
        {
          response: after.collaboration?.submissions[0]?.response,
          beforeSelection: before.selections,
          afterSelection: after.selections,
          nativeFocused,
          reference: mark,
        },
        null,
        2,
      ) + "\n",
    );
    expect(mark).toBeTruthy();
    expect(nativeFocused).toBe(true);
    expect(after.selections).toEqual(before.selections);
    expect(after.chat.draft).toBe(
      "Keep this output selected while we discuss the threshold.",
    );
    await expect(
      primary.locator('.pipeline-node[data-agent-mark="true"]'),
    ).toHaveCount(1);
    await expect(draft).toBeFocused();
    await page.screenshot({ path: join(out, "live-agent-pipeline.png") });
    await writeFile(
      join(out, "live-agent-review.json"),
      JSON.stringify(
        {
          actualAgent: true,
          prompt,
          model: after.workspace.agents[0]?.configuration?.model,
          response: after.collaboration?.submissions[0]?.response,
          reference: mark,
          userSelectionPreserved: true,
          draftPreserved: true,
          composerFocusPreserved: true,
        },
        null,
        2,
      ) + "\n",
    );
  } catch (error) {
    await page
      .screenshot({ path: join(out, "live-agent-failure.png") })
      .catch(() => {});
    throw error;
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
    await writeFile(
      join(out, "live-isolation.json"),
      JSON.stringify(
        {
          authorizedCredentialUnchanged: beforeAuth === (await digest()),
          temporaryProfileRemoved: true,
          syntheticProjectOnly: true,
        },
        null,
        2,
      ) + "\n",
    );
  }
}
void main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
