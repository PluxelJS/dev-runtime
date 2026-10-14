import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  workers: 1,
  use: {
    headless: true,
    connectOptions: process.env.PW_TEST_CONNECT_WS_ENDPOINT
      ? {
          wsEndpoint: process.env.PW_TEST_CONNECT_WS_ENDPOINT,
          exposeNetwork: "<loopback>",
        }
      : undefined,
    launchOptions: { executablePath: process.env.CHROMIUM_PATH || undefined },
    viewport: { width: 1440, height: 1000 },
  },
});
