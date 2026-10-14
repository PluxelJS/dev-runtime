import assert from "node:assert/strict";
import { chromium } from "playwright";

const [service, endpoint] = process.argv.slice(2);
const browser =
  service === "lightpanda"
    ? await chromium.connectOverCDP(endpoint)
    : await chromium.connect(endpoint);
try {
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.goto(
    'data:text/html,<title>Browser smoke</title><button onclick="this.textContent=42">Run</button>',
  );
  assert.equal(await page.title(), "Browser smoke");
  await page.locator("button").click();
  assert.equal(await page.locator("button").textContent(), "42");
  if (service === "playwright") assert.ok((await page.screenshot()).length > 0);
  await context.close();
} finally {
  await browser.close();
}
