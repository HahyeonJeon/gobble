import { createRequire } from "node:module";
import { writeFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import assert from "node:assert/strict";
const require = createRequire(process.cwd() + "/app/package.json");
const { chromium } = require("@playwright/test");
const out = fileURLToPath(new URL(".", import.meta.url));
const url = process.argv[2] || "http://127.0.0.1:55517/sketch.html";
const browser = await chromium.launch({ channel: "chrome", headless: true });
const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
const errors = [];
page.on("pageerror", (e) => errors.push(e.message));
const checks = [];
async function check(name, fn) {
  await fn();
  checks.push(name);
}
try {
  await page.goto(url);
  await page.locator('[data-selection="source"]').click();
  await page.locator("#attach").click();
  await check(
    "Attaching captures source and focuses the single composer",
    async () => {
      assert.match(
        await page.locator("#attachment-label").innerText(),
        /Source.*Line 1/,
      );
      assert.equal(
        await page
          .locator("#draft")
          .evaluate((el) => el === document.activeElement),
        true,
      );
      assert.equal(await page.locator("textarea").count(), 1);
    },
  );
  await page.screenshot({ path: out + "reader-1440.png" });
  await page.locator('[data-selection="output"]').click();
  await check(
    "Changing local selection preserves frozen attachment",
    async () => {
      assert.match(
        await page.locator("#selection-label").innerText(),
        /Output 1/,
      );
      assert.match(
        await page.locator("#attachment-label").innerText(),
        /Source.*Line 1/,
      );
    },
  );
  const scrollBefore = await page
    .locator("#reader")
    .evaluate((el) => el.scrollTop);
  const draftBefore = await page.locator("#draft").inputValue();
  await page.locator("#show-reference").click();
  await check("Reference view disables local selection mutation", async () => {
    assert.equal(
      await page.locator('[data-selection="source"]').isDisabled(),
      true,
    );
    assert.equal(await page.locator("#selection-bar").isVisible(), false);
  });
  await page.screenshot({ path: out + "show-reference.png" });
  await page.locator("#return-view").click();
  await check(
    "Show and Return preserve selection, scroll, attachment, draft and focus",
    async () => {
      assert.equal(
        await page.locator("#reader").evaluate((el) => el.scrollTop),
        scrollBefore,
      );
      assert.equal(await page.locator("#draft").inputValue(), draftBefore);
      assert.match(
        await page.locator("#selection-label").innerText(),
        /Output 1/,
      );
      assert.match(
        await page.locator("#attachment-label").innerText(),
        /Source.*Line 1/,
      );
      assert.equal(
        await page
          .locator("#show-reference")
          .evaluate((el) => el === document.activeElement),
        true,
      );
    },
  );
  await page.locator("#source-change").click();
  await page.locator("#show-reference").click();
  await check(
    "Changed source opens captured original instead of retargeting",
    async () => {
      assert.equal(await page.locator("#captured-view").isVisible(), true);
      assert.match(await page.locator("#captured-view").innerText(), />= 80/);
    },
  );
  await page.screenshot({ path: out + "captured-earlier-version.png" });
  await page.keyboard.press("Escape");
  await page.locator("#refresh").click();
  await check(
    "Refresh replaces local source but preserves frozen attachment",
    async () => {
      assert.equal(await page.locator("#threshold").innerText(), "85");
      assert.match(
        await page.locator("#attachment-version").innerText(),
        /version 1/,
      );
      assert.equal(await page.locator("#attach").isDisabled(), true);
    },
  );
  await page.locator("#send").click();
  await check(
    "Demo send keeps exact original selection and version",
    async () => {
      assert.match(
        await page.locator("#sent-messages").innerText(),
        /Captured version 1/,
      );
      assert.match(await page.locator("#sent-messages").innerText(), />= 80/);
      assert.equal(await page.locator("#attachment").isVisible(), false);
    },
  );
  await page.locator("#concept-b").click();
  await page.locator("#connection-preview").click();
  await check(
    "Alternative concept exposes uncertain connection, no executable control",
    async () => {
      assert.equal(await page.locator("#disconnected").isVisible(), true);
      assert.equal(await page.locator("#reader").isVisible(), false);
      assert.equal(
        await page.getByRole("button", { name: /Run cell|Run all/ }).count(),
        0,
      );
    },
  );
  await page.screenshot({ path: out + "connected-alternative.png" });
  await page.locator("#concept-a").click();
  await page.locator("#focus-pane").click();
  await check("Focus and restore split retain the Notebook state", async () => {
    assert.equal(await page.locator(".method-pane").isVisible(), false);
    await page.locator("#focus-pane").click();
    assert.equal(await page.locator(".method-pane").isVisible(), true);
    assert.equal(await page.locator("#threshold").innerText(), "85");
  });
  await page.locator("#cell-jump summary").click();
  await page.locator('[data-jump="1"]').click();
  await check(
    "Cell jump closes and moves keyboard focus to the target",
    async () => {
      assert.equal(
        await page.locator("#cell-jump").evaluate((el) => el.open),
        false,
      );
      assert.equal(
        await page
          .locator("#cell-1 button")
          .evaluate((el) => el === document.activeElement),
        true,
      );
    },
  );
  for (const [width, height] of [
    [1280, 800],
    [900, 650],
    [640, 600],
  ]) {
    await page.setViewportSize({ width, height });
    await page.reload();
    await page.locator('[data-selection="source"]').click();
    await page.locator("#attach").click();
    await check(
      `Layout ${width}x${height}: controls and composer are visible without page overflow`,
      async () => {
        const geometry = await page.evaluate(() => {
          const names = ["attach", "send", "draft"];
          return {
            scrollWidth: document.documentElement.scrollWidth,
            width: innerWidth,
            controls: names.map((id) => {
              const e = document.getElementById(id),
                b = e.getBoundingClientRect();
              return {
                id,
                left: b.left,
                right: b.right,
                top: b.top,
                bottom: b.bottom,
              };
            }),
          };
        });
        assert.ok(
          geometry.scrollWidth <= geometry.width,
          JSON.stringify(geometry),
        );
        for (const b of geometry.controls)
          assert.ok(
            b.left >= 0 && b.right <= width && b.top >= 0 && b.bottom <= height,
            JSON.stringify(b),
          );
      },
    );
    await page.screenshot({ path: out + `reader-${width}.png` });
  }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.reload();
  await page.locator('[data-selection="source"]').focus();
  await page.keyboard.press("Enter");
  await page.locator("#attach").focus();
  await page.keyboard.press("Enter");
  await check(
    "Keyboard-only preset selection and attachment reach composer",
    async () => {
      assert.equal(
        await page
          .locator("#draft")
          .evaluate((el) => el === document.activeElement),
        true,
      );
    },
  );
  assert.deepEqual(errors, []);
  await writeFile(
    out + "sketch-checks.json",
    JSON.stringify(
      {
        date: "2026-09-08",
        browser: browser.version(),
        subject:
          "Review-only deterministic HTML sketch; not Notebook parser, native Electron or Agent proof",
        checks,
        errors,
      },
      null,
      2,
    ) + "\n",
  );
  console.log(JSON.stringify({ passed: checks.length, errors }));
} finally {
  await browser.close();
}
