// Opt-in expert walkthrough; only a fresh synthetic Project/profile is mutated.
import {
  mkdtemp,
  mkdir,
  copyFile,
  chmod,
  writeFile,
  readFile,
  unlink,
} from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { gzipSync } from "node:zlib";
import { createInterface } from "node:readline";
import {
  launch,
  chooseProject,
  captureWindow,
  focusWindow,
} from "../../../../app/desktop/tests/electron/support";

async function main() {
  const output = resolve(
    "../docs/desktop-workspace/stages/report-live-walkthrough",
  );
  const resume = process.argv.includes("--resume");
  const prior = resume
    ? JSON.parse(await readFile(join(output, "fixture.json"), "utf8"))
    : null;
  const base =
    prior?.base ?? (await mkdtemp(join(tmpdir(), "gobble-report-live-")));
  const profile = join(base, "profile"),
    project = join(base, "Report collaboration study");
  if (!resume) await mkdir(project);
  if (!resume)
    await writeFile(
      join(project, "sample.fastq.gz"),
      gzipSync(
        Array.from(
          { length: 100 },
          (_, i) =>
            `@synthetic-${i}\n${"ACGT".repeat(20)}\n+\n${"I".repeat(80)}\n`,
        ).join(""),
      ),
    );
  const credential = join(profile, "codex/home/auth.json");
  await mkdir(join(profile, "codex/home"), { recursive: true, mode: 0o700 });
  await copyFile(
    "/Users/hahyeon/Library/Application Support/Gobble-stage4-review/codex/home/auth.json",
    credential,
  );
  await chmod(credential, 0o600);
  delete process.env.GOBBLE_CODEX_EXECUTABLE;
  let { application, page } = await launch(profile);
  page.setDefaultTimeout(15000);
  await writeFile(
    join(output, "fixture.json"),
    JSON.stringify({ base, profile, project, syntheticReads: 100 }, null, 2),
  );
  const doc = () =>
    page.evaluate(async () => {
      const ps = await window.gobble.projects.list();
      if (!ps.ok || !ps.value[0]) throw Error("Missing Project");
      const r = await window.gobble.workspace.read({
        projectId: ps.value[0].projectId,
      });
      if (!r.ok) throw Error(r.error.message);
      return r.value;
    });
  const capture = async (name: string) => {
    await captureWindow(application, join(output, "evidence", name + ".png"));
    await writeFile(
      join(output, "evidence", name + ".txt"),
      await page.locator("body").innerText(),
    );
  };
  const snapshot = async () =>
    console.log((await page.locator("body").innerText()).slice(-16000));
  const status = async () => {
    const d = await doc();
    await writeFile(
      join(output, "evidence", "workspace.json"),
      JSON.stringify(d, null, 2),
    );
    console.log(
      JSON.stringify({
        submissions: d?.collaboration?.submissions.map((s) => ({
          id: s.requestId,
          state: s.state,
          model: s.model,
        })),
        tail: (await page.locator("body").innerText()).slice(-9000),
      }),
    );
  };
  const send = async (prompt: string) => {
    await focusWindow(application);
    await page.getByRole("textbox", { name: "Message draft" }).fill(prompt);
    await page.getByRole("button", { name: "Send", exact: true }).click();
  };
  if (!resume) await chooseProject(application, page, project);
  if (!resume)
    await page.getByRole("button", { name: "Account", exact: true }).click();
  if (!resume)
    await page.getByRole("button", { name: "Refresh connection" }).click();
  await snapshot();
  console.log("READY_FOR_COMMANDS");
  let queue = Promise.resolve();
  createInterface({ input: process.stdin }).on("line", (line) => {
    queue = queue.then(async () => {
      try {
        if (line === "close") {
          await capture("final");
          await status();
          await application.close();
          await unlink(credential);
          console.log("CLOSED_AND_TEMP_CREDENTIAL_REMOVED");
          process.exit(0);
        }
        await eval("(async()=>{" + line + "})()");
        console.log("COMMAND_DONE");
      } catch (error) {
        console.log("COMMAND_ERROR", String(error));
        await capture("last-error").catch(() => {});
      }
    });
  });
}
main().catch((e) => {
  console.error(String(e));
  process.exitCode = 1;
});
