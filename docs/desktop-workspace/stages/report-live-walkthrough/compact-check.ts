// Recheck stable compact navigation after the live walkthrough; no credentials or Agent calls.
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import {
  launch,
  captureWindow,
} from "../../../../app/desktop/tests/electron/support";
async function main() {
  const stage = resolve(
    "../docs/desktop-workspace/stages/report-live-walkthrough",
  );
  const { profile } = JSON.parse(
    await readFile(stage + "/fixture.json", "utf8"),
  );
  const { application, page } = await launch(profile);
  try {
    await application.evaluate(({ BrowserWindow }) =>
      BrowserWindow.getAllWindows()
        .find((w) => w.webContents.getURL().startsWith("app://gobble/"))!
        .setSize(1000, 760),
    );
    await page.getByRole("button", { name: "Workspace", exact: true }).click();
    await page.getByRole("button", { name: "Changes", exact: true }).click();
    await page
      .getByRole("region", { name: "Pipeline change review" })
      .getByText("25 Phred", { exact: true })
      .waitFor();
    await page.getByRole("button", { name: "Chat", exact: true }).click();
    if (
      (await page
        .getByRole("textbox", { name: "Message draft" })
        .inputValue()) !==
      "Keep this draft while I review the proposed threshold."
    )
      throw Error("Draft changed");
    await page.getByRole("button", { name: "Workspace", exact: true }).click();
    await page.getByRole("button", { name: "Changes", exact: true }).waitFor();
    if (
      (await page
        .getByRole("region", { name: "Pipeline change review" })
        .count()) !== 0
    )
      throw Error("Navigation defect did not reproduce");
    await captureWindow(
      application,
      stage + "/evidence/23-compact-return-stable.png",
    );
    console.log(
      "Reproduced: compact return settles on Current, while the draft and checked proposal persist.",
    );
  } finally {
    await application.close();
  }
}
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
