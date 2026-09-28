// Opt-in qualification: actual signed-in Agent, synthetic Project, isolated profile.
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
import { execFileSync } from "node:child_process";
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
  const out = resolve(
    "../docs/desktop-workspace/stages/p2b1-visual-proposals/evidence/actual-agent",
  );
  await mkdir(out, { recursive: true });
  const base = await mkdtemp(join(tmpdir(), "gobble-p2b1-agent-"));
  const profile = join(base, "profile"),
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
  const source = await readFile(
    join(project, "trim-review/pipeline.go"),
    "utf8",
  );
  const image = (await readFile("/tmp/p2b-final-image-id", "utf8")).trim();
  const daemon = execFileSync("docker", ["info", "--format", "{{.ID}}"], {
    encoding: "utf8",
  }).trim();
  await writeFile(
    join(project, ".gobble-runtime.json"),
    JSON.stringify({ format: 1, image, daemon }),
  );
  delete process.env.GOBBLE_CODEX_EXECUTABLE;
  let { application, page } = await launch(profile).catch(async (error) => {
    await rm(base, { recursive: true, force: true });
    throw error;
  });
  page.setDefaultTimeout(25_000);
  const record = async (name: string, value: unknown) =>
    writeFile(join(out, name + ".json"), JSON.stringify(value, null, 2) + "\n");
  const focus = async () => {
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
  };
  const document = () =>
    page.evaluate(async () => {
      const projects = await window.gobble.projects.list();
      if (!projects.ok) throw new Error(projects.error.message);
      const result = await window.gobble.workspace.read({
        projectId: projects.value[0]!.projectId,
      });
      if (!result.ok) throw new Error(result.error.message);
      return result.value;
    });
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
    console.log("Actual Agent: current flow checked");
    await openAgentRoster(page);
    await page.getByRole("button", { name: "Add agent", exact: true }).click();
    const editor = page.getByRole("dialog", { name: "Add agent", exact: true });
    await editor.getByLabel("Name", { exact: true }).fill("Research partner");
    await editor.getByLabel("Project access").selectOption("sharedViews");
    await editor
      .getByRole("button", { name: "Add agent", exact: true })
      .click();
    await page
      .getByRole("combobox", { name: "Conversation recipient" })
      .selectOption({ label: "To Research partner" });
    await primary
      .getByRole("button", { name: "Discuss changes", exact: true })
      .click();
    await page
      .getByRole("checkbox", {
        name: "Allow a Pipeline proposal for this message",
      })
      .check();
    const draft = page.getByRole("textbox", { name: "Message draft" });
    const prompt =
      "Please propose a refinement of this analysis: raise the quality threshold from 25 to 30, keep minimum length at 40, and add a FastQC quality-check branch from the trimmed reads. Preserve all existing steps, inputs and connections. Use the scoped Pipeline proposal tools and their authoring guidance. Submit one proposal, then finish your reply while Gobble checks it in the background. Explain it using analysis labels. Do not adopt it or start a Run.";
    await draft.fill(prompt);
    await focus();
    await page.getByRole("button", { name: "Send", exact: true }).click();
    await expect(
      page.locator(".message-state").getByText("completed", { exact: true }),
    ).toHaveCount(1, { timeout: 210_000 });
    const proposedDocument = await document();
    await record("proposal-turn", {
      prompt,
      model: proposedDocument.workspace.agents[0]?.configuration?.model,
      submission: proposedDocument.collaboration?.submissions[0],
    });
    console.log("Actual Agent: proposal turn completed");
    await primary.getByRole("button", { name: "Changes", exact: true }).click();
    const review = page.getByRole("region", { name: "Pipeline change review" });
    await expect(review.getByText("30 Phred", { exact: true })).toBeVisible({
      timeout: 270_000,
    });
    await expect(review.getByText("25 Phred", { exact: true })).toBeVisible();
    await expect(
      review.getByRole("button", { name: "2 Add quality check" }),
    ).toBeVisible();
    await page.screenshot({ path: join(out, "b-setting-review.png") });
    await review.getByRole("button", { name: "2 Add quality check" }).click();
    await expect(
      review.getByText("Quality check added", { exact: true }),
    ).toBeVisible();
    await page.screenshot({ path: join(out, "b-added-step-review.png") });
    await review.getByRole("button", { name: "1 Quality threshold" }).click();
    await review.getByRole("button", { name: "Discuss this change" }).click();
    const before = await document();
    const reviewPrompt =
      "Please read this exact checked comparison and point to the attached quality-threshold change using the Pipeline review reference tool. Explain the before/after values and that the additional quality-check branch has not run yet. Do not change the User selection or draft, create another proposal, adopt, or run anything.";
    await draft.fill(reviewPrompt);
    await focus();
    await page.getByRole("button", { name: "Send", exact: true }).click();
    await expect(draft).toHaveValue("");
    const unsent =
      "Keep minimum length at 40 while we discuss the quality check.";
    await draft.fill(unsent);
    await expect(
      page.locator(".message-state").getByText("completed", { exact: true }),
    ).toHaveCount(2, { timeout: 150_000 });
    const after = await document();
    await record("review-turn", {
      prompt: reviewPrompt,
      submission: after.collaboration?.submissions[1],
      references: after.pipelineReviewMarks,
    });
    expect(after.pipelineReviewMarks).toHaveLength(1);
    expect(after.selections).toEqual(before.selections);
    expect(after.chat.draft).toBe(unsent);
    await expect(
      review.getByRole("button", { name: /Agent reference/ }),
    ).toBeVisible();
    await page.screenshot({ path: join(out, "b-agent-reference.png") });
    await review
      .getByRole("button", { name: "Adopt proposal", exact: true })
      .click();
    await review.getByRole("button", { name: "Confirm adoption" }).click();
    await expect(
      page.getByText("Adopted as current. No Run was started.", {
        exact: true,
      }),
    ).toBeVisible();
    expect(
      await readFile(join(project, "trim-review/pipeline.go"), "utf8"),
    ).toBe(source);
    await page.screenshot({ path: join(out, "b-adopted-review.png") });
    const catalog = JSON.parse(
      await readFile(join(profile, "service/catalog.json"), "utf8"),
    );
    expect(catalog.runs).toHaveLength(0);
    expect(Object.keys(catalog.revisions)).toHaveLength(1);
    await record("result", {
      actualAgent: true,
      image,
      daemon,
      model: after.workspace.agents[0]?.configuration?.model,
      exactReference: after.pipelineReviewMarks,
      userSelectionPreserved: true,
      draftPreserved: true,
      importedSourceUnchanged: true,
      runs: 0,
      adopted: true,
    });
    console.log(
      "Actual Agent: checked comparison, reference and User adoption passed",
    );
  } catch (error) {
    await page.screenshot({ path: join(out, "failure.png") }).catch(() => {});
    await record("failure", {
      error: String(error),
      document: await document().catch(() => null),
    });
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
    await record("isolation", {
      authorizedCredentialUnchanged: beforeAuth === (await digest()),
      temporaryProfileRemoved: true,
      syntheticProjectOnly: true,
    });
  }
}
void main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
