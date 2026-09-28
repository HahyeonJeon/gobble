const { chromium } = require("../../../../../app/node_modules/playwright");
const { strict: assert } = require("node:assert");
const fs = require("node:fs/promises");
const path = require("node:path");
const root = __dirname;
const results = [];
(async () => {
  const browser = await chromium.launch({ channel: "chrome", headless: true });
  const page = await browser.newPage({
    viewport: { width: 1280, height: 840 },
  });
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  try {
    await page.goto("http://127.0.0.1:8774/sketch.html");
    await page.screenshot({ path: path.join(root, "concept-a.png") });
    await page.locator('#target [data-member="align:S03"]').click();
    await page.locator("#target [data-open-logs]").click();
    assert.match(
      await page.locator("#lower-title").innerText(),
      /align:S03 · attempt 2/,
    );
    assert.match(
      await page.locator("#log").innerText(),
      /align:S03 alignment failed/,
    );
    await page.locator("#discuss-log").click();
    await page.locator('[data-attachment="0"]').click();
    const logCapture = JSON.parse(
      await page.locator("#details-body").innerText(),
    );
    assert.equal(logCapture.target.kind, "log");
    assert.equal(
      logCapture.target.range.endExclusive,
      Buffer.byteLength(logCapture.target.excerpt),
    );
    await page.locator("#close-details").click();
    results.push(
      "Exact S03 attempt 2 opens its synthetic log; excerpt capture has matching UTF-8 byte bounds.",
    );

    await page.locator("#list").click();
    await page.locator('#dep-list [data-edge="prepare,align"]').focus();
    await page.keyboard.press("Enter");
    await page.locator("#target [data-discuss]").click();
    await page.locator('[data-attachment="1"]').click();
    const captureA = JSON.parse(
      await page.locator("#details-body").innerText(),
    );
    assert.deepEqual(captureA.target, {
      kind: "edge",
      from: "prepare",
      to: "align",
    });
    assert.equal(captureA.version, "A");
    await page.locator("#close-details").click();
    await page.locator("#draft").fill("Please explain this dependency.");
    await page.locator("#agent-demo").click();
    assert.equal(await page.locator("#reference").isVisible(), false);
    assert.match(
      await page.locator("#target").innerText(),
      /Prepare samples → Align reads/,
    );
    assert.equal(
      await page.locator("#draft").inputValue(),
      "Please explain this dependency.",
    );
    results.push(
      "Keyboard dependency list selects the directed pair; Discuss captures A without sending; simulated Agent arrival preserves selection and draft.",
    );

    await page.locator("#show").click();
    assert.equal(await page.locator("#reference").isVisible(), true);
    assert.equal(
      await page
        .locator('#graph [data-edge="align,report"]')
        .getAttribute("aria-pressed"),
      "true",
    );
    assert.equal(await page.locator("#show").isDisabled(), true);
    assert.equal(
      await page
        .locator("#return")
        .evaluate((e) => e === document.activeElement),
      true,
    );
    await page.screenshot({ path: path.join(root, "concept-a-reference.png") });
    await page.locator("#return").click();
    assert.equal(await page.locator("#reference").isVisible(), false);
    assert.equal(await page.locator("#dep-list").isVisible(), true);
    assert.equal(
      await page
        .locator('#dep-list [data-edge="prepare,align"]')
        .evaluate((e) => e === document.activeElement),
      true,
    );
    assert.equal(
      await page.locator("#draft").inputValue(),
      "Please explain this dependency.",
    );
    results.push(
      "Show marks only the Agent pair; Return restores User dependency-list mode, selected pair, keyboard focus and draft.",
    );

    await page.locator("#advance").click();
    await page.locator("#show").click();
    assert.match(
      await page.locator("#notice").innerText(),
      /Earlier observation/,
    );
    assert.equal(await page.locator(".agent-mark:visible").count(), 0);
    await page.locator('[data-attachment="1"]').click();
    assert.deepEqual(
      JSON.parse(await page.locator("#details-body").innerText()),
      captureA,
    );
    await page.locator("#close-details").click();
    assert.match(await page.locator("#log-source").innerText(), /Snapshot A/);
    results.push(
      "Advancing to B refuses an A pointer; immutable A attachment and existing A log remain labelled with their captured source.",
    );

    await page.locator("#unavailable").click();
    assert.equal(await page.locator("#missing").isVisible(), true);
    assert.equal(
      await page.locator("#target [data-discuss]").isDisabled(),
      true,
    );
    await page.locator("#tasks").click();
    await page.locator('#task-list [data-member="align:S01"]').click();
    await page.locator("#target [data-open-logs]").click();
    await page.locator("#stdout").click();
    assert.match(
      await page.locator("#log").innerText(),
      /align:S01\nAttempt 1 completed/,
    );
    assert.doesNotMatch(await page.locator("#log").innerText(), /S03|failed/);
    results.push(
      "Unavailable topology does not invent a selected group; exact Tasks/logs remain usable with the selected instance identity.",
    );

    await page.reload();
    await page.locator("#concept").selectOption("b");
    assert.equal(await page.locator("#lower-list").isVisible(), true);
    await page.screenshot({ path: path.join(root, "concept-b.png") });
    await page.locator('#lower-list [data-member="align:S03"]').first().click();
    await page.locator("#lower-list [data-open-logs]").click();
    assert.equal(await page.locator("#lower-list").isVisible(), false);
    assert.match(
      await page.locator("#alternate-note").innerText(),
      /instance list was replaced/,
    );
    results.push(
      "Alternative B uses the companion pane for instances; opening logs replaces that list, exposing its navigation tradeoff.",
    );

    for (const viewport of [
      { width: 900, height: 650 },
      { width: 640, height: 840 },
    ]) {
      await page.setViewportSize(viewport);
      await page.reload();
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
        true,
      );
      await page.locator('#target [data-member="align:S03"]').click();
      await page.locator("#target [data-open-logs]").click();
      assert.equal(await page.locator("#log").isVisible(), true);
      if (viewport.height < 710 || viewport.width <= 650)
        await page.locator("#run-pane-tab").click();
      await page.locator("#target [data-discuss]").click();
      assert.equal(
        await page.locator("#chat-tab").getAttribute("aria-pressed"),
        "true",
      );
      await page.locator("#draft").fill("Keep this draft.");
      await page.locator("#agent-demo").click();
      await page.locator("#show").click();
      assert.equal(await page.locator("#return").isVisible(), true);
      for (const selector of ["#return", "#target"]) {
        const box = await page.locator(selector).boundingBox();
        assert(
          box.y >= 0 && box.y + box.height <= viewport.height,
          selector + " must fit inside the viewport",
        );
      }
      await page.screenshot({
        path: path.join(root, `compact-${viewport.width}.png`),
        fullPage: true,
      });
      await page.locator("#return").click();
      assert.equal(await page.locator("#draft").isVisible(), true);
      assert.equal(
        await page.locator("#draft").inputValue(),
        "Keep this draft.",
      );
      assert.equal(
        await page.locator("#chat-tab").getAttribute("aria-pressed"),
        "true",
      );
      results.push(
        `${viewport.width}×${viewport.height}: no page horizontal overflow; exact logs, Discuss, Show and Return reachable; compact Chat state and draft restored.`,
      );
    }
    assert.deepEqual(errors, []);
    const report = {
      date: "2026-09-08",
      browser: await browser.version(),
      status: "passed",
      scenarioCount: results.length,
      scenarios: results,
      pageErrors: errors,
      limits:
        "Standalone synthetic HTML sketch in headless Chrome. Not production Electron, a qualified graph library, screen-reader research, native 150% zoom or real Agent/engine integration.",
    };
    await fs.writeFile(
      path.join(root, "sketch-checks.json"),
      JSON.stringify(report, null, 2) + "\n",
    );
    process.stdout.write(JSON.stringify(report, null, 2) + "\n");
  } finally {
    await browser.close();
  }
})().catch((error) => {
  process.stderr.write(String(error.stack) + "\n");
  process.exitCode = 1;
});
