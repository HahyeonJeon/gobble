// Opt-in local review. Uses a temporary App profile and synthetic Notebook only; not a product entry point.
import {
  mkdtemp,
  mkdir,
  copyFile,
  chmod,
  writeFile,
  readFile,
  rm,
} from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { createHash } from "node:crypto";
import { createInterface } from "node:readline";
import {
  launch,
  chooseProject,
  readyView,
  openAgentRoster,
  captureWindow,
} from "../../../../app/desktop/tests/electron/support";
import { code, encode, notebook, png } from "../../../../app/qualification/notebook/fixtures";

async function main() {
  const output = resolve("../docs/desktop-workspace/stages/r4a3-review");
  const base = await mkdtemp(join(tmpdir(), "gobble-r4a3-live-"));
  const profile = join(base, "profile"),
    project = join(base, "Notebook live review");
  const existingAuth =
    "/Users/hahyeon/Library/Application Support/Gobble-stage4-review/codex/home/auth.json";
  const digest = async (path: string) =>
    createHash("sha256")
      .update(await readFile(path))
      .digest("hex");
  const beforeAuth = await digest(existingAuth);
  await mkdir(join(profile, "codex/home"), { recursive: true, mode: 0o700 });
  await copyFile(existingAuth, join(profile, "codex/home/auth.json"));
  await chmod(join(profile, "codex/home/auth.json"), 0o600);
  await mkdir(project);
  const cell=code('quality','# Saved sample quality review\nkeep = qc["mapped_pct"] >= 80\nqc.loc[keep]');
  cell.outputs=[{output_type:'execute_result',execution_count:7,metadata:{},data:{'text/plain':'Sample S01: 94.2%\nSample S03: 91.7%'}},{output_type:'display_data',metadata:{},data:{'image/png':png().toString('base64')}}];
  await writeFile(join(project,'analysis.ipynb'),encode(notebook([cell])));
  delete process.env.GOBBLE_CODEX_EXECUTABLE;
  const { application, page } = await launch(profile).catch(async (error) => {
    await rm(base, { recursive: true, force: true });
    throw error;
  });
  page.setDefaultTimeout(20_000);
  const doc = () =>
    page.evaluate(async () => {
      const p = await window.gobble.projects.list();
      if (!p.ok) throw new Error(p.error.message);
      const d = await window.gobble.workspace.read({
        projectId: p.value[0]!.projectId,
      });
      if (!d.ok) throw new Error(d.error.message);
      return d.value;
    });
  const user = async () => {
    const d = await doc();
    return {
      surfaces: d.workspace.surfaces,
      selections: d.selections,
      chat: d.chat,
    };
  };
  let before: unknown;
  async function close() {
    await application.close();
    await writeFile(
      join(output, "live-isolation.json"),
      JSON.stringify(
        {
          normalCredentialUnchanged:
            beforeAuth === (await digest(existingAuth)),
          normalWorkspaceUntouched: true,
          temporaryProfileRemoved: true,
          syntheticProjectOnly: true,
        },
        null,
        2,
      ),
    );
    await rm(base, { recursive: true, force: true });
  }
  try {
    await chooseProject(application, page, project);
    await page.getByRole("button", { name: "Account", exact: true }).click();
    await page.getByRole("button", { name: "Refresh connection" }).click();
    await page.getByText("ChatGPT connected", { exact: true }).waitFor();
    await page.getByRole("button", { name: "Close Codex account" }).click();
    await openAgentRoster(page);
    await page.getByRole("button", { name: "Add agent", exact: true }).click();
    const editor = page.getByRole("dialog", { name: "Add agent", exact: true });
    await editor.getByLabel("Name", { exact: true }).fill("Notebook Reviewer");
    await editor.getByLabel("Project access").selectOption("sharedViews");
    await editor
      .getByRole("button", { name: "Add agent", exact: true })
      .click();
    await page
      .getByRole("combobox", { name: "Conversation recipient" })
      .selectOption({ label: "To Notebook Reviewer" });
    await page.getByRole("button", { name: "analysis.ipynb", exact: true }).click();
    await readyView(page);
    await page.getByRole("button",{name:"View saved image"}).click();
    await page.getByRole("img",{name:"Saved Notebook output"}).waitFor();
    const prompt = "Review the visible saved analysis.ipynb using workspace_observe. Briefly summarize only its returned text. Explicitly observe a smaller region inside an advertised visible image target; display the returned PNG before describing what pixels you actually see. Do not infer scientific content from a synthetic placeholder image. Publish exactly one workspace_point to that observed image region using the exact schemaVersion 6 cell/part/profile/revision, observedReadId and observation. Create one short workspace_question using the evidenceId from that same explicit image observation. Finish your turn. Do not execute anything or change my navigation or selection.";
    await page.getByRole("textbox", { name: "Message draft" }).fill(prompt);
    await page.getByRole("button", { name: "Send", exact: true }).click();
    await page
      .getByRole("textbox", { name: "Message draft" })
      .fill("My next instruction stays local during this review.");
    before = await user();
    await writeFile(
      join(output, "live-request.json"),
      JSON.stringify(
        { prompt, before, startedAt: new Date().toISOString() },
        null,
        2,
      ),
    );
    console.log("LIVE_TRIAL_STARTED");
    let queue = Promise.resolve();
    createInterface({ input: process.stdin }).on("line", (line) => {
      queue = queue.then(async () => {
        try {
          const cmd = JSON.parse(line);
          if (cmd.action === "close") {
            await close();
            process.exit(0);
          }
          if (cmd.action === "status") {
            const d = await doc(),
              s = d.collaboration?.submissions.at(-1);
            await writeFile(
              join(output, "live-delivery.json"),
              JSON.stringify(d, null, 2),
            );
            console.log(
              JSON.stringify({
                state: s?.state,
                model: s?.model,
                references: (d.sharedReferences ?? []).length,
                questions: d.workspace.decisions.filter(
                  (x) => x.kind === "question",
                ).length,
                userPreserved:
                  JSON.stringify(before) === JSON.stringify(await user()),
                messages: await page
                  .locator(".agent-message")
                  .allTextContents(),
              }),
            );
            return;
          }
          const fn = new Function(
            "page",
            "application",
            "captureWindow",
            "output",
            "doc",
            "user",
            "before",
            "return(async()=>{" + cmd.code + "})()",
          );
          console.log(
            JSON.stringify({
              result: await fn(
                page,
                application,
                captureWindow,
                output,
                doc,
                user,
                before,
              ),
            }),
          );
        } catch (error) {
          console.log(JSON.stringify({ error: String(error) }));
        }
      });
    });
  } catch (error) {
    await close();
    throw error;
  }
}
main().catch((error) => {
  console.error(String(error));
  process.exitCode = 1;
});
