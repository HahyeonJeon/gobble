(async () => {
  const fs = await import("node:fs/promises");
  const d = await doc();
  await fs.writeFile(
    output + "/live-delivery.json",
    JSON.stringify(d, null, 2),
  );
  await captureWindow(application, output + "/pdf-live-mark.png");
  const primary = page.getByRole("region", {
    name: "Primary pane",
    exact: true,
  });
  await primary.getByRole("button", { name: "Next PDF page" }).click();
  await primary.locator(".surface-view[data-ready=true]").waitFor();
  const base = await user();
  await page
    .locator(".shared-reference-event")
    .getByRole("button", { name: "Show in view", exact: true })
    .click();
  await primary.locator(".observed-reference-panel img").waitFor();
  await captureWindow(application, output + "/pdf-live-show.png");
  const shown = JSON.stringify(base) === JSON.stringify(await user());
  await primary.getByRole("button", { name: "Return to my view" }).click();
  await primary.locator(".surface-view[data-ready=true]").waitFor();
  const result = {
    state: d.collaboration.submissions.at(-1).state,
    model: d.collaboration.submissions.at(-1).model,
    toolset: d.workspace.agents[0].provider.toolsetVersion,
    referenceCount: d.sharedReferences.length,
    questionCount: d.workspace.decisions.filter((x) => x.kind === "question")
      .length,
    arrivalPreservesUserState:
      JSON.stringify(before) ===
      JSON.stringify({
        surfaces: d.workspace.surfaces,
        selections: d.selections,
        chat: d.chat,
      }),
    showPreservesUserState: shown,
    returnPreservesUserState:
      JSON.stringify(base) === JSON.stringify(await user()),
    returnedPage: await primary
      .getByRole("spinbutton", { name: "PDF page number" })
      .inputValue(),
    profile: await application.evaluate(({ app }) => app.getPath("userData")),
    runtime:
      "real pinned Codex 0.153.4 with a byte-forwarding diagnostic wrapper; no scripted provider replies",
  };
  await fs.writeFile(
    output + "/live-review.json",
    JSON.stringify(result, null, 2),
  );
  return result;
})();
