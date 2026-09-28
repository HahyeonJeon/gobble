const { createRequire } = require("node:module");
const fs = require("node:fs");
const assert = require("node:assert/strict");
const root = require("node:path").resolve(__dirname, "../../../../..");
const { chromium, expect } = createRequire(root + "/app/package.json")(
  "@playwright/test",
);
const output =
  root + "/docs/desktop-workspace/proposals/r2-linked-views/review";
const checks = [],
  errors = [];
let browser;
(async () => {
  browser = await chromium.launch({
    executablePath:
      "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    headless: true,
  });
  async function open(width = 1024, colorScheme = "light", touch = false) {
    const page = await browser.newPage({
      viewport: { width, height: 1100 },
      colorScheme,
      hasTouch: touch,
    });
    page.on("pageerror", (e) => errors.push(e.message));
    await page.addInitScript(() => {
      globalThis.__designControls = {};
      globalThis.Tweak = class {
        constructor({ onChange }) {
          this.change = onChange;
        }
        addSelect(object, key) {
          globalThis.__designControls[key] = {
            object,
            key,
            change: this.change,
          };
        }
        addSlider(object, key) {
          this.addSelect(object, key);
        }
      };
    });
    await page.goto("http://127.0.0.1:8768");
    await page.evaluate(() => {
      document.body.style.padding = "0";
      document.body.style.margin = "0";
    });
    const view = page.frameLocator("iframe"),
      frame = page.frames().find((f) => f.parentFrame());
    await view.locator(".r-point").first().waitFor();
    return { page, view, frame };
  }
  const { page, view, frame } = await open();
  const button = (action) => view.locator('[data-action="' + action + '"]');
  const part = (name) => view.locator('[data-part="' + name + '"]');
  const checkbox = (id) =>
    view.getByRole("checkbox", { name: "Select " + id, exact: true });
  const keys = () =>
    view
      .locator("tr.r-selected")
      .evaluateAll((nodes) => nodes.map((n) => n.dataset.key).sort());
  const selected = async (expected) =>
    assert.deepEqual(await keys(), expected.slice().sort());
  const captures = () =>
    view
      .locator(".r-attachment")
      .evaluateAll((nodes) => nodes.map((n) => JSON.parse(n.dataset.capture)));
  async function point(key) {
    return view.locator('.r-point[data-key="' + key + '"]').boundingBox();
  }
  await checkbox("S03").focus();
  await page.keyboard.press("Space");
  await expect(checkbox("S03")).toBeFocused();
  await checkbox("S05").check();
  await selected(["row_3", "row_5"]);
  assert.equal(await view.locator('.r-point[r="7"]').count(), 2);
  await part("sort").selectOption("pc1");
  await selected(["row_3", "row_5"]);
  await part("filter").selectOption("B");
  await expect(part("summary")).toHaveText("2 selected · 2 not shown in plot");
  await part("filter").selectOption("all");
  await selected(["row_3", "row_5"]);
  checks.push(
    "Keyboard checkbox focus, linked exact membership, sort and hidden-selection filter behavior",
  );
  await button("discuss").click();
  const frozen = (await captures())[0];
  await expect(part("draft")).toBeFocused();
  await expect(part("status")).toHaveText("Selection added. Nothing sent.");
  await checkbox("S07").check();
  await button("axes").click();
  await part("y").selectOption("depth");
  await part("filter").selectOption("A");
  assert.deepEqual((await captures())[0], frozen);
  await button("discuss").click();
  assert.equal((await captures()).length, 2);
  await part("draft").fill("Compare these samples.");
  await button("send").click();
  await selected(["row_3", "row_5", "row_7"]);
  assert.equal(await view.locator(".r-agent-mark").count(), 0);
  await button("reveal").first().click();
  await expect(part("peek")).toBeVisible();
  await expect(part("filter")).toHaveValue("all");
  await expect(button("axes")).toHaveText("PC1 × PC2");
  assert.equal(await view.locator(".r-agent-mark").count(), 1);
  await selected(["row_3", "row_5", "row_7"]);
  await expect(button("pan-mode")).toBeDisabled();
  await button("return").click();
  await expect(part("filter")).toHaveValue("A");
  await expect(button("axes")).toHaveText("PC1 × Reads (millions)");
  checks.push(
    "Multiple immutable draft attachments; explicit Send; separate Agent marks; reference peek and return preserve User view and selection",
  );
  await part("draft").fill("Another reference");
  await button("send").click();
  await button("reveal").first().click();
  await expect(button("axes")).toHaveText("PC1 × PC2");
  await button("return").click();
  await button("reveal").nth(1).click();
  await expect(button("axes")).toHaveText("PC1 × Reads (millions)");
  await button("return").click();
  checks.push(
    "Each historical message retains its own reference and presentation",
  );
  await button("discuss").click();
  const retained = await captures();
  await button("close-plot").click();
  await selected(["row_3", "row_5", "row_7"]);
  await button("close-table").click();
  await expect(part("empty")).toBeVisible();
  assert.deepEqual(await captures(), retained);
  await button("reveal").first().click();
  await selected([]);
  await expect(part("table-view")).toBeVisible();
  checks.push(
    "Close one linked view preserves local selection; close last clears local selection; attachment and message reference survive and reopen views",
  );
  await button("return").click();
  await part("filter").selectOption("all");
  await button("axes").click();
  await part("y").selectOption("pc2");
  const p3 = await point("row_3");
  await page.mouse.click(p3.x + p3.width / 2, p3.y + p3.height / 2);
  await selected(["row_3"]);
  const p2 = await point("row_2");
  await page.mouse.move(p2.x + p2.width / 2, p2.y + p2.height / 2);
  await expect(view.getByRole("tooltip")).toContainText("S02");
  const p5 = await point("row_5");
  await page.mouse.move(p5.x - 10, p5.y - 10);
  await page.mouse.down();
  await page.mouse.move(p5.x + p5.width + 10, p5.y + p5.height + 10, {
    steps: 5,
  });
  await page.mouse.up();
  await selected(["row_5"]);
  const p7 = await point("row_7");
  await page.keyboard.down("Shift");
  await page.mouse.move(p7.x - 10, p7.y - 10);
  await page.mouse.down();
  await page.mouse.move(p7.x + p7.width + 10, p7.y + p7.height + 10, {
    steps: 5,
  });
  await page.mouse.up();
  await page.keyboard.up("Shift");
  await selected(["row_5", "row_7"]);
  await button("zoom").click();
  await selected(["row_5", "row_7"]);
  await button("fit").click();
  await button("pan-mode").click();
  const hit = await view.locator(".r-hit").boundingBox();
  await page.mouse.move(hit.x + 70, hit.y + 70);
  await page.mouse.down();
  await page.mouse.move(hit.x + 110, hit.y + 85, { steps: 5 });
  await page.mouse.up();
  await selected(["row_5", "row_7"]);
  await button("fit").click();
  await button("select-mode").click();
  checks.push(
    "Actual pointer point/box/Shift-add, nearest-point tooltip, zoom and pan preserve exact member keys",
  );
  await frame.evaluate(() => {
    const d = globalThis.__designControls.source;
    d.object[d.key] = "changed";
    d.change();
  });
  await expect(button("send")).toBeDisabled();
  await expect(button("discuss")).toBeDisabled();
  assert.equal(await view.locator(".r-agent-mark").count(), 0);
  await button("reveal").first().click();
  await expect(part("status")).toContainText("earlier source");
  checks.push(
    "Changed source blocks stale draft Send and stale local attachment; historical Agent reference never targets current rows",
  );
  await view
    .locator("#gobble-r2")
    .screenshot({ path: output + "/source-changed.png" });
  await page.close();
  const layouts = [];
  for (const width of [1024, 736, 360])
    for (const theme of ["light", "dark"]) {
      const { page, view, frame } = await open(width, theme, width === 360);
      await view
        .getByRole("checkbox", { name: "Select S03", exact: true })
        .check();
      await view
        .getByRole("checkbox", { name: "Select S05", exact: true })
        .check();
      const metrics = await view.locator("#gobble-r2").evaluate((root) => {
        const box = root.getBoundingClientRect();
        const outside = Array.from(
          root.querySelectorAll("button,select,textarea,table"),
        )
          .filter((el) => el.getClientRects().length)
          .filter((el) => {
            const b = el.getBoundingClientRect();
            return b.left < box.left - 1 || b.right > box.right + 1;
          })
          .map((el) => el.textContent || el.getAttribute("aria-label"));
        const texts = Array.from(
          root.querySelectorAll(
            ".r-plot .tick text,.r-plot .axis-title,.r-point-label",
          ),
        )
          .filter((el) => el.getClientRects().length)
          .map((el) => ({
            text: el.textContent,
            b: el.getBoundingClientRect(),
          }));
        const collisions = [];
        for (let i = 0; i < texts.length; i++)
          for (let j = i + 1; j < texts.length; j++) {
            const a = texts[i].b,
              b = texts[j].b;
            if (
              a.left < b.right + 3 &&
              a.right > b.left - 3 &&
              a.top < b.bottom + 3 &&
              a.bottom > b.top - 3
            )
              collisions.push([texts[i].text, texts[j].text]);
          }
        return {
          width: box.width,
          scrollWidth: root.scrollWidth,
          outside,
          collisions,
          points: root.querySelectorAll(".r-point").length,
        };
      });
      assert.deepEqual(metrics.outside, []);
      assert.deepEqual(metrics.collisions, []);
      assert.ok(metrics.scrollWidth <= metrics.width + 1);
      assert.equal(metrics.points, 7);
      await view
        .locator("#gobble-r2")
        .screenshot({ path: output + "/" + theme + "-" + width + ".png" });
      if (width === 360) {
        await view.locator('[data-action="discuss"]').click();
        await expect(view.locator('[data-part="draft"]')).toBeFocused();
        await view.locator('[data-part="draft"]').fill("Keep my draft");
        await view.locator('[data-action="work"]').click();
        await view.locator('[data-action="chat"]').click();
        await expect(view.locator('[data-part="draft"]')).toHaveValue(
          "Keep my draft",
        );
        await view
          .locator("#gobble-r2")
          .screenshot({ path: output + "/" + theme + "-360-chat.png" });
      }
      layouts.push({ width, theme, ...metrics });
      await page.close();
    }
  checks.push(
    "Light/dark at 1024/736/360: all seven plotted samples, no horizontal overflow or overlapping plot labels; narrow Work/Chat preserves composer",
  );
  assert.deepEqual(errors, []);
  fs.writeFileSync(
    output + "/sketch-checks.json",
    JSON.stringify(
      {
        date: "2026-09-08",
        scope:
          "Synthetic interactive design sketch only; no production app, live model or scientific calculation",
        browser: await browser.version(),
        checks,
        layouts,
        errors,
      },
      null,
      2,
    ) + "\n",
  );
  console.log(
    JSON.stringify({ passed: checks.length, layouts: layouts.length, errors }),
  );
})()
  .catch((e) => {
    console.error(e);
    process.exitCode = 1;
  })
  .finally(async () => {
    await browser?.close();
  });
