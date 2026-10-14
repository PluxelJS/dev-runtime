import assert from "node:assert/strict";
import { createServer } from "node:http";
import { writeFile } from "node:fs/promises";
import { chromium } from "playwright";

const [service, endpoint] = process.argv.slice(2);
const html =
  '<title>Browser smoke</title><button onclick="this.textContent=42">Run</button>';
const server = createServer((req, res) => {
  res.setHeader("Content-Type", "text/html");
  res.end(html);
});
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
let browser;
try {
  browser =
    service === "lightpanda"
      ? await chromium.connectOverCDP(endpoint)
      : await chromium.connect(endpoint, { exposeNetwork: "<loopback>" });
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.goto(
    service === "lightpanda"
      ? "data:text/html," + encodeURIComponent(html)
      : `http://127.0.0.1:${server.address().port}`,
  );
  assert.equal(await page.title(), "Browser smoke");
  await page.locator("button").click();
  assert.equal(await page.locator("button").textContent(), "42");
  if (service === "playwright") {
    const png = await page.screenshot();
    assert.equal(png.subarray(1, 4).toString(), "PNG");
    if (process.env.BROWSER_SMOKE_SCREENSHOT)
      await writeFile(process.env.BROWSER_SMOKE_SCREENSHOT, png);
  }
  console.log(
    `${service}: page, JavaScript and DOM passed${service === "playwright" ? ", host loopback and PNG screenshot passed" : ""}`,
  );
  await context.close();
} finally {
  await browser?.close();
  await new Promise((resolve) => server.close(resolve));
}
